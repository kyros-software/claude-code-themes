package invaders

import (
	"bytes"
	"github.com/kyros-software/claude-code-themes/internal/i18n"
	"github.com/kyros-software/claude-code-themes/internal/theme"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kyros-software/claude-code-themes/internal/pet"
)

// counter counts the writes that reach the terminal, which is how the one
// frame, one write rule is checked.
type counter struct {
	mu     sync.Mutex
	writes int
	buf    bytes.Buffer
}

func (c *counter) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.writes++
	return c.buf.Write(p)
}

func (c *counter) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.writes
}

func (c *counter) String() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buf.String()
}

// fakeScreen is the nine lines the loop needs instead of a terminal.
func fakeScreen(t *testing.T, cols, rows int) (screen, chan Key, *counter) {
	t.Helper()
	keys := make(chan Key, 8)
	out := &counter{}
	var mu sync.Mutex
	sc := screen{
		Size: func() (int, int) {
			mu.Lock()
			defer mu.Unlock()
			return cols, rows
		},
		Keys:  keys,
		Out:   out,
		Close: func() {},
	}
	return sc, keys, out
}

func petIn(t *testing.T, xp int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "pet.json")
	s := pet.New()
	s.XP = xp
	if !pet.Save(s, path) {
		t.Fatal("the pet did not save")
	}
	return path
}

// Quitting saves the wave you were on, so the next run picks it up. It is the
// only granularity the file has and it is what makes the auto-pause forgiving.
func TestAQuitKeySavesTheWaveYouWereOn(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	sc, keys, _ := fakeScreen(t, 80, 24)
	f, _ := FieldFor(80, 24)
	g := NewGame(f, "bughunter", 4, Save{Wave: 9, HP: 5, Seed: 1})

	go func() {
		time.Sleep(80 * time.Millisecond)
		keys <- Quit
	}()
	final, code := loop(sc, g, nil, time.Now)
	if code != 0 {
		t.Errorf("quitting exited %d", code)
	}
	if final.Phase == Over {
		t.Error("quitting counted as dying")
	}

	got := final.ToSave(Save{Wave: 9, BestWave: 9})
	if got.Wave != 9 {
		t.Errorf("it saved wave %d, want the 9 it was on", got.Wave)
	}
}

// The whole point of the Stop hook. Claude finishes answering, the game stops
// and says why, and you go and read what it did.
func TestTheGameStopsWhenClaudeStops(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	sc, keys, _ := fakeScreen(t, 80, 24)
	f, _ := FieldFor(80, 24)
	g := NewGame(f, "spark", 2, Save{Wave: 1, Seed: 1})

	go func() {
		time.Sleep(60 * time.Millisecond)
		Touch(time.Now().Add(time.Second))
		time.Sleep(400 * time.Millisecond)
		keys <- Quit
	}()
	final, _ := loop(sc, g, nil, time.Now)

	if final.Phase != Paused {
		t.Errorf("the game is in %v, want paused", final.Phase)
	}
	if final.Banner != BannerClaude {
		t.Errorf("the banner is %q, want the one that says why", final.Banner)
	}
}

// A pause file left behind by a game last week must not pause a fresh one. The
// mtime seen at startup is the baseline, so only something newer counts.
func TestAPauseFileFromLastWeekDoesNotPauseAFreshRun(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	if err := Touch(time.Now().Add(-7 * 24 * time.Hour)); err != nil {
		t.Fatal(err)
	}

	sc, keys, _ := fakeScreen(t, 80, 24)
	f, _ := FieldFor(80, 24)
	g := NewGame(f, "spark", 2, Save{Wave: 1, Seed: 1})
	go func() {
		time.Sleep(450 * time.Millisecond)
		keys <- Quit
	}()
	final, _ := loop(sc, g, nil, time.Now)

	if final.Phase == Paused {
		t.Error("a week-old pause file paused a fresh run")
	}
}

// Too small is a refusal that stays on screen, not a mess drawn anyway. And it
// has to come back on its own when the window grows, or the player has to
// restart to recover from having resized.
func TestATerminalTooSmallIsRefusedAndNothingIsDrawn(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	keys := make(chan Key, 4)
	out := &counter{}
	cols, rows := 40, 10
	var mu sync.Mutex
	sc := screen{
		Size: func() (int, int) {
			mu.Lock()
			defer mu.Unlock()
			return cols, rows
		},
		Keys: keys, Out: out, Close: func() {},
	}
	f, _ := FieldFor(80, 24)
	g := NewGame(f, "spark", 2, Save{Wave: 1, Seed: 1})

	go func() {
		time.Sleep(120 * time.Millisecond)
		mu.Lock()
		cols, rows = 80, 24
		mu.Unlock()
		time.Sleep(200 * time.Millisecond)
		keys <- Quit
	}()
	loop(sc, g, nil, time.Now)

	painted := out.String()
	if !strings.Contains(painted, "40") || !strings.Contains(painted, "10") {
		t.Errorf("the refusal does not name the size it got:\n%q", theStripped(painted))
	}
	if !strings.Contains(painted, "♥") {
		t.Error("it never came back when the window grew")
	}
}

// A window that shrinks must not leave the creature off the floor or the block
// hanging outside the walls. A member that cannot be drawn cannot be shot, and
// one that cannot be shot lands on you.
func TestAResizeThatShrinksTheFieldKeepsEverythingOnScreen(t *testing.T) {
	big, _ := FieldFor(120, 40)
	small, _ := FieldFor(60, 18)
	g := NewGame(big, "wasp", 6, Save{Wave: 7, Seed: 2})
	for i := 0; i < 300; i++ {
		g = Tick(g, Right)
	}

	g = reflow(g, small)

	if g.Ship > small.ShipColMax() {
		t.Errorf("the ship is at column %d of a field %d wide", g.Ship, small.Cols)
	}
	for _, a := range g.Aliens {
		if a.X < 0 || a.X+float64(a.Craft().W) > float64(small.Cols) {
			t.Errorf("a %s spans %g..%g of %d columns",
				a.Craft().Name, a.X, a.X+float64(a.Craft().W), small.Cols)
		}
		if a.Y < 0 || a.Y > float64(small.ShipRow()) {
			t.Errorf("a %s is at row %g of %d", a.Craft().Name, a.Y, small.Rows)
		}
	}
	for _, st := range g.Stones {
		if st.X < 0 || st.X+float64(Rock.W) > float64(small.Cols) {
			t.Errorf("a rock spans %g..%g of %d columns", st.X, st.X+float64(Rock.W), small.Cols)
		}
	}
	if got := len(Render(g, small.Cols)); got != small.Rows+HUDRows+HelpRows {
		t.Errorf("after the resize a frame is %d lines", got)
	}
}

// One buffer, one flush, one write per frame. Anything else is a terminal
// receiving a frame in pieces, which is exactly what flicker is.
func TestEveryFrameIsOneWriteAndOneFlush(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	sc, keys, out := fakeScreen(t, 80, 24)
	f, _ := FieldFor(80, 24)
	g := NewGame(f, "ember", 3, Save{Wave: 1, Seed: 1})

	go func() {
		time.Sleep(300 * time.Millisecond)
		keys <- Quit
	}()
	loop(sc, g, nil, time.Now)

	frames := strings.Count(out.String(), "\033[?2026h")
	if frames == 0 {
		t.Fatal("nothing was drawn")
	}
	if got := out.count(); got != frames {
		t.Errorf("%d frames arrived in %d writes", frames, got)
	}
}

// The screen and the cursor always come back. A game that leaves your terminal
// with no cursor and no scrollback is worse than one that crashes.
func TestTheAlternateScreenIsAlwaysGivenBack(t *testing.T) {
	var out bytes.Buffer
	enter(&out)
	leave(&out)
	painted := out.String()
	for _, want := range []string{"\033[?1049h", "\033[?25l", "\033[?25h", "\033[?1049l"} {
		if !strings.Contains(painted, want) {
			t.Errorf("the pair does not carry %q", want)
		}
	}
	if strings.Index(painted, "\033[?1049h") > strings.Index(painted, "\033[?1049l") {
		t.Error("it leaves the alternate screen before it enters it")
	}

	src := readSource(t, "run.go")
	if !strings.Contains(src, "defer giveBack()") {
		t.Error("the screen is not given back on a panic")
	}
	if !strings.Contains(src, "syscall.SIGTERM") {
		t.Error("a SIGTERM does not give the screen back")
	}
}

// A signal is not a defeat. Neither is a quit. The pet only ever pays for a
// creature that actually died.
func TestAnInterruptedGameIsNotALostGame(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	sc, _, _ := fakeScreen(t, 80, 24)
	f, _ := FieldFor(80, 24)
	g := NewGame(f, "spark", 3, Save{Wave: 4, Seed: 1})

	signals := make(chan os.Signal, 1)
	go func() {
		time.Sleep(80 * time.Millisecond)
		signals <- os.Interrupt
	}()
	final, code := loop(sc, g, signals, time.Now)

	if code != 0 {
		t.Errorf("an interrupt exited %d", code)
	}
	if final.Phase == Over {
		t.Error("an interrupt counted as dying, which would cost the pet a level")
	}
	signal.Stop(signals)
}

// Dying costs the level and quitting does not. Both halves in one test, because
// the whole risk of coupling the game to the pet is that the second half stops
// being true without anybody noticing.
func TestDyingCostsTheLevelAndQuittingDoesNot(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	t.Run("dying near the line takes the level", func(t *testing.T) {
		// Just inside level 4, so the drop fits under the day-of-feeding cap
		// and the level really goes. Deep inside a level it costs a day and the
		// level holds, which is TestADefeatNeverCostsMoreThanADayOfFeeding.
		path := petIn(t, 420)
		was := pet.Load(path).XP
		if lost := concede(path, 12, now); lost <= 0 {
			t.Fatalf("it took %d xp", lost)
		}
		s := pet.Load(path)
		if s.XP >= was {
			t.Errorf("xp went %d -> %d", was, s.XP)
		}
		if pet.LevelFor(s.XP) != 3 {
			t.Errorf("it is level %d, want 3", pet.LevelFor(s.XP))
		}
		if len(s.Log) != 1 || s.Log[0].Event != pet.DefeatEvent {
			t.Errorf("the log says %+v", s.Log)
		}
	})

	t.Run("a larva has nothing to give and the file is not rewritten", func(t *testing.T) {
		path := petIn(t, 10)
		before, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if lost := concede(path, 1, now); lost != 0 {
			t.Errorf("it took %d xp from a larva", lost)
		}
		after, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if !after.ModTime().Equal(before.ModTime()) {
			t.Error("it rewrote pet.json to change nothing")
		}
	})
}

// The game may only ever take. Whatever happens in a run, the pet must never
// come out of it with more XP than it went in with - that is the half of the
// design's original rule that survives, and it is worth a test rather than a
// comment.
func TestTheGameCanOnlyEverTakeXpAndNeverGiveIt(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	for _, xp := range []int{0, 59, 60, 400, 1999, 2000, 4500, 9000} {
		path := petIn(t, xp)
		for i := 0; i < 5; i++ {
			concede(path, i+1, now)
		}
		if got := pet.Load(path).XP; got > xp {
			t.Errorf("starting at %d xp, five defeats left %d", xp, got)
		}
	}
}

func theStripped(s string) string { return strings.ReplaceAll(s, "\033", "^") }

// stutter is a reader that comes back with nothing before it comes back with
// something, which is what a terminal does.
type stutter struct {
	steps []string
	at    int
	done  chan struct{}
}

func (s *stutter) Read(p []byte) (int, error) {
	if s.at >= len(s.steps) {
		<-s.done
		return 0, io.EOF
	}
	step := s.steps[s.at]
	s.at++
	if step == "" {
		// Exactly what os.File does with a read that timed out, which is the
		// shape that killed the keyboard.
		return 0, io.EOF
	}
	return copy(p, step), nil
}

// A read that comes back with nothing is not the end of the terminal, and the
// key reader must not give up on the first one.
//
// This is the dead keyboard, as a test. os.File reports a read that timed out as
// io.EOF, so with VMIN 0 the reader saw one a tenth of a second in, took it for
// a closed terminal and returned - and not one keypress reached the game for the
// rest of the run. The termios is right now and this can no longer happen, but
// the budget means that getting it wrong again is a game that keeps playing
// rather than one nobody can steer.
func TestAnEmptyReadIsNotTheEndOfTheKeyboard(t *testing.T) {
	src := &stutter{steps: []string{"", "", "a", "", " ", "\033[C"}, done: make(chan struct{})}

	keys := make(chan Key, 8)
	stop := make(chan struct{})
	defer close(stop)

	go readKeys(src, keys, stop)

	want := []Key{Left, Fire, Right}
	for i, w := range want {
		select {
		case got := <-keys:
			if got != w {
				t.Errorf("key %d is %d, want %d", i, got, w)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("only %d keys arrived: the reader gave up on an empty read", i)
		}
	}
	close(src.done)
}

// The other half of the auto-pause: a prompt takes the game back out of it. Ten
// turns is ten of these, and a game that had to be un-paused by hand every time
// would be a game nobody plays twice.
func TestAPromptTakesTheGameOutOfThePauseClaudeLeftItIn(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	sc, keys, _ := fakeScreen(t, 80, 24)
	f, _ := FieldFor(80, 24)
	g := NewGame(f, "spark", 2, Save{Wave: 1, Seed: 1})

	go func() {
		time.Sleep(60 * time.Millisecond)
		Touch(time.Now().Add(time.Second))
		time.Sleep(200 * time.Millisecond)
		Resume(time.Now().Add(2 * time.Second))
		time.Sleep(200 * time.Millisecond)
		keys <- Quit
	}()
	final, _ := loop(sc, g, nil, time.Now)

	if final.Phase != Playing {
		t.Errorf("the game is in %v after the prompt, want playing", final.Phase)
	}
	if final.Banner == BannerClaude {
		t.Error("the game is playing and still says Claude stopped it")
	}
}

// But a pause the PLAYER asked for is the player's. Claude starting another turn
// does not hand the keyboard back to a game somebody deliberately stopped to go
// and read something.
func TestAPromptDoesNotLiftThePauseThePlayerAskedFor(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	sc, keys, _ := fakeScreen(t, 80, 24)
	f, _ := FieldFor(80, 24)
	g := NewGame(f, "spark", 2, Save{Wave: 1, Seed: 1})

	go func() {
		time.Sleep(60 * time.Millisecond)
		keys <- Pause
		time.Sleep(200 * time.Millisecond)
		Resume(time.Now().Add(time.Second))
		time.Sleep(200 * time.Millisecond)
		keys <- Quit
	}()
	final, _ := loop(sc, g, nil, time.Now)

	if final.Phase != Paused || final.Banner != BannerPaused {
		t.Errorf("the game is in %v saying %q, want the player's own pause kept",
			final.Phase, final.Banner)
	}
}

// The window's name is what the arena raises the game by, so it is part of
// borrowing the terminal - and it is handed back, or a tab that played once is
// called "ccpet invade" until it is closed.
func TestTheGameNamesTheWindowAndGivesTheNameBack(t *testing.T) {
	var got strings.Builder
	enter(&got)
	if !strings.Contains(got.String(), "\033]0;"+arenaTitle+"\007") {
		t.Errorf("entering wrote %q, with no window title in it", got.String())
	}
	got.Reset()
	leave(&got)
	if !strings.Contains(got.String(), "\033]0;\007") {
		t.Errorf("leaving wrote %q, and never gave the title back", got.String())
	}
}

// A window that gets shorter must not leave the ship below the floor or hanging
// in the middle of the field: both are rows it cannot be steered out of.
func TestAResizeKeepsTheShipBetweenItsRoofAndItsFloor(t *testing.T) {
	big, _ := FieldFor(120, 40)
	small, _ := FieldFor(60, 18)
	g := NewGame(big, "wasp", 6, Save{Wave: 3, Seed: 5})
	for i := 0; i < 200; i++ { // a held arrow, which is a stream of presses
		g = Tick(g, Up)
	}
	if g.Row != big.ShipRoof() {
		t.Fatalf("it did not climb: row %d of a roof at %d", g.Row, big.ShipRoof())
	}

	g = reflow(g, small)
	if g.Row < small.ShipRoof() || g.Row > small.ShipRow() {
		t.Errorf("after the resize the ship is at row %d, and its half is %d..%d",
			g.Row, small.ShipRoof(), small.ShipRow())
	}

	g = reflow(g, big)
	if g.Row < big.ShipRoof() || g.Row > big.ShipRow() {
		t.Errorf("back in the big window it is at row %d of %d..%d",
			g.Row, big.ShipRoof(), big.ShipRow())
	}
}

// killer is a bomb one row above the ship with more than enough in it, which is
// how these tests get a death in three ticks instead of in three minutes.
func killer(g Game) Game {
	g.HP = 1
	g.Bombs = []Bomb{{X: float64(g.Ship + 2), Y: float64(g.Row) - 1, Hurt: 99}}
	return g
}

// Dying is not the end of the evening: the run ends, the screen says so, and
// space starts another one.
func TestTheGameOverScreenOffersAnotherRunAndSpaceTakesIt(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	i18n.Use(i18n.ES)
	defer i18n.Use("")

	sc, keys, out := fakeScreen(t, 80, 24)
	f, _ := FieldFor(80, 24)
	g := NewGame(f, "bughunter", 4, Save{Wave: 6, Seed: 1})
	g.Phase = Over
	g.Banner = BannerOver

	go func() {
		time.Sleep(60 * time.Millisecond)
		keys <- Fire
	}()
	if !askAgain(sc, g) {
		t.Error("space on the game-over screen did not ask for another run")
	}
	painted := out.String()
	if !strings.Contains(theme.Strip(painted), i18n.G().Again) {
		t.Errorf("the game-over screen never says how to play again:\n%s", theme.Strip(painted))
	}
	if !strings.Contains(theme.Strip(painted), "oleada 6") {
		t.Error("the game-over screen does not say which wave it ended on")
	}
}

func TestQuittingFromTheGameOverScreenLeaves(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	sc, keys, _ := fakeScreen(t, 80, 24)
	f, _ := FieldFor(80, 24)
	g := NewGame(f, "spark", 1, Save{Wave: 2, Seed: 1})

	go func() {
		time.Sleep(60 * time.Millisecond)
		keys <- Quit
	}()
	if askAgain(sc, g) {
		t.Error("q on the game-over screen asked for another run")
	}
}

// The whole cycle: die, ask for another, and the second run is a run - it reads
// the keyboard, and the quit that ends it is consumed by it rather than left
// unread.
func TestASecondRunReallyStartsAfterADeath(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	petPath := petIn(t, 420)
	savePath := filepath.Join(t.TempDir(), "invaders.json")
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	sc, keys, _ := fakeScreen(t, 80, 24)
	f, _ := FieldFor(80, 24)
	form, level := pet.CurrentForm(pet.Load(petPath))
	g := killer(NewGame(f, form, level, Save{Wave: 4, Seed: 1}))

	go func() {
		time.Sleep(250 * time.Millisecond) // the bomb lands well inside this
		keys <- Fire                       // another run
		time.Sleep(250 * time.Millisecond)
		keys <- Quit // and this one has to be read by that run
	}()

	state, final, deaths, code := series(sc, g, LoadSave(savePath), petPath, savePath, nil, now, time.Now)
	if code != 0 {
		t.Errorf("it exited %d", code)
	}
	if deaths != 1 {
		t.Errorf("%d deaths, want one", deaths)
	}
	if len(keys) != 0 {
		t.Error("the quit was never read: the second run did not start")
	}
	if final.Phase == Over {
		t.Error("it came back on the game-over screen rather than out of a live run")
	}
	if final.Wave.N != 1 {
		t.Errorf("the second run started on wave %d", final.Wave.N)
	}
	if state.BestWave < 4 {
		t.Errorf("the records lost the wave the first run reached: %+v", state)
	}
	if state.Runs != 1 {
		t.Errorf("%d runs counted, want the one that ended", state.Runs)
	}
}

// The run after a death flies the level the pet has NOW, which is the whole point
// of the wager: the kit the replay gets is the one the death just paid for.
//
// It asserts against pet.LevelFor rather than against a number, because how much
// a defeat costs is pet.Setback's business - it is capped at a day of feeding, so
// a death deep inside a level takes xp and leaves the level standing, and that
// rule belongs to internal/pet and is tested there.
func TestTheRunAfterADeathFliesTheLevelThePetHasNow(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	petPath := petIn(t, 420) // just inside level 4, so the first drop really lands
	savePath := filepath.Join(t.TempDir(), "invaders.json")
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	f, _ := FieldFor(80, 24)

	if was := pet.LevelFor(pet.Load(petPath).XP); was != 4 {
		t.Fatalf("the test pet is level %d, want 4", was)
	}

	concede(petPath, 9, now)
	if now := pet.LevelFor(pet.Load(petPath).XP); now != 3 {
		t.Errorf("after one death the pet is level %d, want 3", now)
	}

	for death := 1; death <= 3; death++ {
		g := revive(f, petPath, savePath)
		level := pet.LevelFor(pet.Load(petPath).XP)
		if g.Level != level {
			t.Errorf("death %d: the next run flies level %d and the pet is level %d",
				death, g.Level, level)
		}
		if g.Kit != KitFor(g.Form, level) {
			t.Errorf("death %d: the next run does not fly the kit its level buys", death)
		}
		if g.Wave.N != 1 || g.Phase != Playing {
			t.Errorf("death %d: the next run starts on wave %d in %v", death, g.Wave.N, g.Phase)
		}
		if g.HP != g.Kit.MaxHP {
			t.Errorf("death %d: it starts with %d of %d life", death, g.HP, g.Kit.MaxHP)
		}
		if g.Power != 0 || g.Quick != 0 || g.Mag != 0 {
			t.Error("a new run kept the upgrades the last one built")
		}
		concede(petPath, 9, now)
	}
}
