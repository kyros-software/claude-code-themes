package voice

// What the hook hands over about a step, cut down to names.

import (
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/kyros-software/claude-code-themes/internal/session"
)

// ContextFrom reads the hook's payload and the session's facts.
func ContextFrom(payload map[string]any, facts session.Facts) Context {
	c := Context{Label: facts.Label, Active: facts.Active}
	c.Session, _ = payload["session_id"].(string)
	c.Session = session.SafeID(c.Session)
	if cwd, _ := payload["cwd"].(string); cwd != "" {
		c.Repo = name(filepath.Base(cwd))
	}
	c.Tool, _ = payload["tool_name"].(string)
	input, _ := payload["tool_input"].(map[string]any)
	if file, _ := input["file_path"].(string); file != "" {
		c.Target = name(filepath.Base(file))
	} else if command, _ := input["command"].(string); command != "" {
		c.Target = program(command)
	}
	return c
}

// name is a file or directory name fit to be shown to the model and kept in
// the trail, or "". A name is whatever somebody called a file in a repo that
// was cloned: it is not trusted to be short, or to be only a name.
func name(s string) string {
	if utf8.RuneCountInString(s) > 40 {
		return ""
	}
	for _, r := range s {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune("-_.+@", r)) {
			return ""
		}
	}
	return s
}

// program is the name of what a command line runs, and nothing after it: the
// arguments are where the tokens and the passwords are.
func program(command string) string {
	// `cd somewhere && go test` is a go, not a cd.
	for strings.HasPrefix(strings.TrimSpace(command), "cd ") {
		_, rest, chained := strings.Cut(command, "&&")
		if !chained {
			break
		}
		command = rest
	}
	for _, word := range strings.Fields(command) {
		if strings.Contains(word, "=") {
			continue // FOO=bar in front of the command
		}
		word = filepath.Base(word)
		if len(word) > 24 {
			return ""
		}
		for _, r := range word {
			if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.') {
				return ""
			}
		}
		return word
	}
	return ""
}
