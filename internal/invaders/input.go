package invaders

// Key decoding, kept away from anything that owns a file descriptor so it can
// be tested without a terminal.

// Decode reads one key out of a buffer and says how many bytes it consumed.
//
// It returns n == 0 when the buffer holds the beginning of an escape sequence
// and not yet all of it, so the caller keeps the bytes and reads more. Raw mode
// is set with VMIN=0 and VTIME=1, which means a read can and does return half of
// an arrow key - and three bytes of garbage in a game where up and down are the
// only controls is a ship that jumps across the field.
func Decode(buf []byte) (k Key, n int) {
	if len(buf) == 0 {
		return None, 0
	}
	switch buf[0] {
	case 0x1b: // ESC
		if len(buf) < 3 {
			// Either the rest is still in flight, or it is a bare Escape and
			// the timeout will hand it back with nothing after it. Waiting is
			// the safe half: the caller drops what it cannot decode.
			return None, 0
		}
		if buf[1] != '[' && buf[1] != 'O' {
			return None, 2
		}
		switch buf[2] {
		case 'A':
			return Up, 3
		case 'B':
			return Down, 3
		}
		return None, 3
	case 'k', 'w', 'K', 'W':
		return Up, 1
	case 'j', 's', 'J', 'S':
		return Down, 1
	case ' ':
		return Fire, 1
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
