// Package theme holds the palette and the width-aware text helpers. One colour
// per kind of data, declared exactly once.
package theme

import (
	"math"
	"os"
	"strconv"
	"strings"
)

// Colour carries both forms: 24-bit RGB and the nearest 256-colour fallback.
// Without COLORTERM the terminal quantises 24-bit colours and near tones
// collapse onto each other, so we pick the fallback ourselves.
type Colour struct {
	R, G, B byte
	X256    byte
}

func hex(s string) Colour {
	v, _ := strconv.ParseUint(s[1:], 16, 32)
	r, g, b := byte(v>>16), byte(v>>8&0xff), byte(v&0xff)
	return Colour{r, g, b, Nearest256(r, g, b)}
}

// Hex builds a Colour from "#rrggbb", quantising its own 256-colour fallback.
//
// It used to leave that fallback at ZERO, on the stated grounds that "without
// COLORTERM it quantises to the nearest cube entry like any other". Nothing
// quantised: seq writes the fallback out as `;5;0`, and colour 0 is black. So
// on any terminal that does not announce truecolor - tmux without the right
// setting, screen, a plain TERM - the entire creature was drawn black on a
// near-black footer, invisible, and the model pill was a black box with black
// ink in it. The ramps came in by the hundred and the field was simply never
// filled.
//
// Hand-picking a hundred fallbacks would indeed be a hundred chances to choose
// badly, and that was the right instinct. Computing them is not: the xterm 256
// palette is a 6x6x6 cube plus a 24-step grey, and "nearest" has an answer.
func Hex(s string) Colour { return hex(s) }

// cubeLevels are the six values each channel takes in the xterm 6x6x6 cube.
var cubeLevels = [6]int{0, 95, 135, 175, 215, 255}

func nearestLevel(v int) int {
	best, bestGap := 0, 1<<30
	for i, level := range cubeLevels {
		gap := v - level
		if gap < 0 {
			gap = -gap
		}
		if gap < bestGap {
			best, bestGap = i, gap
		}
	}
	return best
}

func squaredDistance(r, g, b, r2, g2, b2 int) int {
	dr, dg, db := r-r2, g-g2, b-b2
	return dr*dr + dg*dg + db*db
}

// Nearest256 is the xterm-256 entry closest to a 24-bit colour.
//
// It considers both halves of the palette and takes whichever is nearer. The
// grey ramp matters more than it looks: the ten creature ramps all end in a
// near-grey k.o. step, and the cube's greys are coarse - forcing those onto the
// cube would flatten seven distinct k.o. tones onto about two.
func Nearest256(r, g, b byte) byte {
	ri, gi, bi := nearestLevel(int(r)), nearestLevel(int(g)), nearestLevel(int(b))
	cube := squaredDistance(int(r), int(g), int(b),
		cubeLevels[ri], cubeLevels[gi], cubeLevels[bi])

	// The 24 greys are 8, 18, ... 238, at indices 232..255.
	//
	// The step comes off the plain MEAN of the channels, not off luma. It reads
	// like the wrong statistic and it is the right one: the distance to a grey
	// (v,v,v) expands to 3(v-mean)^2 plus a term that does not involve v, so
	// the grey nearest in value is the grey nearest in distance, and the mean
	// is what "nearest in value" means here. Luma is the right weighting for
	// how bright a colour LOOKS and the wrong one for how far away it is: on
	// #00002a it picked grey 232 when 233 is measurably closer.
	level := (int(r) + int(g) + int(b)) / 3
	step := (level - 8 + 5) / 10
	if step < 0 {
		step = 0
	}
	if step > 23 {
		step = 23
	}
	grey := 8 + 10*step
	if squaredDistance(int(r), int(g), int(b), grey, grey, grey) < cube {
		return byte(232 + step)
	}
	return byte(16 + 36*ri + 6*gi + bi)
}

func c(s string, x256 byte) Colour {
	col := hex(s)
	col.X256 = x256
	return col
}

// Data colours.
var (
	Path     = c("#4dd6c1", 79)  // paths, files, repos
	Ident    = c("#57e389", 78)  // identifiers, additions
	Link     = c("#6fb6ff", 75)  // urls, branches, links
	Number   = c("#e8c46a", 179) // numbers, money, metrics, warnings
	Mode     = c("#b07cf0", 141) // modes and CLI settings
	Bad      = c("#f2777a", 210) // deletions, errors, risk
	Emph     = c("#eceff4", 255) // emphasis
	Paper    = c("#eceff4", 255) // the light ink On() offers: Emph in the dark, white in the light
	Dim      = c("#6b7683", 243) // arrows, separators, units
	Rule     = c("#2c343c", 236) // the vertical separator
	Dir      = c("#8d99a6", 246) // the directory, one grey above Dim
	Quota    = c("#4ea3f5", 75)  // limit bars
	Empty    = c("#1d2b38", 235) // empty half of a limit bar
	CtxEmpty = c("#24382c", 235) // empty half of the context bar
)

// The footer.
var (
	Bg     = c("#0a0d0f", 233) // one tone above black
	Border = c("#161c21", 234) // the hairline that splits it from the thread
	Tail   = c("#3a444e", 238) // the speech-bubble tail
	Text   = c("#c9d1d9", 252) // what the pet says
)

// Creature skins.
var (
	PaleGreen  = c("#d8ffe9", 194)
	DarkGreen  = c("#2f7a52", 29)
	PaleTeal   = c("#d6fffa", 195)
	DarkTeal   = c("#2b7d74", 30)
	PaleBlue   = c("#d5e9ff", 189)
	DarkBlue   = c("#2a5c8a", 24)
	PaleIndigo = c("#cdd2ff", 189)
	DarkIndigo = c("#333c9e", 61)
	PaleGrey   = c("#8891a0", 245)
	DarkGrey   = c("#3d434d", 238)
	Grey       = c("#5a6270", 242)
	Drowned    = c("#5865f2", 63)
)

const (
	Reset = "\033[0m"
	Bold  = "\033[1m"
	Black = "\033[38;5;16m"
	// Soft is "drop the bold, back to the default foreground". A full reset
	// would wipe the footer background, so painted lines swap Reset for this
	// and re-assert the background.
	Soft = "\033[22;39m"
)

// rgb is resolved once: COLORTERM does not change under a running process, and
// every painted glyph would otherwise re-read the environment.
var rgb = func() bool {
	v := strings.ToLower(os.Getenv("COLORTERM"))
	return v == "truecolor" || v == "24bit"
}()

// Truecolor reports whether 24-bit sequences are being emitted.
func Truecolor() bool { return rgb }

// SetTruecolor overrides the detection. Tests only.
func SetTruecolor(on bool) { rgb = on }

func seq(lead int, col Colour) string {
	var b strings.Builder
	b.WriteString("\033[")
	if rgb {
		b.WriteString(strconv.Itoa(lead))
		b.WriteString(";2;")
		b.WriteString(strconv.Itoa(int(col.R)))
		b.WriteByte(';')
		b.WriteString(strconv.Itoa(int(col.G)))
		b.WriteByte(';')
		b.WriteString(strconv.Itoa(int(col.B)))
	} else {
		b.WriteString(strconv.Itoa(lead))
		b.WriteString(";5;")
		b.WriteString(strconv.Itoa(int(col.X256)))
	}
	b.WriteByte('m')
	return b.String()
}

// Fg is the foreground escape for a colour.
func Fg(col Colour) string { return seq(38, col) }

// Bgc is the background escape for a colour.
func Bgc(col Colour) string { return seq(48, col) }

// On is a foreground that stays legible sitting on top of col: whichever of
// the two inks the palette has - black, or Paper's near-white - contrasts with
// it more. Paper and not Emph: in the light palette Emph is the near-black of
// the text, and comparing black against black is no choice at all.
//
// The model pill used to be black on one fixed green, so black was safe by
// construction. It is painted the creature's colour now, and the ramps run
// from a near-white "fresca" down to a near-grey k.o., so the ink has to
// follow the background instead of assuming it.
//
// It MEASURES rather than thresholding. A luma cut-off was tried first and it
// is wrong in the middle of the range, which is where most of the ramp steps
// live: the mid teal #2f9489 reads as "dark" on any sensible cut-off, and yet
// black on it contrasts 5.7 against the light ink's 3.7. Comparing the two
// contrasts has no such band, and it cannot be tuned wrong.
func On(col Colour) string {
	if ContrastRatio(Black8, col) >= ContrastRatio(Paper, col) {
		return Black
	}
	return Fg(Paper)
}

// Black8 is the colour Black paints in, as a Colour: xterm 16 is #000000.
var Black8 = Colour{0, 0, 0, 16}

// ContrastRatio is the WCAG contrast ratio between two colours, 1 for
// identical to 21 for black on white.
func ContrastRatio(a, b Colour) float64 {
	la, lb := relativeLuminance(a), relativeLuminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

func relativeLuminance(c Colour) float64 {
	f := func(v byte) float64 {
		s := float64(v) / 255
		if s <= 0.03928 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*f(c.R) + 0.7152*f(c.G) + 0.0722*f(c.B)
}
