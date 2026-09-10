package invaders

// Key decoding, kept away from anything that owns a file descriptor so it can
// be tested without a terminal.

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

// DecodeAll drains a buffer into the keys it holds and returns whatever tail
// could not be decoded yet.
func DecodeAll(buf []byte) (keys []Key, rest []byte) {
	for len(buf) > 0 {
		k, n := Decode(buf)
		if n == 0 {
			return keys, buf
		}
		if k != None {
			keys = append(keys, k)
		}
		buf = buf[n:]
	}
	return keys, nil
}
