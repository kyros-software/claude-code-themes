package i18n

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// isolate puts the settings file somewhere disposable and clears everything
// that could otherwise answer the question first.
func isolate(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	t.Setenv(Env, "")
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_MESSAGES", "")
	t.Setenv("LANG", "")
	Use("")
	reset()
	// Current caches the file read for the life of the process; these tests
	// change the file, so they go through Setting and resolve, which do not.
	t.Cleanup(func() { Use("") })
	return dir
}

// Nothing configured is Spanish, because that is what the theme spoke before
// it spoke two languages. An upgrade must not reword a statusline by itself.
func TestWithNothingSetItIsSpanish(t *testing.T) {
	isolate(t)
	if got := Current(); got != ES {
		t.Errorf("Current() = %q with nothing configured, want %q", got, ES)
	}
}

func TestTheEnvironmentWins(t *testing.T) {
	isolate(t)
	if err := Save(ES); err != nil {
		t.Fatal(err)
	}
	t.Setenv(Env, "en")
	if got := Current(); got != EN {
		t.Errorf("CCPET_LANG=en gave %q", got)
	}
}

// Somebody who sets CCPET_LANG is very likely to paste their LANG into it.
func TestAWholeLocaleIsAcceptedAsALanguage(t *testing.T) {
	isolate(t)
	for value, want := range map[string]Lang{
		"en_GB.UTF-8": EN,
		"es_ES.UTF-8": ES,
		"EN":          EN,
		"  es  ":      ES,
	} {
		t.Setenv(Env, value)
		if got := Current(); got != want {
			t.Errorf("CCPET_LANG=%q gave %q, want %q", value, got, want)
		}
	}
}

// A language the theme does not speak is not an error and not silence: it is
// the default, the same as never having configured anything.
func TestAnUnknownLanguageFallsBack(t *testing.T) {
	isolate(t)
	t.Setenv(Env, "fr_FR.UTF-8")
	if got := Current(); got != Default {
		t.Errorf("CCPET_LANG=fr gave %q, want the default %q", got, Default)
	}
}

func TestTheFileIsRead(t *testing.T) {
	isolate(t)
	if err := Save(EN); err != nil {
		t.Fatal(err)
	}
	lang, source := Setting()
	if lang != EN {
		t.Errorf("the file says %q after saving en", lang)
	}
	if source != Path() {
		t.Errorf("the setting came from %q, want the file", source)
	}
}

// The file is the theme's, not the language's: a key written by a later
// version has to survive `ccpet lang`.
func TestSavingKeepsTheRestOfTheFile(t *testing.T) {
	dir := isolate(t)
	path := filepath.Join(dir, "ccpet.json")
	if err := os.WriteFile(path, []byte(`{"lang":"es","something":42}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Save(EN); err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if doc["lang"] != "en" {
		t.Errorf("lang = %v", doc["lang"])
	}
	if doc["something"] != 42.0 {
		t.Errorf("the other key is %v, it should still be 42", doc["something"])
	}
}

// A settings file somebody has broken is a theme with no language set, not a
// statusline that fails to draw.
func TestABrokenFileIsNoSettingAtAll(t *testing.T) {
	dir := isolate(t)
	if err := os.WriteFile(filepath.Join(dir, "ccpet.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if lang, _ := Setting(); lang != "" {
		t.Errorf("a broken file gave the setting %q", lang)
	}
	if got := resolve(""); got != Default {
		t.Errorf("resolve(\"\") = %q", got)
	}
}

// auto is stored as auto and answered from the locale every time: somebody who
// asks for the terminal to decide means every session, not this one.
func TestAutoFollowsTheLocale(t *testing.T) {
	isolate(t)
	for locale, want := range map[string]Lang{
		"es_AR.UTF-8": ES,
		"en_US.UTF-8": EN,
		"de_DE.UTF-8": EN,
	} {
		t.Setenv("LANG", locale)
		if got := resolve(Auto); got != want {
			t.Errorf("LANG=%q resolved to %q, want %q", locale, got, want)
		}
	}
}

// C and POSIX name no language, so they are skipped rather than answered.
func TestTheCLocaleIsNotAnAnswer(t *testing.T) {
	isolate(t)
	t.Setenv("LC_ALL", "C")
	t.Setenv("LANG", "es_ES.UTF-8")
	if got := resolve(Auto); got != ES {
		t.Errorf("LC_ALL=C with LANG=es gave %q, want %q", got, ES)
	}
}

func TestUsePinsOverEverything(t *testing.T) {
	isolate(t)
	t.Setenv(Env, "es")
	Use(EN)
	if got := Current(); got != EN {
		t.Errorf("Use(EN) gave %q", got)
	}
	if _, source := Setting(); source != "--lang" {
		t.Errorf("a pinned language reports %q as its source", source)
	}
	Use("")
	if got := Current(); got != ES {
		t.Errorf("after Use(\"\") the environment should be back, got %q", got)
	}
}
