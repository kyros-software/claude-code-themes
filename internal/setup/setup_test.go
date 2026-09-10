package setup

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// This package is the only thing in the project that writes a file the user
// owns and did not ask us to own: ~/.claude/settings.json carries their model,
// their permissions and their MCP servers. Everything here is about not losing
// any of that.

// home sets up an isolated ~/.claude and returns its path.
func home(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	config := filepath.Join(dir, ".claude")
	t.Setenv("HOME", dir)
	t.Setenv("CLAUDE_CONFIG_DIR", config)
	if err := os.MkdirAll(config, 0o755); err != nil {
		t.Fatal(err)
	}
	return config
}

// runtimeRoot is a plausible install: a directory with a bin/ holding a binary
// named for this machine, which is what Command looks for before it decides
// between the symlink and the shim.
func runtimeRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	name := "ccpet-" + runtime.GOOS + "-" + runtime.GOARCH
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	for _, f := range []string{name, "ccpet"} {
		if err := os.WriteFile(filepath.Join(bin, f), []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func write(t *testing.T, path string, doc map[string]any) {
	t.Helper()
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, path string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("settings.json is not json any more: %v\n%s", err, raw)
	}
	return doc
}

func TestOnKeepsEverythingElseInSettings(t *testing.T) {
	config := home(t)
	path := filepath.Join(config, "settings.json")
	write(t, path, map[string]any{
		"model":      "opus",
		"theme":      "Terminal",
		"env":        map[string]any{"FOO": "bar"},
		"mcpServers": map[string]any{"github": map[string]any{"command": "gh-mcp"}},
	})

	if err := On(io.Discard, runtimeRoot(t)); err != nil {
		t.Fatal(err)
	}

	doc := read(t, path)
	for _, key := range []string{"model", "theme", "env", "mcpServers"} {
		if _, ok := doc[key]; !ok {
			t.Errorf("%q was lost", key)
		}
	}
	entry, ok := doc["statusLine"].(map[string]any)
	if !ok {
		t.Fatalf("statusLine = %#v", doc["statusLine"])
	}
	if entry["type"] != "command" || entry["command"] == "" {
		t.Errorf("statusLine entry = %#v", entry)
	}
	if entry["refreshInterval"] != float64(1) {
		t.Errorf("refreshInterval = %#v", entry["refreshInterval"])
	}
}

func TestOnBacksTheFileUpBeforeTouchingIt(t *testing.T) {
	config := home(t)
	path := filepath.Join(config, "settings.json")
	write(t, path, map[string]any{"model": "opus"})

	var out strings.Builder
	if err := On(&out, runtimeRoot(t)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "copia de seguridad") {
		t.Errorf("no backup was announced: %q", out.String())
	}

	entries, err := os.ReadDir(config)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "settings.json.bak.") {
			found = true
			if doc := read(t, filepath.Join(config, e.Name())); doc["model"] != "opus" {
				t.Errorf("the backup is not the file we replaced: %#v", doc)
			}
		}
	}
	if !found {
		t.Error("no backup file was written")
	}
}

func TestBrokenSettingsAreLeftExactlyAsTheyWere(t *testing.T) {
	// The one case where doing nothing is the whole job. Half-written json is
	// somebody's config mid-edit, or mid-crash; parsing it as an empty document
	// and saving that on top would delete their model and their MCP servers.
	config := home(t)
	path := filepath.Join(config, "settings.json")
	broken := []byte(`{"model": "opus", "mcpServers": {`)
	if err := os.WriteFile(path, broken, 0o600); err != nil {
		t.Fatal(err)
	}

	for name, run := range map[string]func() error{
		"On":        func() error { return On(io.Discard, runtimeRoot(t)) },
		"Off":       func() error { return Off(io.Discard) },
		"Install":   func() error { return Install(io.Discard, runtimeRoot(t), true) },
		"Uninstall": func() error { return Uninstall(io.Discard) },
	} {
		if err := run(); err == nil {
			t.Errorf("%s went ahead on an unreadable settings.json", name)
		}
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s deleted the file: %v", name, err)
		}
		if string(after) != string(broken) {
			t.Fatalf("%s rewrote it: %q", name, after)
		}
	}
}

func TestOffRemovesOnlyTheStatusLine(t *testing.T) {
	config := home(t)
	path := filepath.Join(config, "settings.json")
	write(t, path, map[string]any{
		"model":      "opus",
		"statusLine": map[string]any{"type": "command", "command": "x"},
	})

	if err := Off(io.Discard); err != nil {
		t.Fatal(err)
	}
	doc := read(t, path)
	if _, ok := doc["statusLine"]; ok {
		t.Error("statusLine survived")
	}
	if doc["model"] != "opus" {
		t.Errorf("model = %#v", doc["model"])
	}
}

func TestOffOnAnAlreadyOffFileWritesNothing(t *testing.T) {
	config := home(t)
	path := filepath.Join(config, "settings.json")
	write(t, path, map[string]any{"model": "opus"})
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	if err := Off(&out); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Error("it rewrote a file it had nothing to change")
	}
	if !strings.Contains(out.String(), "ya estaba apagada") {
		t.Errorf("out = %q", out.String())
	}
}

func TestAMissingSettingsFileIsCreatedNotAnError(t *testing.T) {
	config := home(t)
	path := filepath.Join(config, "settings.json")
	if err := On(io.Discard, runtimeRoot(t)); err != nil {
		t.Fatal(err)
	}
	if _, ok := read(t, path)["statusLine"]; !ok {
		t.Error("statusLine missing")
	}
}

func hookCommands(t *testing.T, doc map[string]any, event string) []string {
	t.Helper()
	var out []string
	hooks, _ := doc["hooks"].(map[string]any)
	groups, _ := hooks[event].([]any)
	for _, g := range groups {
		group, _ := g.(map[string]any)
		entries, _ := group["hooks"].([]any)
		for _, e := range entries {
			entry, _ := e.(map[string]any)
			cmd, _ := entry["command"].(string)
			out = append(out, cmd)
		}
	}
	return out
}

func TestInstallWiresEveryEventItClaimsAndLeavesForeignHooksAlone(t *testing.T) {
	config := home(t)
	path := filepath.Join(config, "settings.json")
	foreign := "echo not-ours"
	write(t, path, map[string]any{
		"hooks": map[string]any{
			"PostToolUse": []any{map[string]any{
				"matcher": "Bash",
				"hooks":   []any{map[string]any{"type": "command", "command": foreign}},
			}},
		},
	})

	if err := Install(io.Discard, runtimeRoot(t), true); err != nil {
		t.Fatal(err)
	}
	doc := read(t, path)
	for _, event := range hookEvents {
		cmds := hookCommands(t, doc, event)
		ours := 0
		for _, c := range cmds {
			if strings.Contains(c, hookMark) {
				ours++
			}
		}
		if ours != 1 {
			t.Errorf("%s has %d of our hooks: %v", event, ours, cmds)
		}
	}
	if got := hookCommands(t, doc, "PostToolUse"); len(got) != 2 || got[0] != foreign {
		t.Errorf("the foreign hook was disturbed: %v", got)
	}
}

func TestInstallingTwiceDoesNotStackTheHooks(t *testing.T) {
	// dropOurHooks runs first for exactly this: a second install used to leave
	// two copies of every hook, and the pet ate twice per tool call.
	config := home(t)
	path := filepath.Join(config, "settings.json")
	root := runtimeRoot(t)

	for i := 0; i < 3; i++ {
		if err := Install(io.Discard, root, true); err != nil {
			t.Fatal(err)
		}
	}
	doc := read(t, path)
	for _, event := range hookEvents {
		if got := hookCommands(t, doc, event); len(got) != 1 {
			t.Errorf("%s ended up with %d hooks: %v", event, len(got), got)
		}
	}
}

func TestInstallWithoutHooksWiresOnlyTheStatusLine(t *testing.T) {
	config := home(t)
	path := filepath.Join(config, "settings.json")
	if err := Install(io.Discard, runtimeRoot(t), false); err != nil {
		t.Fatal(err)
	}
	doc := read(t, path)
	if _, ok := doc["statusLine"]; !ok {
		t.Error("statusLine missing")
	}
	if _, ok := doc["hooks"]; ok {
		t.Errorf("hooks were installed anyway: %#v", doc["hooks"])
	}
}

func TestUninstallLeavesTheThemeAndForeignHooksStanding(t *testing.T) {
	config := home(t)
	path := filepath.Join(config, "settings.json")
	root := runtimeRoot(t)
	write(t, path, map[string]any{
		"theme": "Terminal",
		"model": "opus",
		"hooks": map[string]any{
			"SessionEnd": []any{map[string]any{
				"hooks": []any{map[string]any{"type": "command", "command": "echo mine"}},
			}},
		},
	})
	if err := Install(io.Discard, root, true); err != nil {
		t.Fatal(err)
	}
	if err := Uninstall(io.Discard); err != nil {
		t.Fatal(err)
	}

	doc := read(t, path)
	if _, ok := doc["statusLine"]; ok {
		t.Error("statusLine survived the uninstall")
	}
	if doc["theme"] != "Terminal" || doc["model"] != "opus" {
		t.Errorf("it took the user's keys with it: %#v", doc)
	}
	if got := hookCommands(t, doc, "SessionEnd"); len(got) != 1 || got[0] != "echo mine" {
		t.Errorf("SessionEnd = %v, want only the foreign hook", got)
	}
	// The two events that were ours alone go away entirely rather than being
	// left as empty lists.
	hooks, _ := doc["hooks"].(map[string]any)
	for _, event := range []string{"PostToolUse", "PreCompact"} {
		if _, ok := hooks[event]; ok {
			t.Errorf("%s was left behind empty: %#v", event, hooks[event])
		}
	}
}

func TestUninstallDropsTheHooksKeyWhenNothingIsLeftInIt(t *testing.T) {
	home(t)
	path := SettingsPath()
	if err := Install(io.Discard, runtimeRoot(t), true); err != nil {
		t.Fatal(err)
	}
	if err := Uninstall(io.Discard); err != nil {
		t.Fatal(err)
	}
	if _, ok := read(t, path)["hooks"]; ok {
		t.Error("an empty hooks object was left in settings.json")
	}
}

func TestStatusReportsWhatIsActuallyWired(t *testing.T) {
	home(t)
	var off strings.Builder
	if err := Status(&off); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(off.String(), "apagada") {
		t.Errorf("a fresh config did not read as off: %q", off.String())
	}

	if err := Install(io.Discard, runtimeRoot(t), true); err != nil {
		t.Fatal(err)
	}
	var on strings.Builder
	if err := Status(&on); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"PostToolUse", "PreCompact", "SessionEnd"} {
		if !strings.Contains(on.String(), want) {
			t.Errorf("%s missing from status: %q", want, on.String())
		}
	}
	if strings.Contains(on.String(), "statusline: apagada") {
		t.Errorf("status still says off: %q", on.String())
	}
}

func TestTheCommandIsAStablePathNotAVersionedOne(t *testing.T) {
	// The whole reason the runtime link exists: a plugin lives under a path
	// with its version in it, and settings.json is a plain string nobody
	// re-resolves. A versioned path in there breaks on the next plugin update.
	config := home(t)
	root := filepath.Join(t.TempDir(), "cache", "themes", "1.2.3")
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	name := "ccpet-" + runtime.GOOS + "-" + runtime.GOARCH
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	if err := os.WriteFile(filepath.Join(root, "bin", name), []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}

	cmd := Command(root)
	if strings.Contains(cmd, "1.2.3") {
		t.Errorf("the version got into settings.json: %q", cmd)
	}
	if !strings.Contains(cmd, "ccpet-statusline") {
		t.Errorf("command = %q, want the stable statusline link", cmd)
	}
	if _, err := os.Lstat(filepath.Join(config, "ccpet-statusline")); err != nil {
		t.Errorf("the link was not made: %v", err)
	}
}

func TestTheCommandFallsBackToTheShimWithoutASymlink(t *testing.T) {
	home(t)
	// A root with no binary for this machine: linkRun finds nothing to point
	// at, so there is no symlink and the command has to name the shim.
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := Command(root)
	if !strings.HasSuffix(cmd, "statusline") || !strings.Contains(cmd, "bin/ccpet") {
		t.Errorf("command = %q, want the shim with its argument", cmd)
	}
}

func TestTildeShortensUnderHomeAndLeavesTheRestAlone(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	if got := tilde(filepath.Join(dir, ".claude", "ccpet")); got != "~/.claude/ccpet" {
		t.Errorf("tilde = %q", got)
	}
	if got := tilde("/opt/ccpet"); got != "/opt/ccpet" {
		t.Errorf("tilde = %q", got)
	}
}

func TestLinkNeverEatsTheDirectoryTheScriptInstallerOwns(t *testing.T) {
	// Two installers, one path. scripts/install.sh puts a real directory at
	// ~/.claude/ccpet; the plugin's SessionStart hook must recognise that and
	// step aside rather than replace it with a symlink to itself.
	config := home(t)
	owned := filepath.Join(config, "ccpet")
	if err := os.MkdirAll(filepath.Join(owned, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(owned, "bin", "marker")
	if err := os.WriteFile(marker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, err := Link(runtimeRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Error("it reported a change it should not have made")
	}
	info, err := os.Lstat(owned)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("the real directory was replaced by a symlink")
	}
	if _, err := os.Stat(marker); err != nil {
		t.Errorf("the installed copy was disturbed: %v", err)
	}
}

func TestLinkPointsAtTheRootAndIsIdempotent(t *testing.T) {
	config := home(t)
	root := runtimeRoot(t)

	changed, err := Link(root)
	if err != nil {
		t.Skipf("no symlinks on this filesystem: %v", err)
	}
	if !changed {
		t.Error("the first link reported no change")
	}
	target, err := os.Readlink(filepath.Join(config, "ccpet"))
	if err != nil {
		t.Fatal(err)
	}
	if target != root {
		t.Errorf("link -> %q, want %q", target, root)
	}

	changed, err = Link(root)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Error("linking the same root twice reported a change")
	}
}

func TestLinkRepointsAfterAPluginUpdate(t *testing.T) {
	config := home(t)
	old := runtimeRoot(t)
	if _, err := Link(old); err != nil {
		t.Skipf("no symlinks on this filesystem: %v", err)
	}

	fresh := runtimeRoot(t)
	changed, err := Link(fresh)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("a new root did not move the link")
	}
	target, err := os.Readlink(filepath.Join(config, "ccpet"))
	if err != nil {
		t.Fatal(err)
	}
	if target != fresh {
		t.Errorf("link -> %q, want %q", target, fresh)
	}
}

func TestSaveLeavesNoTempFileBehind(t *testing.T) {
	config := home(t)
	if err := On(io.Discard, runtimeRoot(t)); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(config)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".settings-") {
			t.Errorf("a temp file survived the rename: %s", e.Name())
		}
	}
}

func TestAnEmptySettingsFileReadsAsAnEmptyDocument(t *testing.T) {
	config := home(t)
	path := filepath.Join(config, "settings.json")
	for _, content := range []string{"", "   \n\t ", "null"} {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		doc, err := load(path)
		if err != nil {
			t.Fatalf("load(%q) = %v", content, err)
		}
		if doc == nil {
			t.Fatalf("load(%q) handed back a nil map, which panics on write", content)
		}
	}
}
