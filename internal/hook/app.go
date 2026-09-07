package hook

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/kyros-software/claude-code-themes/internal/lockfile"
	"github.com/kyros-software/claude-code-themes/internal/pet"
	"github.com/kyros-software/claude-code-themes/internal/session"
	"github.com/kyros-software/claude-code-themes/internal/setup"
)

// One entry point for every event; the event arrives in the stdin JSON.
//
//	PostToolUse  Bash      -> commit / tests / repro-before-fix / docs
//	PostToolUse  TodoWrite -> plan task / plan before code / single-tool task
//	PostToolUse  Edit&co   -> the door to the oracle closes
//	PostToolUse  (others)  -> the name is logged, and nothing else happens
//	PreCompact             -> compact
//	SessionEnd             -> the shape of the session
//
// The Python this replaces needed a bash front end to avoid paying for an
// interpreter on every tool call. A binary that starts in about two
// milliseconds does not, so the front end is gone and this is the whole hook.

var editTools = map[string]bool{
	"Edit": true, "Write": true, "NotebookEdit": true, "MultiEdit": true,
}

// planMinTasks is what counts as a plan for the oracle.
const planMinTasks = 3

// ImpulsivePeak and FeralPeak are the two context peaks the ember branch is
// paid from. The first sits above "cansada" and below "ahogada": high enough
// that it was not a comfortable session, short of the crash. The second is the
// same gesture one notch harder, and it is what picks `feral` over its two
// siblings at level 3.
//
// Both are PEAKS, not crashes, and that is the whole point. See CloseSession.
const (
	ImpulsivePeak = 85
	FeralPeak     = 95
)

// hookState is the per-session scratch this package owns.
type hookState struct {
	Code int `json:"code"`
	// Edited is "something changed since the last green suite was paid for".
	// A suite that passes without a single edit behind it is not work, it is
	// the same suite again - and this package would rather miss a meal than
	// invent one.
	Edited      int `json:"edited"`
	PlanCounted int `json:"plan_counted"`
	Done        int `json:"done"`
	Red         int `json:"red"`
}

func loadHookState(path string) hookState {
	var h hookState
	if path == "" {
		return h
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return h
	}
	if json.Unmarshal(raw, &h) != nil {
		return hookState{}
	}
	return h
}

func saveHookState(path string, h hookState) {
	if path == "" {
		return
	}
	// Through session.Save's writer, for the reason given there: a bare
	// os.WriteFile follows a symlink planted at the path.
	if raw, err := json.Marshal(h); err == nil {
		session.WriteAtomic(path, raw)
	}
}

// toolsUsed reads the distinct tools seen since the previous closed task, and
// clears the log.
//
// It RENAMES the log aside and reads the renamed copy, rather than reading and
// then deleting. Read-then-delete has a gap: a tool logged after the read and
// before the delete is neither counted nor left behind, it is thrown away
// unread. PostToolUse fires on every tool call, so writers are arriving the
// whole time - measured on the old version, 60 rounds out of 60 lost names,
// 17,508 of 24,000.
//
// The rename is most of the fix: it is one atomic step, so an append that has
// already happened either landed in the file we took or lands in the fresh one
// the next writer creates. What it does NOT cover is a writer that opened the
// file before the rename and writes after it - that append goes into the
// renamed inode and is dropped with it. Measured, that last gap was 41 names in
// 24,000. The lock closes it: logTool holds it while it writes, so no append is
// ever in flight across the rename.
//
// What it costs: sniper counts the distinct tools between two closed tasks. A
// dropped tool made a multi-tool task look like a single-tool one and paid out
// a counter that was not earned.
func toolsUsed(path string) map[string]bool {
	out := map[string]bool{}
	if path == "" {
		return out
	}
	release, _ := lockfile.Take(path)
	taken := path + ".taken"
	err := os.Rename(path, taken)
	release()
	if err != nil {
		return out
	}
	raw, err := os.ReadFile(taken)
	os.Remove(taken)
	if err != nil {
		return out
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if name := strings.TrimSpace(line); name != "" && name != "TodoWrite" {
			out[name] = true
		}
	}
	return out
}

func logTool(path, name string) {
	if path == "" || name == "" {
		return
	}
	// The lock is held across open-write-close so the append cannot be in
	// flight while toolsUsed renames the file out from under it. O_APPEND
	// alone makes each write atomic against other writes; it says nothing
	// about a rename happening between this open and this write.
	release, _ := lockfile.Take(path)
	defer release()

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	f.WriteString(name + "\n")
	f.Close()
}

func gitNumstat(dir string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "git", "--no-optional-locks", "-C", dir,
		"show", "--numstat", "--format=", "HEAD").Output()
	if err != nil {
		return ""
	}
	return string(out)
}

func str(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func obj(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

// Run handles one hook event.
func Run(stdin io.Reader, statePath string, now time.Time) int {
	raw, err := io.ReadAll(stdin)
	if err != nil {
		return 0
	}
	var payload map[string]any
	if json.Unmarshal(raw, &payload) != nil || payload == nil {
		return 0
	}

	// A plugin update moves the plugin - the version is in its path - which
	// leaves the stable runtime link dangling until the next SessionStart, and
	// a dangling link is a statusline that has stopped. One lstat per tool call
	// buys the repair.
	setup.Heal()

	event := str(payload["hook_event_name"])
	sessionID := session.SafeID(str(payload["session_id"]))
	hookPath := session.PathFor(sessionID, "hook")
	toolsPath := session.PathFor(sessionID, "tools")
	tool := str(payload["tool_name"])

	// The sniper needs to know HOW MANY distinct tools were used between two
	// closed tasks, i.e. all of them.
	if tool != "" {
		logTool(toolsPath, tool)
	}

	switch event {
	case "PreCompact":
		pet.Update(statePath, func(s *pet.State) bool {
			pet.DecayHunger(s, now)
			pet.Feed(s, "compact", "", now)
			return true
		})
		return 0
	case "SessionEnd":
		if sessionID != "" {
			CloseSession(session.PathFor(sessionID, ""), statePath, now)
			// Everything this session left in $TMPDIR, including the lock
			// beside the tool log and the statusline's git cache. Sweep would
			// get them in a day; this gets them now.
			for _, p := range []string{hookPath, toolsPath, session.PathFor(sessionID, "git")} {
				os.Remove(p)
				os.Remove(lockfile.Path(p))
			}
		}
		return 0
	}

	h := loadHookState(hookPath)
	before := h

	switch {
	case editTools[tool]:
		// Touching code closes the door to the oracle.
		h.Code = 1
		h.Edited = 1
	case tool == "TodoWrite":
		handleTodos(payload, &h, toolsPath, statePath, now)
	case tool == "Bash":
		handleBash(payload, &h, statePath, now)
	}

	if h != before {
		saveHookState(hookPath, h)
	}
	return 0
}

func handleTodos(payload map[string]any, h *hookState, toolsPath, statePath string, now time.Time) {
	rawTodos, _ := obj(payload["tool_input"])["todos"].([]any)
	var done, pending, total int
	var lastDone string
	for _, item := range rawTodos {
		todo, ok := item.(map[string]any)
		if !ok {
			continue
		}
		total++
		switch str(todo["status"]) {
		case "completed":
			done++
			lastDone = str(todo["content"])
		case "pending":
			pending++
		}
	}

	// toolsUsed empties the log as it reads, so it is called once, out here,
	// rather than inside the update.
	soleTool := false
	if done > h.Done {
		soleTool = len(toolsUsed(toolsPath)) == 1
	}

	pet.Update(statePath, func(s *pet.State) bool {
		touched := false

		// oracle: "5 plans written before touching code". A plan is a TodoWrite
		// with several tasks and none closed; it only counts while nothing is
		// edited yet.
		if pending >= planMinTasks && done == 0 && h.Code == 0 && h.PlanCounted == 0 {
			h.PlanCounted = 1
			s.Bump("plans_before_code", 1)
			touched = true
		}

		// cartographer: the longest plan closed in full.
		if total > 0 && done == total {
			s.RecordMax("longest_plan", total)
			touched = true
		}

		previous := h.Done
		h.Done = done
		if done > previous {
			// sniper: "8 tasks closed with a single tool".
			if soleTool {
				s.Bump("single_tool_tasks", 1)
			}
			pet.DecayHunger(s, now)
			pet.Feed(s, "task", truncate(lastDone, 40), now)
			touched = true
		}

		return touched
	})
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

func handleBash(payload map[string]any, h *hookState, statePath string, now time.Time) {
	input := obj(payload["tool_input"])
	command := str(input["command"])
	response := obj(payload["tool_response"])

	var output strings.Builder
	for _, key := range []string{"stdout", "stderr", "output", "content"} {
		output.WriteString(str(response[key]))
	}
	text := output.String()
	failed, _ := response["is_error"].(bool)

	if !failed && IsCommit(command) && !SaidNothingCommitted(text) {
		cwd := str(payload["cwd"])
		if cwd == "" {
			cwd = str(input["cwd"])
		}
		if cwd == "" {
			cwd = "."
		}
		// jardinero, in the original: the commit is already made, so git is
		// asked instead of guessing from the message. Asked BEFORE the lock is
		// taken: it forks git, which is worth up to three seconds, and no other
		// writer should wait behind that.
		docs := IsDocsCommit(gitNumstat(cwd))
		width, hasWidth := FilesChangedCount(text)

		pet.Update(statePath, func(s *pet.State) bool {
			if hasWidth {
				s.RecordMax("widest_commit", width)
			}
			if docs {
				s.MarkDay("docs", now)
			}
			pet.DecayHunger(s, now)
			pet.Feed(s, "commit", CommitRef(text), now)
			return true
		})
		return
	}

	if !IsTestRunner(command, nil) {
		return
	}

	if TestsAreRed(text, failed) {
		// bloodhound: "repro before the fix". A red is remembered; when a green
		// follows, that is a reproduce -> fix cycle.
		h.Red = 1
		return
	}

	repro := h.Red != 0
	h.Red = 0
	paid := h.Edited != 0
	h.Edited = 0

	pet.Update(statePath, func(s *pet.State) bool {
		if repro {
			s.Bump("repro_before_fix", 1)
		}
		// The red -> green cycle above is credited either way: reproducing a
		// bug is worth recording even when the suite itself does not pay out.
		if !paid {
			return true
		}
		pet.DecayHunger(s, now)
		pet.Feed(s, "tests", "", now)
		return true
	})
}

// CloseSession turns facts only the statusline can see - usage peak, duration,
// repo - into counters, and deletes the scratch file.
func CloseSession(factsPath, statePath string, now time.Time) {
	facts := session.Load(factsPath)
	// A file with no structured facts is a session we know nothing about: no
	// counters, and no unlink either.
	if !facts.Structured {
		return
	}

	// Without a credible t0 the duration is not counted: treating a missing one
	// as epoch 0 gave 56-year sessions, which handed out long_sessions and
	// sessions_4h for free.
	duration := int64(-1)
	if facts.T0 > 1e9 {
		duration = now.Unix() - facts.T0
		if duration < 0 {
			duration = 0
		}
	}

	pet.Update(statePath, func(s *pet.State) bool {
		// Every one of these is named for the context and means it. They used to
		// have to say so, because for a while there was a second peak beside this
		// one - the tightest of context, 5h and 7d - and two of the five read it by
		// mistake: a 5h quota at 96 paid out `feral` to somebody whose window never
		// passed 30. There is one peak again.
		if facts.CtxPeak < 40 {
			s.Bump("sessions_under_40", 1)
		}
		if facts.CtxPeak < 60 {
			s.Bump("ctx_low", 1)
		}
		// The mirror of ctx_low, and the only way the ember branch can be reached
		// at all. Its counter used to come from ONE place - blowing the context -
		// and that is the single meal that TAKES XP away, so the arithmetic was
		// closed: every point of "impulsive" cost 15 XP, and every meal that paid
		// it back fed a rival counter. Somebody who blew the context all day ended
		// up at 800 impulsive and still stuck on level 1 at zero XP, with a third
		// of the tree behind a branch nobody could take.
		//
		// The canvas asks for "tira al límite sin frenar", which is a session
		// spent high up, not a session that crashed. So it is the peak that pays.
		if facts.CtxPeak >= ImpulsivePeak {
			s.Bump("impulsive", 1)
		}
		// ctx_maxed picks `feral` at level 3, and it had the same closed arithmetic
		// `impulsive` had: its only source was the overflow, the one meal that
		// TAKES XP away, so every point of it cost 15 XP while every meal that paid
		// that back fed a rival counter. Its two siblings - short_sessions and
		// long_sessions - cost nothing at all, so the branch could not be climbed.
		//
		// Now the three counters that read the ember branch are three notches of
		// one gesture, and none of them is paid for in XP: 85 says you worked high
		// up, 95 says you did it without easing off, 100 says you hit the wall.
		if facts.CtxPeak >= FeralPeak {
			s.Bump("ctx_maxed", 1)
		}
		if facts.CtxPeak >= 99.999 {
			s.Bump("ctx100_sessions", 1)
		}
		if duration >= 0 && duration < 15*60 {
			s.Bump("short_sessions", 1)
			s.Bump("sessions_15min", 1)
		}
		if duration >= 4*3600 {
			s.Bump("long_sessions", 1)
			s.Bump("sessions_4h", 1)
		} else if duration >= 90*60 {
			s.Bump("long_sessions", 1)
		}

		// "5 days running in the same repo": only the day change counts.
		if facts.Repo != "" {
			stamp := facts.Repo + "|" + pet.Today(now)
			if s.RepoDay != stamp {
				if s.RepoDay == facts.Repo+"|"+pet.Yesterday(now) {
					s.Counters["same_repo_days"]++
				} else {
					s.Counters["same_repo_days"] = 1
				}
				s.RepoDay = stamp
			}
		}

		s.HungerPeak = s.Hunger
		return true
	})
	os.Remove(factsPath)
}
