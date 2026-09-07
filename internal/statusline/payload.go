package statusline

import (
	"encoding/json"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/kyros-software/claude-code-themes/internal/theme"
)

// Everything we know about the session, read out of the JSON on stdin.
//
// Two things are NOT in that payload and are dug out here: the permission mode
// (from the tail of the transcript) and the git branch (from git itself).

var permissionModes = map[string]string{
	"bypassPermissions":          "bypass",
	"acceptEdits":                "auto-edit",
	"plan":                       "plan",
	"dangerouslySkipPermissions": "bypass",
}

const transcriptTail = 32768

// Payload is the parsed and derived view of one refresh.
type Payload struct {
	Model     string
	Cwd       string
	Dirname   string
	Effort    string
	Style     string
	Vim       string
	Cost      *float64
	Added     *float64
	Removed   *float64
	Duration  *float64
	APIMs     *float64
	ContextPc *float64
	CtxSize   *float64
	OutTokens *float64
	CacheHit  *float64
	FiveHour  *float64
	SevenDay  *float64
	PromptID  string
	SessionID string

	Permissions string
	Repo        string
	Root        string
	Branch      string
	Dirty       bool
	Label       string
}

func dig(doc map[string]any, keys ...string) any {
	var cur any = doc
	for _, k := range keys {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = m[k]
	}
	return cur
}

// asFloat is the gate every number from the payload passes. A weird value is
// nil, never a panic: a crash here takes the whole statusline down.
func asFloat(v any) *float64 {
	var f float64
	switch n := v.(type) {
	case float64:
		f = n
	case string:
		parsed, err := strconv.ParseFloat(n, 64)
		if err != nil {
			return nil
		}
		f = parsed
	default:
		return nil
	}
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return nil
	}
	return &f
}

// asStr is the only door text from the payload comes through, so the
// flattening lives here: see theme.OneLine for why a newline in a name is a
// broken statusline and not just an ugly one.
func asStr(v any) string {
	if s, ok := v.(string); ok {
		return theme.OneLine(s)
	}
	return ""
}

// defaultStyle is what the CLI puts in the payload when no output style is
// set. It is not the absence of a value: `output_style: {name: Xe}` with
// `Xe = oe?.outputStyle || "default"`, read out of the 2.1.259 binary rather
// than assumed, so the field is ALWAYS there and it is usually this.
const defaultStyle = "default"

// outputStyle keeps the name only when it is worth a band. Painting the field
// raw is what the bash statusline this replaces did, and it spends columns on
// the word "default" to say that nothing is set - which is the one thing band 3
// refuses to do, the same way the folder hides when it only repeats the repo.
func outputStyle(name string) string {
	if name == defaultStyle {
		return ""
	}
	return name
}

var modelSuffix = regexp.MustCompile(`\s*\(.*\)\s*$`)

// ReadStdin parses the refresh payload. An unreadable one is an empty one.
func ReadStdin(r io.Reader) map[string]any {
	raw, err := io.ReadAll(r)
	if err != nil {
		return map[string]any{}
	}
	var doc map[string]any
	if json.Unmarshal(raw, &doc) != nil || doc == nil {
		return map[string]any{}
	}
	return doc
}

var permissionRe = regexp.MustCompile(`"permissionMode"\s*:\s*"([A-Za-z]+)"`)

// permissionMode is not in the payload, but the transcript is. Only the TAIL of
// the file is read: it is a multi-megabyte jsonl and this runs every refresh.
func permissionMode(path string) string {
	if path == "" {
		return ""
	}
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return ""
	}
	offset := info.Size() - transcriptTail
	if offset < 0 {
		offset = 0
	}
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return ""
	}
	tail, err := io.ReadAll(f)
	if err != nil {
		return ""
	}
	matches := permissionRe.FindAllSubmatch(tail, -1)
	if len(matches) == 0 {
		return ""
	}
	return permissionModes[string(matches[len(matches)-1][1])]
}

// repoRoot walks up looking for .git. The Python this replaces asked git for
// the top level, which cost a whole process; a handful of stat calls is free.
func repoRoot(dir string) string {
	for i := 0; i < 64; i++ {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
	return ""
}

// Parse turns the raw payload into everything the bands need.
func Parse(doc map[string]any) *Payload {
	p := &Payload{}

	p.Model = asStr(dig(doc, "model", "display_name"))
	if p.Model == "" {
		p.Model = "?"
	}
	p.Model = modelSuffix.ReplaceAllString(p.Model, "")

	p.Cwd = asStr(dig(doc, "workspace", "current_dir"))
	if p.Cwd == "" {
		p.Cwd = asStr(doc["cwd"])
	}
	if p.Cwd == "" {
		p.Cwd, _ = os.Getwd()
	}
	pretty := p.Cwd
	if home, err := os.UserHomeDir(); err == nil && home != "" && strings.HasPrefix(pretty, home) {
		pretty = "~" + pretty[len(home):]
	}
	parts := strings.Split(strings.TrimRight(pretty, "/"), "/")
	// Only the folder you are in. The full path ate half a band to repeat what
	// the repo in band 2 already says.
	p.Dirname = parts[len(parts)-1]
	if p.Dirname == "" {
		p.Dirname = pretty
	}

	p.Effort = asStr(dig(doc, "effort", "level"))
	p.Style = outputStyle(asStr(dig(doc, "output_style", "name")))
	p.Vim = asStr(dig(doc, "vim", "mode"))
	p.Cost = asFloat(dig(doc, "cost", "total_cost_usd"))
	p.Added = asFloat(dig(doc, "cost", "total_lines_added"))
	p.Removed = asFloat(dig(doc, "cost", "total_lines_removed"))
	p.Duration = asFloat(dig(doc, "cost", "total_duration_ms"))
	p.APIMs = asFloat(dig(doc, "cost", "total_api_duration_ms"))
	p.ContextPc = asFloat(dig(doc, "context_window", "used_percentage"))
	p.CtxSize = asFloat(dig(doc, "context_window", "context_window_size"))
	p.OutTokens = asFloat(dig(doc, "context_window", "total_output_tokens"))
	p.CacheHit = asFloat(dig(doc, "prompt_cache", "hit_ratio"))
	p.FiveHour = asFloat(dig(doc, "rate_limits", "five_hour", "used_percentage"))
	p.SevenDay = asFloat(dig(doc, "rate_limits", "seven_day", "used_percentage"))
	p.PromptID = asStr(doc["prompt_id"])
	p.SessionID = asStr(doc["session_id"])

	p.Permissions = permissionMode(asStr(doc["transcript_path"]))

	// The repo name comes from the payload when there is a remote. Just the
	// name: the owner is always the same one and tells you nothing about where
	// you are, it only spends band.
	if repo, ok := dig(doc, "workspace", "repo").(map[string]any); ok {
		if name := asStr(repo["name"]); name != "" {
			p.Repo = name
		}
	}

	if root := repoRoot(p.Cwd); root != "" {
		p.Root = root
		// The branch is read from .git/HEAD - no fork. The dirty flag is the
		// expensive half and is worked out in Run, where the cache lives.
		// See gitinfo.go.
		p.Branch = branchOf(root)
		if p.Repo == "" {
			trimmed := strings.TrimRight(root, "/")
			bits := strings.Split(trimmed, "/")
			p.Repo = bits[len(bits)-1]
		}
	}

	// The name the payload carries is the CONFIGURED one, and the CLI does not
	// tell us whether it resolved. So the name only survives if we can find the
	// style ourselves - otherwise the band would paint a word for a character
	// that is not loaded. See style.go.
	if p.Style != "" && !styleResolves(p.Style, configDir(), p.Root) {
		p.Style = ""
	}

	p.Label = p.Repo
	if p.Label == "" {
		p.Label = parts[len(parts)-1]
	}
	return p
}
