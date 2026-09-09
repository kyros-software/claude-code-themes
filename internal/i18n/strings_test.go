package i18n

import (
	"reflect"
	"strings"
	"testing"
)

// A language added to the catalogue and left half written prints empty labels
// and blank help - which reads as a broken layout, not a missing translation.
// Every field, in every catalogue, or it is not a language the theme speaks.
func TestNoCatalogueIsHalfWritten(t *testing.T) {
	for name, cat := range map[string]Strings{"spanish": spanish, "english": english} {
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
			}
		}
	}
}

// The formats have to take the same arguments in both, because the call site
// is one call site: a %s that turns into a %d somewhere is a panic in one
// language and not the other.
func TestTheFormatsAgree(t *testing.T) {
	es, en := reflect.ValueOf(spanish), reflect.ValueOf(english)
	for i := 0; i < es.NumField(); i++ {
		field := es.Type().Field(i)
		if es.Field(i).Kind() != reflect.String {
			continue
		}
		a, b := verbs(es.Field(i).String()), verbs(en.Field(i).String())
		if a != b {
			t.Errorf("%s takes %q in Spanish and %q in English", field.Name, a, b)
		}
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
