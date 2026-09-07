package pet

import "github.com/kyros-software/claude-code-themes/internal/lockfile"

// Locking the life file.
//
// pet.json is global - one file for every session, every repo, every window -
// and the hook writes to it on EVERY PostToolUse. Claude Code issues tool calls
// in parallel, so two writers at once is the ordinary case.
//
// Save() has always been atomic: temp file, then rename. That rules out a torn
// or half-written file and it does NOT rule out a lost update, which is the bug
// this fixes. Save writes the whole State, so the loser of a race does not
// merely lose its own increment - it puts back everything it read a moment
// earlier, undoing the winner. Measured before the lock: 100 meals fed
// concurrently landed as 72 XP instead of 800.
//
// The mechanism is internal/lockfile, which explains the .lock file and why a
// failure to take the lock is reported rather than obeyed.

// Update is the only safe way to change the life file: it takes the lock, reads
// the state INSIDE it, hands it to change, and writes back before letting go.
// Load-then-Save from outside this function is the bug it exists to prevent.
//
// change says whether the state is worth writing. Returning false skips the
// write entirely, which is what the callers that only sometimes have something
// to record want.
func Update(path string, change func(*State) bool) bool {
	release, _ := lockfile.Take(path)
	defer release()

	s := Load(path)
	if !change(s) {
		return false
	}
	return Save(s, path)
}
