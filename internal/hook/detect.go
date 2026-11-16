// Package hook turns what you did into what the pet eats.
//
// "Tests green" is not a fact the CLI exposes: it has to be inferred. The rule
// is to rather miss a meal than invent one.
//
//   - The CLI's own is_error beats any pattern: it is the only hard datum there
//     is, and it stands in for the exit code.
//   - Red patterns are searched ONLY in the last lines, where the summary
//     lives. Searching the whole output made a test called test_login_failed
//     paint a green suite red.
//   - No green pattern is needed: if the command was a runner and it exited
//     well, it counts. That is how runners not on the list get in.
//   - PET_TEST_RUNNERS takes an extra regex for yours.
package hook

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// GitCommit is anchored to the start of the command or behind an operator:
// without this, a `grep -rn "git commit"` counted as a commit.
const GitCommit = `(?:^|[;&|(]|&&|\|\|)\s*(?:\w+=\S+\s+)*(?:sudo\s+)?` +
	`git\b(?:\s+-[^\s]+(?:\s+\S+)?)*\s+commit\b`

// NothingCommitted recognises the empty commit, in English and in Spanish -
// git speaks the locale it was given.
const NothingCommitted = `nothing to commit|nada que (?:hacer|confirmar)|no changes added`

// FilesChanged and CommitHash read git's own summary line.
const (
	FilesChanged = `(\d+)\s+files?\s+changed`
	// "[main a1b2c3] message", and nothing looser. It was any bracket
	// anywhere in the output, and a commit in a chain with something else
	// booked whatever that printed first: "[$companyAdmin->id", the tail of
	// a colour escape. That goes into pet.json and onto the panel, so it is
	// git's own line - at the start of one, a branch name, a hash - or "".
	CommitHash = `(?m)^\[([\w./-]+)(?: \([^)]*\))? [0-9a-f]{4,40}\]`
)

// Runners is the list. Note that \w and \s here are ASCII, where the Python
// this replaces had them Unicode-aware; no runner in the list, and no shell
// variable worth the name, is outside ASCII.
const Runners = `\bpytest\b|\bpy\.test\b|\bunittest\b|\btox\b|\bnox\b|\bbehave\b|` +
	`\bnpm\s+(?:run\s+)?test\b|\byarn\s+test\b|\bpnpm\s+(?:run\s+)?test\b|` +
	`\bbun\s+test\b|\bdeno\s+test\b|\bjest\b|\bvitest\b|\bmocha\b|\bava\b|` +
	`\bplaywright\s+test\b|\bcypress\s+run\b|` +
	`\bgo\s+test\b|\bcargo\s+(?:test|nextest)\b|\bctest\b|\bzig\s+test\b|` +
	`\bphpunit\b|\bartisan\s+test\b|\bpest\b|` +
	`\brspec\b|\bcucumber\b|\brake\s+test\b|\bminitest\b|` +
	`\bmvn\s+(?:test|verify)\b|\bgradle\w*\s+\w*test\b|\bsbt\s+test\b|` +
	`\bdotnet\s+test\b|\bswift\s+test\b|\bflutter\s+test\b|\bmix\s+test\b|` +
	`\b(?:make|just|task)\s+\w*test\w*\b|\bnix\s+flake\s+check\b`

// Red is what a failing summary looks like.
const Red = `\bFAILED\b|\bFAILURES\b|\bfailures?[:=]\s*[1-9]|\berrors?[:=]\s*[1-9]|` +
	`\b[1-9]\d*\s+(?:failed|failing|errors?)\b|\bpanic:|` +
	`\bTests?\s+failed\b|\bFAIL\b`

// Zero is a summary NAMING failures to say there were none, which is how most
// runners announce success. It is blanked out before Red is looked for.
//
// Without this, `\bFAILED\b` and `\bFAILURES\b` - case-insensitive, and with
// no number required - matched the word on its own, so eight of the fifteen
// runners checked read their own green summary as a failure:
//
//	cargo test   test result: ok. 10 passed; 0 failed; 0 ignored
//	rspec        10 examples, 0 failures
//	dotnet test  Passed!  - Failed:     0, Passed:    10
//	maven        Tests run: 10, Failures: 0, Errors: 0, Skipped: 0
//	ctest        100% tests passed, 0 tests failed out of 10
//	swift        Executed 10 tests, with 0 failures (0 unexpected)
//	elixir       10 tests, 0 failures
//	junit        [          0 tests failed         ]
//
// The cost was not only the missed meal. A red is REMEMBERED - it is what
// bloodhound's repro-before-fix reads - so a suite that could never come out
// green left the flag set for ever, and the habit it feeds was unreachable
// along with the whole branch behind it.
//
// Go's regexp is RE2 and has no lookbehind, so the zero cannot be excluded
// inside Red itself. Blanking first is the same thing in two steps.
const Zero = `\b0\s+(?:tests?\s+)?(?:failed|failing|failures?|errors?)\b|` +
	`\b(?:failed|failing|failures?|errors?)\s*[:=]\s*0+\b`

// RedTailLines is how much of the output counts as "the summary".
const RedTailLines = 12

// Everything that can sit in front of the real command without being it. They
// are stripped in a loop, not in one pass: `env RUST_LOG=debug cargo test` puts
// a wrapper before an assignment, and a single ordered pass only catches one
// order.
var prefixes = []*regexp.Regexp{
	regexp.MustCompile(`^(?:\w+=\S+\s+)+`), // FOO=1 BAR=2
	regexp.MustCompile(`^(?:sudo|command|env|time|nice|nohup|exec)\s+`),
	regexp.MustCompile(`^(?:python3?|py)\s+-m\s+`), // python -m pytest
	regexp.MustCompile(`^(?:uv|poetry|pipenv|pdm|hatch|rye)\s+run\s+`),
	regexp.MustCompile(`^bundle\s+exec\s+`),
	regexp.MustCompile(`^npx\s+(?:--?y(?:es)?\s+)?`),
}

var (
	splitRe      = regexp.MustCompile(`\|\||&&|[;&|\n]`)
	gitCommitRe  = regexp.MustCompile(GitCommit)
	nothingRe    = regexp.MustCompile(`(?i)` + NothingCommitted)
	filesRe      = regexp.MustCompile(FilesChanged)
	hashRe       = regexp.MustCompile(CommitHash)
	redRe        = regexp.MustCompile(`(?i)` + Red)
	zeroRe       = regexp.MustCompile(`(?i)` + Zero)
	scriptExtRe  = regexp.MustCompile(`\.(sh|bash|zsh|py|rb|js|ts|pl)$`)
	testWordRe   = regexp.MustCompile(`test|spec`)
	builtinRe    = regexp.MustCompile(`(?i)^(?:` + Runners + `)`)
	extraRunners *regexp.Regexp
	extraOnce    bool
)

// RunnersPattern is the built-in list plus whatever PET_TEST_RUNNERS adds, if
// it compiles.
func RunnersPattern() *regexp.Regexp {
	if !extraOnce {
		extraOnce = true
		if extra := strings.TrimSpace(os.Getenv("PET_TEST_RUNNERS")); extra != "" {
			if re, err := regexp.Compile(`(?i)^(?:` + Runners + `|` + extra + `)`); err == nil {
				extraRunners = re
			}
		}
	}
	if extraRunners != nil {
		return extraRunners
	}
	return builtinRe
}

// ResetRunnersPattern drops the cached PET_TEST_RUNNERS. Tests only.
func ResetRunnersPattern() { extraOnce, extraRunners = false, nil }

// stripPrefixes peels wrappers and assignments until what is left starts with
// the command.
func stripPrefixes(segment string) string {
	for i := 0; i < 8; i++ { // bounded: no runaway on odd input
		shorter := segment
		for _, re := range prefixes {
			if loc := re.FindStringIndex(shorter); loc != nil && loc[0] == 0 {
				shorter = shorter[loc[1]:]
			}
		}
		if shorter == segment {
			return segment
		}
		segment = strings.TrimSpace(shorter)
	}
	return segment
}

// basenameForm is the same segment with argv[0] reduced to its basename, so
// `./gradlew test` and `/usr/local/bin/pytest` reach the patterns that expect a
// bare name.
func basenameForm(segment string) (string, bool) {
	fields := strings.Fields(segment)
	if len(fields) == 0 || !strings.Contains(fields[0], "/") {
		return "", false
	}
	base := filepath.Base(fields[0])
	if base == "" || base == "/" || base == "." {
		return "", false
	}
	rest := strings.TrimPrefix(segment, fields[0])
	return base + rest, true
}

// LooksLikeRunner is the last resort for runners not on the list: the
// executable itself is called something with "test" or "spec" (run-tests.sh,
// bin/spec). Shell's `test` is excluded - it compares files, not a suite.
func LooksLikeRunner(segment string) bool {
	fields := strings.Fields(segment)
	if len(fields) == 0 {
		return false
	}
	base := strings.ToLower(filepath.Base(fields[0]))
	base = scriptExtRe.ReplaceAllString(base, "")
	if base == "test" || base == "[" || base == "[[" {
		return false
	}
	return testWordRe.MatchString(base)
}

// IsTestRunner reports whether the command actually runs a suite. The runner
// has to be IN COMMAND POSITION, not just somewhere in the text: without this,
// `echo '{"cmd":"pytest"}'` counted as a green suite.
func IsTestRunner(command string, pattern *regexp.Regexp) bool {
	if pattern == nil {
		pattern = RunnersPattern()
	}
	for _, raw := range splitRe.Split(command, -1) {
		segment := stripPrefixes(strings.TrimSpace(raw))
		candidates := []string{segment}
		if base, ok := basenameForm(segment); ok {
			candidates = append(candidates, base)
		}
		for _, candidate := range candidates {
			if pattern.MatchString(candidate) || LooksLikeRunner(candidate) {
				return true
			}
		}
	}
	return false
}

// IsCommit reports whether the command made a commit.
func IsCommit(command string) bool {
	return gitCommitRe.MatchString(command) && !strings.Contains(command, "--dry-run")
}

// SaidNothingCommitted reports git's "nothing to commit".
func SaidNothingCommitted(output string) bool { return nothingRe.MatchString(output) }

// FilesChangedCount reads "N files changed" out of git's summary.
func FilesChangedCount(output string) (int, bool) {
	m := filesRe.FindStringSubmatch(output)
	if m == nil {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	return n, err == nil
}

// CommitRef reads the short ref out of "[main a1b2c3] message".
func CommitRef(output string) string {
	if m := hashRe.FindStringSubmatch(output); m != nil {
		return m[1]
	}
	return ""
}

// TestsAreRed decides whether the suite failed.
func TestsAreRed(output string, failed bool) bool {
	if failed {
		return true
	}
	lines := strings.Split(output, "\n")
	// splitlines() drops a trailing empty piece; match that.
	if n := len(lines); n > 0 && lines[n-1] == "" {
		lines = lines[:n-1]
	}
	if len(lines) > RedTailLines {
		lines = lines[len(lines)-RedTailLines:]
	}
	// "0 failures" is not a failure. Blanked BEFORE the search rather than
	// excluded from Red, because RE2 has no lookbehind - and blanked one by
	// one rather than all-or-nothing, so "Failures: 2, Errors: 0" keeps the
	// half that is genuinely red.
	tail := zeroRe.ReplaceAllString(strings.Join(lines, "\n"), " ")
	return redRe.MatchString(tail)
}

// docSuffixes and IsDocsCommit: the commit is already made, so git is asked
// instead of guessing from the message.
var docSuffixes = []string{".md", ".rst", ".txt", ".adoc"}

// IsDocsCommit reports a commit that is mostly docs, or a big net deletion.
func IsDocsCommit(numstat string) bool {
	var added, removed, docs, other int
	for _, line := range strings.Split(numstat, "\n") {
		parts := strings.Split(line, "\t")
		if len(parts) != 3 {
			continue
		}
		if n, err := strconv.Atoi(parts[0]); err == nil {
			added += n
		}
		if n, err := strconv.Atoi(parts[1]); err == nil {
			removed += n
		}
		name := strings.ToLower(parts[2])
		isDoc := strings.Contains(name, "/docs/") || strings.HasPrefix(name, "docs/")
		for _, suffix := range docSuffixes {
			if strings.HasSuffix(name, suffix) {
				isDoc = true
			}
		}
		if isDoc {
			docs++
		} else {
			other++
		}
	}
	return (docs > 0 && docs >= other) || (removed > added*2 && removed > 20)
}
