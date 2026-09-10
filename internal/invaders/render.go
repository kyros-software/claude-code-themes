package invaders

import (
	"fmt"
	"strings"

	"github.com/kyros-software/claude-code-themes/internal/i18n"
	"github.com/kyros-software/claude-code-themes/internal/pet"
	"github.com/kyros-software/claude-code-themes/internal/theme"
)

// Grid to painted lines. Everything that emits colour goes through
// internal/theme, and nothing here knows what a terminal is: Render returns
// lines, which is what lets the width guard run without a tty.

// Render is the whole screen: one HUD row, the field, one help row.
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
		magazine(g, w),
		theme.Fg(theme.Number) + fmt.Sprintf(w.HUDScore, g.Score) + theme.Reset,
		theme.Fg(pet.RampOf(g.Form).Body[0]) +
			pet.NameIn(i18n.Current(), g.Form) + theme.Reset +
			theme.Fg(theme.Dim) + fmt.Sprintf(" %s%d ", initial(i18n.S().Level), g.Level) + theme.Reset +
			theme.Fg(theme.Ident) + familyName(g.Kit) + theme.Reset,
		ability(g, w),
	}
	if g.Kits > 0 {
		// Only when you have one: a permanent "kits 0" is a line of HUD spent
		// saying nothing.
		parts = append(parts[:3:3], append([]string{
			theme.Fg(theme.Ident) + fmt.Sprintf("✚%d", g.Kits) + theme.Reset,
		}, parts[3:]...)...)
	}
	if g.Wave.Boss && g.Boss.Alive {
		// The boss's own life, next to the word: a fight against ninety hit
		// points with no bar is a fight you cannot tell you are winning. The
		// sprite's colour ramp says the same thing, and says it too slowly.
		boss := theme.Fg(theme.Bad) + theme.Bold + w.HUDBoss + theme.Reset + " " +
			theme.Bar(float64(g.Boss.HP), float64(max(g.Boss.MaxHP, 1)), 6, theme.Bad, theme.Empty)
		parts = append(parts[:2:2], append([]string{boss}, parts[2:]...)...)
	}
	return joinFit(parts, cols)
}

// magazine is the rounds you have left, or the reload you are waiting out. It is
// third in the HUD, ahead of the score, because it is the number you have to
// know before you press anything.
func magazine(g Game, w i18n.Game) string {
	if g.Loading > 0 {
		left := float64(g.Kit.Reload-g.Loading) / float64(max(g.Kit.Reload, 1))
		return theme.Fg(theme.Dim) + w.Reloading + theme.Reset + " " +
			theme.Bar(left, 1, 6, theme.Number, theme.Empty)
	}
	ink := theme.Ident
	if g.Ammo*3 <= g.Kit.Cap {
		ink = theme.Number
	}
	if g.Ammo == 0 {
		ink = theme.Bad
	}
	return theme.Fg(ink) + "≡" + theme.Reset + " " +
		theme.Fg(ink) + fmt.Sprintf("%d/%d", g.Ammo, g.Kit.Cap) + theme.Reset
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

// grid is the field as painted cells.
//
// The colour is kept beside the glyph rather than wrapped around it so that
// assembly can emit one escape per RUN of colour. A sky of forty stars is one
// colour; wrapping each of them cost about ten kilobytes a frame and two hundred
// a second, which is fine on a local terminal and is not fine down an ssh
// connection.
type grid struct {
	glyph [][]string
	ink   [][]string
	cols  int
}

func newGrid(rows, cols int) grid {
	g := grid{
		glyph: make([][]string, rows),
		ink:   make([][]string, rows),
		cols:  cols,
	}
	for i := range g.glyph {
		g.glyph[i] = make([]string, cols)
		g.ink[i] = make([]string, cols)
	}
	return g
}

// put writes one glyph and the colour it wants, ignoring anything off the field.
func (g grid) put(row, col int, glyph, ink string) {
	if row < 0 || row >= len(g.glyph) || col < 0 || col >= g.cols {
		return
	}
	g.glyph[row][col] = glyph
	g.ink[row][col] = ink
}

// blit writes a row of a sprite, one glyph at a time, in one colour. A blank in
// the art is transparent: the ships are drawn on a grid and their corners are
// spaces, and painting those would rub out whatever is behind them.
func (g grid) blit(row, col int, art, ink string) {
	for i, r := range []rune(art) {
		if r == ' ' {
			continue
		}
		g.put(row, col+i, string(r), ink)
	}
}

// fill is blit for something that is meant to be solid: the blanks are painted
// too. The ship's hull needs it - its middle row is `<o o>`, and with a
// transparent gap a star sailed between the eyes and read as a hole in the hull.
func (g grid) fill(row, col int, art, ink string) {
	for i, r := range []rune(art) {
		g.put(row, col+i, string(r), ink)
	}
}

func (g grid) lines() []string {
	out := make([]string, len(g.glyph))
	for i, row := range g.glyph {
		var b strings.Builder
		open := ""
		for x := 0; x < len(row); x++ {
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

// field paints the playfield, back to front: the sky first and the ship last, so
// nothing is ever painted over the thing the player is steering.
func field(g Game, cols int) []string {
	gr := newGrid(g.Field.Rows, cols)

	drawStars(gr, g)
	drawDrops(gr, g)
	drawStones(gr, g)
	drawFleet(gr, g)
	drawBoss(gr, g)
	drawMotes(gr, g)

	for _, t := range g.Turrets {
		gr.put(t.Row, t.Col, "╫", theme.Fg(theme.Ident))
	}
	for _, b := range g.Bombs {
		gr.put(int(b.Y), int(b.X), "╽", theme.Fg(theme.Bad))
	}
	for _, s := range g.Shots {
		gr.put(int(s.Y), int(s.X), "╿", theme.Fg(theme.Emph))
	}
	drawShip(gr, g)
	return gr.lines()
}

// drawStars is the sky: three depths, one dim colour. It is the cheapest thing
// in the frame and the game looks unfinished without it.
//
// Two glyphs and not three, and never an asterisk: the first version gave the
// fastest stars a "*", which is what an explosion is drawn with, and a screenful
// of them read as debris rather than as depth.
func drawStars(gr grid, g Game) {
	glyphs := [3]string{"·", "·", "."}
	ink := theme.Fg(theme.Rule)
	for _, s := range g.Stars {
		which := 0
		switch {
		case s.V > 0.1:
			which = 2
		case s.V > 0.05:
			which = 1
		}
		gr.put(int(s.Y), int(s.X), glyphs[which], ink)
	}
}

// drawFleet paints the ships in the air, each in the wave's stage tone, sinking
// down the tones as it takes hits - the same trick the pet uses for its own
// seven states.
func drawFleet(gr grid, g Game) {
	pal := Stages[clamp(g.Wave.Stage-1, 0, len(Stages)-1)]
	for _, a := range g.Aliens {
		c := a.Craft()
		hurt := 0
		if a.MaxHP > 0 {
			hurt = (a.MaxHP - a.HP) * len(pal.Tones) / (a.MaxHP + 1)
		}
		ink := theme.Fg(pal.Tones[clamp(hurt, 0, len(pal.Tones)-1)])
		eye := theme.Fg(pal.Eye)
		x, y := int(a.X), int(a.Y)
		for i, row := range c.Rows {
			gr.blit(y+i, x, row, ink)
		}
		// The eyes in the light tone, which is the one thing that breaks the
		// flat colour and what makes a screen of them readable at a glance.
		for i, row := range c.Rows {
			for j, r := range []rune(row) {
				if r == 'o' {
					gr.put(y+i, x+j, "o", eye)
				}
			}
		}
	}
}

func drawStones(gr grid, g Game) {
	ink := theme.Fg(theme.Dir)
	for _, s := range g.Stones {
		for i, row := range Rock.Rows {
			gr.blit(int(s.Y)+i, int(s.X), row, ink)
		}
	}
}

func drawDrops(gr grid, g Game) {
	ink := theme.Fg(theme.Ident)
	for _, d := range g.Drops {
		gr.put(int(d.Y), int(d.X), "✚", ink)
	}
}

// drawMotes paints sparks and meteoroids. A spark fades as it dies; a meteoroid
// is red for the whole of its life, because it can kill you.
func drawMotes(gr grid, g Game) {
	for _, m := range g.Motes {
		glyph, ink := "·", theme.Fg(theme.Dim)
		switch {
		case m.Hurt > 0:
			glyph, ink = "*", theme.Fg(theme.Bad)
		case m.Life > sparkLife/2:
			glyph, ink = "*", theme.Fg(theme.Number)
		}
		gr.put(int(m.Y), int(m.X), glyph, ink)
	}
}

// drawBoss paints the one big sprite off the canvas, worn down through its own
// ramp.
func drawBoss(gr grid, g Game) {
	if !g.Boss.Alive {
		return
	}
	b := Bosses[clamp(g.Boss.Of, 0, len(Bosses)-1)]
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
	gr.blit(y+4, x, b.Legs[(g.Frame/20)%2], ink)
}

// drawShip paints the representation of the creature: the family's silhouette,
// the form's own colour, and eyes that are its health. See ship.go.
func drawShip(gr grid, g Game) {
	vital := g.Vital()
	ramp := pet.RampOf(g.Form)
	ink := theme.Fg(ramp.Body[clamp(vital.Rank, 0, len(ramp.Body)-1)])
	eye := theme.Fg(ramp.Lit(vital.Rank))
	if g.Invuln > 0 {
		// The mole's ability, and the only time the ship is not its own colour:
		// invulnerable has to be visible or it is a mechanic nobody trusts.
		ink = theme.Fg(theme.Emph)
	}

	art := ShipArt(g.Kit.Family, vital)
	row := g.Row
	for i, line := range art {
		if i == 1 {
			// The hull is solid; the crest and the tail are not, so the sky
			// shows between the antennae the way it should.
			gr.fill(row+i, g.Ship, line, ink)
			continue
		}
		gr.blit(row+i, g.Ship, line, ink)
	}
	for j, r := range []rune(art[1]) {
		if r == 'o' || r == '-' || r == 'x' {
			gr.put(row+1, g.Ship+j, string(r), eye)
		}
	}
}

// help is the key row, or the banner when there is one to show.
func help(g Game, cols int) string {
	if banner := Banner(g); banner != "" {
		return paint(theme.Emph, theme.Truncate(banner, cols), theme.Bold)
	}
	return paint(theme.Dim, theme.Truncate(helpRow(i18n.G(), cols), cols), "")
}

// helpRow is the longest key row that fits. The Truncate around it is the last
// resort for a window narrower than either.
func helpRow(w i18n.Game, cols int) string {
	if theme.Width(w.Help) <= cols {
		return w.Help
	}
	return w.Tight
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
	case BannerAgain:
		return fmt.Sprintf(w.GameOver, g.Wave.N) + " · " + w.Again
	case BannerClaude:
		return w.PausedByClaude + " · " + w.Resume
	case BannerPaused:
		return w.Paused + " · " + w.Resume
	case BannerChoose:
		return w.LevelUp
	case BannerKit:
		return w.GotKit
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
