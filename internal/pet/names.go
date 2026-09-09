package pet

import "github.com/kyros-software/claude-code-themes/internal/i18n"

// The pet's names, from the design canvas "Tema Terminal Claude CLI".
//
// The English keys are the ids: they are what pet.json has stored since the
// Python, what the hooks and the evolution tree are written in, and renaming
// them would rewrite every life file in the wild. The Spanish is what a person
// reads, and it is the canvas's own wording - artboard 06 for the tree and 07
// for the lineage the panel prints as "metódico › pauta › refactor".
//
// Which is also why the English reading needs no table of its own: the id IS
// the English name. `bughunter[bloodhound]` is what the canvas drew as
// `cazabugs[sabueso]`, spelled with the words the tree is already written in.
// Only the counters below need translating, because their ids are keys rather
// than words.
//
// A form with no entry here falls back to its id, so the two never fight: a
// missing name is a name that has not been chosen yet, not a crash.

// Names is the 27 forms plus the three temperaments the lineage starts from.
var Names = map[string]string{
	// the fourteen titles, level 6 - "la forma final de cada rama"
	"scalpel": "bisturí", "loom": "telar", "abbot": "abad", "forest": "bosque",
	"wolf": "lobo", "wasp": "avispa", "atlas": "atlas", "sphinx": "esfinge",
	"storm": "tormenta", "falcon": "halcón", "mammoth": "mamut", "worm": "gusano",
	"devil": "diablo", "leviathan": "leviatán",

	// the larva
	"spark": "chispa",

	// the three temperaments, as forms and as the counters the lineage shows
	"pattern": "pauta", "probe": "sonda", "ember": "brasa",
	"methodical": "metódico", "inquisitive": "inquisitivo", "impulsive": "impulsivo",

	// the seven trades
	"refactor": "refactor", "tidy": "pulcro",
	"bughunter": "cazabugs", "architect": "arquitecto",
	"sprinter": "velocista", "marathon": "maratón", "feral": "salvaje",

	// the fourteen marks
	"surgeon": "cirujano", "weaver": "tejedor",
	"monk": "monje", "gardener": "jardinero",
	"bloodhound": "sabueso", "exterminator": "exterminador",
	"cartographer": "cartógrafo", "oracle": "oráculo",
	"bolt": "relámpago", "sniper": "francotirador",
	"ox": "buey", "mole": "topo",
	"gremlin": "gremlin", "kraken": "kraken",

	// the two secrets
	"phoenix": "fénix", "chimera": "quimera",

	// the seven states of the vitality layer, artboard 02. The ids stay
	// English because five places compare against them and one of them is
	// written into the session file on disk; only the reading changes.
	"fresh": "fresca", "lively": "vibrante", "easy": "a gusto",
	"sluggish": "espesa", "tired": "cansada", "drowning": "ahogada",
	"k.o.": "k.o.",
}

// Name is what a person reads for a form or a counter, in the language the
// theme is set to.
func Name(id string) string { return NameIn(i18n.Current(), id) }

// NameIn is Name in a language you name yourself. The atlas the sprites are
// checked against is a Spanish document, so its test asks for Spanish however
// the machine running it is configured.
func NameIn(lang i18n.Lang, id string) string {
	if lang == i18n.EN {
		return id
	}
	if name, ok := Names[id]; ok {
		return name
	}
	return id
}

// CounterNames is what a person reads for a behaviour counter.
//
// Twenty-two numbers decide the whole tree and the panel showed exactly one of
// them - the habit of the mark in progress. The rest moved in silence, so
// there was no way to find out what a mark asks for short of reading the
// source. These are the labels the panel prints; the English keys stay the ids
// pet.json stores, for the same reason the form names do.
var CounterNames = map[string]string{
	// The three temperaments already have a name as forms, and mean the same
	// thing as counters, so they are not repeated here - Name falls through.

	// What the level 3 fork reads.
	"diffs":          "diffs limpios",
	"ctx_low":        "sesiones con contexto bajo",
	"tests":          "suites en verde",
	"plans":          "planes cerrados",
	"short_sessions": "sesiones cortas",
	"long_sessions":  "sesiones largas",
	"ctx_maxed":      "sesiones al límite",

	// What the fourteen marks ask for.
	"diff_streak":       "días seguidos con diff limpio",
	"widest_commit":     "commit más ancho",
	"sessions_under_40": "sesiones bajo el 40%",
	"docs_days":         "días seguidos tocando docs",
	"repro_before_fix":  "reproducir antes de arreglar",
	"test_streak":       "días seguidos en verde",
	"longest_plan":      "plan más largo cerrado",
	"plans_before_code": "planes antes de tocar código",
	"sessions_15min":    "sesiones de menos de 15m",
	"single_tool_tasks": "tareas de una sola herramienta",
	"sessions_4h":       "sesiones de 4h o más",
	"same_repo_days":    "días seguidos en el mismo repo",
	"bypass_turns":      "turnos en bypass",
	"ctx100_sessions":   "veces con el contexto al 100%",
}

// CounterNamesEN is the same list in English. The two are checked against each
// other by a test: a counter added to one and forgotten in the other prints a
// bare id, which is a key leaking into the panel.
var CounterNamesEN = map[string]string{
	"diffs":          "clean diffs",
	"ctx_low":        "sessions on a low context",
	"tests":          "green suites",
	"plans":          "plans closed",
	"short_sessions": "short sessions",
	"long_sessions":  "long sessions",
	"ctx_maxed":      "sessions at the limit",

	"diff_streak":       "days running with a clean diff",
	"widest_commit":     "widest commit",
	"sessions_under_40": "sessions under 40%",
	"docs_days":         "days running touching docs",
	"repro_before_fix":  "reproduced before fixing",
	"test_streak":       "days running in the green",
	"longest_plan":      "longest plan closed",
	"plans_before_code": "plans before touching code",
	"sessions_15min":    "sessions under 15m",
	"single_tool_tasks": "one-tool tasks",
	"sessions_4h":       "sessions of 4h or more",
	"same_repo_days":    "days running in the same repo",
	"bypass_turns":      "turns on bypass",
	"ctx100_sessions":   "times with the context at 100%",
}

// CounterName is CounterNames with the form names as a fallback, so the three
// temperaments read the same whether they are a shape or a habit.
func CounterName(counter string) string {
	table := CounterNames
	if i18n.Current() == i18n.EN {
		table = CounterNamesEN
	}
	if name, ok := table[counter]; ok {
		return name
	}
	return Name(counter)
}
