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
	// frameEvery is the wall clock between ticks. Twenty a second.
	frameEvery = time.Second / TicksPerSecond

	// pauseEvery is how often the pause file is checked, in ticks. Four times a
	// second: at ten the banner could be half a second late, which is long
	// enough to lose the wave you were meant to be let out of.
	pauseEvery = 5

	// emptyReadsBeforeGivingUp is how many reads may come back with nothing
	// before the key reader decides the terminal has gone.
	emptyReadsBeforeGivingUp = 1000

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
	Size  func() (cols, rows int)
	Keys  <-chan Key
	Out   io.Writer
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

	keys := make(chan Key, 8)
	stop := make(chan struct{})
	go readKeys(tm, keys, stop)

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
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)

	state := LoadSave(savePath)
	form, level := pet.CurrentForm(pet.Load(petPath))
	g := NewGame(f, form, level, state)
	if state.Seed == 0 {
		g.Rand = uint64(now.UnixNano()) | 1
	}

	sc := screen{
		Size:  tm.size,
		Keys:  keys,
		Out:   out,
		Close: func() { close(stop) },
	}
	final, code := loop(sc, g, signals, time.Now)

	state = final.ToSave(state)
	giveBack()
	close(stop)

	if err := StoreSave(state, savePath); err != nil {
		fmt.Fprintln(stderr, "ccpet:", err)
	}
	if final.Phase == Over {
		fmt.Fprintln(stdout, Banner(final))
		if lost := concede(petPath, final.Wave.N, now); lost > 0 {
			fmt.Fprintf(stdout, "%s\n", fmt.Sprintf(i18n.G().LostALevel,
				pet.LevelFor(pet.Load(petPath).XP)))
		}
		fmt.Fprintln(stdout, Records(state))
	}
	return code
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
	watch := watchPause()
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

		// Everything that has arrived since the last frame, with an ACTION
		// winning over a direction.
		//
		// One tick can only carry one key, and a terminal cannot say that two
		// are held at once. Movement latches - see driftFor - so a direction
		// dropped here costs nothing, while a dropped shot is a press the player
		// made and the game ignored. So firing wins, and you can shoot without
		// stopping.
		in := None
		for drained := false; !drained; {
			select {
			case k := <-sc.Keys:
				if k == Quit {
					return g, 0
				}
				if in == None || in == Left || in == Right {
					in = k
				}
			default:
				drained = true
			}
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
		if g.Frame%pauseEvery == 0 && watch.fired() && g.Phase == Playing {
			g.Phase = Paused
			g.Banner = BannerClaude
		}

		g = Tick(g, in)
		draw()

		if g.Phase == Over {
			// One last frame with the creature lying down, and then it waits:
			// the records are worth reading before the shell comes back.
			return waitForAKey(sc, g)
		}
	}
}

// waitForAKey holds the final frame until the player presses something.
func waitForAKey(sc screen, g Game) (Game, int) {
	timeout := time.NewTimer(30 * time.Second)
	defer timeout.Stop()
	select {
	case <-sc.Keys:
	case <-timeout.C:
	}
	return g, 0
}

// reflow fits a running game into a field that has changed size, keeping the
// creature on the floor and the block inside the walls.
func reflow(g Game, f Field) Game {
	if f == g.Field {
		return g
	}
	old := g.Field
	g.Field = f
	g.Wave = WaveFor(g.Wave.N, f)

	if g.Ship > f.ShipColMax() {
		g.Ship = f.ShipColMax()
	}

	// The formation is a grid, so a narrower window can leave columns off the
	// right of the screen. They are dropped rather than squeezed: a member that
	// cannot be drawn cannot be shot, and one that cannot be shot lands on you.
	if len(g.Squad.Members) > 0 {
		if g.Squad.X+float64(f.BlockCols()) > float64(f.Cols) {
			g.Squad.X = float64(max(f.Cols-f.BlockCols(), 0))
		}
		cols := f.FormationCols()
		next := make([]Member, 0, len(g.Squad.Members))
		for _, m := range g.Squad.Members {
			if m.Col < cols {
				next = append(next, m)
			}
		}
		g.Squad.Members = next
		if g.Started > len(g.Squad.Members) && old.Cols > f.Cols {
			g.Started = len(g.Squad.Members)
		}
	}
	if g.Boss.Alive && g.Boss.X > float64(f.Cols-BossCols) {
		g.Boss.X = float64(max(f.Cols-BossCols, 0))
	}
	g.Shots = nil
	g.Bombs = nil
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

func enter(w io.Writer) { io.WriteString(w, "\033[?1049h\033[?25l") }
func leave(w io.Writer) { io.WriteString(w, theme.Reset+"\033[?25h\033[?1049l") }

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
func readKeys(r io.Reader, keys chan<- Key, stop <-chan struct{}) {
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
			var decoded []Key
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
