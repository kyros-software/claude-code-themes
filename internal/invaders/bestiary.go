package invaders

import "github.com/kyros-software/claude-code-themes/internal/theme"

// The bestiary, straight off the design canvas.
//
// Do not hand-edit: change the canvas. Forty troop sprites in eight stages from
// "Bichitos por Stage", thirty-five bigger ones in five ranks from "Sprites
// Marcianitos v2". Every row is exactly as many cells wide as it claims - five
// for a troop, nine for a rank - and TestEverySpriteIsTheWidthItClaims is what
// keeps that true after a paste.
//
// Both are drawn the same way: a body in the stage's own tone, and the three
// cells of eyes in the light one. That is the only thing that breaks the flat
// colour, and it is what makes a wave of fifty-five readable at a glance.

// Palette is a stage's or a rank's colours and what its members are worth.
type Palette struct {
	Label  string
	Points int
	Ramp   [7]theme.Colour
	Tones  [3]theme.Colour
	Eye    theme.Colour
}

// Craft is one enemy ship: line art, three to nine cells across, flying on its
// own and shooting back.
//
// This replaced the arcade's marching block, and the block replaced a canvas of
// forty drawn sprites before it. What each version was for is worth keeping
// straight. The canvas sprites were five cells by three and did not read as
// Space Invaders. The block did, at one glyph a member, fifty-five of them
// stepping down together - and its whole shape is that it moves as ONE thing,
// so there is exactly one decision on screen at a time.
//
// A fleet is the other game. Ten kinds, each with its own fall, its own drift
// and its own gun, arriving in ones and twos: what is in front of you is never
// the same twice, and every ship on the field is a separate thing to read. That
// is what the reference plays like, and it is what was asked for.
//
// Every row is exactly W cells wide - TestEverySpriteIsTheWidthItClaims says so
// - because the painter blits rows and a short row would leave the hull open.
type Craft struct {
	Name, Desc string
	Rows       []string
	W, H       int
	HP         int     // before the wave's own bonus
	Fall       float64 // rows per tick
	Drift      float64 // cells per tick, sideways, bouncing off the walls
	Cadence    int     // ticks between its shots
	Points     int
	Stage      int // the first stage it turns up in
}

// Fleet is the ten, in the order they are unlocked. A wave draws from those
// whose Stage it has reached, so wave one is drones and wasps and by stage
// eight the whole zoo is out - the same progression the block had through its
// colours, except now the shapes change too.
var Fleet = []Craft{
	{Name: "zángano", Desc: "el que llega primero, y el que menos vale", Stage: 1,
		W: 3, H: 2, HP: 1, Fall: 0.03, Drift: 0.06, Cadence: 180, Points: 10,
		Rows: []string{"_^_", "<o>"}},
	{Name: "avispa", Desc: "rápida de lado, no aguanta nada", Stage: 1,
		W: 5, H: 2, HP: 1, Fall: 0.045, Drift: 0.125, Cadence: 160, Points: 15,
		Rows: []string{"\\_ _/", "<ooo>"}},
	{Name: "lanza", Desc: "baja recta y deprisa", Stage: 2,
		W: 3, H: 3, HP: 2, Fall: 0.065, Drift: 0.02, Cadence: 140, Points: 20,
		Rows: []string{" ^ ", "/o\\", " v "}},
	{Name: "arpía", Desc: "cruza mientras cae", Stage: 3,
		W: 5, H: 3, HP: 2, Fall: 0.035, Drift: 0.15, Cadence: 120, Points: 25,
		Rows: []string{" ^^^ ", "<-o->", " v v "}},
	{Name: "tejedora", Desc: "la que no está donde apuntaste", Stage: 4,
		W: 5, H: 3, HP: 3, Fall: 0.03, Drift: 0.18, Cadence: 130, Points: 30,
		Rows: []string{"/\\ /\\", "(-o-)", "\\/ \\/"}},
	{Name: "cazador", Desc: "dispara más que ninguno de su tamaño", Stage: 5,
		W: 7, H: 3, HP: 3, Fall: 0.04, Drift: 0.1, Cadence: 100, Points: 35,
		Rows: []string{"\\__ __/", "-<ooo>-", "/  v  \\"}},
	{Name: "yunque", Desc: "lento y duro: hay que dedicarle tiempo", Stage: 5,
		W: 7, H: 3, HP: 5, Fall: 0.02, Drift: 0.05, Cadence: 150, Points: 40,
		Rows: []string{".-----.", "|o-o-o|", "'--v--'"}},
	{Name: "mantis", Desc: "cae a plomo y dispara al caer", Stage: 6,
		W: 5, H: 4, HP: 5, Fall: 0.05, Drift: 0.11, Cadence: 90, Points: 45,
		Rows: []string{" ^ ^ ", "\\o o/", " \\_/ ", "  v  "}},
	{Name: "coraza", Desc: "seis impactos, y ni se despeina", Stage: 7,
		W: 7, H: 4, HP: 6, Fall: 0.025, Drift: 0.06, Cadence: 110, Points: 55,
		Rows: []string{"___ ___", "[o-o-o]", "'-|-|-'", "  v v  "}},
	{Name: "abisal", Desc: "el más grande que no es un jefe", Stage: 8,
		W: 9, H: 4, HP: 8, Fall: 0.02, Drift: 0.08, Cadence: 80, Points: 80,
		Rows: []string{"/^\\   /^\\", "<-o---o->", " \\_|_|_/ ", "   v v   "}},
}

// Rock is an asteroid: it does not shoot and it does not aim, it just comes
// down - and when it breaks it throws meteoroids at everything, including at
// whatever alien happened to be underneath it.
//
// Straight from the reference, and worth stealing: it is the one thing on the
// field that is not on anybody's side.
var Rock = Craft{
	Name: "roca", Desc: "no dispara, pero al romperse reparte",
	W: 5, H: 3, HP: 4, Fall: 0.025, Drift: 0.04, Points: 25,
	Rows: []string{" .-. ", "(o o)", " '-' "},
}

// Boss is one of the thirty-five: five rows of nine cells, two leg frames.
type Boss struct {
	Name, Desc        string
	Rank              int // 1..5, into Ranks
	Top, Upper        string
	Left, Eyes, Right string
	Lower             string
	Legs              [2]string
}

var Stages = [8]Palette{
	{Label: "stage 1 · avanzada", Points: 10,
		Ramp:  [7]theme.Colour{theme.Hex("#7ff0dd"), theme.Hex("#4dd6c1"), theme.Hex("#3bb8a6"), theme.Hex("#2f9489"), theme.Hex("#27736e"), theme.Hex("#1f5a5c"), theme.Hex("#3d4a4c")},
		Tones: [3]theme.Colour{theme.Hex("#7ff0dd"), theme.Hex("#4dd6c1"), theme.Hex("#3bb8a6")},
		Eye:   theme.Hex("#d6fffa")},
	{Label: "stage 2 · zumbido", Points: 15,
		Ramp:  [7]theme.Colour{theme.Hex("#8ff5ae"), theme.Hex("#57e389"), theme.Hex("#45bf71"), theme.Hex("#38995c"), theme.Hex("#2d7549"), theme.Hex("#245c3d"), theme.Hex("#3f4a44")},
		Tones: [3]theme.Colour{theme.Hex("#8ff5ae"), theme.Hex("#57e389"), theme.Hex("#45bf71")},
		Eye:   theme.Hex("#d8ffe9")},
	{Label: "stage 3 · enjambre", Points: 20,
		Ramp:  [7]theme.Colour{theme.Hex("#a5d4ff"), theme.Hex("#6fb6ff"), theme.Hex("#5495e0"), theme.Hex("#4276bd"), theme.Hex("#345c96"), theme.Hex("#2a4875"), theme.Hex("#414a55")},
		Tones: [3]theme.Colour{theme.Hex("#a5d4ff"), theme.Hex("#6fb6ff"), theme.Hex("#5495e0")},
		Eye:   theme.Hex("#d5e9ff")},
	{Label: "stage 4 · falange", Points: 25,
		Ramp:  [7]theme.Colour{theme.Hex("#b8c2ff"), theme.Hex("#8b9cff"), theme.Hex("#6f7fe0"), theme.Hex("#5a67bd"), theme.Hex("#474f96"), theme.Hex("#3a4075"), theme.Hex("#444656")},
		Tones: [3]theme.Colour{theme.Hex("#b8c2ff"), theme.Hex("#8b9cff"), theme.Hex("#6f7fe0")},
		Eye:   theme.Hex("#e2e6ff")},
	{Label: "stage 5 · espectros", Points: 30,
		Ramp:  [7]theme.Colour{theme.Hex("#d3b0ff"), theme.Hex("#b07cf0"), theme.Hex("#8f5fd1"), theme.Hex("#7449ad"), theme.Hex("#5b3789"), theme.Hex("#472b6b"), theme.Hex("#443f52")},
		Tones: [3]theme.Colour{theme.Hex("#d3b0ff"), theme.Hex("#b07cf0"), theme.Hex("#8f5fd1")},
		Eye:   theme.Hex("#efe0ff")},
	{Label: "stage 6 · forja", Points: 35,
		Ramp:  [7]theme.Colour{theme.Hex("#ffdf95"), theme.Hex("#e8c46a"), theme.Hex("#c7a453"), theme.Hex("#a28442"), theme.Hex("#7d6634"), theme.Hex("#64512a"), theme.Hex("#4a4638")},
		Tones: [3]theme.Colour{theme.Hex("#ffdf95"), theme.Hex("#e8c46a"), theme.Hex("#c7a453")},
		Eye:   theme.Hex("#fff4d6")},
	{Label: "stage 7 · rescoldo", Points: 40,
		Ramp:  [7]theme.Colour{theme.Hex("#ffc48f"), theme.Hex("#f2a35e"), theme.Hex("#d1854a"), theme.Hex("#a96a3b"), theme.Hex("#85522f"), theme.Hex("#6b4126"), theme.Hex("#4d4437")},
		Tones: [3]theme.Colour{theme.Hex("#ffc48f"), theme.Hex("#f2a35e"), theme.Hex("#d1854a")},
		Eye:   theme.Hex("#ffe3c9")},
	{Label: "stage 8 · sangre", Points: 50,
		Ramp:  [7]theme.Colour{theme.Hex("#ff9fa2"), theme.Hex("#f2777a"), theme.Hex("#d15d61"), theme.Hex("#a94a4e"), theme.Hex("#85393d"), theme.Hex("#6b2d31"), theme.Hex("#4d3f41")},
		Tones: [3]theme.Colour{theme.Hex("#ff9fa2"), theme.Hex("#f2777a"), theme.Hex("#d15d61")},
		Eye:   theme.Hex("#ffd9da")},
}

var Ranks = [5]Palette{
	{Label: "rango 01 · larvas", Points: 10,
		Ramp:  [7]theme.Colour{theme.Hex("#8ff5ae"), theme.Hex("#57e389"), theme.Hex("#45bf71"), theme.Hex("#38995c"), theme.Hex("#2d7549"), theme.Hex("#245c3d"), theme.Hex("#3f4a44")},
		Tones: [3]theme.Colour{theme.Hex("#8ff5ae"), theme.Hex("#57e389"), theme.Hex("#45bf71")},
		Eye:   theme.Hex("#d8ffe9")},
	{Label: "rango 02 · drones", Points: 20,
		Ramp:  [7]theme.Colour{theme.Hex("#a5d4ff"), theme.Hex("#6fb6ff"), theme.Hex("#5495e0"), theme.Hex("#4276bd"), theme.Hex("#345c96"), theme.Hex("#2a4875"), theme.Hex("#414a55")},
		Tones: [3]theme.Colour{theme.Hex("#a5d4ff"), theme.Hex("#6fb6ff"), theme.Hex("#5495e0")},
		Eye:   theme.Hex("#d5e9ff")},
	{Label: "rango 03 · soldados", Points: 40,
		Ramp:  [7]theme.Colour{theme.Hex("#ffdf95"), theme.Hex("#e8c46a"), theme.Hex("#c7a453"), theme.Hex("#a28442"), theme.Hex("#7d6634"), theme.Hex("#64512a"), theme.Hex("#4a4638")},
		Tones: [3]theme.Colour{theme.Hex("#ffdf95"), theme.Hex("#e8c46a"), theme.Hex("#c7a453")},
		Eye:   theme.Hex("#fff4d6")},
	{Label: "rango 04 · élites", Points: 80,
		Ramp:  [7]theme.Colour{theme.Hex("#d3b0ff"), theme.Hex("#b07cf0"), theme.Hex("#8f5fd1"), theme.Hex("#7449ad"), theme.Hex("#5b3789"), theme.Hex("#472b6b"), theme.Hex("#443f52")},
		Tones: [3]theme.Colour{theme.Hex("#d3b0ff"), theme.Hex("#b07cf0"), theme.Hex("#8f5fd1")},
		Eye:   theme.Hex("#efe0ff")},
	{Label: "rango 05 · jefes", Points: 500,
		Ramp:  [7]theme.Colour{theme.Hex("#ff9fa2"), theme.Hex("#f2777a"), theme.Hex("#d15d61"), theme.Hex("#a94a4e"), theme.Hex("#85393d"), theme.Hex("#6b2d31"), theme.Hex("#4d3f41")},
		Tones: [3]theme.Colour{theme.Hex("#ff9fa2"), theme.Hex("#f2777a"), theme.Hex("#d15d61")},
		Eye:   theme.Hex("#ffd9da")},
}

// Bosses are the thirty-five bigger ones. Rank 5 is the four that close a wave.
var Bosses = []Boss{
	// rango 01 · larvas
	{Name: "mota", Desc: "dos patas y ya", Rank: 1,
		Top: "    ╷    ", Upper: "  ▗▄▄▄▖  ",
		Left: " ▐ ", Eyes: "> <", Right: " ▌ ", Lower: "  ▝▀▀▀▘  ",
		Legs: [2]string{"   ▘ ▝   ", "   ▝ ▘   "}},
	{Name: "chispa", Desc: "antena partida", Rank: 1,
		Top: "  ╲ ◦ ╱  ", Upper: " ▗▟███▙▖ ",
		Left: "▐█ ", Eyes: "> <", Right: " █▌", Lower: " ▝▀▀▀▀▀▘ ",
		Legs: [2]string{" ▘ ▘ ▝ ▝ ", " ▝ ▝ ▘ ▘ "}},
	{Name: "esporo", Desc: "flota bajo", Rank: 1,
		Top: "   ◦ ◦   ", Upper: "  ▟███▙  ",
		Left: " █ ", Eyes: "> <", Right: " █ ", Lower: "  ▜███▛  ",
		Legs: [2]string{"  ▘   ▝  ", "  ▝   ▘  "}},
	{Name: "pulga", Desc: "salta de fila", Rank: 1,
		Top: "  ╱   ╲  ", Upper: " ▄▄▀▀▀▄▄ ",
		Left: "▐█ ", Eyes: "> <", Right: " █▌", Lower: " ▀▀▄▄▄▀▀ ",
		Legs: [2]string{" ▘▘   ▝▝ ", " ▝▝   ▘▘ "}},
	{Name: "ácaro", Desc: "cuerpo estrecho", Rank: 1,
		Top: "   ╭─╮   ", Upper: "  ▗▟█▙▖  ",
		Left: " ▐ ", Eyes: "> <", Right: " ▌ ", Lower: "  ▝▜█▛▘  ",
		Legs: [2]string{"  ▘ ▄ ▝  ", "  ▝ ▄ ▘  "}},
	{Name: "zumbo", Desc: "silueta mordida", Rank: 1,
		Top: " ╲     ╱ ", Upper: " ▗▛▀█▀▜▖ ",
		Left: "▐█ ", Eyes: "> <", Right: " █▌", Lower: " ▝▙▄█▄▟▘ ",
		Legs: [2]string{" ▘  ▝  ▘ ", " ▝  ▘  ▝ "}},
	{Name: "vaina", Desc: "blindaje liso", Rank: 1,
		Top: "  ▄▄▄▄▄  ", Upper: " ▟█████▙ ",
		Left: "█  ", Eyes: "> <", Right: "  █", Lower: " ▜█████▛ ",
		Legs: [2]string{" ▘     ▝ ", " ▝     ▘ "}},
	{Name: "púa", Desc: "cuerno único", Rank: 1,
		Top: "  ╱ │ ╲  ", Upper: "  ▗▟█▙▖  ",
		Left: " ▐ ", Eyes: "> <", Right: " ▌ ", Lower: "  ▝▀▄▀▘  ",
		Legs: [2]string{"  ▘   ▝  ", "  ▝   ▘  "}},
	{Name: "gusano", Desc: "cuatro sensores", Rank: 1,
		Top: " ◦ ◦ ◦ ◦ ", Upper: " ▗▄▄▄▄▄▖ ",
		Left: "▐█ ", Eyes: "> <", Right: " █▌", Lower: " ▝▀▀▀▀▀▘ ",
		Legs: [2]string{" ▘ ▘ ▝ ▝ ", " ▝ ▝ ▘ ▘ "}},
	// rango 02 · drones
	{Name: "sonda", Desc: "sensor en la antena", Rank: 2,
		Top: "  ╲ ◦ ╱  ", Upper: " ▗▟███▙▖ ",
		Left: "▐█ ", Eyes: "> <", Right: " █▌", Lower: " ▝▜███▛▘ ",
		Legs: [2]string{"   ▄ ▄   ", "   ▘ ▝   "}},
	{Name: "orbe", Desc: "gira sobre sí", Rank: 2,
		Top: "   ╭─╮   ", Upper: "  ▟███▙  ",
		Left: " █ ", Eyes: "> <", Right: " █ ", Lower: "  ▜███▛  ",
		Legs: [2]string{"  ▝   ▘  ", "  ▘   ▝  "}},
	{Name: "ojo", Desc: "solo mira", Rank: 2,
		Top: "   ▄▄▄   ", Upper: "  ▟█▀█▙  ",
		Left: " ▐ ", Eyes: "> <", Right: " ▌ ", Lower: "  ▜█▄█▛  ",
		Legs: [2]string{"   ▀▀▀   ", "   ▀ ▀   "}},
	{Name: "mina", Desc: "espera quieta", Rank: 2,
		Top: " ╲ ╱ ╲ ╱ ", Upper: "  ▗▟█▙▖  ",
		Left: " ▐ ", Eyes: "> <", Right: " ▌ ", Lower: "  ▝▜█▛▘  ",
		Legs: [2]string{" ╱ ╲ ╱ ╲ ", " ╲ ╱ ╲ ╱ "}},
	{Name: "baliza", Desc: "marca la oleada", Rank: 2,
		Top: "    │    ", Upper: "  ▄▄█▄▄  ",
		Left: " █ ", Eyes: "> <", Right: " █ ", Lower: "  ▀▀█▀▀  ",
		Legs: [2]string{"   ▘ ▝   ", "   ▝ ▘   "}},
	{Name: "enjambre", Desc: "llega de dos en dos", Rank: 2,
		Top: " ◦◦   ◦◦ ", Upper: " ▟▙   ▟▙ ",
		Left: " █ ", Eyes: "> <", Right: " █ ", Lower: " ▜▛   ▜▛ ",
		Legs: [2]string{" ▘▘   ▝▝ ", " ▝▝   ▘▘ "}},
	{Name: "aguijón", Desc: "baja en picado", Rank: 2,
		Top: "    ▲    ", Upper: "  ▗▟█▙▖  ",
		Left: " ▐ ", Eyes: "> <", Right: " ▌ ", Lower: "  ▝▜█▛▘  ",
		Legs: [2]string{"   ╲ ╱   ", "   ╱ ╲   "}},
	{Name: "espora", Desc: "suelta crías", Rank: 2,
		Top: "  ◦ ◦ ◦  ", Upper: " ▗▛▀▀▀▜▖ ",
		Left: "▐█ ", Eyes: "> <", Right: " █▌", Lower: " ▝▙▄▄▄▟▘ ",
		Legs: [2]string{"  ▘ ▄ ▝  ", "  ▝ ▄ ▘  "}},
	// rango 03 · soldados
	{Name: "casco", Desc: "plancha encima", Rank: 3,
		Top: "  ╔═══╗  ", Upper: " ▟█████▙ ",
		Left: "█  ", Eyes: "> <", Right: "  █", Lower: " ▜█████▛ ",
		Legs: [2]string{" ▘▘   ▝▝ ", " ▝▝   ▘▘ "}},
	{Name: "cuña", Desc: "abre la formación", Rank: 3,
		Top: "    ▲    ", Upper: " ▗▟███▙▖ ",
		Left: "▐█ ", Eyes: "> <", Right: " █▌", Lower: " ▝▜███▛▘ ",
		Legs: [2]string{" ▘ ▘ ▝ ▝ ", " ▝ ▝ ▘ ▘ "}},
	{Name: "baluarte", Desc: "no se rompe", Rank: 3,
		Top: " ▄▄▄ ▄▄▄ ", Upper: "▟▀▀▀▀▀▀▀▙",
		Left: "█  ", Eyes: "> <", Right: "  █", Lower: "▜▄▄▄▄▄▄▄▛",
		Legs: [2]string{" ▄▄   ▄▄ ", " ▄ ▄ ▄ ▄ "}},
	{Name: "lancero", Desc: "dispara recto", Rank: 3,
		Top: "    │    ", Upper: " ▗▟███▙▖ ",
		Left: "▐█ ", Eyes: "> <", Right: " █▌", Lower: " ▝▀▀▀▀▀▘ ",
		Legs: [2]string{" ▘▘   ▝▝ ", " ▝▝   ▘▘ "}},
	{Name: "caparazón", Desc: "el más ancho", Rank: 3,
		Top: " ▄▄▄▄▄▄▄ ", Upper: "▗▟█████▙▖",
		Left: "█  ", Eyes: "> <", Right: "  █", Lower: "▝▜█████▛▘",
		Legs: [2]string{" ▘ ▘ ▝ ▝ ", " ▝ ▝ ▘ ▘ "}},
	{Name: "mandíbula", Desc: "muerde al bajar", Rank: 3,
		Top: " ╲╱   ╲╱ ", Upper: " ▛▀███▀▜ ",
		Left: "▐█ ", Eyes: "> <", Right: " █▌", Lower: " ▙▄███▄▟ ",
		Legs: [2]string{" ▘  ▄  ▝ ", " ▝  ▄  ▘ "}},
	{Name: "pistón", Desc: "avanza a golpes", Rank: 3,
		Top: " ║     ║ ", Upper: " ▟█████▙ ",
		Left: " █ ", Eyes: "> <", Right: " █ ", Lower: " ▜█████▛ ",
		Legs: [2]string{" ▄▄   ▄▄ ", " ▘▘   ▝▝ "}},
	// rango 04 · élites
	{Name: "segador", Desc: "cuatro hojas", Rank: 4,
		Top: " ╲ ╲ ╱ ╱ ", Upper: "▗▟█████▙▖",
		Left: "█  ", Eyes: "> <", Right: "  █", Lower: "▝▜█████▛▘",
		Legs: [2]string{" ▘ ▘ ▝ ▝ ", " ▝ ▝ ▘ ▘ "}},
	{Name: "espectro", Desc: "parpadea al moverse", Rank: 4,
		Top: "  ◦ ▲ ◦  ", Upper: " ▟█▀▀▀█▙ ",
		Left: "▐█ ", Eyes: "> <", Right: " █▌", Lower: " ▜█▄▄▄█▛ ",
		Legs: [2]string{" ▘ ▄ ▄ ▝ ", " ▝ ▄ ▄ ▘ "}},
	{Name: "matriarca", Desc: "suelta larvas", Rank: 4,
		Top: "  ╔═╦═╗  ", Upper: "▟███████▙",
		Left: "█  ", Eyes: "> <", Right: "  █", Lower: "▜███████▛",
		Legs: [2]string{" ▘▘ ▄ ▝▝ ", " ▝▝ ▄ ▘▘ "}},
	{Name: "viuda", Desc: "baja por los lados", Rank: 4,
		Top: " ╱╲   ╱╲ ", Upper: " ▛▀▀█▀▀▜ ",
		Left: "▐█ ", Eyes: "> <", Right: " █▌", Lower: " ▙▄▄█▄▄▟ ",
		Legs: [2]string{"▘ ▘   ▝ ▝", "▝ ▝   ▘ ▘"}},
	{Name: "acechador", Desc: "aparece detrás", Rank: 4,
		Top: "    ◈    ", Upper: " ▗▟███▙▖ ",
		Left: "▐█ ", Eyes: "> <", Right: " █▌", Lower: " ▝▙▄█▄▟▘ ",
		Legs: [2]string{" ▘ ▘ ▝ ▝ ", " ▝ ▝ ▘ ▘ "}},
	{Name: "serpiente", Desc: "cruza la pantalla", Rank: 4,
		Top: " ╲╱╲ ╱╲╱ ", Upper: "▗▄▄███▄▄▖",
		Left: "█  ", Eyes: "> <", Right: "  █", Lower: "▝▀▀███▀▀▘",
		Legs: [2]string{" ▄ ▄ ▄ ▄ ", " ▘ ▝ ▘ ▝ "}},
	{Name: "nulo", Desc: "absorbe un tiro", Rank: 4,
		Top: "  ▄ ▄ ▄  ", Upper: " ▟█████▙ ",
		Left: " █ ", Eyes: "> <", Right: " █ ", Lower: " ▜█████▛ ",
		Legs: [2]string{"  ▘   ▝  ", "  ▝   ▘  "}},
	// rango 05 · jefes
	{Name: "fauce", Desc: "traga los disparos", Rank: 5,
		Top: "╱^╱^ ^╲^╲", Upper: "▟███████▙",
		Left: "█  ", Eyes: "> <", Right: "  █", Lower: "▜█▄▄▄▄▄█▛",
		Legs: [2]string{"▘▘▘   ▝▝▝", "▝▝▝   ▘▘▘"}},
	{Name: "reina", Desc: "reparte la oleada", Rank: 5,
		Top: " ╲ ╔╦╗ ╱ ", Upper: "▟███████▙",
		Left: "█  ", Eyes: "> <", Right: "  █", Lower: "▜███████▛",
		Legs: [2]string{"▘ ▘ ▄ ▝ ▝", "▝ ▝ ▄ ▘ ▘"}},
	{Name: "coloso", Desc: "ocupa media fila", Rank: 5,
		Top: "█▀▀▀▀▀▀▀█", Upper: "▟███████▙",
		Left: "█  ", Eyes: "> <", Right: "  █", Lower: "▜███████▛",
		Legs: [2]string{"▄▄▄   ▄▄▄", "▄ ▄   ▄ ▄"}},
	{Name: "centinela", Desc: "el último", Rank: 5,
		Top: " ^ ╲◈╱ ^ ", Upper: "▗▟█████▙▖",
		Left: "█  ", Eyes: "> <", Right: "  █", Lower: "▝▜█▄▄▄█▛▘",
		Legs: [2]string{"▘▘ ▘ ▝ ▝▝", "▝▝ ▝ ▘ ▘▘"}},
}
