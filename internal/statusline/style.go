package statusline

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/kyros-software/claude-code-themes/internal/config"
)

// Resolving the output style, which is the difference between the band saying
// something true and the band repeating a claim.
//
// The payload hands over the CONFIGURED name, not the loaded one. In the CLI
// (2.1.259) the two are separate steps and only the first reaches us:
//
//	let d = Tn()?.outputStyle || "default"
//	return e[d] ?? null                      // e = the styles that loaded
//	...
//	output_style: { name: Xe }               // Xe = the config, raw
//
// So a name with a typo in it, or one whose file has been deleted, is reported
// exactly like a working one while the system prompt gets nothing at all. The
// band used to paint that name and was therefore lying whenever the lookup on
// the other side had failed. It looks it up itself now.
//
// What this does NOT catch: a style that resolves but is not loaded IN THIS
// SESSION, because the config changed after it started. There is no cheap trace
// that tells those apart - `/output-style` rewrites the same setting and IS
// applied live, so a timestamp cannot separate "stale" from "just switched".
// Reopening the session is the fix, and the band does not pretend to know.

// builtinStyles ship inside the CLI and resolve with no file behind them. Taken
// from its own map (`w8`), not from the docs.
var builtinStyles = map[string]bool{
	"Proactive":   true,
	"Concise":     true,
	"Explanatory": true,
	"Learning":    true,
}

const frontmatterMax = 4096

// declaredName is the CLI's own rule, transcribed:
//
//	let F = basename(d).replace(/\.md$/, "")
//	let B = (p.name != null ? String(p.name) : undefined) || F
//
// The frontmatter name if there is one, the filename without .md otherwise. No
// slugging and no case folding: the lookup on the other side is a plain object
// key, so "criterio" and "Criterio" are two different styles.
func declaredName(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), ".md")

	f, err := os.Open(path)
	if err != nil {
		return base
	}
	defer f.Close()
	// Only the head: the body is the prompt and nothing here needs it.
	buf := make([]byte, frontmatterMax)
	n, _ := f.Read(buf)
	head := string(buf[:n])

	if !strings.HasPrefix(head, "---\n") {
		return base
	}
	end := strings.Index(head[4:], "\n---")
	if end < 0 {
		return base
	}
	for _, line := range strings.Split(head[4:4+end], "\n") {
		rest, ok := strings.CutPrefix(line, "name:")
		if !ok {
			continue
		}
		name := strings.TrimSpace(rest)
		name = strings.Trim(name, `"'`)
		if name != "" {
			return name
		}
		// An empty `name:` falls through to the filename, the way `|| F` does.
		return base
	}
	return base
}

// dirDeclares reports whether any .md in dir declares this name.
func dirDeclares(dir, name string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		if declaredName(filepath.Join(dir, e.Name())) == name {
			return true
		}
	}
	return false
}

// pluginStyleDirs is the loose half of the search, and loose on purpose.
//
// A plugin's styles live under cache/<marketplace>/<plugin>/<version>/, several
// versions deep, and which of them is live takes the enabled-plugin list and the
// marketplace manifests to work out - the whole plugin loader, once a second.
// This does not do that. It asks the weaker question the band actually needs:
// does ANY installed copy declare this name? A stale version answering yes is a
// false positive, and false positives are the safe direction here - the check
// exists to catch a name that is a lie, and hiding a style that works would be a
// worse bug than showing one that does.
func pluginStyleDirs(configDir string) []string {
	var out []string
	cache := filepath.Join(configDir, "plugins", "cache")
	markets, err := os.ReadDir(cache)
	if err != nil {
		return nil
	}
	for _, m := range markets {
		if !m.IsDir() {
			continue
		}
		plugins, err := os.ReadDir(filepath.Join(cache, m.Name()))
		if err != nil {
			continue
		}
		for _, p := range plugins {
			if !p.IsDir() {
				continue
			}
			versions, err := os.ReadDir(filepath.Join(cache, m.Name(), p.Name()))
			if err != nil {
				continue
			}
			for _, v := range versions {
				if v.IsDir() {
					out = append(out, filepath.Join(cache, m.Name(), p.Name(), v.Name(), "output-styles"))
				}
			}
		}
	}
	return out
}

// styleResolves is the question the band asks before painting a name.
//
// The order is cheapest-first and it short-circuits, which matters because this
// runs once a second: a built-in costs a map lookup, the two ordinary
// directories cost a ReadDir each, and the plugin sweep only happens for a name
// that neither of those knew - the rare branch of a branch that is itself only
// taken when a style is set at all.
func styleResolves(name, configDir, repoRoot string) bool {
	if name == "" {
		return false
	}
	if builtinStyles[name] {
		return true
	}
	dirs := []string{filepath.Join(configDir, "output-styles")}
	if repoRoot != "" {
		dirs = append(dirs, filepath.Join(repoRoot, ".claude", "output-styles"))
	}
	for _, dir := range dirs {
		if dirDeclares(dir, name) {
			return true
		}
	}
	for _, dir := range pluginStyleDirs(configDir) {
		if dirDeclares(dir, name) {
			return true
		}
	}
	return false
}

// configDir is ~/.claude, the same way pet.Path spells it. CLAUDE_CONFIG_DIR
// moves it and the CLI honours that, so this does too.
func configDir() string { return config.Dir() }
