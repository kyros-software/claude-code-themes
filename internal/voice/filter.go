package voice

// What comes back from the model, and what of it is fit to be said.

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// echoes says three in four of a line's words are another line's. Every word
// counts, the small ones too: these are lines of six or seven words, and
// "respira. el repo no se va a ir" is "respira. el bug no se va a ir" with
// one of them changed.
func echoes(line, other string) bool {
	words := func(s string) []string {
		var out []string
		for _, w := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) }) {
			if utf8.RuneCountInString(w) >= 2 {
				out = append(out, w)
			}
		}
		return out
	}
	mine, theirs := words(line), map[string]bool{}
	for _, w := range words(other) {
		theirs[w] = true
	}
	if len(mine) < 4 {
		return strings.EqualFold(line, other)
	}
	same := 0
	for _, w := range mine {
		if theirs[w] {
			same++
		}
	}
	return same*4 >= len(mine)*3
}

// aside is the model talking about the job instead of doing it - "hmm, no:
// eso sería un recuento, prohibido", "aquí va la versión buena" - which it
// does out loud now and then, in lines short enough to pass for the pet's.
var aside = regexp.MustCompile(`\b(frases?|palabras|lista|recuento|encargo|versión|prohibid[oa]|caracteres|instrucci[oó]n(es)?|entorno|carpeta|directorio|statusline|mascota|tmp)\b|\b(sin|ni|no es|no hay|no tiene)( un| el)? (repo|repositorio|git)\b|\bya (la|lo|las|los) (dije|he dicho)\b|[()\[\]]`)

// garbled is what is not a sentence at all, which a small model at its lowest
// effort hands over about once in a hundred lines: "standard view",
// "ese-tsc-no-cierra-solo-caminando-lobo", a path. Under three words, a word
// that is three words hyphenated together, or a slash.
func garbled(line string) bool {
	fields := strings.Fields(line)
	if len(fields) < 3 || strings.ContainsAny(line, "/\\") {
		return true
	}
	// Cut off before its end: "sí, ya tiene su..."
	if strings.HasSuffix(line, "...") || strings.HasSuffix(line, "…") {
		return true
	}
	for _, w := range fields {
		if strings.Count(strings.Trim(w, "-"), "-") >= 2 {
			return true
		}
	}
	return false
}

// speaker is the model signing the line: "sabueso: lo que huelo es...".
var speaker = regexp.MustCompile(`^[\p{L}]{3,14}:\s+`)

var bullet = regexp.MustCompile(`^(\d{1,2}[.)]|[-*•])\s+`)

// pick is the lines of an answer that can be said: clean, lower case like the
// repertoire, not said before and not twice in the same answer. All of them:
// which three are said is choose's business.
func pick(out string, past []string) []string {
	var lines []string
	for _, raw := range strings.Split(out, "\n") {
		// "1. ", "- ", "* ": told not to number them, and it will anyway.
		// Only a list marker: "9 commits y sigo" keeps its nine.
		raw = bullet.ReplaceAllString(strings.TrimSpace(raw), "")
		raw = speaker.ReplaceAllString(raw, "")
		line := strings.ToLower(Clean(raw))
		// "aquí van las frases:" is a heading, whatever it was told.
		if line == "" || strings.HasSuffix(line, ":") || aside.MatchString(line) || garbled(line) ||
			said(past, line) || said(lines, line) {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

func said(past []string, line string) bool {
	for _, p := range past {
		if echoes(line, p) {
			return true
		}
	}
	return false
}

// choose is the best `wanted` of the lines that could be said: the ones that
// are a remark and not an inventory. Asked not to, the model still hands back
// "apt, compose, docker... ¿y si lo haces a ciegas?" and "dos horas de sesión
// y el contexto a punto de reventar" - the list it was given, and the fact it
// was given, said back. Those are dropped; of the rest, the ones that name
// one thing go before the ones that name two.
func choose(lines, things, told []string) []string {
	type ranked struct {
		line   string
		weight int
	}
	var keep []ranked
	for _, line := range lines {
		named := naming(line, things)
		switch {
		case named >= 3, named == 2 && listed(line, things):
			continue // an inventory
		case parrots(line, told):
			continue // the fact, said back
		case invented(line, told):
			continue // a count or a date nobody gave it: "ocho horas de surco"
		}
		keep = append(keep, ranked{line, named})
	}
	sort.SliceStable(keep, func(i, j int) bool { return keep[i].weight < keep[j].weight })

	// One line a subject. Told that he has been making commits all day, four
	// of its five lines are about commits, and three of those in a row is one
	// remark made three times. A line about nothing on the list - pure
	// character - has no subject and never clashes.
	subjects := subjectsOf(things, told)
	var out, spare []string
	used := map[string]bool{}
	for _, k := range keep {
		clash := false
		mine := about(k.line, subjects)
		for _, subject := range mine {
			clash = clash || used[subject]
		}
		if clash {
			spare = append(spare, k.line)
			continue
		}
		for _, subject := range mine {
			used[subject] = true
		}
		out = append(out, k.line)
	}
	// Better a second line on the same subject than a turn with nothing to say.
	out = append(out, spare...)
	if len(out) > wanted {
		out = out[:wanted]
	}
	return out
}

// counted is a number, or a when: the thing the model was not given and
// supplies anyway - "ocho compactaciones", "a las cinco de la mañana", "la
// migración de ayer", "plan.md lleva días igual". Any of them is fine when
// the facts said so, and made up when they did not; see invented.
//
// "dos" is let through. It is as often a figure of speech - dos cabezas, dos
// cosas a la vez - as a count, and dropping it took good lines with it.
var counted = regexp.MustCompile(`\d|\b(tres|cuatro|cinco|seis|siete|ocho|nueve|diez|once|doce|quince|veinte|treinta|cuarenta|cincuenta|cien|mil)\b`)

var dated = regexp.MustCompile(`\b(ayer|anteayer|anoche|días|semanas?|meses|años)\b`)

// invented says a line states a count or a when that nothing it was told
// carries.
func invented(line string, told []string) bool {
	heard := strings.ToLower(strings.Join(told, " "))
	for _, re := range []*regexp.Regexp{counted, dated} {
		for _, m := range re.FindAllString(line, -1) {
			if !strings.Contains(heard, m) {
				return true
			}
		}
	}
	return false
}

// filler is the words of a fact that are not what it is about.
var filler = map[string]bool{
	"ahora": true, "mismo": true, "anda": true, "trabaja": true, "lleva": true, "hace": true,
	"está": true, "tiene": true, "mucho": true, "muchas": true, "muchos": true, "varias": true,
	"varios": true, "alguna": true, "algún": true, "seguidas": true, "seguidos": true, "punto": true,
}

// stem is a word cut short enough for "commits" to be "commit" and
// "compactado" to be "compactar". Crude, and enough for seven-word lines.
func stem(w string) string {
	if r := []rune(w); len(r) > 5 {
		return string(r[:5])
	}
	return w
}

// subjectsOf is what a turn's lines can be about: the things in hand, and
// the words of the facts that carry them.
func subjectsOf(things, told []string) map[string]bool {
	out := map[string]bool{}
	for _, thing := range things {
		for _, s := range stems(thing) {
			out[stem(s)] = true
		}
	}
	for _, fact := range told {
		for _, w := range words(fact) {
			if utf8.RuneCountInString(w) >= 5 && !filler[w] {
				out[stem(w)] = true
			}
		}
	}
	return out
}

// about is which of the subjects a line touches.
func about(line string, subjects map[string]bool) []string {
	var out []string
	seen := map[string]bool{}
	for _, w := range words(line) {
		if s := stem(w); subjects[s] && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// stems is the ways a thing can be named in a line: "auth.py" and "auth".
func stems(thing string) []string {
	thing = unaccent.Replace(strings.ToLower(thing))
	if base, _, ok := strings.Cut(thing, "."); ok && base != "" {
		return []string{thing, base}
	}
	return []string{thing}
}

// unaccent is for telling what a line names, not for anything shown: the
// file is guia.md and the pet, which can spell, says "la guía".
var unaccent = strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u")

func words(s string) []string {
	return strings.FieldsFunc(unaccent.Replace(strings.ToLower(s)), func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-')
	})
}

// naming is how many of the things the session is handling a line names.
func naming(line string, things []string) int {
	in := map[string]bool{}
	// A program whose name is a Spanish word is not named by the word: "tú
	// tecleas y yo cargo" is a buey complaining, not a remark about cargo.
	homonym := map[string]bool{"cargo": true, "sed": true, "ve": true, "es": true, "top": true}
	for _, w := range words(line) {
		in[w] = true
	}
	n := 0
	for _, thing := range things {
		if homonym[strings.ToLower(thing)] {
			continue
		}
		for _, stem := range stems(thing) {
			if in[strings.Trim(stem, ".")] || strings.Contains(unaccent.Replace(strings.ToLower(line)), stem) && strings.Contains(stem, ".") {
				n++
				break
			}
		}
	}
	return n
}

// listed says two things are named side by side - "auth.py y pytest", "cargo,
// parser" - which is a list of two, as against "pytest verde y auth roto",
// which is a sentence about both.
func listed(line string, things []string) bool {
	ws := words(strings.NewReplacer(".py", "", ".rs", "", ".go", "", ".ts", "", ".js", "", ".yml", "", ".md", "", ".css", "", ".sql", "").Replace(strings.ToLower(line)))
	is := map[string]bool{}
	for _, thing := range things {
		for _, stem := range stems(thing) {
			is[stem] = true
		}
	}
	// "el readme, la guía y el changelog" is a list with its articles on.
	var bare []string
	for _, w := range ws {
		switch w {
		case "el", "la", "los", "las", "un", "una", "tu", "tus", "ese", "esa":
		default:
			bare = append(bare, w)
		}
	}
	ws = bare
	for i, w := range ws {
		if !is[w] {
			continue
		}
		if i+1 < len(ws) && is[ws[i+1]] {
			return true
		}
		if i+2 < len(ws) && (ws[i+1] == "y" || ws[i+1] == "e" || ws[i+1] == "con") && is[ws[i+2]] {
			return true
		}
	}
	return false
}

// parrots says nearly every word of a line that means something is in what
// the model was told: the fact handed back, with nothing added.
func parrots(line string, told []string) bool {
	heard := map[string]bool{}
	for _, fact := range told {
		for _, w := range words(fact) {
			heard[w] = true
		}
	}
	// By the first four letters: "paras" for "para", "commit" for "commits".
	short := func(w string) string {
		if r := []rune(w); len(r) > 4 {
			return string(r[:4])
		}
		return w
	}
	for w := range heard {
		heard[short(w)] = true
	}
	mine, same := 0, 0
	for _, w := range words(line) {
		if utf8.RuneCountInString(w) < 4 {
			continue
		}
		mine++
		if heard[short(w)] {
			same++
		}
	}
	return mine >= 3 && same*5 >= mine*4 // four in five
}

// Clean turns what the model sent into something that can go on a terminal, or
// into "" if it cannot: one line, printable, no wrapping quotes, short enough.
//
// The length is a refusal and not a cut. A line chopped at 57 characters is a
// sentence without its end, and the repertoire is right there to say instead.
func Clean(raw string) string {
	line := firstLine(raw)
	// Refused whole, not scrubbed: an escape sequence without its ESC is
	// still "[2J" in the middle of the footer.
	for _, r := range line {
		if !unicode.IsPrint(r) {
			return ""
		}
	}
	line = strings.TrimSpace(strings.Trim(strings.TrimSpace(line), "\"'`«»“”‘’"))
	if line == "" || utf8.RuneCountInString(line) > MaxRunes {
		return ""
	}
	return line
}

func firstLine(raw string) string {
	for _, l := range strings.Split(raw, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			return l
		}
	}
	return ""
}
