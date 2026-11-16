package hook

import (
	"testing"

	"github.com/kyros-software/claude-code-themes/internal/voice"
)

func TestWhatKindOfStepAToolCallWas(t *testing.T) {
	bash := func(command string) map[string]any {
		return map[string]any{"tool_input": map[string]any{"command": command}}
	}
	file := func(path string) map[string]any {
		return map[string]any{"tool_input": map[string]any{"file_path": path}}
	}
	for _, tc := range []struct {
		tool    string
		payload map[string]any
		want    string
	}{
		{"Edit", file("/src/main.go"), voice.KindEdit},
		{"Write", file("/repo/README.md"), voice.KindDoc},
		{"Read", file("/src/main.go"), voice.KindRead},
		{"Grep", map[string]any{}, voice.KindRead},
		{"TodoWrite", map[string]any{}, voice.KindPlan},
		{"Bash", bash("go test ./..."), voice.KindTest},
		{"Bash", bash("git commit -m x"), voice.KindCommit},
		{"Bash", bash("sudo apt install jq"), voice.KindSystem},
		{"Bash", bash("cd /srv && npm install left-pad"), voice.KindSystem},
		{"Bash", bash("docker compose up -d"), voice.KindSystem},
		{"Bash", bash("ls -la"), voice.KindOther},
		{"Bash", bash("grep service app.log"), voice.KindOther},
		{"Bash", bash("echo curl"), voice.KindOther},
		{"Bash", bash("npm run build"), voice.KindOther},
		{"mcp__github__get_me", map[string]any{}, voice.KindOther},
	} {
		if got := kindOf(tc.tool, tc.payload); got != tc.want {
			t.Errorf("%s %v is %q, want %q", tc.tool, tc.payload["tool_input"], got, tc.want)
		}
	}
}
