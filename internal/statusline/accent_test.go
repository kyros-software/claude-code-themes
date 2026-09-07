package statusline

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kyros-software/claude-code-themes/internal/pet"
	"github.com/kyros-software/claude-code-themes/internal/session"
	"github.com/kyros-software/claude-code-themes/internal/theme"
)

// Band 1 and the creature are one reading in one colour.
//
// The bar was already on the pet's LADDER - same rung, same number - but the
// ladder is one scale shared by all forty-one forms, and since the atlas the
// hue belongs to the branch: a blue cazabugs sat beside a green bar that was
// saying the same thing in a different language. These fix the stronger claim:
// not "the same state" but the same colour, the one the torso is drawn in.

// bodyOf is the colour render.go paints the creature with: its branch's ramp at
// the step the state picks.
func bodyOf(form string, usage float64) theme.Colour {
	return pet.RampOf(form).Body[pet.StateFor(usage).Rank]
}

func TestThePillAndTheBarWearTheCreaturesColour(t *testing.T) {
	// One form per ramp, so all ten hues get a turn, and the seven states on
	// top of them.
	forms := []string{"spark", "pattern", "tidy", "probe", "architect",
		"ember", "marathon", "feral", "phoenix", "chimera"}
	for _, form := range forms {
		for _, usage := range []float64{0, 20, 45, 63, 78, 90, 100} {
			body := bodyOf(form, usage)
			p := &Payload{Model: "Opus 5", ContextPc: ptr(usage)}
			band := assemble(engine(p, nil, &body), 140)

			if !strings.Contains(band, theme.Bgc(body)) {
				t.Errorf("%s at %v: the pill is not the creature's colour", form, usage)
			}
			if !strings.Contains(band, theme.Bar(usage, 100, 16, body, theme.Empty)) {
				t.Errorf("%s at %v: the bar is not the creature's colour", form, usage)
			}
		}
	}
}

// Two different forms in the same state must come out as two different band 1s.
// Under the old rule they were identical, which is exactly what made the colour
// stop carrying any information about WHICH creature it was.
func TestTwoFormsInTheSameStateGetDifferentBands(t *testing.T) {
	blue, amber := bodyOf("probe", 45), bodyOf("marathon", 45)
	if blue == amber {
		t.Fatal("the two ramps chosen for this test are the same colour")
	}
	p := &Payload{Model: "Opus 5", ContextPc: ptr(45)}
	if assemble(engine(p, nil, &blue), 140) == assemble(engine(p, nil, &amber), 140) {
		t.Error("a cazabugs and a maraton draw the same band 1")
	}
}

// With the pet switched off there is no torso to borrow, and band 1 has to
// keep the palette it always had rather than going colourless.
func TestBandOneKeepsItsOldPaletteWithNoPet(t *testing.T) {
	p := &Payload{Model: "Opus 5", ContextPc: ptr(45)}
	band := assemble(engine(p, nil, nil), 140)
	if !strings.Contains(band, theme.Bgc(theme.Ident)) {
		t.Error("without a pet the pill lost its own colour")
	}
	if !strings.Contains(band, wantBar(p)) {
		t.Error("without a pet the bar lost the state ladder")
	}
}

// The colour the card hands over must be the colour the card actually drew
// itself in, or band 1 is matching something nobody can see.
func TestTheCardHandsOverTheColourItDrewWith(t *testing.T) {
	for _, form := range []string{"spark", "bughunter", "marathon", "leviathan"} {
		for _, usage := range []float64{10, 50, 95} {
			body := bodyOf(form, usage)
			rows := pet.DrawCard(form, pet.StateFor(usage), 0, false)
			if !strings.Contains(strings.Join(rows[:], "\n"), theme.Fg(body)) {
				t.Errorf("%s at %v: the sprite is not drawn in the colour the card reports",
					form, usage)
			}
		}
	}
}

// The pill prints dark text on its background. That was safe while the
// background was one fixed green; a ramp runs down to a near-grey k.o., and
// black ink on #39404a is a smudge. Every step of every ramp has to come out
// readable.
func TestThePillStaysLegibleOnEveryStepOfEveryRamp(t *testing.T) {
	// WCAG AA is 4.5 for body text and 3.0 for bold, which the pill is. The
	// seventy cells here actually measure 4.43 at their worst - architect's
	// "espesa" - so the line is drawn just under that: high enough that it is
	// a real guard, low enough that it is not a snapshot of today's palette.
	// A new ramp step that lands below this is one the pill cannot carry.
	const minContrast = 4.0
	for name, ramp := range pet.Ramps {
		for step, body := range ramp.Body {
			ink := theme.On(body)
			fg := theme.Emph
			if ink == theme.Black {
				fg = theme.Black8
			}
			if got := theme.ContrastRatio(fg, body); got < minContrast {
				t.Errorf("%s step %d (#%02x%02x%02x): contrast %.2f, unreadable",
					name, step, body.R, body.G, body.B, got)
			}
		}
	}
}

// --- the rung, on the way to disk -------------------------------------------

// xpFor is the XP that opens a level, read off pet.Levels rather than written
// down: these tests only care that the pet is AT level 5 or 6, not what that
// costs this week.
func xpFor(level int) int {
	best := 0
	for _, l := range pet.Levels {
		if l.Level <= level && l.XP > best {
			best = l.XP
		}
	}
	return best
}

// RenderCard persists nothing, so a RememberForm call in there would have
// mutated a state nobody writes - which is how the first version of this went
// in, silently doing nothing. RememberShape owns the write, and these pin both
// halves: that it lands, and that it does not land once a second.

func TestTheRungIsWrittenToTheFile(t *testing.T) {
	path := t.TempDir() + "/pet.json"
	s := pet.New()
	s.XP = xpFor(6)
	s.Counters = map[string]int{"inquisitive": 20, "tests": 20, "test_streak": pet.TitleAsks["wasp"]}
	pet.Save(s, path)

	form, _ := pet.CurrentForm(pet.Load(path))
	if form != "wasp" {
		t.Fatalf("el estado de partida es %s, se esperaba wasp", form)
	}
	RememberShape(Card{id: form}, pet.Load(path), path)
	if got := pet.Load(path).FormSeen; got != "wasp" {
		t.Fatalf("form_seen quedo en %q", got)
	}

	// The streak breaks and the shape holds its rung.
	back := pet.Load(path)
	back.Counters["test_streak"] = 0
	pet.Save(back, path)
	if got, _ := pet.CurrentForm(pet.Load(path)); got != "wasp" {
		t.Errorf("tras romperse la racha sale %s, se esperaba wasp", got)
	}
}

// The statusline refreshes about once a second. Restating a field that did not
// change would be a write per second, on a file the hooks are also writing.
func TestTheRungIsNotRewrittenOnEveryRefresh(t *testing.T) {
	path := t.TempDir() + "/pet.json"
	s := pet.New()
	s.XP = xpFor(5)
	s.Counters = map[string]int{"inquisitive": 20, "tests": 20, "test_streak": 20}
	pet.Save(s, path)

	form, _ := pet.CurrentForm(pet.Load(path))
	RememberShape(Card{id: form}, pet.Load(path), path)

	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		RememberShape(Card{id: form}, pet.Load(path), path)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	// Save writes a temp file and renames it, so a rewrite is a new inode and a
	// new mtime even when the bytes are identical.
	if !after.ModTime().Equal(before.ModTime()) {
		t.Error("veinte refrescos sin cambio de forma reescribieron pet.json")
	}
}

// A lateral move on the same rung DOES have to be written: the floor hands back
// whatever name it stored, so a stale sibling there would resurrect the wrong
// shape the next time a streak fell.
func TestALateralMoveIsWrittenTo(t *testing.T) {
	path := t.TempDir() + "/pet.json"
	s := pet.New()
	s.XP = xpFor(5)
	s.Counters = map[string]int{"inquisitive": 20, "tests": 20, "test_streak": 20}
	pet.Save(s, path)
	if got := pet.Load(path).FormSeen; got != "exterminator" {
		t.Fatalf("el punto de partida es %q", got)
	}

	// The streak goes and the other habit is what is left: a real sideways
	// move, counters and all. A card that disagreed with the counters would be
	// a state no run can produce.
	moved := pet.Load(path)
	moved.Counters["test_streak"] = 0
	moved.Counters["repro_before_fix"] = 10
	pet.Save(moved, path)

	on := pet.Load(path)
	form, _ := pet.CurrentForm(on)
	if form != "bloodhound" {
		t.Fatalf("tras el lateral la forma es %s", form)
	}
	RememberShape(Card{id: form}, on, path)
	if got := pet.Load(path).FormSeen; got != "bloodhound" {
		t.Errorf("el movimiento lateral no se guardo: form_seen = %q", got)
	}
}

// A walk that has fallen must never lower the rung already on disk. This is
// the end-to-end shape of the bug: with the statusline switched off, a pet
// stood at rung 6 for weeks with form_seen never written, and then one blown
// context dropped it to rung 3 and the next save recorded the FALL as the
// truth - so the floor came into being pointing at the wrong rung and the
// avispa was gone for good.
//
// Note what is not being tested here any more: a Card claiming a rung the
// counters do not support. Save derives the rung from the state now, so a card
// cannot assert one, which is the point of having one writer.
func TestAFallenWalkNeverLowersTheStoredRung(t *testing.T) {
	path := t.TempDir() + "/pet.json"
	s := pet.New()
	s.XP = xpFor(6)
	s.Counters = map[string]int{"inquisitive": 20, "tests": 20, "test_streak": pet.TitleAsks["wasp"]}
	pet.Save(s, path)
	if got := pet.Load(path).FormSeen; got != "wasp" {
		t.Fatalf("el punto de partida es %q, se esperaba wasp", got)
	}

	// The context blows: the clean streak goes to zero and the walk collapses
	// to the trade.
	fallen := pet.Load(path)
	fallen.Counters["test_streak"] = 0
	pet.Save(fallen, path)

	if got := pet.Load(path).FormSeen; got != "wasp" {
		t.Errorf("la caida se grabo como verdad: form_seen = %q", got)
	}
	if got, _ := pet.CurrentForm(pet.Load(path)); got != "wasp" {
		t.Errorf("tras la caida la mascota es %s", got)
	}
}

// --- the bracket: where the pet is heading ----------------------------------

// The bracket names the mark the pet IS WEARING, with the trade it is a
// variant of outside it. `cazabugs[sabueso]` is a pet that is a sabueso, and a
// sabueso is one of the two shapes a cazabugs can take.
//
// It held the opposite once - the mark being headed FOR - which printed
// `cazabugs[sabueso]` on a level 4 pet that was not a sabueso and might never
// be. Two bright words with an all-but-invisible bracket between them read as
// one compound name, and the tense the bracket was carrying reached nobody.
func TestTheBracketNamesTheMarkThePetIsWearing(t *testing.T) {
	path := t.TempDir() + "/pet.json"
	s := pet.New()
	s.XP = xpFor(5)
	s.Counters = map[string]int{
		"inquisitive": 20, "tests": 20, // spark -> probe -> bughunter
		"repro_before_fix": pet.Unlocks["bloodhound"].Threshold,
	}
	pet.Save(s, path)

	card, _ := RenderCard(&Payload{Model: "M", ContextPc: ptr(7)},
		session.Facts{}, rateFacts{}, false, false, path, time.Now())
	if card.Worn != "sabueso" {
		t.Errorf("wearing %q, want sabueso", card.Worn)
	}
	if card.Form != "cazabugs" {
		t.Errorf("the trade outside the bracket is %q, want cazabugs", card.Form)
	}
	band := theme.Strip(assemble(petBand(card, 140), 140))
	if !strings.Contains(band, "cazabugs[sabueso]") {
		t.Errorf("band 4 is %q", band)
	}
}

// A trade has not forked yet, so there is nothing to put in a bracket. This is
// the case that used to lie: a forecast printed as if it were a fact.
func TestATradeWearsNoBracket(t *testing.T) {
	path := t.TempDir() + "/pet.json"
	s := pet.New()
	s.XP = xpFor(4)
	s.Counters = map[string]int{
		"inquisitive": 20, "tests": 20,
		"repro_before_fix": 23, // well past bloodhound's threshold, and level 4
	}
	pet.Save(s, path)

	card, _ := RenderCard(&Payload{Model: "M", ContextPc: ptr(7)},
		session.Facts{}, rateFacts{}, false, false, path, time.Now())
	if card.Form != "cazabugs" || card.Level != 4 {
		t.Fatalf("the pet is %s/%d, want cazabugs/4", card.Form, card.Level)
	}
	if card.Worn != "" {
		t.Errorf("a level 4 pet claims to wear %q", card.Worn)
	}
	if band := theme.Strip(assemble(petBand(card, 140), 140)); strings.Contains(band, "[") {
		t.Errorf("band 4 drew a bracket on a trade: %q", band)
	}
}

// A title is the end of its branch and needs no context: `lobo`, on its own.
func TestATitleWearsNoBracketEither(t *testing.T) {
	path := t.TempDir() + "/pet.json"
	s := pet.New()
	s.XP = xpFor(6)
	s.Counters = map[string]int{
		"inquisitive": 20, "tests": 20,
		"repro_before_fix": pet.Unlocks["bloodhound"].Threshold * 4,
	}
	pet.Save(s, path)

	card, _ := RenderCard(&Payload{Model: "M", ContextPc: ptr(7)},
		session.Facts{}, rateFacts{}, false, false, path, time.Now())
	if card.Form != "lobo" {
		t.Fatalf("the pet is %s, want lobo", card.Form)
	}
	if card.Worn != "" {
		t.Errorf("a title claims to wear %q", card.Worn)
	}
}

// The brackets are painted Dim, not Rule. Rule is the vertical bar's colour
// and is meant not to be seen - 1.54:1 against the background, against 11.8:1
// for the two words it sits between - so what reached the eye was two bright
// words with nothing between them, read as one compound word. The punctuation
// carrying the whole meaning was the only part invisible.
func TestTheBracketsAreVisible(t *testing.T) {
	card := Card{Form: "cazabugs", Worn: "sabueso", Level: 5,
		State: "fresca", Vital: pet.StateFor(7)}
	band := assemble(petBand(card, 140), 140)

	rule := theme.Fg(theme.Rule) + "["
	if strings.Contains(band, rule) {
		t.Error("the brackets are still painted in the separator's colour")
	}
	if !strings.Contains(band, theme.Fg(theme.Dim)+"[") {
		t.Errorf("the brackets are not Dim: %q", band)
	}
}

func TestTheBracketGoesBeforeTheStateWhenColumnsRunOut(t *testing.T) {
	// The bracket is the first thing to give up its columns after the bubble.
	//
	// It is part of the name now rather than a forecast, so dropping it costs
	// something real - which of the two shapes the trade took. What makes it
	// affordable is that the half left standing is still TRUE: a sabueso is a
	// cazabugs, so `cazabugs` on its own is less specific and not wrong. The
	// state cannot be dropped the same way - there is no vaguer version of
	// "fresca" - so it outranks the bracket.
	card := Card{Form: "cazabugs", Worn: "sabueso", Level: 5,
		State: "fresca", Vital: pet.StateFor(7), Bubble: "algo"}

	full := theme.Strip(assemble(petBand(card, 140), 140))
	if !strings.Contains(full, "cazabugs[sabueso]") || !strings.Contains(full, "fresca") {
		t.Fatalf("the wide band is missing something: %q", full)
	}
	tight := theme.Strip(assemble(petBand(card, 140), 26))
	if strings.Contains(tight, "[") {
		t.Errorf("the bracket outlived the squeeze: %q", tight)
	}
	if !strings.Contains(tight, "fresca") {
		t.Errorf("the state went before the bracket did: %q", tight)
	}
}
