package pet

import (
	"os"
	"testing"
	"time"

	"github.com/kyros-software/claude-code-themes/internal/i18n"
)

// TestMain pins the language for this package: these tests assert on wording,
// and the theme now speaks two. Spanish, because that is the default and the
// assertions were written against it - a developer with CCPET_LANG=en in their
// shell would otherwise fail every one of them.
func TestMain(m *testing.M) {
	os.Setenv(i18n.Env, string(i18n.ES))
	os.Exit(m.Run())
}

// A counter named in one language and forgotten in the other prints its raw id
// in the panel - "sessions_under_40" in a column of sentences.
func TestEveryCounterIsNamedInBothLanguages(t *testing.T) {
	for counter := range CounterNames {
		if CounterNamesEN[counter] == "" {
			t.Errorf("%s has no English name", counter)
		}
	}
	for counter := range CounterNamesEN {
		if CounterNames[counter] == "" {
			t.Errorf("%s has no Spanish name", counter)
		}
	}
}

// And every counter the tree actually reads is in the table at all, in both:
// the fork section prints these names, so a gap there is a nameless runner.
func TestTheCountersTheTreeReadsAreNamed(t *testing.T) {
	for _, lang := range []i18n.Lang{i18n.ES, i18n.EN} {
		i18n.Use(lang)
		for form, unlock := range Unlocks {
			if name := CounterName(unlock.Counter); name == unlock.Counter {
				t.Errorf("%s: %s asks for %q, which reads as its own id",
					lang, form, unlock.Counter)
			}
		}
	}
	i18n.Use("")
}

// The two repertoires have to be the same shape, not the same words: the
// do-not-repeat memory is SaidMemory deep and the "repertoire exhausted"
// restart counts on a form having more than one thing to say.
func TestBothVoicesHaveTheSameShape(t *testing.T) {
	for form, lines := range Repertoire {
		en, ok := RepertoireEN[form]
		if !ok {
			t.Errorf("%s says nothing in English", form)
			continue
		}
		if len(en) != len(lines) {
			t.Errorf("%s has %d lines in Spanish and %d in English", form, len(lines), len(en))
		}
		for i, line := range en {
			if line == "" {
				t.Errorf("%s line %d is empty in English", form, i)
			}
		}
	}
	for form := range RepertoireEN {
		if _, ok := Repertoire[form]; !ok {
			t.Errorf("%s says something in English and nothing in Spanish", form)
		}
	}
}

// Every event that opens the pet's mouth says something in both languages. An
// event with no line in one of them is a bubble that never appears there, and
// nothing anywhere would report it.
func TestEveryEventSpeaksInBothLanguages(t *testing.T) {
	events := []Event{EventHungry, EventLevelUp, EventStreak, EventCtxBlown}
	for _, lang := range []i18n.Lang{i18n.ES, i18n.EN} {
		i18n.Use(lang)
		for _, e := range events {
			if line := shared(e, 4, 3, 8); line == "" {
				t.Errorf("%s: %q says nothing", lang, e)
			}
		}
		// And a big meal, through Speak, which is the path the statusline takes.
		s := New()
		if line := Speak(s, EventBigMeal, "bughunter", time.Unix(1788200000, 0), func(int) int { return 0 }); line == "" {
			t.Errorf("%s: a big meal says nothing", lang)
		}
	}
	i18n.Use("")
}

// The names are the ids in English, which is the whole reason there is no
// English table: the tree is written in English words already.
func TestEnglishNamesAreTheIds(t *testing.T) {
	i18n.Use(i18n.EN)
	defer i18n.Use("")
	for id := range Names {
		if got := Name(id); got != id {
			t.Errorf("Name(%q) = %q in English, want the id", id, got)
		}
	}
}

// And NameIn ignores the setting, which is what the atlas test leans on.
func TestNameInIgnoresTheSetting(t *testing.T) {
	i18n.Use(i18n.EN)
	defer i18n.Use("")
	if got := NameIn(i18n.ES, "bughunter"); got != "cazabugs" {
		t.Errorf("NameIn(ES, bughunter) = %q", got)
	}
}
