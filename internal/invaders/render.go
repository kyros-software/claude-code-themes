package invaders

import (
	"fmt"
	"strings"

	"github.com/kyros-software/claude-code-themes/internal/i18n"
	"github.com/kyros-software/claude-code-themes/internal/pet"
	"github.com/kyros-software/claude-code-themes/internal/theme"
)

// Grid to painted lines. Everything that emits colour goes through
// internal/theme, and the creature at the bottom is drawn by internal/pet, whose
// own tests already guarantee every row is nine cells wide in every state.

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
		parts = append(parts[:2:2],
			append([]string{theme.Fg(theme.Bad) + theme.Bold + w.HUDBoss + theme.Reset},
				parts[2:]...)...)
	}
	return joinFit(parts, cols)
}

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

// joinFit joins the parts it can afford, most important first, and drops the
// rest.
//
// Not a truncation: theme.Truncate counts the bytes of an escape sequence as
// visible width and will happily cut inside one, which puts a literal "[38" on
// screen. Dropping whole parts is also the better answer for a player: a HUD
// that loses the score at sixty columns still reads, one that ends mid-word does
// not.
func joinFit(parts []string, cols int) string {
	sep := theme.Fg(theme.Rule) + " · " + theme.Reset
	for n := len(parts); n > 0; n-- {
		if line := strings.Join(parts[:n], sep); theme.Width(line) <= cols {
			return line
		}
	}
	return ""
}

// grid is the field as painted cells, plus the whole painted rows that
// internal/pet hands back for the creature.
//
// The blocks are separate because pet.DrawCompact returns a row at a time, not a
// cell at a time - it is the only painter in the repo that knows how to draw the
// creature and re-implementing it here to get cells would be a second copy of
// the one thing internal/pet is for. So a block claims nine columns of its row
// and the assembly steps over them.
type grid struct {
	glyph  [][]string // what is in the cell, unpainted
	ink    [][]string // the escape that colours it
	blocks []map[int]string
	cols   int
}

func newGrid(rows, cols int) grid {
	g := grid{
		glyph:  make([][]string, rows),
		ink:    make([][]string, rows),
		blocks: make([]map[int]string, rows),
		cols:   cols,
	}
	for i := range g.glyph {
		g.glyph[i] = make([]string, cols)
		g.ink[i] = make([]string, cols)
		g.blocks[i] = map[int]string{}
	}
	return g
}

// block lays an already-painted nine-cell row of the creature into the grid.
func (g grid) block(row, col int, painted string) {
	if row < 0 || row >= len(g.blocks) || col < 0 || col+ShipCols > g.cols {
		return
	}
	g.blocks[row][col] = painted
}

// put writes one glyph and the colour it wants, ignoring anything off the field.
//
// The colour is kept beside the glyph rather than wrapped around it so that
// assembly can emit one escape per RUN of colour. A row of eleven identical
// sprites is one colour and sixty-odd cells; wrapping each of them cost about
// ten kilobytes a frame and two hundred a second, which is fine on a local
// terminal and is not fine down an ssh connection.
func (g grid) put(row, col int, glyph, ink string) {
	if row < 0 || row >= len(g.glyph) || col < 0 || col >= g.cols {
		return
	}
	g.glyph[row][col] = glyph
	g.ink[row][col] = ink
}

// blit writes a row of a sprite, one glyph at a time, in one colour. A blank in
// the art is transparent: the sprites are drawn on a grid and their corners are
// spaces, and painting those would rub out whatever is behind them.
func (g grid) blit(row, col int, art, ink string) {
	for i, r := range []rune(art) {
		if r == ' ' {
			continue
		}
		g.put(row, col+i, string(r), ink)
	}
}

func (g grid) lines() []string {
	out := make([]string, len(g.glyph))
	for i, row := range g.glyph {
		var b strings.Builder
		open := ""
		for x := 0; x < len(row); x++ {
			if painted, ok := g.blocks[i][x]; ok {
				if open != "" {
					b.WriteString(theme.Reset)
					open = ""
				}
				b.WriteString(painted)
				x += ShipCols - 1
				continue
			}
			if row[x] == "" {
				if open != "" {
					b.WriteString(theme.Reset)
					open = ""
				}
				b.WriteByte(' ')
				continue
			}
			if ink := g.ink[i][x]; ink != open {
				b.WriteString(ink)
				open = ink
			}
			b.WriteString(row[x])
		}
		if open != "" {
			b.WriteString(theme.Reset)
		}
		out[i] = b.String()
	}
	return out
}

// field paints the playfield: the block at the top, the creature at the bottom,
// and whatever is in the air between them.
func field(g Game, cols int) []string {
	gr := newGrid(g.Field.Rows, cols)

	drawSquad(gr, g)
	drawBoss(gr, g)

	for _, t := range g.Turrets {
		gr.put(g.Field.ShipRow()-1, t.Col, "╫", theme.Fg(theme.Ident))
	}
	for _, b := range g.Bombs {
		gr.put(int(b.Y), int(b.X), "╽", theme.Fg(theme.Bad))
	}
	for _, s := range g.Shots {
		gr.put(int(s.Y), int(s.X), "╿", theme.Fg(theme.Emph))
	}

	// The creature is opaque on its nine columns, and it goes in last so
	// nothing is ever painted over it.
	sprite := pet.DrawCompact(g.Form, g.Vital(), g.Frame/8, g.Ready > 0)
	for i, row := range sprite {
		gr.block(g.Field.ShipRow()+i, g.Ship, row)
	}
	return gr.lines()
}

// drawSquad paints the block: every living member in its stage's tone, with the
// three cells of eyes in the light one.
func drawSquad(gr grid, g Game) {
	frame := g.Squad.Frame()
	for _, m := range g.Squad.Members {
		t := Troops[m.Species]
		pal := Stages[t.Stage-1]
		// A member that has taken a hit and lived shows it by sinking down the
		// ramp, which is the same trick the pet uses for its own seven states.
		tone := pal.Tones[clamp(g.Wave.HP-m.HP, 0, len(pal.Tones)-1)]
		ink, eye := theme.Fg(tone), theme.Fg(pal.Eye)

		x, y := g.Squad.At(m)
		gr.blit(y, x, t.Top, ink)
		gr.blit(y+1, x, t.Left, ink)
		gr.blit(y+1, x+1, t.Eyes, eye)
		gr.blit(y+1, x+4, t.Right, ink)
		gr.blit(y+2, x, t.Legs[frame], ink)
	}
}

// drawBoss paints the one big sprite, worn down through its own ramp.
func drawBoss(gr grid, g Game) {
	if !g.Boss.Alive {
		return
	}
	b := Bosses[g.Boss.Of]
	pal := Ranks[b.Rank-1]
	hurt := 0
	if g.Boss.MaxHP > 0 {
		hurt = (g.Boss.MaxHP - g.Boss.HP) * len(pal.Ramp) / (g.Boss.MaxHP + 1)
	}
	ink := theme.Fg(pal.Ramp[clamp(hurt, 0, len(pal.Ramp)-1)])
	eye := theme.Fg(pal.Eye)

	x, y := int(g.Boss.X), int(g.Boss.Y)
	gr.blit(y, x, b.Top, ink)
	gr.blit(y+1, x, b.Upper, ink)
	gr.blit(y+2, x, b.Left, ink)
	gr.blit(y+2, x+3, b.Eyes, eye)
	gr.blit(y+2, x+6, b.Right, ink)
	gr.blit(y+3, x, b.Lower, ink)
	gr.blit(y+4, x, b.Legs[(g.Frame/10)%2], ink)
}

// help is the key row, or the banner when there is one to show.
func help(g Game, cols int) string {
	if banner := Banner(g); banner != "" {
		return paint(theme.Emph, theme.Truncate(banner, cols), theme.Bold)
	}
	return paint(theme.Dim, theme.Truncate(i18n.G().Help, cols), "")
}

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
		return fmt.Sprintf(w.RivalArrives, bossName(g))
	case BannerBossOff:
		return fmt.Sprintf(w.BossDown, bossName(g))
	case BannerRevived:
		return w.Revived
	case BannerLanded:
		return w.Landed
	case BannerOver:
		return fmt.Sprintf(w.GameOver, g.Wave.N)
	case BannerClaude:
		return w.PausedByClaude + " · " + w.Resume
	case BannerPaused:
		return w.Paused + " · " + w.Resume
	}
	return ""
}

// bossName is what the wave's boss is called, off the canvas.
func bossName(g Game) string { return Bosses[clamp(g.Wave.BossOf, 0, len(Bosses)-1)].Name }

// Records is the line printed when a run ends.
func Records(s Save) string {
	return fmt.Sprintf(i18n.G().Records, s.BestWave, s.BestScore, s.Runs)
}

// TooSmall is the refusal, with both pairs of numbers in it.
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
