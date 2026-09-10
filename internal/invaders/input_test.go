package invaders

import (
	"bytes"
	"math/rand"
	"testing"
)

// Every key the help row promises has to work, including all four arrows, the vi
// set and the wasd set - a game where up is only ever "k" is a game nobody can
// pick up.
//
// The one disagreement is `s`: wasd wants it for down and this uses it for the
// brake, because a ship that latches needs one and the down arrow and `j` both
// already say down.
func TestEveryKeyTheHelpRowPromisesIsDecoded(t *testing.T) {
	for _, c := range []struct {
		in   string
		want Key
	}{
		{"\033[D", Left}, {"\033OD", Left}, {"a", Left}, {"h", Left}, {"A", Left},
		{"\033[C", Right}, {"\033OC", Right}, {"d", Right}, {"l", Right}, {"L", Right},
		{"\033[B", Down}, {"\033OB", Down}, {"j", Down}, {"J", Down},
		{"\033[A", Up}, {"\033OA", Up}, {"w", Up}, {"k", Up}, {"K", Up},
		{"s", Stop}, {"S", Stop},
		{" ", Fire}, {"x", Ability}, {"z", Ability},
		{"r", Rearm}, {"e", Heal}, {"f", Heal},
		{"1", One}, {"2", Two}, {"3", Three},
		{"p", Pause}, {"P", Pause},
		{"q", Quit}, {"Q", Quit}, {"\003", Quit},
	} {
		got, n := Decode([]byte(c.in))
		if got != c.want {
			t.Errorf("%q decoded to %d, want %d", c.in, got, c.want)
		}
		if n != len(c.in) {
			t.Errorf("%q consumed %d bytes of %d", c.in, n, len(c.in))
		}
	}
}

// Raw mode is VMIN 0 with VTIME 1, so a read comes back after a tenth of a
// second with whatever arrived - which is regularly one or two bytes of a
// three-byte arrow key. Decoding those as separate keys turns one press of the
// up arrow into an escape, a bracket and an A, and in a game steered with four
// arrows that is a ship jumping across the field.
func TestAnEscapeSequenceSplitAcrossTwoReadsIsOneKeyAndNotThree(t *testing.T) {
	full := []byte("\033[D")
	for cut := 1; cut < len(full); cut++ {
		head, tail := full[:cut], full[cut:]

		keys, rest := DecodeAll(head)
		if len(keys) != 0 {
			t.Errorf("cut at %d: the first half already decoded to %v", cut, keys)
		}
		if !bytes.Equal(rest, head) {
			t.Errorf("cut at %d: the first half was eaten rather than kept", cut)
		}

		keys, rest = DecodeAll(append(rest, tail...))
		if len(keys) != 1 || keys[0] != Left {
			t.Errorf("cut at %d: the two halves decoded to %v", cut, keys)
		}
		if len(rest) != 0 {
			t.Errorf("cut at %d: %q was left over", cut, rest)
		}
	}
}

// A whole burst decodes to the keys it holds, in order, which is what the reader
// goroutine hands the loop.
func TestABurstOfKeysDecodesInOrder(t *testing.T) {
	keys, rest := DecodeAll([]byte("aa\033[Cp q"))
	want := []Key{Left, Left, Right, Pause, Fire, Quit}
	if len(rest) != 0 {
		t.Errorf("%q was left over", rest)
	}
	if len(keys) != len(want) {
		t.Fatalf("got %v, want %v", keys, want)
	}
	for i := range want {
		if keys[i] != want[i] {
			t.Errorf("key %d is %d, want %d", i, keys[i], want[i])
		}
	}
}

// Anything at all can arrive on a terminal - a paste, a mouse report, a broken
// UTF-8 tail - and none of it may panic or lock the decoder up.
func TestJunkNeverPanicsAndNeverLies(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for i := 0; i < 20000; i++ {
		buf := make([]byte, r.Intn(8))
		for j := range buf {
			buf[j] = byte(r.Intn(256))
		}
		k, n := Decode(buf)
		if len(buf) == 0 {
			continue
		}
		if n < 0 || n > len(buf) {
			t.Fatalf("%q consumed %d of %d", buf, n, len(buf))
		}
		// The only thing allowed to consume nothing is an unfinished escape,
		// because anything else that consumed nothing would spin for ever.
		if n == 0 && buf[0] != 0x1b {
			t.Fatalf("%q consumed nothing and does not start with an escape", buf)
		}
		if k > Quit {
			t.Fatalf("%q decoded to key %d, which does not exist", buf, k)
		}
	}
}

// A long paste of junk still terminates. DecodeAll's loop only advances when
// Decode consumes something, so this is the shape that would hang.
func TestDecodingJunkAlwaysTerminates(t *testing.T) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		r := rand.New(rand.NewSource(2))
		buf := make([]byte, 4096)
		for i := range buf {
			buf[i] = byte(r.Intn(256))
		}
		DecodeAll(buf)
	}()
	<-done
}

// Ctrl-C has to quit from inside the game, because ISIG is off in raw mode:
// nothing else is going to turn it into a signal, and a game you cannot leave
// with the key everybody reaches for is a game that has to be killed.
func TestCtrlCQuitsBecauseIsigIsOff(t *testing.T) {
	if k, _ := Decode([]byte{0x03}); k != Quit {
		t.Errorf("ctrl-c decoded to %d, want quit", k)
	}
	src := readSource(t, "term_unix.go")
	if !contains(src, "syscall.ISIG") {
		t.Error("raw mode does not clear ISIG, so ctrl-c never reaches Decode")
	}
}

func contains(haystack, needle string) bool {
	return bytes.Contains([]byte(haystack), []byte(needle))
}
