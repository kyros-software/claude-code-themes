package statusline

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/kyros-software/claude-code-themes/internal/pet"
	"github.com/kyros-software/claude-code-themes/internal/theme"
)

// The four bands.
//
//	1 ENGINE  model, context, limits, rate       -> what changes every turn
//	2 WORK    repo, branch, diff, cost, clock    -> what the session has spent
//	3 WHERE   the directory, the output style    -> read out of the corner of
//	                                                the eye: what barely moves
//	4 PET     trade, level, state, mark, bubble  -> see petBand, below

// sixSigFigs renders a float the way "%g" does: six significant digits, no
// trailing zeros.
func sixSigFigs(f float64) string { return strconv.FormatFloat(f, 'g', 6, 64) }

func ctxLabel(size *float64) string {
	if size == nil {
		return ""
	}
	n := float64(int64(*size))
	if n >= 1000000 {
		return sixSigFigs(n/1000000.0) + "M ctx"
	}
	return sixSigFigs(n/1000.0) + "k ctx"
}

func formatDuration(ms *float64) string {
	if ms == nil {
		return ""
	}
	seconds := int(*ms / 1000)
	minutes, secs := seconds/60, seconds%60
	hours, mins := minutes/60, minutes%60
	if hours > 0 {
		return fmt.Sprintf("%dh %02dm", hours, mins)
	}
	if mins > 0 {
		return fmt.Sprintf("%dm %02ds", mins, secs)
	}
	return fmt.Sprintf("%ds", secs)
}

// round matches Python's round(): half-to-even, not half-up. With a context at
// 42.5% the two disagree - 42 against 43 - and the number is on screen once a
// second, so it has to be the same number.
func round(f float64) int { return int(math.RoundToEven(f)) }

// engine is band 1. accent is the creature's own body colour, or nil when the
// pet is switched off.
//
// The pill and the context bar are painted with it, which is the whole trick:
// the top-left of the statusline and the creature on the right are one reading
// in one colour, not two colour schemes describing the same session. The pill
// used to be a fixed green - the identifier green, borrowed for want of
// anything better - and the bar carried the state ladder, so a turquoise pet
// sat next to a green pill and an indigo bar and the eye had three things to
// reconcile. Now the hue says WHICH creature and the step says HOW it is, and
// both ends of the line say it together.
func engine(p *Payload, tps *float64, accent *theme.Colour) []segment {
	skin := theme.Ident
	if accent != nil {
		skin = *accent
	}
	pill := theme.Bgc(skin) + theme.On(skin) + theme.Bold
	out := []segment{
		seg(0, pill+" "+p.Model+" "+theme.Reset).truncatable(pill, " "+p.Model+" "),
	}

	if p.ContextPc != nil {
		// The bar IS the context, in length and in number, and so is the
		// creature: one measurement on one line. Anything else has been tried
		// and read as a bug. When the bar was the context and only borrowed the
		// neck's colour, ctx 48 beside a 5h at 67 drew a half-full bar next to
		// the word "espesa", which is the reading for 67. When the bar was
		// promoted to the neck to close that gap, it printed "82% 5h" three
		// columns before the band printed "5h 82%" again, and the context - the
		// one figure of the three that belongs to this session, and the only
		// one you can do anything about - was left with a bare "7%".
		//
		// The quotas keep the reading they had all along: bare numbers at the
		// end of the band, painted off this same ladder, so a 5h at 95 arrives
		// in the pet's own drowning indigo. What they no longer do is speak for
		// a session they know nothing about. See pet.ContextLoad.
		//
		// The colour is the creature's own body, which is the trick this band
		// turns: the top-left of the statusline and the pet on the right are
		// one reading in one colour, not two colour schemes describing the same
		// session. It used to be the state ladder's colour, which agreed with
		// the creature about how the session felt but not about what was doing
		// the feeling - one scale for every pet, while since the atlas the hue
		// belongs to the branch. A blue cazabugs sat beside a green bar that
		// meant the same thing.
		fill := pet.StateFor(pet.ContextLoad(p.ContextPc)).Colour
		if accent != nil {
			fill = *accent
		}
		out = append(out, seg(1,
			theme.Bar(*p.ContextPc, 100, 16, fill, theme.Empty)+" "+
				theme.Fg(theme.Emph)+theme.Bold+strconv.Itoa(round(*p.ContextPc))+"%"+theme.Reset,
		).withSep(" "))
	}

	if label := ctxLabel(p.CtxSize); label != "" {
		out = append(out, seg(3, theme.Fg(theme.Dim)+label+theme.Reset).
			truncatable(theme.Fg(theme.Dim), label).
			withSep(" "+theme.Fg(theme.Rule)+"·"+theme.Reset+" "))
	}

	if p.Effort != "" {
		out = append(out, seg(2, theme.Fg(theme.Mode)+p.Effort+theme.Reset).
			truncatable(theme.Fg(theme.Mode), p.Effort))
	}

	// The two rate limits, as bare numbers - no bars. They used to have their
	// own bars down in band 3, and when band 3 became just the folder they had
	// nowhere to live.
	//
	// This is the whole reading of the quotas now, and it is enough. They are
	// the ACCOUNT's, not this session's, so they say nothing about how the
	// window in front of you feels and the creature does not read them. What
	// they do say - how much of the day is left - takes a number, not a
	// silhouette.
	//
	// The number is still painted off the SAME ladder as the bar and the pet,
	// which is what keeps it from being a footnote: a 5h at 95 arrives in the
	// pet's own drowning indigo, so the thing that is about to stop you is the
	// loudest colour on the line even while the creature is green.
	first := true
	for _, limit := range []struct {
		value *float64
		tag   string
	}{{p.FiveHour, "5h"}, {p.SevenDay, "7d"}} {
		if limit.value == nil {
			continue
		}
		// Clamped, like every other reading of a percentage on this line.
		// pet.ContextLoad already rejects NaN and pins the context to [0,100]
		// for the bar; this printed whatever arrived, so a payload carrying
		// -5 or 400 drew a sane bar beside "5h -5%". One band, one rule.
		value := pet.ContextLoad(limit.value)
		text := strconv.Itoa(round(value)) + "%"
		s := seg(4, theme.Fg(theme.Dim)+limit.tag+" "+theme.Reset+
			theme.Fg(pet.StateFor(value).Colour)+text+theme.Reset)
		// The 7d hugs the 5h: the two read as one block, as they always did.
		if !first {
			s = s.withSep("  ")
		}
		out = append(out, s)
		first = false
	}

	switch {
	case tps != nil:
		text := strconv.FormatFloat(*tps, 'f', 1, 64)
		if *tps >= 100 {
			text = strconv.Itoa(round(*tps))
		}
		out = append(out, seg(5, theme.Fg(theme.Number)+text+theme.Reset+
			theme.Fg(theme.Dim)+" tok/s"+theme.Reset))
	case p.CacheHit != nil:
		// It relieves the rate, it does not sit next to it: the design gives
		// the band one speed slot, and while the model is talking tok/s owns it.
		text := strconv.Itoa(round(*p.CacheHit*100)) + "%"
		out = append(out, seg(6, theme.Fg(theme.Number)+text+theme.Reset+
			theme.Fg(theme.Dim)+" cache"+theme.Reset))
	}

	if p.Permissions != "" {
		// The CLI paints its own line under the prompt box announcing the
		// mode - "bypass permissions on (shift+tab to cycle)" - so spelling
		// "bypass" here put the same word on the same footer twice. Bypass
		// gets a mark instead: the signal stays, the word does not repeat,
		// and the busiest band gets five columns back. The other two keep
		// their name, which has no obvious glyph and would only be a riddle.
		paint := theme.Fg(theme.Mode) + theme.Bold
		text := p.Permissions
		if p.Permissions == "bypass" {
			paint = theme.Fg(theme.Bad) + theme.Bold
			text = BypassMark
		}
		out = append(out, seg(3, paint+text+theme.Reset).
			truncatable(paint, text))
	}

	if p.Vim != "" {
		paint := theme.Fg(theme.Dim) + theme.Bold
		if p.Vim == "INSERT" {
			paint = theme.Fg(theme.Mode) + theme.Bold
		}
		out = append(out, seg(7, paint+p.Vim+theme.Reset).truncatable(paint, p.Vim))
	}
	return out
}

func intOf(f *float64) string {
	if f == nil {
		return "0"
	}
	return strconv.FormatFloat(*f, 'f', -1, 64)
}

func work(p *Payload) []segment {
	out := []segment{
		seg(0, theme.Fg(theme.Path)+p.Label+theme.Reset).
			truncatable(theme.Fg(theme.Path), p.Label),
	}

	if p.Branch != "" {
		dirty := ""
		if p.Dirty {
			dirty = " " + theme.Fg(theme.Number) + "✳" + theme.Reset
		}
		out = append(out, seg(1,
			theme.Fg(theme.Dim)+"("+theme.Reset+
				theme.Fg(theme.Link)+p.Branch+theme.Reset+dirty+
				theme.Fg(theme.Dim)+")"+theme.Reset).withSep(" "))
	}

	if p.Added != nil || p.Removed != nil {
		out = append(out, seg(2,
			theme.Fg(theme.Ident)+"+"+intOf(p.Added)+theme.Reset+
				theme.Fg(theme.Rule)+"/"+theme.Reset+
				theme.Fg(theme.Bad)+"−"+intOf(p.Removed)+theme.Reset))
	}

	if p.Cost != nil {
		text := "$" + strconv.FormatFloat(*p.Cost, 'f', 2, 64)
		out = append(out, seg(3, theme.Fg(theme.Number)+text+theme.Reset).
			truncatable(theme.Fg(theme.Number), text))
	}

	// The session clock, come up from band 3. It belongs next to the cost:
	// both are what the session has SPENT, one in money and one in hours, and
	// they are the two you read together at the end of a long one.
	if text := formatDuration(p.Duration); text != "" {
		out = append(out, seg(4, theme.Fg(theme.Dim)+text+theme.Reset).
			truncatable(theme.Fg(theme.Dim), text))
	}
	return out
}

// petBand is band 4, the pet's own line. The canvas gives it the trade, the
// level, the XP and the state, with whatever the pet has to say behind them.
// It sits under
// the three data bands and alongside the card, so the four rows on the left
// line up with the four on the right.
//
// How it feels is NOT here, even though the canvas draws it twice. On a real
// terminal the same word ends up on the same footer within a few columns of
// itself and reads as a bug; the canvas never had to look at it running. The
// card's copy is the one that stays, because it is the one with a whole row
// to itself.
//
// Below BubbleMin columns it collapses to just the trade name - the canvas:
// "con menos de 100 columnas se cae sola y solo queda el oficio".
func petBand(c Card, columns int) []segment {
	out := []segment{
		seg(0, theme.Fg(theme.Path)+c.Form+theme.Reset).
			truncatable(theme.Fg(theme.Path), c.Form),
	}
	if columns < BubbleMin {
		return out
	}

	// The mark, glued to the trade it is a variant of: `cazabugs[sabueso]`
	// is a pet that IS a sabueso. See Card.Worn.
	//
	// No separator, because it is not a second thing on the line: it is one
	// name in two parts.
	//
	// The brackets are painted Dim and not Rule. Rule is the colour of the
	// vertical bar, which is meant not to be seen - 1.54:1 against the
	// background, against 11.8:1 for the two words it sits between - so what
	// reached the eye was two bright words with nothing in between, and it
	// read as one compound word. The punctuation carrying the whole meaning
	// was the only part invisible.
	if c.Worn != "" {
		bracket := theme.Fg(theme.Dim) + "[" + theme.Reset +
			theme.Fg(theme.Number) + c.Worn + theme.Reset +
			theme.Fg(theme.Dim) + "]" + theme.Reset
		out = append(out, seg(5, bracket).
			truncatable(theme.Fg(theme.Dim), "["+c.Worn+"]").withSep(""))
	}

	level := "nivel " + strconv.Itoa(c.Level)
	out = append(out, seg(2,
		theme.Fg(theme.Dim)+"nivel "+theme.Reset+
			theme.Fg(theme.Emph)+strconv.Itoa(c.Level)+theme.Reset).
		truncatable(theme.Fg(theme.Dim), level).withSep(" "))

	// How it feels, in words, beside how far it has come. It used to crown the
	// card, which cost the creature the row its crest needed; here it costs a
	// few columns of a line that had them, and it sits next to the level, which
	// is the other thing you read about the pet in one glance.
	if c.State != "" {
		paint := theme.Fg(c.Vital.Colour)
		if c.Jumped {
			paint += theme.Bold
		}
		out = append(out, seg(4, paint+c.State+theme.Reset).
			truncatable(paint, c.State))
	}

	// The mark being filled, by name. There used to be a twelve-cell bar in
	// front of it - XP in green while a level was still opening, the habit in
	// amber once one was not - and it is gone by request: band 1 already
	// carries a bar, and a second one beside the state read as the same
	// measurement twice. The name survives because it says something the bar
	// could not, which is WHICH mark is being filled; while it is XP that is
	// still opening, there is no name and nothing here at all.
	//
	// Card.Done and Card.Span are still filled in - `pet` shows the numbers,
	// and the panel draws its own bar from them.
	if c.Span > 0 && c.Mark != "" {
		out = append(out, seg(3,
			theme.Fg(theme.Number)+c.Mark+theme.Reset).
			truncatable(theme.Fg(theme.Number), c.Mark).withSep(" "))
	}

	if c.Bubble != "" {
		out = append(out, seg(6,
			theme.Fg(theme.Tail)+"◗"+theme.Reset+" "+
				theme.Fg(theme.Text)+c.Bubble+theme.Reset).
			truncatable(theme.Fg(theme.Text), c.Bubble))
	}
	return out
}

// BypassMark stands in for the word "bypass", which the CLI already spells out
// on its own line. It is two cells wide, not one, and that is now measured
// rather than assumed: theme.Width counts cells, so a wide glyph here costs the
// band the two columns it actually takes instead of sliding the card out of
// true. See internal/theme/width.go.
const BypassMark = "⚡"

// quota is band 3: where you are, and under what character.
//
// The 5h and 7d bars and the session clock used to live here too. They went up
// to bands 1 and 2, which is where they belong, and left the band with the
// folder alone - and the folder hides at the root of a repo, so the row was
// blank most of the time. An empty left half is what blankAnchor is for, but an
// anchor is a fix for a row with nothing in it, not a reason to keep it empty.
//
// The OUTPUT STYLE is what moved in. It earns the room band 1 could not spare:
// it is the slowest-moving thing on the footer - it changes when you change it
// and not once a turn - and it is the one field that says which CHARACTER is
// answering, which the rest of the statusline has no way to tell you. Band 1
// carries what changes every turn and band 2 what the session has spent; this
// is the band you read out of the corner of your eye, and a name that only
// moves when you move it belongs exactly there.
//
// Both elements are bare names with no label, and the colour does the telling
// them apart: the folder in grey, because it is somewhere, and the style in
// Mode purple, the same as `xhigh` and `plan`, because it is a CLI setting.
// Reading order is where-then-who, and when the band runs short the style goes
// first: the folder is what this band was for.
//
// The band can still come out empty - at the root of a repo with no style set,
// which is most sessions - and that is not a regression, it is the same rule
// both elements follow. Nothing here shows unless it says something.
func quota(p *Payload) []segment {
	var out []segment
	// If the folder is named like the repo you are at its root, and then it
	// adds nothing: band 2 already says it. It only shows when you are inside.
	if p.Dirname != p.Label {
		out = append(out, seg(1, theme.Fg(theme.Dir)+p.Dirname+theme.Reset).
			truncatable(theme.Fg(theme.Dir), p.Dirname))
	}

	// Payload.Style is already "" for the default, so there is no name here
	// that is not worth the columns. See outputStyle in payload.go.
	//
	// It is LOWERCASED, and that is the footer's voice rather than a fact about
	// the style: everything else in this seat arrives lowercase already -
	// `xhigh`, `plan`, `auto-edit`, the pet's `cazabugs` and `vibrante` - so a
	// capitalised name is the one word on the line that shouts. Doing it here
	// and not in Parse keeps Payload.Style the real name, and doing it in the
	// band rather than renaming the style covers the two built-ins as well:
	// `Explanatory` and `Learning` ship capitalised and nobody can rename those.
	//
	// The folder next to it is NOT lowercased, and the difference is the point:
	// a folder is a name on disk that has to match what `ls` says, while a style
	// name is a setting's label, and labels here are lowercase.
	if p.Style != "" {
		style := strings.ToLower(p.Style)
		out = append(out, seg(2, theme.Fg(theme.Mode)+style+theme.Reset).
			truncatable(theme.Fg(theme.Mode), style))
	}
	return out
}
