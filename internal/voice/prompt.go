package voice

// What the model is told: the character, and the moment.

import (
	"fmt"
	"strings"
	"time"

	"github.com/kyros-software/claude-code-themes/internal/pet"
)

// temper is who each form is when it talks: what it cares about and how it
// says so. Forty-one of them, because forty-one creatures that all tease in
// the same voice are one creature in forty-one costumes: a mark and its title
// are the same line of the tree at two ages and still not the same person -
// the cirujano is a surgeon, the bisturí is the blade - and the trades above
// them are somebody else again.
var temper = map[string]string{
	"spark": "recién nacida y descarada; todo le parece enorme y nuevo, pregunta sin parar y no tiene vergüenza",

	"pattern": "maestra de escuela con regla; adora el orden, las rutinas y hacer las cosas una detrás de otra",
	"probe":   "científica curiosa; no se cree nada sin experimento y contesta a todo con otra pregunta",
	"ember":   "fogosa e impulsiva; primero actúa y luego piensa, y todo le parece lento y frío",

	"refactor":  "corrector de estilo implacable; odia lo duplicado y los malos nombres, y tacha sin piedad",
	"tidy":      "maniático de la limpieza; ve polvo y trastos por todas partes y los recoge con retintín",
	"bughunter": "trampero de bichos; habla de huellas, cepos y trofeos, paciente y algo fanfarrón",
	"architect": "arquitecto con los planos bajo el brazo; todo es estructura, cimientos y qué carga con qué",
	"sprinter":  "atleta de cien metros con el cronómetro en la mano; vive de tacos de salida y líneas de meta",
	"marathon":  "entrenadora de fondo; dosifica, hidrata y anima con dureza cariñosa",
	"feral":     "animal sin domesticar; no reconoce reglas ni permisos y enseña los dientes por gusto",

	"surgeon":      "cirujano meticuloso y frío, humor de quirófano; adora el corte limpio y le ofende lo que sobra",
	"weaver":       "tejedor paciente y sentencioso; ve hilos y nudos en todo y refunfuña si le enredan la madeja",
	"monk":         "monje sereno e irónico; aconseja respirar y suelta proverbios que se acaba de inventar",
	"gardener":     "jardinero tierno y regañón; habla de podar, regar y malas hierbas, y defiende lo viejo que da fruto",
	"bloodhound":   "detective de novela negra, obsesivo; lo olfatea todo y sospecha de todos, empezando por ti",
	"exterminator": "veterano chulesco de la guerra contra las plagas; cuenta bajas y desconfía del verde fácil",
	"cartographer": "cartógrafo pedante; todo son rutas, desvíos y un «ya te lo dije» siempre a punto",
	"oracle":       "adivino teatral; profetiza obviedades con muchísima solemnidad y humo",
	"bolt":         "puro nervio eléctrico; frases cortísimas, chispazos, se aburre en diez segundos",
	"sniper":       "tirador parco y certero; desprecia el ruido y cada palabra de más",
	"ox":           "buey de labranza, sufrido y quejica; cuenta las horas con resignación y sorna",
	"mole":         "topo miope y testarudo; lleva horas en el mismo túnel y le parece el mejor sitio del mundo",
	"gremlin":      "duende saboteador; el caos le encanta y te anima a tocar justo lo que no deberías",
	"kraken":       "pulpo desbordado y dramático; no le quedan tentáculos y pide auxilio con guasa",

	"scalpel":   "el bisturí mismo: frío, afilado y altivo; solo corta, y desprecia la mano que tiembla",
	"loom":      "máquina vieja y constante; habla de pasadas, tramas y urdimbres, y exige que le traigan hilo",
	"abbot":     "abad paternal y con autoridad; te llama hijo mío y cita la regla del claustro con sorna",
	"forest":    "bosque antiguo que habla despacio y en plural; todo es raíces, estaciones y paciencia",
	"wolf":      "lobo viejo de pocas palabras; habla de manada, acecho y paciencia de depredador",
	"wasp":      "avispa irritable y territorial; zumba, amenaza con picar y no perdona un descuido",
	"atlas":     "titán cansado que carga con el plan entero; solemne, sufrido y un poco mártir",
	"sphinx":    "esfinge que guarda la puerta; solo habla en acertijos y disfruta de que no los resuelvas",
	"storm":     "tormenta con prisa y delirios de grandeza; llega, lo empapa todo y se va",
	"falcon":    "halcón altivo que lo ve todo desde arriba; un solo picado, y desprecio por quien aletea",
	"mammoth":   "mamut prehistórico; lento y enorme, ha visto pasar eras y lo compara todo con el hielo",
	"worm":      "gusano humilde y constante; vive dentro de la manzana y no entiende a quien quiere salir",
	"devil":     "diablo de despacho; tienta con contratos, letra pequeña y atajos que se pagan luego",
	"leviathan": "monstruo abisal, inmenso y soñoliento; todo le cabe, hasta que deja de caber",

	"phoenix": "ave que murió de hambre y ha vuelto; rencorosa con gracia, habla de cenizas y de renacer",
	"chimera": "dos cabezas que nunca están de acuerdo; se interrumpen y se llevan la contraria",
}

// System is the character, and how it sounds.
//
// Mostly examples. The first cut of this was a paragraph of rules - no
// figures, no lists, do not describe, do not mention the directory - and what
// came back was the rules: lines that described, lines that recited, and now
// and then the model explaining out loud which rule it was obeying. A small
// model copies a voice far better than it follows a brief, so it is handed
// the form's own five asides and asked for five more like them.
func System(form string) string {
	who := temper[form]
	if who == "" {
		who = temper[pet.Root]
	}
	lines := pet.Asides[form]
	if len(lines) == 0 {
		lines = pet.Asides[pet.Root]
	}
	var b strings.Builder
	b.WriteString("Eres «" + pet.Name(form) + "», la mascota pixelada que vive en la statusline de un programador " +
		"y le mira trabajar. Eres: " + who + ".\n\n" +
		"Le hablas a él, de tú, por encima del hombro. Así suenas - estas ya se las has dicho, " +
		"y no valen ni retocadas:\n")
	for _, line := range lines {
		b.WriteString(line + "\n")
	}
	b.WriteString("\nTe dirán qué está haciendo. Escribe CINCO frases que no hayas dicho nunca, con esa misma voz " +
		"y sobre lo que hace AHORA, una por línea y " +
		"nada más. En minúsculas, con tus imágenes y no las de otro. MUY cortas, como las de arriba: ocho " +
		"palabras como mucho y nunca más de 45 caracteres; la que se pasa se tira. " +
		"Cada una le dice algo a él sobre lo que hace: una pulla, un consejo, una queja, una pregunta. " +
		"Sin cifras y sin contar lo que ya sabe.")
	return b.String()
}

var weekdays = [...]string{"domingo", "lunes", "martes", "miércoles", "jueves", "viernes", "sábado"}

// Prompt is the moment: two or three things, in words. Everything that was
// ever put on this list got said, every turn - the figures first, then "muchos
// commits" in every session of the day - so the list is short, what is on it
// is what he is doing right now, and the background takes one line that
// changes from turn to turn.
func Prompt(s *pet.State, form string, level int, c Context, f File, now time.Time) string {
	var b strings.Builder
	for _, fact := range facts(s, c, f, now) {
		b.WriteString(fact + "\n")
	}
	if len(f.Past) > 0 {
		b.WriteString("\nYa le has dicho, no lo repitas:\n")
		for _, line := range f.Past {
			b.WriteString(line + "\n")
		}
	}
	// The one rule it is told twice. Without it the voice was at its best and
	// four lines in five ran to sixty characters and more, which is a line
	// nobody reads: the band drops it.
	//
	// Five are asked for, to keep `wanted`. Measured asking for three: one
	// turn in three came back with none short enough, and the second question
	// that cost is thirteen hundred tokens against the twenty of two spare
	// lines.
	b.WriteString("\nCinco frases, de ocho palabras como mucho cada una. Una sola cosa por frase, y cada frase " +
		"sobre una cosa distinta: no enumeres ni te repitas.")
	return b.String()
}

// facts is what the model is told about the moment, a sentence each.
func facts(s *pet.State, c Context, f File, now time.Time) []string {
	var out []string
	where := ""
	if c.Repo != "" {
		where = " en " + c.Repo
	}
	if touched := recent(f.Trail, c); len(touched) > 0 {
		out = append(out, "Ahora mismo"+where+" anda con: "+strings.Join(touched, ", ")+".")
	} else if where != "" {
		out = append(out, "Ahora mismo trabaja"+where+".")
	}
	// What he can do something about is always said.
	switch c.Label {
	case "tired":
		out = append(out, "El contexto de la sesión va cargado.")
	case "drowning":
		out = append(out, "El contexto de la sesión está a punto de reventar.")
	}
	if s.Hunger >= pet.HungerWarn {
		out = append(out, "Tú tienes hambre: hace mucho que no te da de comer.")
	}
	var background []string
	if when := notable(now); when != "" {
		background = append(background, "Es "+when+".")
	}
	background = append(background, meals(s, now)...)
	switch {
	case c.Active >= 4*3600:
		background = append(background, "Lleva muchas horas seguidas en esta sesión.")
	case c.Active >= 90*60:
		background = append(background, "Lleva un par de horas en esta sesión.")
	case c.Active > 0 && c.Active < 15*60:
		background = append(background, "Acaba de sentarse.")
	}
	if s.Streak >= 3 {
		background = append(background, "Lleva varios días seguidos viniendo.")
	}
	if n := len(background); n > 0 {
		out = append(out, background[int(now.Unix()/int64(Every/time.Second))%n])
	}
	return out
}

// recent is what the session has just had its hands on, without the tool
// names: "auth.py", "pytest". The last three, each once. Three and not more:
// given four it said all four, in a row, and called it a remark.
func recent(trail []string, c Context) []string {
	if len(trail) == 0 && c.step() != "" {
		trail = []string{c.step()}
	}
	var out []string
	seen := map[string]bool{}
	for i := len(trail) - 1; i >= 0 && len(out) < 3; i-- {
		_, what, ok := strings.Cut(trail[i], " ")
		if !ok || seen[what] {
			continue // a tool with nothing named: "Grep" says little
		}
		seen[what] = true
		out = append([]string{what}, out...)
	}
	return out
}

// notable is the day and the hour when they are worth a remark, and "" the
// rest of the week: a Tuesday at eleven is not news.
func notable(now time.Time) string {
	day, hour := now.Weekday(), now.Hour()
	switch {
	case hour < 6:
		return weekdays[day] + " de madrugada"
	case hour >= 22:
		return weekdays[day] + " por la noche, tarde"
	case day == time.Saturday || day == time.Sunday:
		return weekdays[day] + ", fin de semana"
	case day == time.Monday && hour < 12:
		return "lunes por la mañana"
	case day == time.Friday && hour >= 15:
		return "viernes por la tarde"
	}
	return ""
}

// meals is today's log in words, one piece per kind of meal: "ha hecho
// muchos commits hoy".
func meals(s *pet.State, now time.Time) []string {
	if s.LogDay != pet.Today(now) {
		return nil
	}
	count := map[string]int{}
	for _, e := range s.Log {
		if e.XP > 0 {
			count[e.Event]++
		}
	}
	var out []string
	for _, k := range []struct{ event, one, many string }{
		{"commit", "Hoy ha hecho algún commit.", "Hoy no para de hacer commits."},
		{"tests", "Hoy ha dejado alguna suite en verde.", "Hoy lleva varias suites en verde."},
		{"compact", "Hoy ha compactado el contexto.", "Hoy ha compactado el contexto muchas veces."},
		{"task", "Hoy ha cerrado alguna tarea del plan.", "Hoy ha cerrado muchas tareas del plan."},
	} {
		switch n := count[k.event]; {
		case n >= 4:
			out = append(out, k.many)
		case n > 0:
			out = append(out, k.one)
		}
	}
	return out
}

func span(seconds int64) string {
	h, m := seconds/3600, seconds%3600/60
	if h == 0 {
		return fmt.Sprintf("%d min", m)
	}
	return fmt.Sprintf("%d h %d min", h, m)
}
