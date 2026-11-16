package hook

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kyros-software/claude-code-themes/internal/voice"
)

// systemRe is a command that works on the machine rather than on the code:
// installing, configuring, running services. What the gremlin is the shape of.
// Matched where a command starts, not anywhere: `grep service app.log` is not
// one.
var systemRe = regexp.MustCompile(`(^|[;&|(])\s*(sudo\s+)?(apt|apt-get|dnf|yum|pacman|brew|snap|` +
	`systemctl|service|docker|docker-compose|kubectl|helm|terraform|ansible|` +
	`chmod|chown|ln|crontab|curl|wget|ssh|scp|rsync|` +
	`(npm|pnpm|yarn|pip|pip3|pipx|uv|cargo|gem|composer)\s+(install|add|i|update|upgrade|remove|uninstall))(\s|$)`)

var readTools = map[string]bool{
	"Read": true, "Grep": true, "Glob": true, "LS": true,
	"WebSearch": true, "WebFetch": true, "Agent": true, "Task": true,
}

var planTools = map[string]bool{
	"TodoWrite": true, "ExitPlanMode": true, "EnterPlanMode": true,
	"TaskCreate": true, "TaskUpdate": true,
}

// kindOf says what kind of step a tool call was, for the session's shape. It
// is decided here, where the command line is, and only the answer leaves: the
// voice package is never handed the command itself.
func kindOf(tool string, payload map[string]any) string {
	input := obj(payload["tool_input"])
	switch {
	case editTools[tool]:
		name := strings.ToLower(filepath.Base(str(input["file_path"])))
		for _, suffix := range docSuffixes {
			if strings.HasSuffix(name, suffix) {
				return voice.KindDoc
			}
		}
		return voice.KindEdit
	case readTools[tool]:
		return voice.KindRead
	case planTools[tool]:
		return voice.KindPlan
	case tool == "Bash":
		command := str(input["command"])
		switch {
		case IsTestRunner(command, RunnersPattern()):
			return voice.KindTest
		case IsCommit(command):
			return voice.KindCommit
		case systemRe.MatchString(command):
			return voice.KindSystem
		}
	}
	return voice.KindOther
}
