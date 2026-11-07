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
//   - Primero el evento, luego la forma. The four shared lines belong to
//     specific events and everyone says them; a form's own repertoire is its
//     voice for a good meal. A mark or a title borrows the voice of the trade
//     it grew out of (see voiceOf); the three temperaments and the two
//     secrets, which hang off no trade, only ever say the shared ones.
//   - Sin repetir. The last SaidMemory phrases live in pet.json and do not come
//     back until the repertoire is exhausted.
//   - Silencio por defecto. At most one line every SpeechCooldown, and without
//     an event it says nothing: the statusline is not a chat.

// SaidMemory is how many phrases back the pet remembers not to repeat.
const SaidMemory = 3

// SpeechCooldown is the floor between two lines.
const SpeechCooldown = 5 * time.Minute

// Event is what opens the pet's mouth.
type Event string

// The five things worth saying something about.
const (
	EventNothing  Event = ""
	EventBigMeal  Event = "big_meal"
	EventHungry   Event = "hungry"
	EventLevelUp  Event = "level_up"
	EventStreak   Event = "streak"
	EventCtxBlown Event = "ctx_blown"
)

// Repertoire is what each form says after a good meal. Forms absent from this
// table fall back to the shared lines.
var Repertoire = map[string][]string{
	"spark": {
		"acabo de nacer y ya tienes 40 tests rojos",
		"no sé hacer nada todavía, pero mira qué mono",
		"me han dicho que aquí se come compactando",
	},
	"refactor": {
		"he borrado 40 líneas y nadie las va a echar de menos",
		"eso estaba escrito tres veces. ahora una.",
		"si vuelves a duplicar ese bloque me como el linter",
	},
	"tidy": {
		"38% de contexto. respiro por la nariz.",
		"te he ordenado los imports. no hace falta que digas nada.",
		"hay polvo en ese fichero desde 2023",
	},
	"bughunter": {
		"el bug no era el código, era el jueves",
		"reproducido. ahora ya es personal.",
		"14 tests verdes. uno iba de suerte.",
	},
	"architect": {
		"antes de tocar nada, un plano",
		"esto tiene tres capas y dos son la misma",
		"te he escrito un doc. lo leerás en marzo.",
	},
	"sprinter": {
		"hecho. ¿tenías otra?",
		"11 minutos, y la mitad los gastaste leyendo",
		"no he leído el fichero entero. tampoco hacía falta.",
	},
	"marathon": {
		"cuatro horas. yo sigo, de ti no estoy seguro.",
		"esto ya no es una sesión, es un piso compartido",
		"he perdido la cuenta de los commits. tú también.",
	},
	"feral": {
		"99% de contexto. me gusta vivir así.",
		"permisos en bypass. qué puede salir mal.",
		"he tocado prod. tranquilo. era el seed.",
	},
}

// Shared lines. They do not depend on the form, and they carry the number the
// event is about.
func shared(e Event, level, streak, hunger int) string {
	switch e {
	case EventHungry:
		return "tengo hambre. un /compact y nos entendemos."
	case EventLevelUp:
		return fmt.Sprintf("nivel %d. ya soy alguien.", level)
	case EventStreak:
		return fmt.Sprintf("%s de racha. mañana no me falles.", days(streak))
	case EventCtxBlown:
		return "100% de contexto. avisé. no pasa nada."
	}
	return ""
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
	} else if line := shared(e, LevelFor(s.XP), s.Streak, s.Hunger); line != "" {
		options = []string{line}
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
		// decir - a form has three lines and SaidMemory is three, so the
		// filter empties every third time, and starting over from the whole
		// list let it say the same line twice in a row. Everything else about
		// the order goes unnoticed; that does not.
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
// else that of the nearest trade it grew out of.
//
// The canvas gave lines to the root and the seven trades only, and read
// literally that left the fourteen marks and the fourteen titles mute after a
// good meal - for good, since a title is where every pet ends up. A `scalpel`
// is still a `refactor`, so it talks like one. (v3, 7-nov-2026: the canvas is
// no longer the authority on this; the people who keep the pet are.)
//
// The root is skipped on the way up: its lines are a newborn's, and a
// temperament has outgrown them.
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
