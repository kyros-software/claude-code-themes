package invaders

import (
	"fmt"
	"strings"

	"github.com/kyros-software/claude-code-themes/internal/i18n"
	"github.com/kyros-software/claude-code-themes/internal/pet"
	"github.com/kyros-software/claude-code-themes/internal/theme"
)

// Grid to painted lines. Everything that emits colour in here goes through
// internal/theme, and the creatures - yours and the rival's - are drawn by
// internal/pet, whose own tests already guarantee every row is nine cells wide
// in every state.

// traitInk is the colour a trait is drawn in. The glyph says which body it is
// and the colour says which trait, so thirty-five kinds are legible at a glance
// without anybody reading a legend.
var traitInk = [traitCount]theme.Colour{
	Plain:    theme.Dim,
	Weaver:   theme.PaleTeal,
	Darter:   theme.Bad,
	Plated:   theme.Grey,
	Splitter: theme.Quota,
}

// Render is the whole screen: one HUD row, the field, one help row.
//
// It returns lines and writes nothing, which is what lets the width guard run
// without a terminal.
func Render(g Game, cols int) []string {
	out := make([]string, 0, g.Field.Rows+HUDRows+HelpRows)
	out = append(out, hud(g, cols))
	out = append(out, field(g, cols)...)
	return append(out, help(g, cols))
}

// hud is the one row above the field.
func hud(g Game, cols int) string {
	w := i18n.G()
	parts := []string{
		theme.Fg(theme.Emph) + fmt.Sprintf(w.HUDWave, g.Wave.N) + theme.Reset,
		theme.Fg(theme.Bad) + "♥" + theme.Reset + " " +
			theme.Bar(float64(g.HP), float64(g.Kit.MaxHP), 8, theme.Bad, theme.Empty),
		theme.Fg(theme.Number) + fmt.Sprintf(w.HUDScore, g.Score) + theme.Reset,
		theme.Fg(pet.RampOf(g.Form).Body[0]) +
			pet.NameIn(i18n.Current(), g.Form) + theme.Reset +
			theme.Fg(theme.Dim) + fmt.Sprintf(" %s%d ", initial(i18n.S().Level), g.Level) + theme.Reset +
			theme.Fg(theme.Ident) + familyName(g.Kit) + theme.Reset,
		ability(g, w),
	}
	if g.Wave.Boss {
		// Straight after the life, because it is the one thing that changes how
		// you should be playing.
		parts = append(parts[:2:2],
			append([]string{theme.Fg(theme.Bad) + theme.Bold + w.HUDBoss + theme.Reset},
				parts[2:]...)...)
	}
	return joinFit(parts, cols)
}

// joinFit joins the parts it can afford, most important first, and drops the
// rest.
//
// Not a truncation: theme.Truncate counts the bytes of an escape sequence as
// visible width and will happily cut inside one, which puts a literal "[38" on
// screen. It is right for the plain text the statusline hands it and wrong for
// anything already painted. Dropping whole parts is also the better answer for a
// player: a HUD that loses the score at sixty columns still reads, one that ends
// mid-word does not.
func joinFit(parts []string, cols int) string {
	sep := theme.Fg(theme.Rule) + " · " + theme.Reset
	for n := len(parts); n > 0; n-- {
		if line := strings.Join(parts[:n], sep); theme.Width(line) <= cols {
			return line
		}
	}
	return ""
}

// familyName is what the HUD calls the weapon.
func familyName(k Kit) string {
	if name := i18n.G().Families[k.Family]; name != "" {
		return name
	}
	return k.Family
}

func ability(g Game, w i18n.Game) string {
	if g.Ready == 0 {
		return theme.Fg(theme.Emph) + w.HUDAbility + " " + w.HUDReady + theme.Reset
	}
	left := float64(g.Kit.Cooldown-g.Ready) / float64(max(g.Kit.Cooldown, 1))
	return theme.Fg(theme.Dim) + w.HUDAbility + theme.Reset + " " +
		theme.Bar(left, 1, 6, theme.Emph, theme.Empty)
}

// field paints the playfield.
//
// The creatures are opaque: their nine columns on their five rows are the
// sprite's own painted cells and nothing is drawn behind them. pet.Draw hands
// back whole painted rows rather than cells, and re-implementing its painter so
// an enemy could show through a crest would be a second copy of the one thing
// internal/pet exists for.
func field(g Game, cols int) []string {
	rows := make([][]string, g.Field.Rows)
	blocks := make([]map[int]string, g.Field.Rows)
	for i := range rows {
		rows[i] = make([]string, cols)
		blocks[i] = map[int]string{}
	}

	put := func(row, x int, s string) {
		if row < 0 || row >= len(rows) || x < 0 || x >= cols {
			return
		}
		rows[row][x] = s
	}

	for _, t := range g.Turrets {
		put(t.Row, ShipCols+1, theme.Fg(theme.Ident)+"╫"+theme.Reset)
	}
	for _, b := range g.Bolts {
		put(b.Row, int(b.X), theme.Fg(theme.Bad)+"◄"+theme.Reset)
	}
	for _, s := range g.Shots {
		put(s.Row, int(s.X), theme.Fg(theme.Emph)+"·"+theme.Reset)
	}

	for _, e := range g.Enemies {
		if e.Boss() {
			drawCreature(blocks, e.Rival, e.Vital(), g.Frame/8, false, e.Row, int(e.X), cols)
			continue
		}
		ink := theme.Fg(traitInk[e.Trait])
		for i, r := range []rune(e.Glyph()) {
			put(e.Row, int(e.X)+i, ink+string(r)+theme.Reset)
		}
	}

	// The ship goes on last so nothing is ever painted over the creature.
	drawCreature(blocks, g.Form, g.Vital(), g.Frame/8, g.Ready > 0, g.Ship, 0, cols)

	out := make([]string, len(rows))
	for i := range rows {
		out[i] = assemble(rows[i], blocks[i], cols)
	}
	return out
}

// drawCreature lays a five-row sprite into the row blocks.
//
// step is the frame count divided down. pet.Draw's walk cycle is step%12 < 4,
// calibrated for a statusline that refreshes once a second; handed a raw
// twenty-per-second frame counter the feet strobe.
func drawCreature(blocks []map[int]string, form string, v pet.Vital, step int, dim bool, top, x, cols int) {
	if x < 0 || x+ShipCols > cols {
		return
	}
	sprite := pet.Draw(form, v, step, dim)
	for i, line := range sprite {
		row := top + i
		if row < 0 || row >= len(blocks) {
			continue
		}
		blocks[row][x] = line
	}
}

// assemble turns a row of cells and its sprite blocks into one painted line.
func assemble(cells []string, blocks map[int]string, cols int) string {
	var b strings.Builder
	for x := 0; x < cols; x++ {
		if block, ok := blocks[x]; ok {
			b.WriteString(block)
			x += ShipCols - 1
			continue
		}
		if cells[x] == "" {
			b.WriteByte(' ')
			continue
		}
		b.WriteString(cells[x])
	}
	return b.String()
}

// help is the key row, or the banner when there is one to show.
func help(g Game, cols int) string {
	if banner := Banner(g); banner != "" {
		return paint(theme.Emph, theme.Truncate(banner, cols), theme.Bold)
	}
	return paint(theme.Dim, theme.Truncate(i18n.G().Help, cols), "")
}

// paint colours plain text. The truncation happens BEFORE the escapes go on, so
// there is never a cut inside one.
func paint(col theme.Colour, plain, weight string) string {
	return theme.Fg(col) + weight + plain + theme.Reset
}

// Banner is the message a state is showing, in words. The tick only ever sets an
// id: it has no catalogue and no language.
func Banner(g Game) string {
	w := i18n.G()
	switch g.Banner {
	case BannerCleared:
		return fmt.Sprintf(w.WaveCleared, g.Wave.N)
	case BannerBoss:
		return fmt.Sprintf(w.RivalArrives, rivalName(g))
	case BannerBossOff:
		return fmt.Sprintf(w.BossDown, rivalName(g))
	case BannerRevived:
		return w.Revived
	case BannerOver:
		return fmt.Sprintf(w.GameOver, g.Wave.N)
	case BannerClaude:
		return w.PausedByClaude + " · " + w.Resume
	case BannerPaused:
		return w.Paused + " · " + w.Resume
	}
	return ""
}

// rivalName is what to call the rival a wave is holding, falling back to the one
// the wave would have if it has already been beaten.
func rivalName(g Game) string {
	for _, e := range g.Enemies {
		if e.Boss() {
			return pet.NameIn(i18n.Current(), e.Rival)
		}
	}
	return pet.NameIn(i18n.Current(), RivalFor(g.Wave.N, g.Form))
}

// Records is the line printed when a run ends.
func Records(s Save) string {
	return fmt.Sprintf(i18n.G().Records, s.BestWave, s.BestScore, s.Runs)
}

// TooSmall is the refusal, with both pairs of numbers in it so the player can
// see how far off they are rather than guessing.
func TooSmall(cols, rows int) string {
	return fmt.Sprintf(i18n.G().TooSmall, MinCols, MinRows, cols, rows)
}

// initial is the first letter of a word, by rune. Slicing the string would be
// one translation away from cutting a two-byte letter in half.
func initial(word string) string {
	for _, r := range word {
		return string(r)
	}
	return ""
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
