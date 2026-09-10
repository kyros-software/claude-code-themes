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
	Reloading  string

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
	// LevelUp is the three-way choice the score buys, and GotKit is a health
	// kit caught. Both are banners, so both have to fit on one row at sixty
	// columns.
	LevelUp string
	GotKit  string
	// The level-up box, which is a box and not a line because a line at the
	// bottom of the screen with the field frozen behind it reads as a crash -
	// and was reported as one.
	UpChoose string
	UpPower  string
	UpRate   string
	UpMag    string
	// Again is the offer on the game-over screen. It goes on the same row as
	// GameOver, so the two of them together have to fit sixty columns.
	Again string

	// The refusals.
	TooSmall string // takes the columns and rows wanted, then the ones there are
	NoTTY    string
	NoTmux   string

	// The arena: the switch that lets the game open itself while Claude works.
	ArenaIsOn  string
	ArenaIsOff string
	ArenaUsage string
	NoTerminal string

	// The help row, twice: the whole thing, and one that fits in a narrow
	// window. The renderer prints the longest that fits rather than truncating,
	// because a key row that ends mid-word has stopped being a key row - and
	// with four arrows, a brake, a magazine and a health kit to explain, the full
	// one is ninety-three columns in Spanish.
	//
	// The tight one drops the separators rather than any of the keys. Nine
	// groups at sixty columns leaves six characters each, and " · " is three of
	// them; a key row missing a key is worse than a dense one.
	Help  string
	Tight string

	// Families is one name per weapon family, keyed by the id in
	// internal/invaders/kit.go. A map and not thirteen fields because the ids
	// belong to the resolver; a name missing from either language is caught by
	// the guards in strings_test.go, which learned to walk a map the day this
	// file arrived.
	//
	// The enemies are NOT here. The fleet and the thirty-five bosses carry the
	// names they were drawn with, the way a Zaku is a Zaku in every language:
	// translating "zángano" would be translating a proper noun.
	Families map[string]string
	// Abilities is one name per ability id, and the HUD prints it whether the
	// ability is ready or not: the space bar is the same for everybody and this
	// is the key that is not, so it says what it is.
	Abilities map[string]string
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
	Reloading:  "recargando",

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
	LevelUp:        "mejora: 1 potencia · 2 cadencia · 3 cargador",
	UpChoose:       "ESCOGE UNA MEJORA",
	UpPower:        "potencia · +1 de daño",
	UpRate:         "cadencia · disparas más seguido",
	UpMag:          "cargador · +3 balas",
	GotKit:         "botiquín a bordo · e para gastarlo",
	Again:          "espacio otra · q salir",

	TooSmall: "hacen falta %dx%d y este terminal es %dx%d",
	NoTTY:    "invade necesita un terminal de verdad, no una tubería",
	NoTmux:   "no hay tmux: abre otra pestaña y lanza ccpet invade ahí",

	ArenaIsOn:  "arena: encendida · el juego se abre solo mientras Claude trabaja",
	ArenaIsOff: "arena: apagada · el juego solo se abre si lo abres tú",
	ArenaUsage: "uso: ccpet arena [on|off]",
	NoTerminal: "no hay ningún emulador de terminal en el que abrir el juego (o dime cuál con CCPET_ARENA_TERM)",

	Help:  "↑↓←→ mover · s parar · espacio tirar · x habilidad · r recargar · e curar · p pausa · q salir",
	Tight: "↑↓←→ ␣tiro x poder r carga e cura s alto p pausa q salir",

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
	Abilities: map[string]string{
		"sweep":    "barrido",
		"turret":   "torreta",
		"turret2":  "dos torretas",
		"invuln":   "intocable",
		"threerow": "barrido triple",
		"blast":    "fogonazo",
		"chimera":  "quimera",
		"pulse":    "empujón",
		"shield":   "escudo",
		"mark":     "dardos",
		"rush":     "desboque",
		"mirror":   "espejo",
		"net":      "red",
		"dash":     "embestida",
		"lance":    "lanza",
		"frenzy":   "furia",
	},
}

var englishGame = Game{
	HUDWave:    "wave %d",
	HUDBoss:    "BOSS",
	HUDScore:   "score %d",
	HUDLife:    "life",
	HUDAbility: "ability",
	HUDReady:   "ready",
	Reloading:  "reloading",

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
	LevelUp:        "upgrade: 1 power · 2 rate · 3 magazine",
	UpChoose:       "CHOOSE AN UPGRADE",
	UpPower:        "power · +1 damage",
	UpRate:         "rate · shoot more often",
	UpMag:          "magazine · +3 rounds",
	GotKit:         "health kit aboard · e to use it",
	Again:          "space for another · q to quit",

	TooSmall: "needs %dx%d and this terminal is %dx%d",
	NoTTY:    "invade needs a real terminal, not a pipe",
	NoTmux:   "no tmux here: open another tab and run ccpet invade in it",

	ArenaIsOn:  "arena: on · the game opens by itself while Claude works",
	ArenaIsOff: "arena: off · the game opens when you open it",
	ArenaUsage: "usage: ccpet arena [on|off]",
	NoTerminal: "no terminal emulator here to open the game in (or name one in CCPET_ARENA_TERM)",

	Help:  "↑↓←→ move · s stop · space fire · x ability · r reload · e heal · p pause · q quit",
	Tight: "↑↓←→ ␣fire x power r load e heal s stop p pause q quit",

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
	Abilities: map[string]string{
		"sweep":    "sweep",
		"turret":   "turret",
		"turret2":  "two turrets",
		"invuln":   "untouchable",
		"threerow": "triple sweep",
		"blast":    "blast",
		"chimera":  "chimera",
		"pulse":    "shove",
		"shield":   "shield",
		"mark":     "darts",
		"rush":     "overdrive",
		"mirror":   "mirror",
		"net":      "net",
		"dash":     "dash",
		"lance":    "lance",
		"frenzy":   "frenzy",
	},
}
