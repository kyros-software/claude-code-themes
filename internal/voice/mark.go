package voice

// The tie-break: which of two or three branches a session wears.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/kyros-software/claude-code-themes/internal/pet"
	"github.com/kyros-software/claude-code-themes/internal/session"
)

// lines is the fourteen lines of the tree as the model is offered them: the
// mark's id, and the kind of work it is the shape of. In the tree's own order.
var lines = []struct{ mark, work string }{
	{"surgeon", "cambios pequeños y precisos, un arreglo quirúrgico, commits limpios"},
	{"weaver", "cambios anchos que tocan muchos ficheros a la vez, refactors grandes"},
	{"monk", "sesión tranquila y ligera, leer y preguntar con poco ruido"},
	{"gardener", "documentación, README, comentarios, cuidar lo que ya existe"},
	{"bloodhound", "depurar: seguir el rastro de un fallo y reproducirlo antes de arreglarlo"},
	{"exterminator", "tests: escribirlos, pasarlos, dejar la suite en verde"},
	{"cartographer", "planificar: repartir un trabajo largo en tareas y recorrerlas"},
	{"oracle", "explorar y diseñar antes de tocar código, investigar, decidir"},
	{"bolt", "algo rápido: entrar, un cambio corto y salir"},
	{"sniper", "una sola tarea con una sola herramienta, directo al blanco"},
	{"ox", "sesión muy larga, horas de trabajo seguido sobre lo mismo"},
	{"mole", "cavar hondo en un mismo sitio: un módulo, un fichero, una y otra vez"},
	{"gremlin", "tocar el sistema: instalar, configurar, scripts, automatizar a lo bruto"},
	{"kraken", "demasiadas cosas a la vez, el contexto desbordado"},
}

// chooseMark puts the rules' doubt to the model: which of these few lines of
// the tree fits the session. "" if the answer is not one of them.
func chooseMark(c Context, f File, among, taken []string) string {
	offered := map[string]bool{}
	for _, mark := range among {
		offered[mark] = true
	}
	var b strings.Builder
	b.WriteString("Formas posibles:\n")
	for _, l := range lines {
		if offered[l.mark] {
			fmt.Fprintf(&b, "- %s (%s): %s\n", l.mark, pet.Name(l.mark), l.work)
		}
	}
	b.WriteString("\nEsta sesión:\n")
	if c.Repo != "" {
		b.WriteString("- repo: " + c.Repo + "\n")
	}
	if len(f.Trail) > 0 {
		b.WriteString("- últimas herramientas, de la más antigua a la más reciente: " + strings.Join(f.Trail, ", ") + "\n")
	} else if step := c.step(); step != "" {
		b.WriteString("- lo último que ha hecho: " + step + "\n")
	}
	if c.Label != "" {
		b.WriteString("- el contexto está: " + pet.Name(c.Label) + "\n")
	}
	if c.Active >= 60 {
		fmt.Fprintf(&b, "- lleva %s de trabajo\n", span(c.Active))
	}
	if len(taken) > 0 {
		b.WriteString("\nOtras sesiones abiertas llevan: " + strings.Join(taken, ", ") +
			". Si dudas entre dos, elige la que no lleve ninguna.\n")
	}
	b.WriteString("\nResponde solo con el identificador.")

	out, err := ask("Eliges qué forma toma la mascota de la statusline de un programador, según el trabajo "+
		"que está haciendo esta sesión. Respondes con UNA palabra: el identificador de la lista, tal cual, "+
		"sin explicar nada.", b.String())
	if err != nil {
		return ""
	}
	if picked := parseMark(out); offered[picked] {
		return picked
	}
	return ""
}

// parseMark finds the one line the answer names, by id or by its Spanish name.
// An answer that names none, or two, is no answer.
func parseMark(out string) string {
	words := strings.FieldsFunc(strings.ToLower(firstLine(out)), func(r rune) bool {
		return !unicode.IsLetter(r)
	})
	found := ""
	for _, word := range words {
		for _, l := range lines {
			if word == l.mark || word == pet.Name(l.mark) {
				if found != "" && found != l.mark {
					return ""
				}
				found = l.mark
			}
		}
	}
	return found
}

// others is the lines the other live sessions wear, so that they can be
// spread: four windows open should not be four of the same creature because
// each was read alone. The rules weigh by it, and the model is told.
func others(mine string, now time.Time) []string {
	paths, _ := filepath.Glob(filepath.Join(session.TmpDir(), session.Prefix+"voice-*"))
	seen := map[string]bool{}
	var out []string
	for _, path := range paths {
		if path == mine || strings.HasSuffix(path, ".lock") {
			continue
		}
		info, err := os.Stat(path)
		if err != nil || now.Sub(info.ModTime()) > ShapeEvery {
			continue // a session that has not used a tool in a while
		}
		mark := Load(path).Mark
		if _, ok := pet.Unlocks[mark]; ok && !seen[mark] {
			seen[mark] = true
			out = append(out, mark)
		}
	}
	sort.Strings(out)
	return out
}
