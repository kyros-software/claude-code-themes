package theme

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kyros-software/claude-code-themes/internal/config"
)

// The light palette, for a terminal with a white background.
//
// The dark one cannot just be inverted. Its data colours are pastels that read
// as light on black and vanish on white (#4dd6c1 on white is 1.79:1), and its
// footer is a near-black slab that, in a white terminal, is the one dark thing
// on the screen. So every variable here has a light twin, and the switch swaps
// them all at once, before anything is painted.
//
// Where the twins come from, so nobody picks one by eye:
//
//   - Data colours keep the dark palette's HUES (OKLCH) at one perceptual
//     lightness, L 0.52, and one chroma, 0.135 - 0.155 for Bad and the warm
//     ones, which read as no alarm below that. Every one is >= 4.5:1 on the
//     footer and on the white canvas above it.
//   - The footer is #f3f3f3, with a #e5e5e5 hairline: the base tone Windows
//     paints in light mode, and the background of the light terminal itself,
//     so the footer is the same surface as the conversation above it and not
//     a band of its own - a near-white band under a white terminal was the
//     first thing that looked wrong.
//   - Greys are neutral, like Windows'. The dark ones lean blue.
//
// The terminal palette these are taken from is claude-terminal-theme's
// palette.json, block `light`; claude-code-themes' "Terminal Light" theme is
// generated from the same values.

type twin struct {
	at          *Colour
	dark, light Colour
}

var (
	twins []twin
	light bool
	hooks []func()
)

// Light reports whether the light palette is in use.
func Light() bool { return light }

// Use switches every palette variable to its dark or its light twin, and
// tells the packages that copied colours out of here to copy them again.
func Use(l bool) {
	light = l
	for _, t := range twins {
		if l {
			*t.at = t.light
		} else {
			*t.at = t.dark
		}
	}
	for _, h := range hooks {
		h()
	}
}

// OnChange registers a function that recomputes what was copied out of this
// package - pet.Vitals holds colours by value, the layout a painted separator.
// In a real run the palette is chosen in init, before those copies are made,
// so this only matters to whoever switches it later: the tests.
func OnChange(f func()) { hooks = append(hooks, f) }

func init() {
	for _, p := range []struct {
		at  *Colour
		hex string
	}{
		{&Path, "#037a6c"},
		{&Ident, "#057e3f"},
		{&Link, "#1b6bb3"},
		{&Number, "#826404"},
		{&Mode, "#7952a8"},
		{&Bad, "#b03942"},
		{&Emph, "#1b1b1b"},
		{&Paper, "#ffffff"},
		{&Dim, "#6b6b6b"},
		{&Rule, "#c8c8c8"},     // 1.6:1 on the footer, the dark one's own contrast
		{&Dir, "#555555"},      // one grey more legible than Dim, as in the dark
		{&Quota, "#036bb9"},    // the most chroma sRGB allows at L 0.52, as in the dark one
		{&Empty, "#c3d3e4"},    // 1.38:1 on the footer, the dark one's 1.35
		{&CtxEmpty, "#b6cabb"}, // 1.56:1, the dark one's own

		{&Bg, "#f3f3f3"},
		{&Border, "#e5e5e5"},
		{&Tail, "#8a8a8a"},
		{&Text, "#1b1b1b"},

		// Pale is the faded step of each ramp, Dark its lit eye: in the light
		// the pale end is the one that sinks into the background.
		{&PaleGreen, "#77ad85"},
		{&DarkGreen, "#004a23"},
		{&PaleTeal, "#74aba0"},
		{&DarkTeal, "#00483f"},
		{&PaleBlue, "#82a2c3"},
		{&DarkBlue, "#053e6e"},
		{&PaleIndigo, "#929cc9"},
		{&DarkIndigo, "#2e3576"},
		{&PaleGrey, "#9a9fa5"},
		{&DarkGrey, "#383d47"},
		{&Grey, "#6b6b6b"},
		{&Drowned, "#4b58b7"},
	} {
		twins = append(twins, twin{p.at, *p.at, Hex(p.hex)})
	}
	// Under `go test` nothing is detected: the tests draw in the dark palette
	// unless they ask for the light one, whatever the machine running them has
	// in its settings.json.
	if !testing.Testing() {
		Use(DetectLight())
	}
}

// DetectLight decides the palette for this run. STATUSLINE_THEME=light|dark
// forces it, like the other STATUSLINE_ switches; otherwise it follows the
// theme Claude Code is set to, so `/theme` moves the footer along with
// everything else. Resolved once, like COLORTERM: the theme does not change
// under a running refresh.
func DetectLight() bool {
	switch strings.ToLower(os.Getenv("STATUSLINE_THEME")) {
	case "light":
		return true
	case "dark":
		return false
	}
	return LightTheme(config.Dir())
}

// LightTheme reports whether the theme in dir/settings.json is a light one.
//
// The built-ins say it in their name - light, light-daltonized, light-ansi. A
// custom theme does not: "custom:terminal-light" is a file in dir/themes whose
// `base` is what it is drawn over, and the base is what decides. If that file
// cannot be read, the name is the last clue. No settings, or no theme in them,
// is Claude Code's default, which is dark.
func LightTheme(dir string) bool {
	raw, err := os.ReadFile(filepath.Join(dir, "settings.json"))
	if err != nil {
		return false
	}
	var s struct {
		Theme string `json:"theme"`
	}
	if json.Unmarshal(raw, &s) != nil {
		return false
	}
	slug, custom := strings.CutPrefix(s.Theme, "custom:")
	if !custom {
		return strings.HasPrefix(s.Theme, "light")
	}
	// Only a plain file name: a slug is not allowed to walk out of themes/.
	if slug != "" && slug == filepath.Base(slug) && !strings.HasPrefix(slug, ".") {
		if raw, err := os.ReadFile(filepath.Join(dir, "themes", slug+".json")); err == nil {
			var t struct {
				Base string `json:"base"`
			}
			if json.Unmarshal(raw, &t) == nil && t.Base != "" {
				return strings.HasPrefix(t.Base, "light")
			}
		}
	}
	return strings.Contains(slug, "light")
}
