package pet

import (
	"os"
	"strings"

	"github.com/kyros-software/claude-code-themes/internal/theme"
)

// Sprite x state. Not one silhouette written twice.

// Nothing is derived any more: the canvas draws every row of every state, and
// Sprite carries the flat pair and all four feet rows verbatim. What used to be
// slump(), flat(), stepMap and stillMap were rules that held for most forms and
// quietly lied about the rest.

// At 1 fps, feet shuffling non-stop in the corner of your eye are tiring, so by
// default it walks 4 seconds out of every 12. STATUSLINE_PET_WALK=1 restores
// the continuous shuffle the design calls for.
func walkAlways() bool {
	switch strings.ToLower(os.Getenv("STATUSLINE_PET_WALK")) {
	case "1", "on", "yes":
		return true
	}
	return false
}

// Compact rows for the statusline. The design canvas calls this 9b and says it
// is the one applied: five rows in the panels, three in the statusline, where
// two rows of terminal are worth more than a crest.
//
// The crest goes (forms are told apart by the Upper row, which already varies)
// and the feet move into the lower contour:
//
//	 ▗▟███▙▖      row 1 = Upper
//	▐█ > < █▌     row 2 = Face
//	 ▘▝▜█▛▝▘      row 3 = legs + a three-cell body + legs
//
// The canvas fixes the four states of that third row, and they are the
// acceptance test: walk ` ▘▝▜█▛▝▘ ` <-> ` ▝▘▜█▛▘▝ `, sunk ` ▖▗▜█▛▗▖ `, k.o.
// ` ▄▄▀▀▀▄▄ `.
const CompactRows = 3

// corners are the glyphs that cap a lower contour; they are dropped before the
// body is squeezed to three cells.
const corners = "▝▘▗▖"

// squeeze reduces a lower contour to the three cells the compact row has room
// for: the two shoulders and one cell of body.
func squeeze(lower string) string {
	body := []rune(strings.TrimSpace(lower))
	for len(body) > 0 && strings.ContainsRune(corners, body[0]) {
		body = body[1:]
	}
	for len(body) > 0 && strings.ContainsRune(corners, body[len(body)-1]) {
		body = body[:len(body)-1]
	}
	switch len(body) {
	case 0:
		return "   "
	case 1:
		return string([]rune{body[0], body[0], body[0]})
	case 2:
		return string([]rune{body[0], body[0], body[1]})
	}
	return string([]rune{body[0], body[len(body)/2], body[len(body)-1]})
}

// DrawCard returns the four rows of the statusline's card: the crest and then
// the three compact rows.
//
// The crest used to have nowhere to go. The card's top row was the state's name
// -- "a gusto", "ahogada" - and the silhouette got the three below it, so the
// one row that names the evolution was dropped on the floor. The state's name
// is now a word in band 4, beside the level, where it costs a few columns of a
// line that had room; the row it vacated goes back to the creature.
//
// That is the atlas's own reading of what identifies a form: "la marca de
// arriba y el numero de patas". The statusline now carries both.
func DrawCard(form string, v Vital, step int, dimEyes bool) [4]string {
	sprite, ok := Sprites[form]
	if !ok {
		sprite = Sprites[Root]
	}
	rows := DrawCompact(form, v, step, dimEyes)
	return [4]string{
		theme.Fg(RampOf(form).Body[v.Rank]) + sprite.Crest + theme.Reset,
		rows[0], rows[1], rows[2],
	}
}

// DrawCompact returns the three coloured rows the statusline uses.
//
// Three rows cannot carry the crest, which is half of what tells the forms
// apart - so the COLOUR carries it. Two evolutions with the same torso are a
// cazabugs in blue and a kraken in red, and that is legible at a glance in a
// way a one-cell difference in a contour never was.
func DrawCompact(form string, v Vital, step int, dimEyes bool) [3]string {
	sprite, ok := Sprites[form]
	if !ok {
		sprite = Sprites[Root]
	}
	ramp := RampOf(form)
	body := theme.Fg(ramp.Body[v.Rank])

	upper := sprite.Upper
	if !v.HeadUp {
		upper = sprite.UpperFlat
	}

	// The lower contour squeezed to three cells, with the form's own feet on
	// either side: the compact row the canvas calls 9b.
	feet := []rune(feetFor(sprite, v, step))
	legs := string(feet[1:3])
	row3 := " " + legs + squeeze(loweFor(sprite, v)) + string([]rune{feet[6], feet[7]}) + " "

	return [3]string{
		body + upper + theme.Reset,
		faceRow(sprite, ramp, v, step, dimEyes),
		body + row3 + theme.Reset,
	}
}

// loweFor is the lower contour for a state: flat from "tired" down.
func loweFor(sprite Sprite, v Vital) string {
	if v.HeadUp {
		return sprite.Lower
	}
	return sprite.LowerFlat
}

// feetFor picks one of the four feet rows the canvas draws. Every one of them
// keeps the form's own foot COUNT and columns - "lo tumba en k.o. sin perder la
// cuenta de patas" - so the number of legs stays readable when everything else
// has flattened.
func feetFor(sprite Sprite, v Vital, step int) string {
	switch {
	case v.Rank >= 6:
		return sprite.KO
	case !v.Walks:
		return sprite.Still
	case walkAlways() || step%12 < 4:
		if step%2 != 0 {
			return sprite.Step
		}
		return sprite.Feet
	}
	return sprite.Still
}

// Draw returns the five coloured rows, SpriteWidth visible columns each.
// step advances on every refresh; the walk cycle and the blink come out of it.
func Draw(form string, v Vital, step int, dimEyes bool) [5]string {
	sprite, ok := Sprites[form]
	if !ok {
		sprite = Sprites[Root]
	}
	ramp := RampOf(form)
	body := theme.Fg(ramp.Body[v.Rank])

	upper, lower := sprite.Upper, sprite.Lower
	if !v.HeadUp {
		upper, lower = sprite.UpperFlat, sprite.LowerFlat
	}
	// The crest never goes. It is the form's mark, and the canvas holds it
	// steady through all seven states - it used to be blanked from "tired"
	// down, which took the evolution's name off it exactly when the rest of
	// the silhouette had flattened into everyone else's.
	return [5]string{
		body + sprite.Crest + theme.Reset,
		body + upper + theme.Reset,
		faceRow(sprite, ramp, v, step, dimEyes),
		body + lower + theme.Reset,
		body + feetFor(sprite, v, step) + theme.Reset,
	}
}

// eyesFor picks the pair of glyphs for a frame. They belong to the STATE, all
// seven of them, for every form: > < then o o, ▬ ▬, _ _ and x x. One frame in
// seven it blinks, but only while it is still walking.
func eyesFor(v Vital, step int) [2]rune {
	if v.Walks && step%7 == 3 {
		return [2]rune{'_', '_'}
	}
	return v.Eyes
}

// faceRow paints the face with its eyes in place. Shared by both renderers:
// the face is the one row the compact form keeps whole.
func faceRow(sprite Sprite, ramp Ramp, v Vital, step int, dimEyes bool) string {
	eyes := eyesFor(v, step)
	body := theme.Fg(ramp.Body[v.Rank])
	eyeCol := theme.Fg(ramp.Lit(v.Rank))
	// Hungry eyes go out early: same glyph, sunk to where a drowning pet's
	// would be, whatever the state actually is.
	if dimEyes {
		eyeCol = theme.Fg(ramp.Body[2])
	}

	var row strings.Builder
	row.WriteString(body)
	for i, r := range []rune(sprite.Face) {
		switch i {
		case sprite.EyeCols[0]:
			row.WriteString(eyeCol)
			row.WriteRune(eyes[0])
			row.WriteString(body)
		case sprite.EyeCols[1]:
			row.WriteString(eyeCol)
			row.WriteRune(eyes[1])
			row.WriteString(body)
		default:
			row.WriteRune(r)
		}
	}
	row.WriteString(theme.Reset)
	return row.String()
}
