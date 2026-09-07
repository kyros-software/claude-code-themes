package panel

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kyros-software/claude-code-themes/internal/pet"
)

// The CLI surface. Run is what every hook and every `ccpet <something>` lands
// on, and it had no test: the argument handling, the exit codes and the meal
// path were all unmeasured.

func run(t *testing.T, statePath string, args ...string) (string, string, int) {
	t.Helper()
	var out, errOut bytes.Buffer
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	code := Run(args, &out, &errOut, statePath, now)
	return out.String(), errOut.String(), code
}

func statePath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "pet.json")
}

func TestAMealIsEatenAndReported(t *testing.T) {
	path := statePath(t)
	out, _, code := run(t, path, "commit", "arreglado el suelo")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out, "+12 xp") {
		t.Errorf("la salida no dice lo que comio: %q", out)
	}
	s := pet.Load(path)
	if s.XP != 12 {
		t.Errorf("xp = %d", s.XP)
	}
	if s.Counters["methodical"] != 1 || s.Counters["diff_streak"] != 1 {
		t.Errorf("el commit no alimento sus habitos: %v", s.Counters)
	}
}

// A meal on cooldown is not an error, it is a sentence.
func TestAMealOnCooldownSaysWhenNotFails(t *testing.T) {
	path := statePath(t)
	if _, _, code := run(t, path, "feed"); code != 0 {
		t.Fatalf("la primera comida salio con %d", code)
	}
	out, _, code := run(t, path, "feed")
	if code != 0 {
		t.Errorf("exit %d, comer con cooldown no es un fallo", code)
	}
	if !strings.Contains(out, "le toca en") {
		t.Errorf("no dice cuando le toca: %q", out)
	}
}

func TestSomethingThatIsNotFoodIsAUsageError(t *testing.T) {
	_, errOut, code := run(t, statePath(t), "ensalada")
	if code != 2 {
		t.Errorf("exit %d, se esperaba 2", code)
	}
	if !strings.Contains(errOut, "no es comida") || !strings.Contains(errOut, "commit") {
		t.Errorf("el error no ayuda: %q", errOut)
	}
}

func TestCountAndRecordMoveTheRightCounters(t *testing.T) {
	path := statePath(t)
	run(t, path, "count", "tests")      // sin numero: uno
	run(t, path, "count", "tests", "4") // con numero
	run(t, path, "record", "longest_plan", "7")
	run(t, path, "record", "longest_plan", "3") // un maximo no baja

	s := pet.Load(path)
	if s.Counters["tests"] != 5 {
		t.Errorf("tests = %d, se esperaba 5", s.Counters["tests"])
	}
	if s.Counters["longest_plan"] != 7 {
		t.Errorf("longest_plan = %d, se esperaba 7", s.Counters["longest_plan"])
	}
}

func TestTheArgumentGuards(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want int
	}{
		{[]string{"count"}, 2},            // sin contador
		{[]string{"record"}, 2},           // sin contador
		{[]string{"count", "x", "no"}, 2}, // numero ilegible
		{[]string{"day"}, 2},              // sin marca
		{[]string{"session"}, 0},          // sin id: no hay nada que cerrar
	} {
		if _, _, code := run(t, statePath(t), tc.args...); code != tc.want {
			t.Errorf("%v -> %d, se esperaba %d", tc.args, code, tc.want)
		}
	}
}

// An evolution is announced, and only when it happens.
func TestAMealThatEvolvesSaysSo(t *testing.T) {
	path := statePath(t)
	s := pet.New()
	s.XP = 59 // one meal short of level 2
	s.Counters = map[string]int{"methodical": 9}
	pet.Save(s, path)

	out, _, _ := run(t, path, "commit")
	if !strings.Contains(out, "evoluciona") {
		t.Errorf("no anuncio la evolucion: %q", out)
	}
	// The next meal does not evolve it, and must not say it did.
	out2, _, _ := run(t, path, "compact")
	if strings.Contains(out2, "evoluciona") {
		t.Errorf("anuncio una evolucion que no hubo: %q", out2)
	}
}

func TestRoughlyReadsLikeAPersonSaysIt(t *testing.T) {
	for _, tc := range []struct {
		d    time.Duration
		want string
	}{
		{30 * time.Second, "menos de un minuto"},
		{time.Minute, "1 min"},
		{59 * time.Minute, "59 min"},
		{time.Hour, "1h"},
		{90 * time.Minute, "1h 30m"},
		{4*time.Hour + 5*time.Minute, "4h 05m"},
	} {
		if got := roughly(tc.d); got != tc.want {
			t.Errorf("roughly(%v) = %q, se esperaba %q", tc.d, got, tc.want)
		}
	}
}
