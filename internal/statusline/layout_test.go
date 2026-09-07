package statusline

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/kyros-software/claude-code-themes/internal/pet"
	"github.com/kyros-software/claude-code-themes/internal/theme"
)

// Packing invariants. Wrapping knocks the prompt box out of square, so no line
// may ever be wider than the width it was given.

func elements(n int) []segment {
	out := make([]segment, 0, n)
	for i := 0; i < n; i++ {
		text := "element-" + string(rune('0'+i))
		out = append(out, seg(i, text).truncatable("", text))
	}
	return out
}

func TestABandNeverExceedsItsWidth(t *testing.T) {
	for width := 4; width < 90; width += 3 {
		if got := theme.Width(assemble(elements(6), width)); got > width {
			t.Errorf("width %d produced %d columns", width, got)
		}
	}
}

func TestTheLowestPriorityGoesFirst(t *testing.T) {
	line := theme.Strip(assemble(elements(4), 30))
	if !strings.Contains(line, "element-0") {
		t.Error("the highest-priority element was dropped")
	}
	if strings.Contains(line, "element-3") {
		t.Error("the lowest-priority element survived")
	}
}

func TestTheLastSurvivorIsTruncatedWithAnEllipsis(t *testing.T) {
	// No paint/plain pair: the uncoloured path strips and cuts, and adds no
	// reset of its own.
	line := assemble([]segment{seg(0, "averylongsingleelement")}, 8)
	if line != "averylo…" {
		t.Errorf("truncation = %q", line)
	}
	if theme.Width(line) != 8 {
		t.Errorf("truncated width = %d", theme.Width(line))
	}
}

func TestAnEmptyBandIsAnEmptyString(t *testing.T) {
	if assemble(nil, 40) != "" {
		t.Error("nil band produced output")
	}
}

func TestATruncatedElementKeepsItsColour(t *testing.T) {
	paint := theme.Fg(theme.Path)
	s := seg(0, paint+"a-very-long-repository-name"+theme.Reset).
		truncatable(paint, "a-very-long-repository-name")
	line := assemble([]segment{s}, 10)
	if !strings.HasPrefix(line, paint) || !strings.HasSuffix(line, theme.Reset) {
		t.Errorf("colour was lost: %q", line)
	}
	if theme.Width(line) != 10 {
		t.Errorf("width = %d", theme.Width(line))
	}
}

// --- the whole render ------------------------------------------------------

var payload = map[string]any{
	"model":     map[string]any{"display_name": "Opus 5 (1M context)"},
	"workspace": map[string]any{"current_dir": ".", "repo": map[string]any{"name": "claude-code-themes"}},
	"effort":    map[string]any{"level": "high"},
	"cost": map[string]any{"total_cost_usd": 12.3456, "total_lines_added": 1200,
		"total_lines_removed": 450, "total_duration_ms": 5400000,
		"total_api_duration_ms": 90000},
	"context_window": map[string]any{"used_percentage": 42.5,
		"context_window_size": 1000000, "total_output_tokens": 900},
	"prompt_cache": map[string]any{"hit_ratio": 0.87},
	"rate_limits": map[string]any{
		"five_hour": map[string]any{"used_percentage": 33},
		"seven_day": map[string]any{"used_percentage": 61}},
	"session_id": "layout-test", "prompt_id": "p1",
}

func render(t *testing.T, doc map[string]any, columns int) []string {
	t.Helper()
	t.Setenv("COLUMNS", itoa(columns))
	t.Setenv("COLORTERM", "truecolor")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("TMPDIR", t.TempDir())
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run(bytes.NewReader(raw), &out); err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}

func TestTheFooterIsFourBandsPlusTheRule(t *testing.T) {
	// The canvas: four bands on the left, four rows of card on the right. Two
	// rows less than the six the pet used to take.
	lines := render(t, payload, 140)
	if len(lines) != 5 { // the rule on top plus the four bands
		t.Fatalf("footer is %d lines, want 5", len(lines))
	}
	last := theme.Strip(lines[len(lines)-1])
	if !strings.Contains(last, "nivel ") {
		t.Errorf("band 4 does not carry the level: %q", last)
	}
}

func TestBandFourCollapsesToTheTradeWhenNarrow(t *testing.T) {
	// "con menos de 100 columnas se cae sola y solo queda el oficio"
	lines := render(t, payload, 90)
	last := theme.Strip(lines[len(lines)-1])
	if strings.Contains(last, "nivel ") {
		t.Errorf("band 4 kept the level below 100 columns: %q", last)
	}
	if strings.TrimSpace(last) == "" {
		t.Error("band 4 vanished entirely")
	}
}

func TestNoRenderedLineIsWiderThanTheTerminal(t *testing.T) {
	for _, columns := range []int{200, 140, 120, 100, 80, 70, 60, 55, 50, 40, 30, 20} {
		for _, line := range render(t, payload, columns) {
			if got := theme.Width(line); got > columns {
				t.Errorf("at %d columns a line came out %d wide: %q",
					columns, got, theme.Strip(line))
			}
		}
	}
}

func TestAHostilePayloadStillPrintsItsRows(t *testing.T) {
	hostile := []map[string]any{
		{},
		{"model": nil},
		{"context_window": map[string]any{"used_percentage": "x"}},
		{"cost": map[string]any{}},
		{"workspace": map[string]any{"repo": "not a map"}},
		{"session_id": "../../etc/passwd"},
		{"rate_limits": map[string]any{"five_hour": map[string]any{"used_percentage": nil}}},
		{"model": map[string]any{"display_name": strings.Repeat("x", 400)}},
	}
	for i, doc := range hostile {
		lines := render(t, doc, 140)
		if len(lines) < 4 {
			t.Errorf("payload %d produced %d lines", i, len(lines))
		}
	}
}

func TestRoundingMatchesPythonsBankersRule(t *testing.T) {
	// round(42.5) is 42, not 43. The number is on screen once a second.
	for _, tc := range []struct {
		in   float64
		want int
	}{{42.5, 42}, {43.5, 44}, {0.5, 0}, {1.5, 2}, {2.5, 2}, {42.6, 43}} {
		if got := round(tc.in); got != tc.want {
			t.Errorf("round(%v) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestContextLabelAndDuration(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	if got := ctxLabel(f(1000000)); got != "1M ctx" {
		t.Errorf("ctxLabel(1e6) = %q", got)
	}
	if got := ctxLabel(f(200000)); got != "200k ctx" {
		t.Errorf("ctxLabel(2e5) = %q", got)
	}
	if ctxLabel(nil) != "" {
		t.Error("a missing size produced a label")
	}
	for _, tc := range []struct {
		ms   float64
		want string
	}{{5400000, "1h 30m"}, {90000, "1m 30s"}, {1000, "1s"}, {20000000, "5h 33m"}} {
		if got := formatDuration(f(tc.ms)); got != tc.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tc.ms, got, tc.want)
		}
	}
}

func TestTheStateIsSaidOnceNotTwice(t *testing.T) {
	// The canvas draws it on the card AND in band 4, but on a real terminal the
	// same word lands on the same footer twice and reads as a bug. Band 4 is
	// the copy that stays: the card's top row went back to the creature's
	// crest, which is what the atlas says names the form.
	lines := render(t, payload, 140)
	footer := theme.Strip(strings.Join(lines, "\n"))

	said := ""
	for _, v := range pet.Vitals {
		if name := pet.Names[v.Label]; name != "" && strings.Contains(footer, name) {
			said = name
			break
		}
	}
	if said == "" {
		t.Fatalf("the footer carries no state at all:\n%s", footer)
	}
	if n := strings.Count(footer, said); n != 1 {
		t.Errorf("%q appears %d times on one footer, want once:\n%s", said, n, footer)
	}

	// And it is the band that carries it, not the card - the card's four rows
	// are all sprite now.
	last := theme.Strip(lines[len(lines)-1])
	row := []rune(last)
	band := strings.TrimSpace(string(row[:len(row)-cardWidth-cardGap]))
	if !strings.Contains(band, said) {
		t.Errorf("band 4 lost the state: %q", band)
	}
	for _, line := range lines {
		r := []rune(theme.Strip(line))
		card := string(r[len(r)-cardWidth:])
		if strings.Contains(card, said) {
			t.Errorf("the card kept a copy of the state: %q", card)
		}
	}
}

func TestAtTheTopTheBandIsTradeAndLevel(t *testing.T) {
	// The mark is worn, so there is no bar left to draw - but the state is in
	// this band now, so it is not as bare as it was.
	band := petBand(Card{
		Form:  "exterminador",
		Level: 5,
		Vital: pet.StateFor(20),
	}, 140)
	line := theme.Strip(assemble(band, 140))
	for _, want := range []string{"exterminador", "nivel 5"} {
		if !strings.Contains(line, want) {
			t.Errorf("band 4 %q is missing %q", line, want)
		}
	}
}

// Band 4 names the mark being filled and draws NO bar for it. The bar used to
// be there - twelve cells, XP in green and the habit in amber - and it came out
// by request: band 1 already carries one, and a second beside the state read as
// the same measurement twice.
func TestBandFourNamesTheMarkWithoutDrawingABar(t *testing.T) {
	band := petBand(Card{
		Form:  "bughunter",
		Level: 5,
		Done:  12,
		Span:  15,
		Mark:  "exterminador",
		Vital: pet.StateFor(20),
	}, 140)
	line := theme.Strip(assemble(band, 140))
	if !strings.Contains(line, "exterminador") {
		t.Errorf("band 4 does not name the mark it is filling: %q", line)
	}
	if strings.ContainsAny(line, "█░") {
		t.Errorf("band 4 still draws a bar: %q", line)
	}
}

// And with XP still opening a level there is no mark to name, so nothing is
// added at all - no bar, no stray word.
func TestBandFourDrawsNothingWhileTheProgressIsXP(t *testing.T) {
	band := petBand(Card{
		Form:  "bughunter",
		Level: 4,
		Done:  433,
		Span:  500,
		State: "fresca ✦",
		Vital: pet.StateFor(20),
	}, 140)
	line := theme.Strip(assemble(band, 140))
	if strings.ContainsAny(line, "█░") {
		t.Errorf("the XP bar came back to band 4: %q", line)
	}
	if strings.TrimSpace(line) != "bughunter nivel 4 │ fresca ✦" {
		t.Errorf("band 4 is %q, want the trade, the level and the state and nothing else", line)
	}
}

func TestBandFourStillCollapsesToTheTradeWhenNarrow(t *testing.T) {
	// The extra element must not survive the floor the canvas sets.
	band := petBand(Card{
		Form:  "bughunter",
		Level: 5,
		Vital: pet.StateFor(20),
	}, 90)
	line := theme.Strip(assemble(band, 90))
	if strings.TrimSpace(line) != "bughunter" {
		t.Errorf("below 100 columns band 4 is %q, want just the trade", line)
	}
}

func TestTheBypassMarkIsMeasuredAndNotAssumed(t *testing.T) {
	// This used to demand a one-cell glyph, because Width() counted runes and a
	// wide one would measure 1 and draw 2. The mark is in fact two cells - it
	// always was - and now that cells are what gets counted, the requirement is
	// no longer "keep it narrow" but "keep the band honest about it".
	if theme.Width(BypassMark) != 2 {
		t.Errorf("the mark measures %d, want 2", theme.Width(BypassMark))
	}
	// The invariant that actually matters: band 1 never claims fewer columns
	// than it draws, mark and all.
	for _, width := range []int{40, 60, 80, 140} {
		band := assemble(engine(&Payload{Model: "Opus 5", Permissions: "bypass"}, nil, nil), width)
		if got := theme.Width(band); got > width {
			t.Errorf("at %d columns the band came out %d wide", width, got)
		}
	}
}

func TestTheBypassBadgeDoesNotSpellTheWord(t *testing.T) {
	// The CLI already writes "bypass permissions on" under the prompt box.
	band := assemble(engine(&Payload{Model: "Opus 5", Permissions: "bypass"}, nil, nil), 140)
	line := theme.Strip(band)
	if strings.Contains(line, "bypass") {
		t.Errorf("band 1 still spells it out: %q", line)
	}
	if !strings.Contains(line, BypassMark) {
		t.Errorf("the mark is missing: %q", line)
	}
}

func TestTheOtherModesKeepTheirName(t *testing.T) {
	for _, mode := range []string{"plan", "auto-edit"} {
		line := theme.Strip(assemble(
			engine(&Payload{Model: "Opus 5", Permissions: mode}, nil, nil), 140))
		if !strings.Contains(line, mode) {
			t.Errorf("%q lost its name: %q", mode, line)
		}
	}
}

// Band 3: the folder and the output style. Both follow the same rule - nothing
// shows unless it says something - which is the whole reason the band had room.

func TestTheDefaultStyleIsNotAName(t *testing.T) {
	// "default" is what the CLI sends when nothing is set. Painting it spends
	// columns to say that nothing is set.
	if got := outputStyle("default"); got != "" {
		t.Errorf("the default survived as %q", got)
	}
	for _, name := range []string{"Criterio", "Explanatory", "Learning"} {
		if got := outputStyle(name); got != name {
			t.Errorf("outputStyle(%q) = %q", name, got)
		}
	}
}

func TestBandThreeCarriesTheStyle(t *testing.T) {
	line := theme.Strip(assemble(
		quota(&Payload{Dirname: "themes", Label: "themes", Style: "Criterio"}), 80))
	if !strings.Contains(line, "criterio") {
		t.Errorf("the style is missing from band 3: %q", line)
	}
	// At the root the folder still says nothing band 2 has not.
	if strings.Contains(line, "themes") {
		t.Errorf("the folder came back at the root: %q", line)
	}
}

func TestBandThreeIsStillEmptyWithNothingToSay(t *testing.T) {
	// The root of a repo with no style set, which is most sessions.
	if line := assemble(quota(&Payload{Dirname: "themes", Label: "themes"}), 80); line != "" {
		t.Errorf("band 3 drew %q with nothing to draw", line)
	}
}

func TestBandThreeReadsWhereThenWho(t *testing.T) {
	line := theme.Strip(assemble(
		quota(&Payload{Dirname: "internal", Label: "themes", Style: "Criterio"}), 80))
	where, who := strings.Index(line, "internal"), strings.Index(line, "criterio")
	if where < 0 || who < 0 {
		t.Fatalf("band 3 lost an element: %q", line)
	}
	if where > who {
		t.Errorf("the style came before the folder: %q", line)
	}
}

func TestTheStyleGoesBeforeTheFolderWhenNarrow(t *testing.T) {
	// The band was the folder's before it was the style's.
	line := theme.Strip(assemble(
		quota(&Payload{Dirname: "internal", Label: "themes", Style: "Criterio"}), 10))
	if strings.Contains(line, "criterio") {
		t.Errorf("the style survived the squeeze: %q", line)
	}
	if !strings.Contains(line, "internal") {
		t.Errorf("the folder was dropped instead: %q", line)
	}
}

func TestTheStyleIsPaintedAsASetting(t *testing.T) {
	// Two bare names side by side are told apart by colour alone, so the folder
	// has to stay grey and the style has to wear Mode, the same purple as
	// `xhigh` and `plan`.
	band := assemble(quota(&Payload{Dirname: "internal", Label: "themes", Style: "Criterio"}), 80)
	if !strings.Contains(band, theme.Fg(theme.Mode)+"criterio") {
		t.Errorf("the style is not painted as a CLI setting: %q", band)
	}
	if !strings.Contains(band, theme.Fg(theme.Dir)+"internal") {
		t.Errorf("the folder lost its grey: %q", band)
	}
}

func TestTheStyleComesOffThePayload(t *testing.T) {
	// Parse now checks the name against disk, so the test brings its own
	// config dir instead of leaning on whatever this machine happens to have.
	t.Setenv("CLAUDE_CONFIG_DIR", writeStyle(t, "criterio.md", "Criterio"))
	for _, tc := range []struct {
		raw  string
		want string
	}{
		{`{"output_style":{"name":"Criterio"}}`, "Criterio"},
		{`{"output_style":{"name":"Nolotengo"}}`, ""},
		{`{"output_style":{"name":"default"}}`, ""},
		{`{}`, ""},
		{`{"output_style":{}}`, ""},
		{`{"output_style":"Criterio"}`, ""},
	} {
		var doc map[string]any
		if err := json.Unmarshal([]byte(tc.raw), &doc); err != nil {
			t.Fatalf("%s: %v", tc.raw, err)
		}
		if got := Parse(doc).Style; got != tc.want {
			t.Errorf("%s gave style %q, want %q", tc.raw, got, tc.want)
		}
	}
}

func TestTheStyleSpeaksInTheFootersVoice(t *testing.T) {
	// The seat is a CLI setting's seat, and every other name in it - `xhigh`,
	// `plan`, the pet's own words - arrives lowercase. The two built-in styles
	// ship capitalised and cannot be renamed, so the band does it.
	for _, name := range []string{"Criterio", "Explanatory", "Learning"} {
		line := theme.Strip(assemble(quota(&Payload{Style: name}), 80))
		if line != strings.ToLower(name) {
			t.Errorf("%q was painted as %q", name, line)
		}
	}
	// The folder keeps its case: it has to match what `ls` says.
	line := theme.Strip(assemble(quota(&Payload{Dirname: "Internal", Label: "themes"}), 80))
	if !strings.Contains(line, "Internal") {
		t.Errorf("the folder was lowercased too: %q", line)
	}
	// And Payload keeps the real name; only the band shrinks it. Parse drops a
	// style it cannot find on disk (see payload.go), so the name has to resolve
	// against a config dir this test owns - not against whatever the machine
	// running it happens to have installed.
	t.Setenv("CLAUDE_CONFIG_DIR", writeStyle(t, "criterio.md", "Criterio"))
	if got := Parse(map[string]any{
		"output_style": map[string]any{"name": "Criterio"},
	}).Style; got != "Criterio" {
		t.Errorf("Parse lowercased the name: %q", got)
	}
}
