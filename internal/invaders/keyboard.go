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
// ourDelay is how long a key has to be held before it starts repeating, and it
// is a compromise between two things that pull opposite ways.
//
// Short is what a game wants: the delay is dead time at the start of every hold,
// and at the desktop's own five hundred milliseconds a held arrow gave one column
// and then looked broken. Eighty was the first try and it broke something else -
// a window manager chord like Cinnamon's ctrl+super+up, which switches workspace,
// is held for something like a tenth of a second by a human hand, and at eighty
// it fired two or three times: you switch workspace and come straight back, which
// reads as the shortcut not working at all.
//
// Two hundred is longer than a deliberate chord and much shorter than the wait
// that started this. Anybody who would rather their keyboard was left alone
// entirely has CCPET_NO_XSET.
const (
	ourDelay = 200
	ourRate  = 40
)

// noRepeat are the keys that must NOT repeat while the game has the focus, by X
// keycode.
//
// This is the fix for the complaint that outlasted two others: "sigue habiendo
// problemas con pararse mientras se mueve". It is not the game's doing, and it is
// worth writing down exactly what was measured. Holding an arrow, a terminal
// receives:
//
//	2560ms  ESC[D          the press
//	3058ms  ESC[D ESC[D…   the autorepeat, every 30ms
//	4058ms  space          one tap of the fire key
//	        (nothing)      and not one arrow again, with the key still held down
//
// X repeats the last key pressed and only that one, and a press of anything else
// cancels the repeat for good - releasing the new key does not bring the old one
// back. So every shot stopped the ship until the player let go of the arrow and
// pressed it again.
//
// With the fire key's own repeat turned off, the same test gives the arrow back
// 29 milliseconds after the shot and it keeps repeating. That is the whole fix:
// none of these keys has any use for a repeat - a held space bar is not a faster
// gun, the cadence decides that - so their repeat is switched off while the
// window has the focus and switched back exactly as it was when it loses it.
//
// The UP AND DOWN ARROWS are in the list too, and that is the second half of the
// same story. Reported from play: "estoy apretando a la izquierda y, mientras
// sigo apretando, hago abajo o arriba" - and the ship stops. Measured the same
// way, holding left and tapping up:
//
//	with up repeating      …ESC[D ESC[D ESC[A                 and nothing after it
//	without up repeating   …ESC[D ESC[D ESC[A ESC[D ESC[D…    left carries straight on
//
// So the vertical is a STEP rather than a glide: one row per press, no repeat.
// That is the trade, and it is the right way round - strafing is what wants to be
// continuous, the ship's half of the field is nine rows against forty-odd
// columns, and a step is precise where a glide is not. Left and right keep their
// repeat, and so do the letters that strafe.
//
// The codes are evdev's, which is to say physical positions: right for anybody on
// a qwerty-shaped layout, which the letters below assume. On a layout that moves
// them, the wrong keys lose their repeat while the game is focused - harmless,
// scoped to the focus, and no worse than the behaviour this replaces.
var noRepeat = []int{
	65,  // space, fire
	53,  // x, ability
	52,  // z, ability
	27,  // r, reload
	26,  // e, health kit
	41,  // f, health kit
	10,  // 1, upgrade
	11,  // 2
	12,  // 3
	33,  // p, pause
	24,  // q, quit
	111, // up
	116, // down
	25,  // w, up
	45,  // k, up
	44,  // j, down
}

// noXset switches the whole thing off for anybody who would rather it did not
// touch their keyboard at all. The game still plays: holding an arrow just waits
// out whatever delay the desktop is set to.
const noXset = "CCPET_NO_XSET"

// keyboard remembers what the desktop had, so it can be given back.
type keyboard struct {
	delay, rate int
	repeats     map[int]bool // whether each action key repeated before we touched it
	ours        bool         // theirs was read and ours may be set
	fast        bool         // ours is set right now
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
	// And which of the action keys repeated before, one bit each, so that
	// restoring puts back what was there rather than what we assume was there.
	k.repeats = map[int]bool{}
	bits := repeatBits(string(out))
	for _, kc := range noRepeat {
		k.repeats[kc] = bits[kc]
	}
	return k
}

// repeatBits reads xset's per-key table:
//
//	auto repeating keys:  00ffffffdffffbbf
//	                      fadfffefffedffff
//	                      9fffffffffffffff
//	                      fff7ffffffffffff
//
// Thirty-two bytes, one bit per keycode, lowest keycode in the lowest bit of the
// first byte - the same layout XGetKeyboardControl hands out.
func repeatBits(out string) [256]bool {
	var on [256]bool
	lines := strings.Split(out, "\n")
	for i, line := range lines {
		if !strings.Contains(line, "auto repeating keys") {
			continue
		}
		hex := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "auto repeating keys:"))
		for j := i + 1; j < len(lines) && len(hex) < 64; j++ {
			hex += strings.TrimSpace(lines[j])
		}
		for b := 0; b*2+1 < len(hex) && b < 32; b++ {
			v, err := strconv.ParseUint(hex[b*2:b*2+2], 16, 8)
			if err != nil {
				break
			}
			for bit := 0; bit < 8; bit++ {
				on[b*8+bit] = v&(1<<uint(bit)) != 0
			}
		}
		break
	}
	return on
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

// quicken shortens the delay and stops the action keys repeating. It is a no-op
// if it is already done or if there was nothing to read in the first place.
func (k *keyboard) quicken() {
	if !k.ours || k.fast {
		return
	}
	args := []string{"r", "rate", strconv.Itoa(ourDelay), strconv.Itoa(ourRate)}
	for _, kc := range noRepeat {
		args = append(args, "-r", strconv.Itoa(kc))
	}
	if exec.Command("xset", args...).Run() == nil {
		k.fast = true
	}
}

// restore puts the desktop's own settings back, per key: what was repeating
// repeats again and what was not does not. Safe to call twice, and called from
// every path out of a run.
func (k *keyboard) restore() {
	if !k.ours || !k.fast {
		return
	}
	exec.Command("xset", k.restoreArgs()...).Run()
	k.fast = false
}

func (k *keyboard) restoreArgs() []string {
	args := []string{"r", "rate", strconv.Itoa(k.delay), strconv.Itoa(k.rate)}
	for _, kc := range noRepeat {
		if k.repeats[kc] {
			args = append(args, "r", strconv.Itoa(kc))
			continue
		}
		args = append(args, "-r", strconv.Itoa(kc))
	}
	return args
}
