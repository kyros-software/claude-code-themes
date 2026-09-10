package i18n

import "fmt"

// Strings is every word the runtime prints that is not the pet's own name or
// its voice - the panel's row labels, the setup messages, the help.
//
// A struct and not a map of keys: a mistyped field does not compile, and a
// language added later cannot quietly forget half of it. The pet's names live
// in pet/names.go and its phrases in pet/speech.go, next to the tree and the
// rules they belong to; only the chrome is here.
type Strings struct {
	// The panel's rows and the words inside them.
	Level        string // "nivel", the word in front of the number
	LevelRow     string // the row label, padded by the caller
	MarkRow      string
	HungerRow    string
	StreakRow    string
	Toward       string // "para", between a number and what it buys
	EatingItself string
	Best         string
	Ate          string
	Today        string

	// The panel's sections.
	Level5Mark     string
	Met            string
	OnTheWay       string
	LeadsNowhere   string
	Habits         string
	Sessions       string
	Food           string
	Ready          string
	In             string // "en", in front of a duration still to wait
	IfContextBlows string

	// Feeding, from the panel.
	// LogDefeat labels the one log row that is not a meal: a run of invade
	// lost. It is here and not in Foods because a defeat is not something you
	// can order - see pet.Setback.
	LogDefeat  string
	AteAlready string // takes a duration
	WontEat    string // takes the meal's name
	Evolves    string // takes two form names
	NotFood    string // takes the word tried and the list of meals

	// Durations, spelled the way each language says them out loud.
	LessThanAMinute string
	AgoMinutes      string // takes minutes
	AgoHours        string // takes hours and minutes

	// setup.
	Backup            string
	StatuslineOn      string
	PickTheme         string
	AlreadyOff        string
	StatuslineOff     string
	StatuslineIsOff   string
	FoodHooksNone     string
	FoodHooks         string
	PluginHooksNote   string
	StatuslineWired   string
	HooksWired        string
	HooksNotInstalled string
	SettingsClean     string

	// The command line itself.
	Usage      string
	SetupUsage string
	LangUsage  string
	LangIs     string // takes the language and where it was decided
	LangSaved  string

	// days spells a small number of days, because "cinco días de racha" reads
	// better than "5 días de racha" and English says "five days" for the same
	// reason. Beyond the table it falls back to the numeral.
	days []string
	// Day and Days are the singular and plural of the streak row's unit.
	Day  string
	Days string
}

// S is the catalogue for the language in use.
func S() Strings {
	if Current() == EN {
		return english
	}
	return spanish
}

// Days spells a count of days: "cinco días", "five days", "12 días".
func (s Strings) NDays(n int) string {
	if n >= 0 && n < len(s.days) {
		return s.days[n]
	}
	unit := s.Days
	if n == 1 {
		unit = s.Day
	}
	return fmt.Sprintf("%d %s", n, unit)
}

var spanish = Strings{
	Level:        "nivel",
	LevelRow:     "nivel",
	MarkRow:      "marca",
	HungerRow:    "hambre",
	StreakRow:    "racha",
	Toward:       "para",
	EatingItself: "se está comiendo",
	Best:         "mejor",
	Ate:          "comió",
	Today:        "hoy",

	Level5Mark:     "la marca del nivel 5",
	Met:            "cumplido",
	OnTheWay:       "en camino",
	LeadsNowhere:   "no lleva a nada desde aquí",
	Habits:         "hábitos",
	Sessions:       "sesiones",
	Food:           "comida",
	Ready:          "listo",
	In:             "en",
	IfContextBlows: "si revientas el contexto",

	LogDefeat:  "derrota en invade",
	AteAlready: "ya ha comido. le toca en %s",
	WontEat:    "no le entra %s ahora mismo",
	Evolves:    "evoluciona: %s › %s",
	NotFood:    "ccpet: %q no es comida. Prueba: %s\n",

	LessThanAMinute: "menos de un minuto",
	AgoMinutes:      "hace %dm",
	AgoHours:        "hace %dh %02dm",

	Backup:            "copia de seguridad: %s\n",
	StatuslineOn:      "statusline encendida -> %s\n",
	PickTheme:         "elige también el tema, si no lo has hecho: /theme -> Terminal",
	AlreadyOff:        "la statusline ya estaba apagada",
	StatuslineOff:     "statusline apagada",
	StatuslineIsOff:   "statusline: apagada",
	FoodHooksNone:     "hooks de comida en settings.json: ninguno",
	FoodHooks:         "hooks de comida en settings.json: %s\n",
	PluginHooksNote:   "(si lo instalas como plugin, sus hooks son suyos y no salen aquí)",
	StatuslineWired:   "  statusLine conectada",
	HooksWired:        "  hooks conectados: PostToolUse (todas), PreCompact, SessionEnd, UserPromptSubmit, Stop",
	HooksNotInstalled: "  hooks NO instalados (pasa --hooks si los quieres)",
	SettingsClean:     "  settings.json limpio (el tema no se toca: cámbialo con /theme)",

	SetupUsage: "uso: ccpet setup {on|off|status|install|install-hooks|uninstall} [raíz]",
	LangUsage:  "uso: ccpet lang [es|en|auto]",
	LangIs:     "idioma: %s (%s)\n",
	LangSaved:  "idioma: %s\n",
	Usage: `ccpet - la statusline del tema Terminal y su mascota.

  ccpet                       el panel de la mascota
  ccpet feed                  darle de comer (+3 xp, -2 hambre, uno cada 4 h)
  ccpet <evento>              una comida: tests | commit | compact | task | overflow
  ccpet count <contador> [n]  suma a un contador de comportamiento
  ccpet day <nombre>          cuenta días SEGUIDOS, no veces
  ccpet record <contador> <n> guarda el máximo de un contador
  ccpet session <fichero>     cierra una sesión: sus datos pasan a contadores

  ccpet statusline            pinta un refresco (payload por stdin)
  ccpet hook                  atiende un evento de hook (payload por stdin)

  ccpet lang [es|en|auto]     idioma del tema (sin argumento, dice cuál es)
  ccpet link                  apunta ~/.claude/ccpet a este plugin
  ccpet setup on|off|status   enciende o apaga la statusline en settings.json
  ccpet setup install         instalación sin plugin (install-hooks incluye los hooks)
  ccpet setup uninstall       deshacerlo
  ccpet invade                el juego: tu mascota es el cañón
  ccpet arena on|off          abre el juego mientras Claude trabaja
  ccpet version               imprime la versión
  ccpet help                  esta ayuda (también -h y --help)
`,

	days: []string{"cero días", "un día", "dos días", "tres días", "cuatro días",
		"cinco días", "seis días", "siete días"},
	Day:  "día",
	Days: "días",
}

var english = Strings{
	Level:        "level",
	LevelRow:     "level",
	MarkRow:      "mark",
	HungerRow:    "hunger",
	StreakRow:    "streak",
	Toward:       "to",
	EatingItself: "eating itself at",
	Best:         "best",
	Ate:          "ate",
	Today:        "today",

	Level5Mark:     "the level 5 mark",
	Met:            "done",
	OnTheWay:       "on its way",
	LeadsNowhere:   "leads nowhere from here",
	Habits:         "habits",
	Sessions:       "sessions",
	Food:           "food",
	Ready:          "ready",
	In:             "in",
	IfContextBlows: "if you blow the context",

	LogDefeat:  "lost at invade",
	AteAlready: "already fed. next one in %s",
	WontEat:    "it will not take %s right now",
	Evolves:    "evolves: %s › %s",
	NotFood:    "ccpet: %q is not food. Try: %s\n",

	LessThanAMinute: "under a minute",
	AgoMinutes:      "%dm ago",
	AgoHours:        "%dh %02dm ago",

	Backup:            "backup: %s\n",
	StatuslineOn:      "statusline on -> %s\n",
	PickTheme:         "pick the theme too, if you have not: /theme -> Terminal",
	AlreadyOff:        "the statusline was already off",
	StatuslineOff:     "statusline off",
	StatuslineIsOff:   "statusline: off",
	FoodHooksNone:     "food hooks in settings.json: none",
	FoodHooks:         "food hooks in settings.json: %s\n",
	PluginHooksNote:   "(installed as a plugin, its hooks are its own and do not show here)",
	StatuslineWired:   "  statusLine wired",
	HooksWired:        "  hooks wired: PostToolUse (all), PreCompact, SessionEnd, UserPromptSubmit, Stop",
	HooksNotInstalled: "  hooks NOT installed (pass --hooks if you want them)",
	SettingsClean:     "  settings.json clean (the theme is left alone: change it with /theme)",

	SetupUsage: "usage: ccpet setup {on|off|status|install|install-hooks|uninstall} [root]",
	LangUsage:  "usage: ccpet lang [es|en|auto]",
	LangIs:     "language: %s (%s)\n",
	LangSaved:  "language: %s\n",
	Usage: `ccpet - the Terminal theme's statusline and its pet.

  ccpet                       the pet's panel
  ccpet feed                  feed it by hand (+3 xp, -2 hunger, one every 4 h)
  ccpet <event>               a meal: tests | commit | compact | task | overflow
  ccpet count <counter> [n]   add to a behaviour counter
  ccpet day <name>            count CONSECUTIVE DAYS, not occurrences
  ccpet record <counter> <n>  keep a counter's maximum
  ccpet session <file>        close a session: its facts become counters

  ccpet statusline            print one refresh (payload on stdin)
  ccpet hook                  handle a hook event (payload on stdin)

  ccpet lang [es|en|auto]     the theme's language (with no argument, says which)
  ccpet link                  point ~/.claude/ccpet at this plugin
  ccpet setup on|off|status   turn the statusline on or off in settings.json
  ccpet setup install         install without the plugin (install-hooks adds the hooks)
  ccpet setup uninstall       undo it
  ccpet invade                the shooter: your pet is the cannon
  ccpet arena on|off          open the game while Claude works
  ccpet version               print the version
  ccpet help                  this help (also -h and --help)
`,

	days: []string{"zero days", "one day", "two days", "three days", "four days",
		"five days", "six days", "seven days"},
	Day:  "day",
	Days: "days",
}
