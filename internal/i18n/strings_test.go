package i18n

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

// catalogues is every catalogue struct in this package, paired by language.
// Both guards below walk it, so a catalogue added later is one line here instead
// of an edit to two reflective tests - which is how the second one nearly went
// uncovered: Game is not a Strings and could not be dropped into the
// map[string]Strings this used to be.
func catalogues() []map[string]any {
	return []map[string]any{
		{"spanish": spanish, "english": english},
		{"spanish": spanishGame, "english": englishGame},
	}
}

// A language added to the catalogue and left half written prints empty labels
// and blank help - which reads as a broken layout, not a missing translation.
// Every field, in every catalogue, or it is not a language the theme speaks.
func TestNoCatalogueIsHalfWritten(t *testing.T) {
	for _, set := range catalogues() {
		for name, cat := range set {
			v := reflect.ValueOf(cat)
			for i := 0; i < v.NumField(); i++ {
				field := v.Type().Field(i)
				switch value := v.Field(i); value.Kind() {
				case reflect.String:
					if strings.TrimSpace(value.String()) == "" {
						t.Errorf("%s: %s is empty", name, field.Name)
					}
				case reflect.Slice:
					if value.Len() == 0 {
						t.Errorf("%s: %s is empty", name, field.Name)
					}
				case reflect.Map:
					// A map was invisible to this guard until the game's
					// Families and Traits arrived: an empty map passed, and so
					// did every blank value inside a full one. Both print
					// nothing, which is the whole thing this test is for.
					if value.Len() == 0 {
						t.Errorf("%s: %s is empty", name, field.Name)
					}
					for _, key := range value.MapKeys() {
						entry := value.MapIndex(key)
						if entry.Kind() == reflect.String && strings.TrimSpace(entry.String()) == "" {
							t.Errorf("%s: %s[%v] is empty", name, field.Name, key)
						}
					}
				}
			}
		}
	}
}

// The formats have to take the same arguments in both, because the call site
// is one call site: a %s that turns into a %d somewhere is a panic in one
// language and not the other.
func TestTheFormatsAgree(t *testing.T) {
	for _, set := range catalogues() {
		es, en := reflect.ValueOf(set["spanish"]), reflect.ValueOf(set["english"])
		for i := 0; i < es.NumField(); i++ {
			field := es.Type().Field(i)
			if es.Field(i).Kind() != reflect.String {
				continue
			}
			a, b := verbs(es.Field(i).String()), verbs(en.Field(i).String())
			if a != b {
				t.Errorf("%s.%s takes %q in Spanish and %q in English",
					es.Type().Name(), field.Name, a, b)
			}
		}
	}
}

// The two maps in Game are keyed by ids the game invents, and a key that exists
// in one language and not the other is a name that prints blank in exactly one
// of them - which no amount of reading the catalogue top to bottom would catch.
func TestTheMapsHaveTheSameKeysInBothLanguages(t *testing.T) {
	for _, set := range catalogues() {
		es, en := reflect.ValueOf(set["spanish"]), reflect.ValueOf(set["english"])
		for i := 0; i < es.NumField(); i++ {
			if es.Field(i).Kind() != reflect.Map {
				continue
			}
			name := es.Type().Field(i).Name
			for _, key := range es.Field(i).MapKeys() {
				if !en.Field(i).MapIndex(key).IsValid() {
					t.Errorf("%s[%v] is in Spanish and not in English", name, key)
				}
			}
			for _, key := range en.Field(i).MapKeys() {
				if !es.Field(i).MapIndex(key).IsValid() {
					t.Errorf("%s[%v] is in English and not in Spanish", name, key)
				}
			}
		}
	}
}

// And a guard on the guards: a catalogue struct declared in this package and
// left out of catalogues() is covered by nothing at all, which is the exact
// hole Game would have fallen into.
func TestEveryCatalogueStructIsInTheGuardList(t *testing.T) {
	declared := 0
	for _, file := range []string{"strings.go", "game.go"} {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, line := range strings.Split(string(raw), "\n") {
			if strings.HasPrefix(line, "type ") && strings.HasSuffix(line, " struct {") {
				declared++
			}
		}
	}
	if declared != len(catalogues()) {
		t.Errorf("%d catalogue structs are declared and %d are guarded",
			declared, len(catalogues()))
	}
}

// verbs is the format's arguments, in order: "%dh %02dm" is "dd".
func verbs(format string) string {
	var out []byte
	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			continue
		}
		i++
		for i < len(format) && strings.ContainsRune("0123456789.+-# ", rune(format[i])) {
			i++
		}
		if i < len(format) && format[i] != '%' {
			out = append(out, format[i])
		}
	}
	return string(out)
}

func TestDaysAreSpelledOutAndThenCounted(t *testing.T) {
	for _, c := range []struct {
		lang Lang
		n    int
		want string
	}{
		{ES, 0, "cero días"},
		{ES, 1, "un día"},
		{ES, 5, "cinco días"},
		{ES, 12, "12 días"},
		{EN, 1, "one day"},
		{EN, 5, "five days"},
		{EN, 12, "12 days"},
	} {
		Use(c.lang)
		if got := S().NDays(c.n); got != c.want {
			t.Errorf("%s NDays(%d) = %q, want %q", c.lang, c.n, got, c.want)
		}
	}
	Use("")
}
