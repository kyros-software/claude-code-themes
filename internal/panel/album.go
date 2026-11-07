package panel

import (
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/kyros-software/claude-code-themes/internal/pet"
	"github.com/kyros-software/claude-code-themes/internal/theme"
)

// The album: every form the pet has worn, life after life, and what is still
// to see. v3's answer to "hay muchísimas formas y he visto muy pocas" - the
// forms existed, and nothing on screen ever said so.

// albumWidth is where the "still to see" rows wrap.
const albumWidth = 78

// treeOrder is every form of the tree in design order, breadth first: the
// temperaments, then the trades, the marks and the titles, each in the order
// the tree lists them.
func treeOrder() []string {
	out := []string{pet.Root}
	for i := 0; i < len(out); i++ {
		out = append(out, pet.Tree[out[i]]...)
	}
	return out
}

// hint is what leads to a form: the habit for a fork, the mark for a title.
func hint(form string) string {
	switch pet.Tier(form) {
	case 2, 3:
		return pet.CounterName(pet.BranchBy[form])
	case 5:
		return pet.CounterName(pet.Unlocks[form].Counter)
	case 6:
		return "de " + pet.Name(pet.Parent[form])
	}
	return ""
}

func showAlbum(out io.Writer, statePath string) int {
	s := pet.Load(statePath)
	dim, reset, emph := theme.Fg(theme.Dim), theme.Reset, theme.Fg(theme.Emph)
	joiner := " " + theme.Fg(theme.Rule) + "›" + reset + dim + " "

	seen := map[string]bool{}
	lives := map[int][]string{}
	var gens []int
	for _, e := range s.Album {
		if _, ok := lives[e.Gen]; !ok {
			gens = append(gens, e.Gen)
		}
		lives[e.Gen] = append(lives[e.Gen], pet.Name(e.Form))
		seen[e.Form] = true
	}
	sort.Ints(gens)

	total := len(pet.Sprites)
	var b strings.Builder
	nl := func(line string) { b.WriteString(line + "\n") }

	nl("")
	nl("  " + theme.Fg(theme.Path) + theme.Bold + "álbum" + reset + "  " +
		emph + strconv.Itoa(len(seen)) + reset + dim + "/" + strconv.Itoa(total) + " formas · generación " + reset +
		emph + strconv.Itoa(max(s.Generation, 1)) + reset)
	nl("")
	for _, g := range gens {
		nl("  " + dim + theme.PadRight("generación "+strconv.Itoa(g), 15) + reset +
			theme.Fg(theme.Number) + strings.Join(lives[g], reset+joiner+theme.Fg(theme.Number)) + reset)
	}

	// What is still to see, by rung, each with the habit that leads there.
	groups := []struct {
		tier  int
		label string
	}{{2, "temperamentos"}, {3, "oficios"}, {5, "marcas"}, {6, "títulos"}}
	missing := 0
	var rows []string
	for _, g := range groups {
		var items []string
		for _, form := range treeOrder() {
			if pet.Tier(form) != g.tier || seen[form] {
				continue
			}
			items = append(items, theme.Fg(theme.Emph)+pet.Name(form)+reset+dim+" ("+hint(form)+")"+reset)
		}
		missing += len(items)
		if len(items) == 0 {
			continue
		}
		rows = append(rows, "    "+dim+g.label+reset)
		line := "      "
		for i, item := range items {
			sep := ""
			if i > 0 {
				sep = dim + " · " + reset
			}
			if theme.Width(line+sep+item) > albumWidth && i > 0 {
				rows = append(rows, line)
				line, sep = "      ", ""
			}
			line += sep + item
		}
		rows = append(rows, line)
	}
	secrets := 0
	for _, form := range pet.Secrets {
		if !seen[form] {
			secrets++
		}
	}
	missing += secrets

	if missing > 0 {
		nl("")
		nl("  " + dim + "por ver " + reset + emph + strconv.Itoa(missing) + reset)
		for _, row := range rows {
			nl(row)
		}
		if secrets > 0 {
			nl("    " + dim + "secretas" + reset)
			nl("      " + dim + strconv.Itoa(secrets) + " sin descubrir" + reset)
		}
	}
	nl("")
	io.WriteString(out, b.String())
	return 0
}

// albumLine is the one-row summary for the main panel.
func albumLine(s *pet.State) string {
	dim, reset, emph := theme.Fg(theme.Dim), theme.Reset, theme.Fg(theme.Emph)
	seen := map[string]bool{}
	for _, e := range s.Album {
		seen[e.Form] = true
	}
	line := "  " + dim + "álbum  " + reset + emph + strconv.Itoa(len(seen)) + reset +
		dim + "/" + strconv.Itoa(len(pet.Sprites)) + " · generación " + reset +
		emph + strconv.Itoa(max(s.Generation, 1)) + reset
	if pet.AtTitle(s) && s.TitleDays > 0 {
		left := pet.TitleStay - s.TitleDays + 1
		day := "días"
		if left == 1 {
			day = "día"
		}
		line += dim + " · huevo en " + reset + emph + strconv.Itoa(left) + reset + dim + " " + day + " de uso" + reset
	}
	return line + dim + "   /pet album" + reset
}
