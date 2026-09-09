// Package i18n is what language the theme speaks, and where that was decided.
//
// Everything the runtime prints that is not a number used to be Spanish, hard
// coded at the point of printing: the pet's names, its voice, the panel's row
// labels, the setup messages and the help. That is the theme's own wording and
// it stays - but it was also the ONLY wording, which made the whole statusline
// unusable to anyone who does not read Spanish.
//
// The language is resolved once and read everywhere. Nothing prints a literal
// any more; the packages that have text keep their two versions of it and ask
// here which one to use.
package i18n

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kyros-software/claude-code-themes/internal/config"
)

// Lang is a language the theme speaks.
type Lang string

// The two of them, plus the word that means "work it out from the locale".
// Auto is never the ANSWER - Current always returns ES or EN - it is only ever
// something a person can write in the setting.
const (
	ES   Lang = "es"
	EN   Lang = "en"
	Auto Lang = "auto"
)

// Default is what the theme speaks when nobody has said otherwise.
//
// Spanish, because that is what it spoke before this package existed and an
// upgrade must not silently reword a statusline somebody is used to. English
// is one `ccpet lang en` away, and `ccpet lang auto` hands the decision to the
// locale for anyone who would rather it followed the terminal.
const Default = ES

// Env is the variable that overrides everything, for a single session or a
// single command: `CCPET_LANG=en ccpet`.
const Env = "CCPET_LANG"

var (
	pinned Lang // set by Use, wins over the environment

	// The settings file, read once per process. `ccpet statusline` is a fresh
	// process on every refresh, so this caches nothing across time - it only
	// keeps one panel, which asks S() a few dozen times, from opening the same
	// file a few dozen times.
	cacheM sync.Mutex
	cached Lang
	read1  bool
)

// Current is the language to print in. ES or EN, never Auto.
//
// The environment is re-read on every call because it costs nothing and
// because a test that sets it has to be believed immediately; the file is read
// once per process, since the statusline refreshes about once a second and
// none of them are going to find a different answer.
func Current() Lang {
	if pinned != "" {
		return pinned
	}
	if l, ok := parse(os.Getenv(Env)); ok {
		return resolve(l)
	}
	cacheM.Lock()
	defer cacheM.Unlock()
	if !read1 {
		cached, read1 = read(), true
	}
	return resolve(cached)
}

// reset drops the cached file, for tests that write a new one.
func reset() {
	cacheM.Lock()
	cached, read1 = "", false
	cacheM.Unlock()
}

// Use pins the language for the rest of the process, whatever the environment
// and the file say. `ccpet lang en <command>` uses it, and so does every test
// that asserts on wording.
func Use(l Lang) {
	if l == "" {
		pinned = ""
		return
	}
	pinned = resolve(l)
}

// Setting is what has been configured and where, for `ccpet lang` to print.
// The language may be Auto here; that is the point of the question.
func Setting() (lang Lang, source string) {
	if pinned != "" {
		return pinned, "--lang"
	}
	if l, ok := parse(os.Getenv(Env)); ok {
		return l, Env
	}
	if l := read(); l != "" {
		return l, Path()
	}
	return "", "default"
}

// Path is the file the setting lives in: ~/.claude/ccpet.json, beside the
// pet's own pet.json and under CLAUDE_CONFIG_DIR like everything else.
func Path() string { return filepath.Join(config.Dir(), "ccpet.json") }

// Save writes the setting. Auto is a value like any other - it is stored, and
// resolved on the way out - and the empty language deletes the key, which is
// how a setting is unset without deleting a file that may hold other keys.
//
// Unknown keys survive: this file is small now and will not stay that way.
func Save(l Lang) error {
	path := Path()
	doc := map[string]any{}
	if raw, err := os.ReadFile(path); err == nil {
		if json.Unmarshal(raw, &doc) != nil || doc == nil {
			doc = map[string]any{}
		}
	}
	if l == "" {
		delete(doc, "lang")
	} else {
		doc["lang"] = string(l)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(doc, "", " ")
	if err != nil {
		return err
	}
	// Written whole and renamed into place, the way pet.json is: a statusline
	// reading this file mid-write must never see half of it.
	tmp, err := os.CreateTemp(filepath.Dir(path), ".ccpet-*.json")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := tmp.Write(append(raw, '\n')); err != nil {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return err
	}
	if err := os.Rename(name, path); err != nil {
		os.Remove(name)
		return err
	}
	// The process has already answered Current() at least once by now; without
	// this, `ccpet lang en` would print its confirmation in Spanish.
	cacheM.Lock()
	cached, read1 = l, true
	cacheM.Unlock()
	return nil
}

// read is the file, and a broken file is no setting at all rather than an
// error: a language is not worth failing a statusline over.
func read() Lang {
	raw, err := os.ReadFile(Path())
	if err != nil {
		return ""
	}
	var doc struct {
		Lang string `json:"lang"`
	}
	if json.Unmarshal(raw, &doc) != nil {
		return ""
	}
	l, _ := parse(doc.Lang)
	return l
}

// parse accepts what a person would plausibly write: "en", "EN", "auto", and
// a whole locale like "es_ES.UTF-8", which is what somebody copying their LANG
// into CCPET_LANG ends up with.
func parse(s string) (Lang, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	switch {
	case s == "":
		return "", false
	case s == string(Auto):
		return Auto, true
	case strings.HasPrefix(s, "es"):
		return ES, true
	case strings.HasPrefix(s, "en"):
		return EN, true
	}
	// Anything else is a language this theme does not speak. Not an error, and
	// not silence either: it falls through to the default, the same as a file
	// that was never written.
	return "", false
}

// resolve turns a setting into a language to print in.
func resolve(l Lang) Lang {
	switch l {
	case ES, EN:
		return l
	case Auto:
		return fromLocale()
	}
	return Default
}

// fromLocale reads the terminal's own idea of the language, in the order the
// C library reads it. Only a locale that actually names Spanish gives Spanish;
// C and POSIX name no language at all, so they are skipped rather than treated
// as an answer.
func fromLocale() Lang {
	for _, name := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		v := strings.TrimSpace(os.Getenv(name))
		if v == "" || v == "C" || v == "POSIX" || strings.HasPrefix(v, "C.") {
			continue
		}
		if strings.HasPrefix(strings.ToLower(v), "es") {
			return ES
		}
		return EN
	}
	return Default
}
