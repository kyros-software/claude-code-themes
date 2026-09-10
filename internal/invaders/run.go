package invaders

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kyros-software/claude-code-themes/internal/i18n"
	"github.com/kyros-software/claude-code-themes/internal/pet"
	"github.com/kyros-software/claude-code-themes/internal/theme"
)

// The loop: the only file in this package that talks to a terminal, and the only
// one that is allowed to write the pet.

const (
	// frameEvery is the wall clock between ticks: forty a second, twenty-five
	// milliseconds apart.
	frameEvery = time.Second / TicksPerSecond

	// pauseEvery is how often the pause file is checked, in ticks. Four times a
	// second: at ten the banner could be half a second late, which is long
	// enough to lose the wave you were meant to be let out of.
	pauseEvery = TicksPerSecond / 4

	// emptyReadsBeforeGivingUp is how many reads may come back with nothing
	// before the key reader decides the terminal has gone.
	emptyReadsBeforeGivingUp = 1000

	// againWait is how long the game-over screen waits for an answer before it
	// gives the terminal back. Long enough to walk away from a lost run and come
	// back to it, short enough that an abandoned window does not hold a shell
	// for the rest of the day.
	againWait = 2 * time.Minute

	// sizeEvery is how often the terminal is re-measured, in ticks.
	//
	// Polled rather than driven by SIGWINCH: it is one ioctl a second against a
	// signal handler, a channel and a race to reason about, and it is the same
	// call the fake screen in the tests already provides. The cost of being a
	// second late to a resize is one smeared frame.
	sizeEvery = TicksPerSecond
)

// screen is everything the loop needs from a terminal.
//
// A struct of functions rather than an interface with a mock: the fake is nine
// lines in a test, and this way the loop has no idea whether it is driving a tty.
type screen struct {
	Size func() (cols, rows int)
	Keys <-chan Event
	Out  io.Writer
	// Focus is the window taking or losing the focus, which is not a game event
	// at all: it is when the keyboard's autorepeat is borrowed and given back.
	// See keyboard.go.
	Focus func(has bool)
	Close func()
}

// Run is ccpet invade.
func Run(args []string, stdout, stderr io.Writer, petPath, savePath string, now time.Time) int {
	if len(args) > 0 && args[0] == "--split" {
		return split(stdout, stderr)
	}

	tm, err := openTerm()
	if err != nil {
		fmt.Fprintln(stderr, "ccpet:", i18n.G().NoTTY)
		return 2
	}
	defer tm.Close()

	cols, rows := tm.size()
	f, ok := FieldFor(cols, rows)
	if !ok {
		fmt.Fprintln(stderr, "ccpet:", TooSmall(cols, rows))
		return 2
	}

	restore, err := tm.raw()
	if err != nil {
		fmt.Fprintln(stderr, "ccpet:", i18n.G().NoTTY)
		return 2
	}

	keys := make(chan Event, 16)
	stop := make(chan struct{})
	go readKeys(tm, keys, stop)

	// The heartbeat. It is how a prompt in any session can tell that a game is
	// already running and raise that window instead of opening a second one, and
	// it is kept whether the arena is on or not: a game somebody started by hand
	// is still the one game. The claim goes back on the way out, in that order,
	// so the beater cannot write the file after it has been dropped.
	heart := make(chan struct{})
	defer Unbeat()
	defer close(heart)
	go beating(heart)

	// The alternate screen and the cursor come back whatever happens: a normal
	// quit, a signal, or a panic in the tick. A game that leaves your terminal
	// with no cursor is worse than a game that crashes.
	out := tm
	enter(out)
	given := false
	giveBack := func() {
		if given {
			return
		}
		given = true
		leave(out)
		restore()
	}
	defer giveBack()

	signals := make(chan os.Signal, 1)
	// SIGHUP as well as the other two: closing the window is how a game in the
	// arena usually ends, and it has to give the keyboard back like every other
	// way out.
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(signals)

	// The autorepeat, borrowed for as long as this window has the focus. Nothing
	// happens here on a desktop with no xset, no DISPLAY, or a CCPET_NO_XSET in
	// the environment.
	kb := borrowKeyboard()
	defer kb.restore()

	state := LoadSave(savePath)
	form, level := pet.CurrentForm(pet.Load(petPath))
	g := NewGame(f, form, level, state)
	if state.Seed == 0 {
		g.Rand = uint64(now.UnixNano()) | 1
	}

	sc := screen{
		Size: tm.size,
		Keys: keys,
		Out:  out,
		Focus: func(has bool) {
			if has {
				kb.quicken()
				return
			}
			kb.restore()
		},
		Close: func() { close(stop) },
	}
	state, final, deaths, code := series(sc, g, state, petPath, savePath, signals, now, time.Now)

	giveBack()
	close(stop)

	if deaths > 0 {
		// One line whatever the evening was, and the level it says is the one
		// the pet is on NOW: five runs is five setbacks, and reporting each of
		// them would be five lines saying the same thing worse.
		fmt.Fprintln(stdout, Banner(over(final)))
		fmt.Fprintf(stdout, "%s\n", fmt.Sprintf(i18n.G().LostALevel,
			pet.LevelFor(pet.Load(petPath).XP)))
		fmt.Fprintln(stdout, Records(state))
	}
	return code
}

// series plays one run after another for as long as the player asks for another,
// and returns the last of them with the number that ended in a death.
//
// It is the loop around the loop, and it lives here rather than in the tick for
// the same reason concede does: between two runs the pet is READ AGAIN. The death
// that just happened has taken a level off it, so the next run flies the kit it
// has now - which is the whole point of the wager, and it would be invisible if a
// replay reused the kit the process started with.
func series(sc screen, g Game, state Save, petPath, savePath string,
	signals <-chan os.Signal, now time.Time, clock func() time.Time) (Save, Game, int, int) {
	deaths := 0
	for {
		final, code := loop(sc, g, signals, clock)
		state = final.ToSave(state)
		if err := StoreSave(state, savePath); err != nil {
			return state, final, deaths, code
		}
		if final.Phase != Over {
			// A quit, a signal or a terminal that went away. None of them is a
			// defeat and none of them asks a question.
			return state, final, deaths, code
		}

		deaths++
		concede(petPath, final.Wave.N, now)
		if !askAgain(sc, final) {
			return state, final, deaths, code
		}
		g = revive(final.Field, petPath, savePath)
	}
}

// over is the state as the game-over line wants it, for the one printed to the
// shell on the way out.
func over(g Game) Game {
	g.Banner = BannerOver
	return g
}

// askAgain holds the last frame with the offer on it and waits.
//
// It draws once and then blocks. There is nothing left to animate - the fleet is
// gone, the ship has its eyes out - and a screen that keeps repainting a dead
// game is a screen that keeps a laptop's fan on while somebody decides.
func askAgain(sc screen, g Game) bool {
	g.Banner = BannerAgain
	w := bufio.NewWriterSize(sc.Out, 1<<16)
	frame(w, Render(g, g.Field.Cols))

	timeout := time.NewTimer(againWait)
	defer timeout.Stop()
	for {
		select {
		case e := <-sc.Keys:
			switch e.Key {
			case Fire, One:
				// The mouse button counts: it arrives as a Fire, and somebody
				// who has been steering with the pointer will click.
				return true
			case Quit, Pause:
				return false
			}
		case <-timeout.C:
			// Nobody is there. Give the terminal back rather than sit on it for
			// the rest of the day.
			return false
		}
	}
}

// revive is the run after a death: the pet is read again for the level it has
// now, and the save for the records the last run left behind.
func revive(f Field, petPath, savePath string) Game {
	form, level := pet.CurrentForm(pet.Load(petPath))
	return NewGame(f, form, level, LoadSave(savePath))
}

// concede is the whole of the game's power over the pet: one level, once, on a
// death, through pet.Update.
//
// It is here and not in the tick because the tick runs twenty times a second and
// this has to happen exactly once. It is never called for a quit, a Ctrl-C or a
// pause: leaving the game is not losing it.
func concede(petPath string, wave int, now time.Time) int {
	lost := 0
	pet.Update(petPath, func(s *pet.State) bool {
		lost = pet.Setback(s, fmt.Sprintf("invade %s%d", initial(i18n.G().HUDWave), wave), now)
		return lost > 0
	})
	return lost
}

// loop runs the game against whatever screen it is handed, which is what makes
// the pacing, the auto-pause and the resize testable with no terminal at all.
func loop(sc screen, g Game, signals <-chan os.Signal, now func() time.Time) (Game, int) {
	ticker := time.NewTicker(frameEvery)
	defer ticker.Stop()

	w := bufio.NewWriterSize(sc.Out, 1<<16)
	watch := watchFile(PausePath())
	wake := watchFile(ResumePath())
	cols, _ := sc.Size()
	small := false

	draw := func() {
		if small {
			c, r := sc.Size()
			frame(w, []string{TooSmall(c, r)})
			return
		}
		frame(w, Render(g, cols))
	}
	draw()

	for {
		select {
		case <-signals:
			// A signal is not a defeat. The run is saved where it stands and
			// the terminal goes back.
			return g, 0
		case <-ticker.C:
		}

		// Everything that has arrived since the last frame, sorted into the two
		// things a frame can carry: where you are going and what you did.
		//
		// It used to carry one key and prefer the action, which threw away a
		// direction every time a shot and an arrow arrived in the same
		// twenty-five milliseconds - which is most of the time, since a held
		// arrow repeats at about the frame rate. Whichever of the two was
		// dropped, something the player did did not happen.
		move, act := None, None
		aimed, aimX, aimY := false, 0, 0
		for drained := false; !drained; {
			select {
			case e := <-sc.Keys:
				switch e.Key {
				case FocusIn, FocusOut:
					// Not a key: the window's own news, and the tick has no
					// business hearing it.
					if sc.Focus != nil {
						sc.Focus(e.Key == FocusIn)
					}
					// And whatever the mouse was doing, it is not doing it now.
					// A button released outside the window is a release nobody
					// ever reports, and a trigger held down for ever after it is
					// a game that shoots on its own.
					g.Trigger = false
				case Quit:
					return g, 0
				case MouseAt:
					// Only the last one matters: the pointer is where it is now,
					// not where it has been.
					aimed, aimX, aimY = true, e.X, e.Y
				case Click:
					// A press of the button both fires and holds the trigger
					// down, and the trigger is not a key: nothing cancels it but
					// the release.
					act = Fire
					g.Trigger = true
				case Release:
					g.Trigger = false
				case Left, Right, Up, Down, Stop:
					// The brake is a movement key, not an action: it belongs
					// with the arrows or it never reaches moveShip at all.
					move = e.Key
					g.Trigger = false
				default:
					act = e.Key
					// A key means the hand is on the keyboard, so the mouse is
					// not holding anything down any more. Whoever is playing
					// with the keys does their own shooting.
					g.Trigger = false
				}
			default:
				drained = true
			}
		}
		if aimed {
			g = g.AimAt(aimX, aimY-HUDRows)
		}

		if g.Frame%sizeEvery == 0 {
			c, r := sc.Size()
			if f, ok := FieldFor(c, r); ok {
				small = false
				cols = c
				g = reflow(g, f)
			} else {
				small = true
			}
		}
		if g.Frame%pauseEvery == 0 {
			// Both files are asked on the same beat and the phase decides which
			// answer matters: a Stop only stops a game that is playing, and a
			// prompt only wakes one that a Stop put to sleep. A pause the PLAYER
			// asked for with p is the player's, and Claude does not lift it -
			// which is the whole reason the banner is looked at here.
			stopped, woken := watch.fired(), wake.fired()
			if stopped && g.Phase == Playing {
				g.Phase = Paused
				g.Banner = BannerClaude
			}
			if woken && g.Phase == Paused && g.Banner == BannerClaude {
				g.Phase = Playing
				g.Banner = ""
			}
		}

		g = TickWith(g, move, act)
		draw()

		if g.Phase == Over {
			// The last frame is drawn - the ship with its eyes out - and that is
			// where this stops. What happens next is a question, and asking it is
			// series' job.
			return g, 0
		}
	}
}

// reflow fits a running game into a field that has changed size, keeping the
// ship on the floor and everything else inside the walls.
//
// A ship or an alien left outside a narrowed field is not a cosmetic problem: a
// thing that cannot be drawn cannot be shot, and one that cannot be shot reaches
// the floor for free.
func reflow(g Game, f Field) Game {
	if f == g.Field {
		return g
	}
	g.Field = f
	g.Wave = WaveFor(g.Wave.N, f)

	if g.Ship > f.ShipColMax() {
		g.Ship = f.ShipColMax()
	}
	if g.Ship < 0 {
		g.Ship = 0
	}
	// The row as well, now that the ship has one of its own: a shorter window
	// can leave it below the floor, and a taller one can leave it hanging in the
	// middle of the field where it has no business being.
	g.Row = clamp(g.Row, f.ShipRoof(), f.ShipRow())

	aliens := make([]Alien, 0, len(g.Aliens))
	for _, a := range g.Aliens {
		if right := float64(f.Cols - a.Craft().W); a.X > right {
			a.X = right
		}
		if a.X < 0 {
			a.X = 0
		}
		if a.Y > float64(f.ShipRow()) {
			a.Y = float64(f.ShipRow())
		}
		aliens = append(aliens, a)
	}
	g.Aliens = aliens

	stones := make([]Stone, 0, len(g.Stones))
	for _, s := range g.Stones {
		if right := float64(f.Cols - Rock.W); s.X > right {
			s.X = right
		}
		if s.X < 0 {
			s.X = 0
		}
		stones = append(stones, s)
	}
	g.Stones = stones

	if g.Boss.Alive && g.Boss.X > float64(f.Cols-BossCols) {
		g.Boss.X = float64(max(f.Cols-BossCols, 0))
	}
	// The sky is regenerated rather than squeezed: the stars are decoration and
	// a whole new one is cheaper than moving forty of them.
	g.Stars = sky(f, &g.Rand)
	g.Shots = nil
	g.Bombs = nil
	g.Motes = nil
	g.Drops = nil
	return g
}

// frame writes one whole repaint: one buffer, one flush, one write.
//
// Cursor home and an erase-to-end on every row, never a clear-screen: \033[2J is
// the flash. The synchronised-output pair around it is ignored by terminals that
// do not have it. No diffing: sixty by eighteen is about ten kilobytes a frame
// and two hundred a second, which is nothing, and a shadow buffer buys a whole
// class of stale-cell bugs for no measurable gain.
func frame(w *bufio.Writer, lines []string) {
	w.WriteString("\033[?2026h\033[H")
	for i, line := range lines {
		w.WriteString(line)
		w.WriteString("\033[K")
		if i < len(lines)-1 {
			w.WriteString("\r\n")
		}
	}
	w.WriteString("\033[?2026l")
	w.Flush()
}

// enter and leave are the terminal the game borrows: the alternate screen, the
// cursor, and the window's name.
//
// The name is not decoration - it is the handle the arena raises the game's
// window by, there being no way to find a window by the process inside it under
// a terminal whose windows all belong to one server. It is given back on the way
// out with an empty title, which is how a terminal is told to go back to
// deciding its own.
// The 1004 in there is focus reporting: the terminal sends CSI I when the window
// takes the focus and CSI O when it loses it. It is the only way the game can
// know it is being played rather than merely open, and it is what scopes the
// borrowed autorepeat to this window.
// The 1003 and the 1006 are the mouse: report every movement of the pointer, in
// the SGR encoding, which is the one that can count past column 223.
//
// It is how the game this is modelled on steers, and it is the only way a
// terminal can express two directions at once - a keyboard cannot, and the two
// measurements in keyboard.go are why. It costs the terminal's own text selection
// while the game is running, which every full-screen program that reads the mouse
// costs; shift and drag still selects in most of them.
func enter(w io.Writer) {
	io.WriteString(w, "\033]0;"+arenaTitle+"\007\033[?1049h\033[?25l\033[?1004h")
	if mouseWanted() {
		io.WriteString(w, "\033[?1003h\033[?1006h")
	}
}

func leave(w io.Writer) {
	io.WriteString(w, "\033[?1006l\033[?1003l")
	io.WriteString(w, theme.Reset+"\033[?1004l\033[?25h\033[?1049l\033]0;\007")
}

// NoMouse turns the pointer off for anybody who would rather have their
// terminal's own text selection back, or who keeps knocking the mouse.
const NoMouse = "CCPET_NO_MOUSE"

func mouseWanted() bool { return os.Getenv(NoMouse) == "" }

// beating keeps the heartbeat fresh for as long as the game is on screen.
func beating(stop <-chan struct{}) {
	Beat(time.Now())
	ticker := time.NewTicker(beatEvery)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case now := <-ticker.C:
			Beat(now)
		}
	}
}

// readKeys turns the terminal into a channel of keys.
//
// The channel is small and full sends are dropped: somebody leaning on the arrow
// key must not be able to stall the frame that is meant to be showing them the
// result.
//
// A read that comes back with nothing is NOT the end of the terminal, and taking
// it for one is how the first version of this shipped with a dead keyboard: with
// VMIN 0 and VTIME 1 os.File reports a timed-out read as io.EOF, so the goroutine
// returned a tenth of a second in and never read another byte. Raw mode blocks
// on a byte now, and an empty read is treated as the nothing it is.
func readKeys(r io.Reader, keys chan<- Event, stop <-chan struct{}) {
	buf := make([]byte, 0, 16)
	chunk := make([]byte, 16)
	empty := 0
	for {
		select {
		case <-stop:
			return
		default:
		}
		n, err := r.Read(chunk)
		if n > 0 {
			empty = 0
			var decoded []Event
			decoded, buf = DecodeAll(append(buf, chunk[:n]...))
			for _, k := range decoded {
				select {
				case keys <- k:
				case <-stop:
					return
				default:
				}
			}
			continue
		}
		// A terminal that has gone away reports it over and over and instantly,
		// so a budget of empty reads tells that apart from a terminal that is
		// merely quiet - and it costs microseconds to spend. With VMIN 1 the
		// budget is never touched at all; it is here so that getting the termios
		// wrong is a game that keeps playing rather than a dead keyboard, which
		// is exactly how this shipped the first time.
		if empty++; empty > emptyReadsBeforeGivingUp {
			return
		}
		if err == nil {
			continue
		}
	}
}
