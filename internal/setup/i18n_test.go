package setup

import (
	"io"
	"os"
	"strings"
	"testing"

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

// Every message setup prints, in English, through the three commands that
// print them. The catalogue's own test proves no field is empty; this one
// proves the fields are actually reached - a Fprintln left with its literal
// would pass the first and fail this.
func TestSetupSpeaksEnglish(t *testing.T) {
	home(t)
	i18n.Use(i18n.EN)
	defer i18n.Use("")

	var off strings.Builder
	if err := Status(&off); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(off.String(), "statusline: off") ||
		!strings.Contains(off.String(), "food hooks in settings.json: none") {
		t.Errorf("status, off:\n%s", off.String())
	}

	var on strings.Builder
	if err := On(&on, runtimeRoot(t)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(on.String(), "statusline on ->") ||
		!strings.Contains(on.String(), "pick the theme too") {
		t.Errorf("on:\n%s", on.String())
	}

	var again strings.Builder
	if err := Off(io.Discard); err != nil {
		t.Fatal(err)
	}
	if err := Off(&again); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(again.String(), "was already off") {
		t.Errorf("off twice:\n%s", again.String())
	}

	// And nothing Spanish survives any of them.
	all := off.String() + on.String() + again.String()
	for _, unwanted := range []string{"apagada", "encendida", "comida", "tema"} {
		if strings.Contains(all, unwanted) {
			t.Errorf("setup still says %q in English:\n%s", unwanted, all)
		}
	}
}

// The line that reports what was wired names the events one by one, in both
// languages, so it goes stale the moment a fourth is added - and it did, the day
// Stop arrived for the game's auto-pause. Driven off hookEvents so it cannot
// drift again.
func TestTheHooksWiredLineNamesEveryEventItWired(t *testing.T) {
	for _, lang := range []i18n.Lang{i18n.ES, i18n.EN} {
		i18n.Use(lang)
		line := i18n.S().HooksWired
		for _, event := range hookEvents {
			if !strings.Contains(line, event) {
				t.Errorf("%s: %q does not name %s", lang, line, event)
			}
		}
	}
	i18n.Use("")
}
