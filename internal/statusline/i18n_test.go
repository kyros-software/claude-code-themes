package statusline

import (
	"os"
	"strings"
	"testing"

	"github.com/kyros-software/claude-code-themes/internal/i18n"
	"github.com/kyros-software/claude-code-themes/internal/pet"
	"github.com/kyros-software/claude-code-themes/internal/theme"
)

// TestMain pins the language for this package: these tests assert on wording,
// and the theme now speaks two. Spanish, because that is the default and the
// assertions were written against it - a developer with CCPET_LANG=en in their
// shell would otherwise fail every one of them.
func TestMain(m *testing.M) {
	os.Setenv(i18n.Env, string(i18n.ES))
	os.Exit(m.Run())
}

// Band 4 carries the one word the footer spells out, and it used to spell it
// in Spanish whatever the theme was set to. The rest of the band is names and
// numbers, so this is the whole of the statusline's translation.
func TestBandFourNamesTheLevelInTheSetLanguage(t *testing.T) {
	defer i18n.Use("")
	for lang, want := range map[i18n.Lang]string{
		i18n.ES: "bughunter nivel 4 │ fresca ✦",
		i18n.EN: "bughunter level 4 │ fresh ✦",
	} {
		i18n.Use(lang)
		band := petBand(Card{
			Form:  "bughunter",
			Level: 4,
			Done:  433,
			Span:  500,
			State: pet.Name("fresh") + " ✦",
			Vital: pet.StateFor(20),
		}, 140)
		if got := strings.TrimSpace(theme.Strip(assemble(band, 140))); got != want {
			t.Errorf("%s: band 4 is %q, want %q", lang, got, want)
		}
	}
}

// The whole card in English: the trade, the mark it wears and the state all
// come through pet.Name, so one forgotten call is one Spanish word left in an
// otherwise English footer.
func TestTheCardIsEnglishThroughout(t *testing.T) {
	i18n.Use(i18n.EN)
	defer i18n.Use("")
	for _, id := range []string{"bughunter", "bloodhound", "fresh", "drowning"} {
		if name := pet.Name(id); name != id {
			t.Errorf("pet.Name(%q) = %q with the theme in English", id, name)
		}
	}
}
