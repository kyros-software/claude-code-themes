package invaders

// The randomness, so that the tick can stay a pure function.

// next advances a splitmix64 state and returns the next value.
//
// Not math/rand: a *rand.Rand is a pointer to mutable state, and a tick that
// takes a state by value and returns a new one cannot hold one without two
// copies of a run sharing the same stream. Sixty-four bits of seed in the save
// file, four lines of arithmetic, and a resumed run plays the sequence it was in.
func next(r *uint64) uint64 {
	*r += 0x9E3779B97F4A7C15
	z := *r
	z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
	z = (z ^ (z >> 27)) * 0x94D049BB133111EB
	return z ^ (z >> 31)
}

// roll is a number in [0, n). Zero and below give zero, because a caller asking
// for a choice out of nothing is a bug that must not be a panic mid-frame.
func roll(r *uint64, n int) int {
	if n <= 0 {
		return 0
	}
	return int(next(r) % uint64(n))
}

// chance is true n times in a hundred.
func chance(r *uint64, percent int) bool { return roll(r, 100) < percent }
