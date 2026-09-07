package statusline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kyros-software/claude-code-themes/internal/pet"
	"github.com/kyros-software/claude-code-themes/internal/theme"
)

func ptr(v float64) *float64 { return &v }

// wantBar is the exact bar band 1 should have drawn: same width, same trough,
// the fill measuring THIS SESSION'S CONTEXT and coloured off the state that
// same context puts the pet in. Comparing the whole bar rather than hunting for
// an escape is what makes this survive a context of 0, where there is no filled
// cell to find, and the quota numbers, which are painted off the same ladder.
//
// Length and colour have both come from the context, then both from the neck,
// and now both from the context again - never from two different numbers,
// which is the one arrangement that has been tried and failed twice. The bar
// measuring the context while borrowing the neck's colour drew a half-full bar
// at ctx 48 next to the word "espesa", the reading for 67. The bar measuring
// the neck printed "82% 5h" three columns before the band printed "5h 82%"
// again, and left the context - this session's own number - as a bare "7%".
func wantBar(p *Payload) string {
	ctx := pet.ContextLoad(p.ContextPc)
	return theme.Bar(ctx, 100, 16, pet.StateFor(ctx).Colour, theme.Empty)
}

func TestTheBarAndThePetAreColouredOffTheSameLadder(t *testing.T) {
	// The bug: band 1 ran its own three-step scale on its own palette - green
	// under 60, amber under 85, red above - while the pet ran the seven-state
	// comfort curve. Same session, same footer, two colours that agreed about
	// nothing.
	for _, ctx := range []float64{0, 10, 22, 23, 45, 50, 63, 70, 78, 85, 89, 95, 99, 100} {
		for _, quota := range []*float64{nil, ptr(0), ptr(46), ptr(80), ptr(100)} {
			p := &Payload{Model: "M", ContextPc: ptr(ctx), FiveHour: quota}
			if band := assemble(engine(p, nil, nil), 140); !strings.Contains(band, wantBar(p)) {
				t.Errorf("ctx %v with 5h %v: the bar is not the pet's %q",
					ctx, deref(quota), pet.StateFor(ctx).Label)
			}
		}
	}
}

func TestAQuotaAboveTheContextChangesNothing(t *testing.T) {
	// The context low and a quota higher. This drew a green bar beside a
	// turquoise pet once, and then a turquoise bar reading 46 beside a window
	// at 22. Both halves are the window now.
	p := &Payload{Model: "M", ContextPc: ptr(22), FiveHour: ptr(46), SevenDay: ptr(11)}
	if got := pet.StateFor(pet.ContextLoad(p.ContextPc)).Label; got != "fresh" {
		t.Fatalf("a window at 22 gave %q, want fresh", got)
	}
	band := assemble(engine(p, nil, nil), 140)
	if !strings.Contains(band, wantBar(p)) {
		t.Error("the bar is not the pet's colour")
	}
	byNeck := theme.Bar(46, 100, 16, pet.StateFor(46).Colour, theme.Empty)
	if strings.Contains(band, byNeck) {
		t.Error("the bar is still painted off the quota")
	}
	plain := theme.Strip(band)
	// The bar's number is the window, and it needs no tag: there is only one
	// number it could be.
	if !strings.Contains(plain, "22%") {
		t.Errorf("the context percentage went missing: %q", plain)
	}
	if strings.Contains(plain, "22% 5h") || strings.Contains(plain, "46% 5h") {
		t.Errorf("the bar is still naming a neck: %q", plain)
	}
	// And the quota is where it belongs: at the end of the band, as itself.
	if !strings.Contains(plain, "5h 46%") {
		t.Errorf("the quota lost its own reading: %q", plain)
	}
}

func TestAFullWindowReadsAsTheEndOfTheRoad(t *testing.T) {
	// The case the first report was about: a full window with the quotas idle.
	// The old weighted mean put that at 58 and painted the pet "a gusto"
	// turquoise next to a red bar.
	p := &Payload{Model: "M", ContextPc: ptr(100), FiveHour: ptr(20), SevenDay: ptr(10)}

	card := pet.StateFor(pet.ContextLoad(p.ContextPc))
	if card.Label != "k.o." {
		t.Errorf("a full context reads as %q", card.Label)
	}
	band := assemble(engine(p, nil, nil), 140)
	if !strings.Contains(band, wantBar(p)) {
		t.Error("the bar is not the card's")
	}
	if !strings.Contains(band, theme.Fg(card.Colour)) {
		t.Error("the band did not use the card's colour")
	}
}

func TestASqueezedQuotaLeavesTheCreatureAlone(t *testing.T) {
	// An empty window with the 5h nearly spent. The creature is FRESH, and
	// that is the whole change: what the quota measures is the account's day,
	// not how this session feels, and two windows open at once would otherwise
	// report the same mood off a number neither of them earned.
	//
	// What stops that being a lie is the quota's own number, painted off the
	// same ladder: it arrives in drowning indigo, so the thing about to stop
	// you is the loudest colour on the line while the pet is green.
	p := &Payload{Model: "M", ContextPc: ptr(20), FiveHour: ptr(95), SevenDay: ptr(10)}

	if got := pet.StateFor(pet.ContextLoad(p.ContextPc)).Label; got != "fresh" {
		t.Errorf("a window at 20 with the 5h at 95 gave %q, want fresh", got)
	}
	band := assemble(engine(p, nil, nil), 140)
	plain := theme.Strip(band)
	if !strings.Contains(band, wantBar(p)) {
		t.Error("the bar is not the window's")
	}
	if !strings.Contains(plain, "20%") {
		t.Errorf("the context reading went missing: %q", plain)
	}
	if !strings.Contains(plain, "5h 95%") {
		t.Errorf("the quota is not shown: %q", plain)
	}
	if !strings.Contains(band, theme.Fg(pet.StateFor(95).Colour)) {
		t.Error("the quota at 95 is not painted off the state ladder")
	}
}

func TestAnEmptyLeftHalfIsAnchoredAgainstTheTrim(t *testing.T) {
	// Claude Code trims the leading spaces off every line. A row whose left
	// half is empty is then "spaces, then the pet", and the trim drops the
	// lot - that slice of the sprite lands at column 0 and the creature comes
	// apart across the screen. Band 3 is empty at the root of a repo, which is
	// most of the time.
	if got := theme.Width(anchored("")); got != 1 {
		t.Errorf("an empty row anchored to %d cells, want 1", got)
	}
	if !strings.HasPrefix(anchored(""), blankAnchor) {
		t.Error("an empty row was left with nothing for the trim to bite on")
	}
	if !strings.HasPrefix(anchored("   "), blankAnchor) {
		t.Error("a row of spaces is just as empty and needs the same anchor")
	}
	// A row with anything real in it is handed back untouched: the anchor
	// would cost it a column.
	for _, row := range []string{"x", theme.Fg(theme.Dim) + "statusline" + theme.Reset} {
		if anchored(row) != row {
			t.Errorf("a row with content was anchored: %q", anchored(row))
		}
	}
}

func TestEveryRowIsTheSameWidthWithAnEmptyBandThree(t *testing.T) {
	// At the root of a repo band 3 has nothing to say. All four rows still
	// have to end on the same column or the card goes crooked.
	t.Setenv("COLUMNS", "100")
	t.Setenv("STATUSLINE_PET", "1")
	t.Setenv("STATUSLINE_BACKGROUND", "0")
	t.Setenv("STATUSLINE_RULE", "0")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(t.TempDir(), ".claude"))
	t.Setenv("TMPDIR", t.TempDir())

	var out strings.Builder
	payload := `{"model":{"display_name":"M"},` +
		`"workspace":{"current_dir":"/tmp"},` +
		`"context_window":{"used_percentage":22},` +
		`"session_id":"widthrow","prompt_id":"1"}`
	if err := Run(strings.NewReader(payload), &out); err != nil {
		t.Fatal(err)
	}

	var widths []int
	for _, row := range strings.Split(strings.TrimRight(out.String(), "\n"), "\n") {
		widths = append(widths, theme.Width(row))
	}
	if len(widths) != 4 {
		t.Fatalf("got %d rows, want 4: %q", len(widths), out.String())
	}
	for i, w := range widths {
		if w != widths[0] {
			t.Errorf("row %d is %d cells, row 0 is %d", i, w, widths[0])
		}
	}
}

func deref(f *float64) any {
	if f == nil {
		return "none"
	}
	return *f
}

func TestABubbleDroppedForWidthIsNotBooked(t *testing.T) {
	// The bug: band 4 drops the bubble before anything else when it runs
	// short, but the phrase had already been marked as said and the
	// five-minute cooldown already started. The pet talked into the void and
	// then held its tongue for a line nobody read.
	//
	// COLUMNS stays above BubbleMin so the pet does try to speak; the right
	// pad is what leaves band 4 with no room for the bubble.
	for _, c := range []struct {
		pad     string
		onudge  string
		visible bool
	}{{"6", "fits", true}, {"40", "does not fit", false}} {
		home := t.TempDir()
		if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
			t.Fatal(err)
		}
		statePath := filepath.Join(home, ".claude", "pet.json")
		// Hunger past the warning line is an event that fires on any refresh.
		if err := os.WriteFile(statePath, []byte(
			`{"xp":950,"hunger":8,"streak":2,"level_seen":5,`+
				`"counters":{"impulsive":9,"long_sessions":9}}`), 0o600); err != nil {
			t.Fatal(err)
		}

		t.Setenv("HOME", home)
		t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(home, ".claude"))
		t.Setenv("TMPDIR", t.TempDir())
		t.Setenv("COLUMNS", "110")
		t.Setenv("STATUSLINE_RIGHT_PAD", c.pad)
		t.Setenv("STATUSLINE_PET", "1")
		t.Setenv("STATUSLINE_BACKGROUND", "0")
		t.Setenv("STATUSLINE_RULE", "0")

		var out strings.Builder
		if err := Run(strings.NewReader(
			`{"model":{"display_name":"M"},"workspace":{"current_dir":"/tmp"},`+
				`"context_window":{"used_percentage":22},`+
				`"session_id":"bubble`+c.pad+`","prompt_id":"1"}`), &out); err != nil {
			t.Fatal(err)
		}

		shown := strings.Contains(out.String(), "◗")
		if shown != c.visible {
			t.Errorf("pad %s (%s): bubble on screen = %v, want %v",
				c.pad, c.onudge, shown, c.visible)
		}

		raw, err := os.ReadFile(statePath)
		if err != nil {
			t.Fatal(err)
		}
		var doc map[string]any
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatal(err)
		}
		booked := doc["said_at"] != nil && doc["said_at"].(float64) != 0
		if booked != c.visible {
			t.Errorf("pad %s (%s): the line was booked = %v, but it was shown = %v",
				c.pad, c.onudge, booked, c.visible)
		}
	}
}

// The bar and the creature are ONE measurement, and that measurement is this
// session. Two sessions open at once with the same account behind them must be
// able to disagree about how they feel, because they do.

func TestTwoSessionsWithOneAccountCanDisagree(t *testing.T) {
	// The report this came from, with its real numbers: one window at 6% and
	// one at 64%, the 5h quota at 81 for both. They both said "cansada".
	quota, idle := ptr(81.0), ptr(21.0)
	fresh := &Payload{Model: "M", ContextPc: ptr(6), FiveHour: quota, SevenDay: idle}
	busy := &Payload{Model: "M", ContextPc: ptr(64), FiveHour: quota, SevenDay: idle}

	a := pet.StateFor(pet.ContextLoad(fresh.ContextPc)).Label
	b := pet.StateFor(pet.ContextLoad(busy.ContextPc)).Label
	if a != "fresh" {
		t.Errorf("the window at 6%% says %q, want fresh", a)
	}
	if b != "sluggish" {
		t.Errorf("the window at 64%% says %q, want sluggish", b)
	}
	// And the two bands do not look alike either.
	if !strings.Contains(assemble(engine(fresh, nil, nil), 140), wantBar(fresh)) ||
		!strings.Contains(assemble(engine(busy, nil, nil), 140), wantBar(busy)) {
		t.Error("a band is not drawing its own session")
	}
}

func TestTheBandSaysTheQuotaOnceAndOnlyOnce(t *testing.T) {
	// The duplication that gave this away: the bar printed "82% 5h" and three
	// columns later the band printed "5h 82%" again, while the context - 7% -
	// was the only figure on the line with no bar under it.
	p := &Payload{Model: "M", ContextPc: ptr(7), FiveHour: ptr(82), SevenDay: ptr(21),
		CtxSize: ptr(1000000)}
	plain := theme.Strip(assemble(engine(p, nil, nil), 140))

	if strings.Count(plain, "82%") != 1 {
		t.Errorf("the 5h quota is printed %d times: %q", strings.Count(plain, "82%"), plain)
	}
	if !strings.Contains(plain, "5h 82%") {
		t.Errorf("the quota lost its reading: %q", plain)
	}
	// The context has the bar and the old shape back: the percentage, then the
	// window size hanging off it.
	if !strings.Contains(plain, "7% · 1M ctx") {
		t.Errorf("the context did not get its bar and its shape back: %q", plain)
	}
}

func TestWithNoContextFigureThereIsNoBarAtAll(t *testing.T) {
	// An old CLI sends no percentage. A bar claiming zero would be a
	// measurement nobody took - and the quotas cannot stand in for it.
	band := theme.Strip(assemble(engine(&Payload{Model: "M", FiveHour: ptr(90)}, nil, nil), 140))
	if strings.Contains(band, "░") || strings.Contains(band, "█") {
		t.Errorf("a bar was drawn out of nothing: %q", band)
	}
	if !strings.Contains(band, "5h 90%") {
		t.Errorf("the quota went missing with it: %q", band)
	}
}

func TestTheBarAndTheWordNeverDisagree(t *testing.T) {
	// The whole point, swept: whatever number the bar prints, the pet's state
	// is the state for that number, and no quota can move either one.
	for _, ctx := range []float64{0, 20, 48, 63, 64, 78, 88, 100} {
		for _, h5 := range []float64{0, 20, 46, 67, 80, 95, 100} {
			p := &Payload{Model: "M", ContextPc: ptr(ctx), FiveHour: ptr(h5)}
			card := pet.StateFor(pet.ContextLoad(p.ContextPc))
			if card.Label != pet.StateFor(ctx).Label {
				t.Errorf("ctx %v / 5h %v: the quota moved the state to %q",
					ctx, h5, card.Label)
			}
			if !strings.Contains(assemble(engine(p, nil, nil), 140), wantBar(p)) {
				t.Errorf("ctx %v / 5h %v: the bar is not the pet's", ctx, h5)
			}
		}
	}
}
