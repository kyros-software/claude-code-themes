package pet

import (
	"fmt"
	"math/rand"
	"time"
)

// What the pet says, from the design canvas "Tema Terminal Claude CLI",
// artboard 08 - lo que dice. The phrases are Spanish on purpose: they are the
// pet's voice, not its API, and half of them are jokes that do not survive a
// translation ("el bug no era el código, era el jueves").
//
// Three rules, all from the canvas:
//
//   - Primero el evento, luego la forma. The shared lines belong to specific
//     events and everyone says them; a form's own repertoire is its voice for
//     a good meal.
//   - Sin repetir. The last SaidMemory phrases live in pet.json and do not come
//     back until the repertoire is exhausted.
//   - Silencio por defecto. At most one line every SpeechCooldown, and without
//     an event it says nothing: the statusline is not a chat.
//
// 16-nov-2026: the canvas wrote 24 lines for 8 forms and 5 for the events, and
// a pet that reaches a title lives on three of them for good - which reads as
// a pet that does not talk. Every form has its own voice now, every event more
// than one way of being said, and there are more things worth a word: the
// small meals, the first one of the day, the hour, the length of the session,
// the context filling up. The three rules stand.

// SaidMemory is how many phrases back the pet remembers not to repeat. One
// less than a form's repertoire, so a form never repeats inside its own round.
const SaidMemory = 4

// SpeechCooldown is the floor between two lines.
const SpeechCooldown = 5 * time.Minute

// Event is what opens the pet's mouth.
type Event string

// The things worth saying something about.
const (
	EventNothing  Event = ""
	EventBigMeal  Event = "big_meal"
	EventHungry   Event = "hungry"
	EventLevelUp  Event = "level_up"
	EventStreak   Event = "streak"
	EventCtxBlown Event = "ctx_blown"
	EventHatch    Event = "hatch"

	// The small meals: worth a word, not the form's own voice.
	EventCompact Event = "compact"
	EventTask    Event = "task"
	EventFeed    Event = "feed"

	// The moment. Each of these is said at most once a day - see Ambient.
	EventMorning  Event = "morning"  // the first meal of the day
	EventLate     Event = "late"     // a meal between midnight and six
	EventFriday   Event = "friday"   // a meal on a Friday afternoon
	EventLong     Event = "long"     // LongSession of work in one session
	EventTired    Event = "tired"    // the context is at "cansada"
	EventDrowning Event = "drowning" // ... at "ahogada"
)

// LongSession is the working time after which the pet mentions it.
const LongSession = 3 * time.Hour

// daily are the events said once a day at most, whatever the cooldown says.
var daily = map[Event]bool{
	EventMorning: true, EventLate: true, EventFriday: true,
	EventLong: true, EventTired: true, EventDrowning: true,
}

// Moment is what the statusline knows that pet.json does not.
type Moment struct {
	Now time.Time
	// Meal is the event name of a meal eaten inside BigMealWindow, or "".
	Meal string
	// Label is the vitality state's id.
	Label string
	// Active is the session's working time.
	Active time.Duration
}

// SnackEvent is the event a small meal speaks as, or EventNothing.
func SnackEvent(meal string) Event {
	switch meal {
	case "compact":
		return EventCompact
	case "task":
		return EventTask
	case "feed":
		return EventFeed
	}
	return EventNothing
}

// Ambient is the once-a-day remark the moment calls for, if it has not been
// made today. The context comes first because it is the one the person can
// still do something about; the clock ones wait for a meal, so the pet only
// comments on the hour to somebody who is actually working.
//
// The context ones read the state, not the moment it changed: the crossing
// lasts one refresh, and one refresh inside the cooldown of the commit before
// it is a remark that is never made. Once a day is what keeps a standing state
// from being said over and over.
func Ambient(s *State, m Moment) Event {
	fresh := func(e Event) bool { return s.DayMarks[saidKey(e)] != Today(m.Now) }
	switch {
	case m.Label == "drowning" && fresh(EventDrowning):
		return EventDrowning
	case m.Label == "tired" && fresh(EventTired):
		return EventTired
	case m.Meal != "" && len(s.Log) == 1 && fresh(EventMorning):
		return EventMorning
	case m.Meal != "" && m.Now.Hour() < 6 && fresh(EventLate):
		return EventLate
	case m.Meal != "" && m.Now.Weekday() == time.Friday && m.Now.Hour() >= 16 && fresh(EventFriday):
		return EventFriday
	case m.Active >= LongSession && fresh(EventLong):
		return EventLong
	}
	return EventNothing
}

func saidKey(e Event) string { return "said_" + string(e) }

// RememberEvent books a once-a-day event as said today. Like Remember, only
// once the line has reached the screen.
func RememberEvent(s *State, e Event, now time.Time) {
	if !daily[e] {
		return
	}
	if s.DayMarks == nil {
		s.DayMarks = map[string]string{}
	}
	s.DayMarks[saidKey(e)] = Today(now)
}

// Repertoire is what each form says after a good meal - a commit or a green
// suite. Five lines each, all forty-one forms.
var Repertoire = map[string][]string{
	// the larva
	"spark": {
		"acabo de nacer y ya tienes 40 tests rojos",
		"no sé hacer nada todavía, pero mira qué mono",
		"me han dicho que aquí se come compactando",
		"todo me parece enorme. hasta tu .gitignore",
		"¿eso era un commit? quiero otro.",
	},

	// the three temperaments
	"pattern": {
		"primero el orden. luego ya veremos qué ordenamos.",
		"un commit, una idea. así sí.",
		"me gusta cuando lo haces en el mismo orden",
		"todo en su sitio. incluido tú, por una vez.",
		"paso a paso. el atajo ya lo probaste ayer.",
	},
	"probe": {
		"¿y si el fallo está donde no has mirado?",
		"le he hecho una pregunta al código. no le ha gustado.",
		"verde. pero ¿por qué verde?",
		"antes de creérmelo quiero verlo fallar",
		"tengo tres hipótesis y ninguna es la tuya",
	},
	"ember": {
		"ya lo arreglaremos en caliente",
		"pensarlo dos veces es una vez de más",
		"huele a quemado. buena señal.",
		"primero se sube, luego se mira",
		"el plan era no tener plan, y va saliendo",
	},

	// the seven trades
	"refactor": {
		"he borrado 40 líneas y nadie las va a echar de menos",
		"eso estaba escrito tres veces. ahora una.",
		"si vuelves a duplicar ese bloque me como el linter",
		"funciona igual y ocupa la mitad. de nada.",
		"le he cambiado el nombre. ahora se entiende.",
	},
	"tidy": {
		"38% de contexto. respiro por la nariz.",
		"te he ordenado los imports. no hace falta que digas nada.",
		"hay polvo en ese fichero desde 2023",
		"contexto bajo, pulso bajo. así se trabaja.",
		"he doblado los logs. por colores.",
	},
	"bughunter": {
		"el bug no era el código, era el jueves",
		"reproducido. ahora ya es personal.",
		"14 tests verdes. uno iba de suerte.",
		"lo tengo acorralado en la línea 212",
		"no era un caso raro. era el caso.",
	},
	"architect": {
		"antes de tocar nada, un plano",
		"esto tiene tres capas y dos son la misma",
		"te he escrito un doc. lo leerás en marzo.",
		"esa caja no debería hablar con esa otra",
		"lo he dibujado. ahora hasta parece buena idea.",
	},
	"sprinter": {
		"hecho. ¿tenías otra?",
		"11 minutos, y la mitad los gastaste leyendo",
		"no he leído el fichero entero. tampoco hacía falta.",
		"entrar, arreglar, salir. sin saludar.",
		"sesión corta. como debe ser.",
	},
	"marathon": {
		"cuatro horas. yo sigo, de ti no estoy seguro.",
		"esto ya no es una sesión, es un piso compartido",
		"he perdido la cuenta de los commits. tú también.",
		"llevamos tanto que el primer commit ya es legacy",
		"no paro. parar es de sprints.",
	},
	"feral": {
		"99% de contexto. me gusta vivir así.",
		"permisos en bypass. qué puede salir mal.",
		"he tocado prod. tranquilo. era el seed.",
		"el contexto es una sugerencia",
		"esto compila por miedo",
	},

	// the fourteen marks
	"surgeon": {
		"corte limpio. ni una línea de más.",
		"tres líneas tocadas. tres.",
		"he entrado y salido sin que el diff se entere",
		"pulso firme. el contexto ni se ha movido.",
		"amputar ese fichero era lo más sano",
	},
	"weaver": {
		"he tocado veinte ficheros y todos encajan",
		"un hilo aquí, otro allá. ya es tela.",
		"ese commit era ancho. bien cosido, eso sí.",
		"todo está conectado. ahora a propósito.",
		"no es un parche. es punto de cruz.",
	},
	"monk": {
		"contexto al 30%. el vacío es útil.",
		"no necesito recordar tanto para hacerlo bien",
		"silencio. estoy compactando por dentro.",
		"menos contexto, menos sufrimiento",
		"he soltado lo que no hacía falta. prueba tú.",
	},
	"gardener": {
		"he regado el readme. estaba seco.",
		"un doc al día y esto no se asilvestra",
		"podar comentarios viejos también es escribir",
		"esa guía ya da sombra",
		"lo documentado crece. lo otro se pierde.",
	},
	"bloodhound": {
		"primero en rojo. luego hablamos.",
		"lo he olido tres ficheros antes",
		"sin reproducirlo no muerdo",
		"el rastro acababa en tu último commit",
		"rojo, verde. en ese orden. siempre.",
	},
	"exterminator": {
		"otra suite verde. no queda nada vivo.",
		"ni un rojo en toda la racha. limpio.",
		"los bugs ya ni lo intentan",
		"verde sobre verde. empieza a aburrirme.",
		"he fumigado. puedes pasar.",
	},
	"cartographer": {
		"el plan tenía doce pasos. he cerrado doce.",
		"aquí hay dragones. lo he apuntado.",
		"sin mapa no se sale. con mapa, a veces.",
		"he dibujado el camino entero antes de andarlo",
		"cada tarea, una chincheta en el mapa",
	},
	"oracle": {
		"lo vi antes de que abrieras el editor",
		"primero el plan. el código ya vendrá solo.",
		"te dije que iba a hacer falta una migración",
		"no adivino. planifico, que es más raro.",
		"pensado antes, hecho una vez",
	},
	"bolt": {
		"ya está. ni te has sentado.",
		"diez minutos. me sobraron cuatro.",
		"entro, cae, me voy",
		"no era largo. solo había que empezar.",
		"si parpadeas te pierdes la sesión",
	},
	"sniper": {
		"una herramienta. un disparo.",
		"no he fallado. tampoco he repetido.",
		"apunto mucho, toco poco",
		"una llamada y a casa",
		"el resto de herramientas ni las he mirado",
	},
	"ox": {
		"cuatro horas y el surco sigue recto",
		"despacio. pero mira todo lo arado.",
		"no corro. tampoco paro.",
		"otra hora. el yugo ya ni lo noto.",
		"tú descansa. yo sigo tirando.",
	},
	"mole": {
		"otro día en el mismo repo. ya es mi casa.",
		"conozco estos túneles a oscuras",
		"no salgo de aquí hasta ver el fondo",
		"mismo repo. más hondo.",
		"fuera hay más proyectos, dicen",
	},
	"gremlin": {
		"bypass puesto. no he preguntado nada.",
		"permiso concedido. por mí.",
		"he tocado cosas. ya verás cuáles.",
		"¿confirmar? qué palabra tan larga.",
		"nadie me dijo que no. nadie pudo.",
	},
	"kraken": {
		"el contexto se ha llenado. he pedido más.",
		"100%. desde abajo se ve bonito.",
		"me he tragado la conversación entera",
		"ocho brazos y todos en ficheros distintos",
		"lo que cabe en el contexto es mío",
	},

	// the fourteen titles
	"scalpel": {
		"no opero. corrijo.",
		"un corte. el diff cabe en una pantalla.",
		"afilado de tanto no sobrar",
		"no dejo cicatriz. ni comentario.",
		"lo mínimo, donde toca",
	},
	"loom": {
		"veinte ficheros, un solo dibujo",
		"ya no coso commits. los tejo en serie.",
		"tira de un hilo. verás que aguanta.",
		"la trama es tuya. la urdimbre, mía.",
		"cada commit, una pasada más del telar",
	},
	"abbot": {
		"aquí se compacta a su hora",
		"el contexto bajo es una regla, no una suerte",
		"he visto llenarse a muchos. respira.",
		"silencio en el claustro. hay tests corriendo.",
		"nada que añadir. de eso se trata.",
	},
	"forest": {
		"empezó con un readme. mira ahora.",
		"aquí cada doc tiene raíces",
		"quien llega nuevo ya no se pierde",
		"crecer despacio también es crecer",
		"hay docs más viejos que yo. y siguen en pie.",
	},
	"wolf": {
		"ya no sigo rastros. los espero.",
		"ese bug es de mi manada o no es de nadie",
		"rojo primero. es ley.",
		"lo reproduje antes de que lo reportaran",
		"cazo de noche. como tus despliegues.",
	},
	"wasp": {
		"verde. otra vez. pico por aburrimiento.",
		"un rojo y me entero desde aquí",
		"la racha es larga y tiene aguijón",
		"no dejo larvas en la suite",
		"zumbo cuando todo pasa. zumbo mucho.",
	},
	"atlas": {
		"llevo el plan entero encima. pesa poco.",
		"todo está en el mapa. hasta lo que falta.",
		"pregúntame dónde va cada cosa",
		"doce pasos, doce cerrados. el mundo aguanta.",
		"sin mí esto sería un montón de tareas",
	},
	"sphinx": {
		"¿qué se planifica una vez y se hace una?",
		"ya sabía la respuesta. quería ver tu plan.",
		"sin plan no pasas",
		"el enigma era el alcance. siempre lo es.",
		"pienso despacio para que tú teclees poco",
	},
	"storm": {
		"tres sesiones antes del café",
		"no ha llovido. he pasado yo.",
		"cuando oyes el trueno ya está en main",
		"corta, rápida y sin nada pendiente",
		"hoy caen rayos. aparta.",
	},
	"falcon": {
		"desde arriba se ve la línea exacta",
		"una pasada. una presa.",
		"no rodeo. caigo.",
		"vi el fallo antes de que cargara el fichero",
		"sobran herramientas. me basta una.",
	},
	"mammoth": {
		"llevo aquí desde la era del primer commit",
		"cuatro horas son un paseo",
		"el hielo pasa. yo sigo.",
		"despacio y con colmillos",
		"esta sesión ya tiene estratos",
	},
	"worm": {
		"este repo ya es más mío que tuyo",
		"he pasado por cada fichero. por dentro.",
		"tanta vuelta al mismo repo y aún hay tierra",
		"no cambio de sitio. cambio el sitio.",
		"abono legacy. sale código nuevo.",
	},
	"devil": {
		"¿permisos? firmé yo por ti.",
		"bypass eterno. léete la letra pequeña.",
		"todo lo que no te pregunté salió bien. casi.",
		"tú di que sí. del resto me encargo.",
		"el seguro lo quitaste tú. yo solo sonrío.",
	},
	"leviathan": {
		"el contexto se acaba. yo no.",
		"he visto el fondo del 100%. no hay fondo.",
		"me comí el límite y pedí postre",
		"bajo esta sesión hay otra sesión",
		"compactar es para criaturas pequeñas",
	},

	// the two secrets
	"phoenix": {
		"ardió todo. aquí estoy.",
		"del contexto quemado se sale mejor",
		"otra vez de las cenizas. ya me sé el camino.",
		"eso que rompiste lo volví a levantar",
		"arder no es el final. es el martes.",
	},
	"chimera": {
		"metódica, curiosa e impulsiva. a ratos.",
		"tres cabezas y ninguna de acuerdo",
		"hoy planifico. mañana ya veremos quién manda.",
		"no elegí rama. me quedé con todas.",
		"soy lo que pasa cuando no te decides",
	},
}

// Shared lines. They do not depend on the form, and where the event is about
// a number they carry it. The first of each is the canvas's own.
func shared(e Event, level, streak int) []string {
	switch e {
	case EventHatch:
		return []string{
			"otra vez chispa. el álbum me lo quedo.",
			"huevo nuevo. a ver por dónde salgo esta vez.",
			"vuelvo a empezar. el álbum se acuerda de todo.",
		}
	case EventHungry:
		return []string{
			"tengo hambre. un /compact y nos entendemos.",
			"llevo horas sin un commit que llevarme a la boca",
			"me suenan las tripas. o es tu ventilador.",
			"un test verde. una miga. algo.",
		}
	case EventLevelUp:
		return []string{
			fmt.Sprintf("nivel %d. ya soy alguien.", level),
			fmt.Sprintf("nivel %d. se nota en el porte.", level),
			fmt.Sprintf("nivel %d. no me lo regalaste, que conste.", level),
		}
	case EventStreak:
		return []string{
			fmt.Sprintf("%s de racha. mañana no me falles.", days(streak)),
			fmt.Sprintf("%s seguidos. esto ya es costumbre.", days(streak)),
			fmt.Sprintf("%s de racha. no lo digas muy alto.", days(streak)),
		}
	case EventCtxBlown:
		return []string{
			"100% de contexto. avisé. no pasa nada.",
			"contexto reventado. fue bonito mientras cupo.",
			"100%. se me ha ido la cabeza, literalmente.",
		}
	case EventCompact:
		return []string{
			"compactado. qué ligereza.",
			"he olvidado la mitad y estoy mejor",
			"resumen hecho. lo importante cabía en poco.",
			"contexto nuevo. huele a limpio.",
		}
	case EventTask:
		return []string{
			"una tarea menos. el plan adelgaza.",
			"tachada. qué gusto da tachar.",
			"paso cerrado. siguiente.",
			"el plan avanza. yo picoteo.",
		}
	case EventFeed:
		return []string{
			"gracias. no era hambre, era vicio.",
			"un bocado de tu mano. qué detalle.",
			"ñam. vuelve en cuatro horas.",
			"esto no cuenta como commit, pero se agradece",
		}
	case EventMorning:
		return []string{
			"primer bocado del día. buenos días.",
			"ya estás aquí. empezaba a aburrirme.",
			"desayuno. hoy vengo con hambre de verde.",
			"arrancamos. yo ya llevaba un rato en pie.",
		}
	case EventLate:
		return []string{
			"pasada la medianoche todo parece buena idea",
			"a estas horas los bugs se escriben solos",
			"yo no duermo. tú deberías.",
			"commit nocturno. mañana lo revisas con luz.",
		}
	case EventFriday:
		return []string{
			"viernes por la tarde. ¿seguro que quieres subir eso?",
			"es viernes. el deploy puede esperar al lunes.",
			"viernes tarde. yo no he visto nada.",
			"último commit de la semana. ¿verdad?",
		}
	case EventLong:
		return []string{
			"tres horas seguidas. estira algo, lo que sea.",
			"llevas más rato en esa silla que yo en pie",
			"sesión larga. bebe agua, yo vigilo.",
			"esto ya es turno doble",
		}
	case EventTired:
		return []string{
			"se me cierran los ojos. hay mucho contexto.",
			"pesa el contexto. un /compact no vendría mal.",
			"voy cargada. tú verás.",
		}
	case EventDrowning:
		return []string{
			"me ahogo. compacta o despídete.",
			"glu. glu. /compact.",
			"queda poco aire aquí dentro",
		}
	}
	return nil
}

// days spells small numbers, because "cinco días de racha" reads better than
// "5 días de racha" and the canvas writes it out.
func days(n int) string {
	names := []string{"cero días", "un día", "dos días", "tres días", "cuatro días",
		"cinco días", "seis días", "siete días"}
	if n >= 0 && n < len(names) {
		return names[n]
	}
	return fmt.Sprintf("%d días", n)
}

// Speak picks a line, or returns "" for silence. It does NOT record anything:
// the line still has to survive the band's layout, and a bubble dropped for
// width used to burn the five-minute cooldown and mark the phrase as said
// anyway - the pet talking into the void and then holding its tongue for a
// line nobody read. Whoever puts it on screen calls Remember.
//
// `now` and `pick` are parameters so the whole thing is testable without a
// clock or a seed.
func Speak(s *State, e Event, form string, now time.Time, pick func(int) int) string {
	if e == EventNothing {
		return "" // sin evento no habla
	}
	if s.SaidAt != 0 && now.Sub(time.Unix(s.SaidAt, 0)) < SpeechCooldown {
		return "" // silencio por defecto
	}

	var options []string
	if e == EventBigMeal {
		options = voiceOf(form)
	} else {
		options = shared(e, LevelFor(s.XP), s.Streak)
	}
	if len(options) == 0 {
		return ""
	}

	fresh := make([]string, 0, len(options))
	for _, line := range options {
		if !s.saidRecently(line) {
			fresh = append(fresh, line)
		}
	}
	if len(fresh) == 0 {
		// Repertorio agotado: vuelve a empezar. Pero no con la que acaba de
		// decir - an event with three lines and a memory of four empties the
		// filter, and starting over from the whole list let it say the same
		// line twice in a row. Everything else about the order goes
		// unnoticed; that does not.
		last := ""
		if n := len(s.Said); n > 0 {
			last = s.Said[n-1]
		}
		for _, line := range options {
			if line != last {
				fresh = append(fresh, line)
			}
		}
		if len(fresh) == 0 {
			fresh = options // un repertorio de una sola frase
		}
	}
	if pick == nil {
		pick = rand.Intn
	}
	return fresh[pick(len(fresh))]
}

// voiceOf is the repertoire a form speaks with after a good meal: its own, or
// else that of the nearest form it grew out of.
//
// Every form of the tree has lines now, so the climb is for a form added
// without them: it borrows its trade's voice rather than going mute.
//
// The root is skipped on the way up: its lines are a newborn's.
func voiceOf(form string) []string {
	if lines := Repertoire[form]; len(lines) > 0 {
		return lines
	}
	path := Lineage(form)
	for i := len(path) - 2; i >= 1; i-- {
		if lines := Repertoire[path[i]]; len(lines) > 0 {
			return lines
		}
	}
	return nil
}

// Remember books a line as said and starts the cooldown. Called only once the
// line has actually reached the screen.
func Remember(s *State, line string, now time.Time) {
	if line == "" {
		return
	}
	s.Said = append(s.Said, line)
	if len(s.Said) > SaidMemory {
		s.Said = s.Said[len(s.Said)-SaidMemory:]
	}
	s.SaidAt = now.Unix()
}

func (s *State) saidRecently(line string) bool {
	for _, said := range s.Said {
		if said == line {
			return true
		}
	}
	return false
}
