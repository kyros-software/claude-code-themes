package i18n

// Game is every word ccpet invade prints: the HUD, the banners, the refusals
// and one name per weapon family and per enemy trait.
//
// Its own struct and not more fields on Strings. The statusline and the panel do
// not want the game's vocabulary in scope, and Strings is already a screenful:
// a catalogue you cannot read top to bottom stops being the thing that keeps a
// language honest and becomes a place where blanks hide.
type Game struct {
	// The HUD.
	HUDWave    string // takes the wave
	HUDBoss    string
	HUDScore   string // takes the score
	HUDLife    string
	HUDAbility string
	HUDReady   string

	// The banners.
	Paused         string
	PausedByClaude string
	Resume         string
	WaveCleared    string // takes the wave
	RivalArrives   string // takes the rival's name
	BossDown       string // takes the rival's name
	Revived        string
	Landed         string
	GameOver       string // takes the wave it ended on
	// LostALevel is the one line that has to land. The run is a game and the
	// pet is not: a player who does not understand that dying cost the creature
	// a level will read the panel tomorrow and think the theme has a bug.
	LostALevel string // takes the level it dropped to
	Records    string // takes the best wave, the best score and the runs

	// The refusals.
	TooSmall string // takes the columns and rows wanted, then the ones there are
	NoTTY    string
	NoTmux   string

	// The help row.
	Help string

	// Families is one name per weapon family, keyed by the id in
	// internal/invaders/kit.go, and Traits one per enemy trait. Maps and not
	// fields because the ids belong to the resolver and the bestiary; a name
	// missing from either is caught by the guards in strings_test.go, which
	// learned to walk a map the day this file arrived.
	Families map[string]string
	Traits   map[string]string
}

// G is the game's catalogue for the language in use.
func G() Game {
	if Current() == EN {
		return englishGame
	}
	return spanishGame
}

var spanishGame = Game{
	HUDWave:    "oleada %d",
	HUDBoss:    "JEFE",
	HUDScore:   "puntos %d",
	HUDLife:    "vida",
	HUDAbility: "habilidad",
	HUDReady:   "lista",

	Paused:         "en pausa",
	PausedByClaude: "en pausa: Claude ha terminado de responder",
	Resume:         "p para seguir",
	WaveCleared:    "oleada %d limpia",
	RivalArrives:   "baja %s",
	BossDown:       "%s cae · vida al máximo",
	Revived:        "el fénix te levanta, una sola vez",
	Landed:         "han aterrizado. se acabó",
	GameOver:       "fin de la partida en la oleada %d",
	LostALevel:     "tu bicho baja al nivel %d",
	Records:        "mejor oleada %d · mejores puntos %d · partidas %d",

	TooSmall: "hacen falta %dx%d y este terminal es %dx%d",
	NoTTY:    "invade necesita un terminal de verdad, no una tubería",
	NoTmux:   "no hay tmux: abre otra pestaña y lanza ccpet invade ahí",

	Help: "←→ mover · espacio disparar · x habilidad · p pausa · q salir",

	Families: map[string]string{
		"single":   "único",
		"steady":   "pausado",
		"seeker":   "buscador",
		"rapid":    "rápido",
		"twin":     "gemelo",
		"sweep":    "barrido",
		"homing":   "rastreador",
		"turret":   "torreta",
		"burst":    "ráfaga",
		"cannon":   "cañón",
		"overload": "sobrecarga",
		"phoenix":  "fénix",
		"chimera":  "quimera",
	},
	Traits: map[string]string{
		"plain":    "recto",
		"weaver":   "ondulante",
		"darter":   "lanzado",
		"plated":   "blindado",
		"splitter": "divisible",
	},
}

var englishGame = Game{
	HUDWave:    "wave %d",
	HUDBoss:    "BOSS",
	HUDScore:   "score %d",
	HUDLife:    "life",
	HUDAbility: "ability",
	HUDReady:   "ready",

	Paused:         "paused",
	PausedByClaude: "paused: Claude has finished answering",
	Resume:         "p to resume",
	WaveCleared:    "wave %d cleared",
	RivalArrives:   "%s is coming down",
	BossDown:       "%s down · life full",
	Revived:        "the phoenix picks you up, once and once only",
	Landed:         "they have landed. that is that",
	GameOver:       "game over on wave %d",
	LostALevel:     "your creature drops to level %d",
	Records:        "best wave %d · best score %d · runs %d",

	TooSmall: "needs %dx%d and this terminal is %dx%d",
	NoTTY:    "invade needs a real terminal, not a pipe",
	NoTmux:   "no tmux here: open another tab and run ccpet invade in it",

	Help: "←→ move · space fire · x ability · p pause · q quit",

	Families: map[string]string{
		"single":   "single",
		"steady":   "steady",
		"seeker":   "seeker",
		"rapid":    "rapid",
		"twin":     "twin",
		"sweep":    "sweep",
		"homing":   "homing",
		"turret":   "turret",
		"burst":    "burst",
		"cannon":   "cannon",
		"overload": "overload",
		"phoenix":  "phoenix",
		"chimera":  "chimera",
	},
	Traits: map[string]string{
		"plain":    "straight",
		"weaver":   "weaving",
		"darter":   "darting",
		"plated":   "plated",
		"splitter": "splitting",
	},
}
