package invaders

// Key and mouse decoding, kept away from anything that owns a file descriptor so
// it can be tested without a terminal.

// Event is one thing the terminal reported: a key, or where the pointer is.
//
// The pointer is here because it is the only way a terminal can say that two
// directions are wanted AT ONCE. A keyboard cannot: there is no key-up event, and
// X repeats the last key pressed and only that one, so pressing up while holding
// left kills the left - which is measured, twice, in keyboard.go. A pointer has
// no such problem, it is two numbers, and it is how the game this is modelled on
// steers: the ship is centred on it.
type Event struct {
	Key  Key
	X, Y int // cells, zero-based, for MouseAt
}

// Decode reads one key out of a buffer and says how many bytes it consumed.
//
// It returns n == 0 when the buffer holds the beginning of an escape sequence
// and not yet all of it, so the caller keeps the bytes and reads more. Raw mode
// is set with VMIN=0 and VTIME=1, which means a read can and does return half of
// an arrow key - and three bytes of garbage in a game steered with two keys is a
// creature that jumps across the screen.
func Decode(buf []byte) (k Key, n int) {
	if len(buf) == 0 {
		return None, 0
	}
	switch buf[0] {
	case 0x1b: // ESC
		if len(buf) < 3 {
			return None, 0
		}
		if buf[1] != '[' && buf[1] != 'O' {
			return None, 2
		}
		switch buf[2] {
		case 'D':
			return Left, 3
		case 'C':
			return Right, 3
		case 'B':
			return Down, 3
		case 'A':
			return Up, 3
		case 'I':
			// The window took the focus, and the terminal is telling us because
			// run.go asked it to. It is not a key and it never reaches the tick.
			return FocusIn, 3
		case 'O':
			return FocusOut, 3
		}
		return None, 3
	case 'a', 'A', 'h', 'H':
		return Left, 1
	case 'd', 'D', 'l', 'L':
		return Right, 1
	case 'w', 'W', 'k', 'K':
		return Up, 1
	case 'j', 'J':
		return Down, 1
	case 's', 'S':
		// The brake, and the one key where wasd and hjkl disagree: wasd wants
		// this for down. The down arrow and j both do that, and something has to
		// stop a ship that latches.
		return Stop, 1
	case ' ':
		return Fire, 1
	case 'x', 'X', 'z', 'Z':
		return Ability, 1
	case 'r', 'R':
		return Rearm, 1
	case 'e', 'E', 'f', 'F':
		// e for the kit, and f as well: the reference's own README names both,
		// and somebody who has played that one will try f.
		return Heal, 1
	case '1':
		return One, 1
	case '2':
		return Two, 1
	case '3':
		return Three, 1
	case 'p', 'P':
		return Pause, 1
	case 'q', 'Q', 0x03: // q and Ctrl-C
		return Quit, 1
	}
	return None, 1
}

// DecodeAll drains a buffer into the events it holds and returns whatever tail
// could not be decoded yet.
func DecodeAll(buf []byte) (events []Event, rest []byte) {
	for len(buf) > 0 {
		if mouse, n, ok := decodeMouse(buf); ok {
			if n == 0 {
				return events, buf
			}
			events = append(events, mouse...)
			buf = buf[n:]
			continue
		}
		k, n := Decode(buf)
		if n == 0 {
			return events, buf
		}
		if k != None {
			events = append(events, Event{Key: k})
		}
		buf = buf[n:]
	}
	return events, nil
}

// decodeMouse reads one SGR mouse report - the shape run.go asks for with
// CSI ?1006h - and says whether the buffer even begins with one.
//
//	ESC [ < button ; column ; row M    a press, or a move
//	ESC [ < button ; column ; row m    a release
//
// A move reports where the pointer is; a press reports that and pulls the
// trigger. Both come out of here as separate events, because they are separate
// things: the ship follows the pointer whether or not anybody is shooting.
func decodeMouse(buf []byte) (events []Event, n int, ok bool) {
	if len(buf) < 3 || buf[0] != 0x1b || buf[1] != '[' {
		return nil, 0, false
	}
	if buf[2] != '<' {
		return nil, 0, false
	}
	// Three numbers and a letter, or not enough bytes yet.
	nums, i := [3]int{}, 3
	for f := 0; f < 3; f++ {
		start := i
		for i < len(buf) && buf[i] >= '0' && buf[i] <= '9' {
			nums[f] = nums[f]*10 + int(buf[i]-'0')
			i++
		}
		if i == start || i >= len(buf) {
			return nil, 0, true // the beginning of one, and not yet all of it
		}
		if f < 2 {
			if buf[i] != ';' {
				return nil, i + 1, true // not a report we know; swallow it
			}
			i++
		}
	}
	end := buf[i]
	i++
	if end != 'M' && end != 'm' {
		return nil, i, true
	}

	button, col, row := nums[0], nums[1]-1, nums[2]-1
	events = append(events, Event{Key: MouseAt, X: col, Y: row})

	// Bit 32 is motion with the button still down, which is a move and not a
	// second press; bit 64 is the wheel, which steers nothing here. Both have to
	// be ruled out before the low two bits mean a button at all - the wheel is
	// reported as button 64, whose low bits look exactly like the left one.
	moving, wheel := button&0b100000 != 0, button&0b1000000 != 0
	switch {
	case moving || wheel:
	case button&0b11 == 0 && end == 'M':
		events = append(events, Event{Key: Fire})
	case button&0b11 == 0 && end == 'm':
		events = append(events, Event{Key: Release})
	case button&0b11 == 2 && end == 'M':
		// The right button reloads, which is what the game this is modelled on
		// does with it.
		events = append(events, Event{Key: Rearm})
	}
	return events, i, true
}
