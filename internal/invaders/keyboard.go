package invaders

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// The keyboard's autorepeat, borrowed while the game has the focus.
//
// This is the piece that makes holding an arrow work at all. A terminal has no
// key-up event and no way to say a key is still down: what it has is the
// operating system's autorepeat, which streams the key while it is held. So
// "holding" is a stream of presses and "releasing" is the stream stopping - see
// moveShip - and the whole thing rests on how quickly the stream starts.
//
// On this desktop it starts after HALF A SECOND (xset says "auto repeat delay:
// 500"), which is what a held arrow felt like: one column, a pause long enough to
// think something had broken, and then a smooth glide. Eighty milliseconds is
// what a game wants.
//
// It is a global X setting, so it is borrowed rather than taken: shortened when
// the game's window takes the focus and put back the moment it loses it, which is
// why run.go turns on focus reporting. Typing in another window is never affected,
// and every exit path restores it - a normal quit, a signal, the window being
// closed, or a panic in the tick.
const (
	ourDelay = 80
	ourRate  = 40
)

// noXset switches the whole thing off for anybody who would rather it did not
// touch their keyboard at all. The game still plays: holding an arrow just waits
// out whatever delay the desktop is set to.
const noXset = "CCPET_NO_XSET"

// keyboard remembers what the desktop had, so it can be given back.
type keyboard struct {
	delay, rate int
	ours        bool // theirs was read and ours may be set
	fast        bool // ours is set right now
}

// borrowKeyboard reads the current setting, and says nothing if there is nothing
// to read: no X, no xset, or somebody who asked to be left alone.
func borrowKeyboard() *keyboard {
	k := &keyboard{}
	if os.Getenv(noXset) != "" || os.Getenv("DISPLAY") == "" {
		return k
	}
	if _, err := exec.LookPath("xset"); err != nil {
		return k
	}
	out, err := exec.Command("xset", "q").Output()
	if err != nil {
		return k
	}
	delay, rate, ok := repeatIn(string(out))
	if !ok {
		return k
	}
	k.delay, k.rate, k.ours = delay, rate, true
	return k
}

// repeatIn reads the two numbers out of xset's report:
//
//	auto repeat delay:  500    repeat rate:  33
func repeatIn(out string) (delay, rate int, ok bool) {
	for _, line := range strings.Split(out, "\n") {
		if !strings.Contains(line, "auto repeat delay") {
			continue
		}
		fields := strings.Fields(line)
		for i, f := range fields {
			switch {
			case f == "delay:" && i+1 < len(fields):
				delay, _ = strconv.Atoi(fields[i+1])
			case f == "rate:" && i+1 < len(fields):
				rate, _ = strconv.Atoi(fields[i+1])
			}
		}
		return delay, rate, delay > 0 && rate > 0
	}
	return 0, 0, false
}

// quicken shortens the delay, and is a no-op if it is already short or if there
// was nothing to read in the first place.
func (k *keyboard) quicken() {
	if !k.ours || k.fast {
		return
	}
	if setRepeat(ourDelay, ourRate) {
		k.fast = true
	}
}

// restore puts the desktop's own setting back. It is safe to call twice, and it
// is called from every path out of a run.
func (k *keyboard) restore() {
	if !k.ours || !k.fast {
		return
	}
	setRepeat(k.delay, k.rate)
	k.fast = false
}

func setRepeat(delay, rate int) bool {
	return exec.Command("xset", "r", "rate", strconv.Itoa(delay), strconv.Itoa(rate)).Run() == nil
}
