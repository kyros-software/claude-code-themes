// Package session holds the per-session scratch state, in $TMPDIR.
//
// What only becomes knowable by comparing one statusline refresh with the
// previous one - token rate, context peak, bypass turns, session length - lives
// here. The SessionEnd hook turns it into behaviour counters and deletes it.
//
// All three files share the Prefix so a single sweep reaches them all.
package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Prefix marks every file this package owns.
const Prefix = "claude-statusline-"

// MaxAge is how long a leftover survives before the sweep takes it.
const MaxAge = 24 * time.Hour

// Facts is what the statusline carries from one refresh to the next.
type Facts struct {
	Label string `json:"label"`

	// CtxPeak is the highest the context window got. It is the ONE peak, which
	// it has not always been: while the state read the tightest of the three
	// necks there was a `peak` beside it carrying that, and the counters named
	// for the context - ctx_low, sessions_under_40, ctx100_sessions, and the
	// two ember rungs in the hook - had to be careful to read this one, or they
	// would credit "3 sessions touching 100% of context" to somebody who never
	// filled the window and only ran out of 5h quota. Two of them were not
	// careful. The state is the context again, so there is nothing to be
	// careful about: see pet.ContextLoad.
	CtxPeak float64 `json:"ctx_peak"`

	T0 int64 `json:"t0"`

	// Active is the session's working time in seconds, and Seen the last moment
	// anything happened in it. Every write of these facts is activity - a new
	// turn, the model answering, the context moving - and the gap since Seen is
	// added to Active unless it is longer than IdleGap. The wall clock from T0
	// to SessionEnd counted a terminal left open overnight as a marathon:
	// measured, 164 of some 650 sessions on one machine were "4 h or more".
	Active   int64    `json:"active"`
	Seen     int64    `json:"seen"`
	Repo     string   `json:"repo"`
	PromptID string   `json:"prompt_id"`
	APIMs    *float64 `json:"api_ms"`
	TPS      *float64 `json:"tps"`
	TPSAt    float64  `json:"tps_at"`

	// Structured says the file actually carried session facts. A file with
	// only a bare label is one this version knows nothing about, and closing a
	// session on it would hand out counters for free.
	Structured bool `json:"-"`
}

// asFloat reads a number out of the raw document. It exists for the legacy
// `peak` key, which no longer has a field of its own to unmarshal into.
func asFloat(v any) *float64 {
	f, ok := v.(float64)
	if !ok || f != f {
		return nil
	}
	return &f
}

// IdleGap is the longest pause that still counts as working time.
const IdleGap = 15 * 60

// Touch books activity at now: the gap since the last activity is working
// time unless it is longer than IdleGap, which is somebody away from the desk.
func (f *Facts) Touch(now int64) {
	if f.Seen > 0 {
		if gap := now - f.Seen; gap > 0 && gap <= IdleGap {
			f.Active += gap
		}
	}
	if now > f.Seen {
		f.Seen = now
	}
}

// SafeID whitelists a session id, because it becomes part of a path.
func SafeID(id string) string {
	if id == "" || len(id) > 64 {
		return ""
	}
	for _, r := range id {
		ok := r == '_' || r == '-' ||
			(r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		if !ok {
			return ""
		}
	}
	return id
}

// TmpDir is where the scratch files live.
func TmpDir() string {
	if d := os.Getenv("TMPDIR"); d != "" {
		return d
	}
	return "/tmp"
}

// PathFor is the scratch file for a session. kind "" is the statusline's own.
func PathFor(id, kind string) string {
	safe := SafeID(id)
	if safe == "" {
		return ""
	}
	if kind != "" {
		kind += "-"
	}
	return filepath.Join(TmpDir(), Prefix+kind+safe)
}

// Load never fails. A file that is not json is read as a bare label, which is
// what the very first version of this wrote.
func Load(path string) Facts {
	var f Facts
	if path == "" {
		return f
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return f
	}
	text := strings.TrimSpace(string(raw))
	if text == "" {
		return f
	}
	if !strings.HasPrefix(text, "{") {
		f.Label = text
		return f
	}
	var doc map[string]any
	if json.Unmarshal([]byte(text), &doc) != nil {
		return Facts{}
	}
	// v1 keys, kept readable so a session that spans an upgrade is not lost.
	if v, ok := doc["etq"]; ok {
		doc["label"] = v
	}
	if v, ok := doc["pico"]; ok {
		doc["peak"] = v
	}
	if v, ok := doc["api"]; ok {
		doc["api_ms"] = v
	}
	if v, ok := doc["pid"]; ok {
		doc["prompt_id"] = v
	}
	if v, ok := doc["tps_t"]; ok {
		doc["tps_at"] = v
	}
	repacked, err := json.Marshal(doc)
	if err != nil {
		return Facts{}
	}
	if json.Unmarshal(repacked, &f) != nil {
		return Facts{}
	}
	_, hasPeak := doc["peak"]
	_, hasT0 := doc["t0"]
	f.Structured = hasPeak || hasT0
	// A file older than the split between the two peaks carries only `peak`,
	// and back then `peak` WAS the context. Falling back to it keeps a session
	// that spans an upgrade from reading as a context that never rose above
	// zero, which would hand out ctx_low and sessions_under_40 for free. A file
	// from the neck versions carries both, and its `ctx_peak` is the right one
	// to take, which is exactly what this does.
	if _, has := doc["ctx_peak"]; !has {
		if v := asFloat(doc["peak"]); v != nil {
			f.CtxPeak = *v
		}
	}
	return f
}

// Save writes the facts. A failure is not worth reporting: the next refresh
// will try again.
//
// Temp file then rename, the way pet.Save has always done it, and not the bare
// os.WriteFile this used to be. These files live in $TMPDIR, which on a shared
// box is a directory other people can write to: a plain WriteFile FOLLOWS a
// symlink planted at the path and truncates whatever is on the far end. The
// name has a session UUID in it so nobody is guessing it, but a writer that
// cannot be aimed elsewhere costs three lines and settles the question.
func Save(path string, f Facts) bool {
	if path == "" {
		return false
	}
	raw, err := json.Marshal(f)
	if err != nil {
		return false
	}
	return WriteAtomic(path, raw)
}

// WriteAtomic puts bytes at path without ever writing THROUGH what is already
// there. O_EXCL on the temp file means it is ours or it does not exist, and the
// rename replaces the destination whatever it was - symlink included, which is
// then simply gone rather than followed.
func WriteAtomic(path string, raw []byte) bool {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+"-*")
	if err != nil {
		return false
	}
	name := tmp.Name()
	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		os.Remove(name)
		return false
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		os.Remove(name)
		return false
	}
	if tmp.Close() != nil || os.Rename(name, path) != nil {
		os.Remove(name)
		return false
	}
	return true
}

// Sweep drops the leftovers of sessions that died without a SessionEnd.
//
// It matches on the name and the age, so on a shared $TMPDIR it can in
// principle line up with somebody else's file. Two things keep that harmless:
// /tmp carries the sticky bit, so the kernel refuses the unlink of a file we do
// not own, and the Lstat below means a symlink is never followed to something
// outside the directory - only the link itself would go.
func Sweep(now time.Time) {
	dir := TmpDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	cutoff := now.Add(-MaxAge)
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), Prefix) {
			continue
		}
		path := filepath.Join(dir, e.Name())
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() || !info.ModTime().Before(cutoff) {
			continue
		}
		os.Remove(path)
	}
}
