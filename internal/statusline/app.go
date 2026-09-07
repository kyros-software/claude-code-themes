// Package statusline renders the four bands and the pet's card.
//
//	row 0   band 1 - engine    state label
//	row 1   band 2 - work      upper body
//	row 2   band 3 - quota     face
//	row 3   band 4 - the pet   legs
//
// Four bands against four rows of card. The pet used to be a six-row block
// pinned to the right of three bands; the design canvas compresses it to three
// rows and gives it a band of its own, which buys back two rows of terminal.
package statusline

import (
	"io"
	"strings"
	"time"

	"github.com/kyros-software/claude-code-themes/internal/pet"
	"github.com/kyros-software/claude-code-themes/internal/session"
	"github.com/kyros-software/claude-code-themes/internal/theme"
)

// minWidthForPet is where the card stops fitting and the bands take the width.
const minWidthForPet = 55

// blankAnchor is U+2800, BRAILLE PATTERN BLANK: a character that paints as
// nothing but is not a space.
//
// Claude Code TRIMS THE LEADING SPACES off every line of the statusline. A row
// whose left half came out empty is then just "spaces, then the pet", and the
// trim drops the lot: that row's slice of the sprite lands at column 0 while
// the other three stay put, and the creature comes apart across the screen.
//
// Band 3 is the row this happens to - at the root of a repo the folder name
// says nothing band 2 has not - and it happens on band 4 too on a pet with
// neither bar nor bubble. One character the trim will not eat holds the row
// where it belongs.
const blankAnchor = "\u2800"

// spokenIn hands back the bubble if the assembled band really carries it, and
// "" if the band dropped it for width.
func spokenIn(band, bubble string) string {
	if bubble == "" || !strings.Contains(theme.Strip(band), bubble) {
		return ""
	}
	return bubble
}

// anchored pins a row that would otherwise be nothing but padding.
func anchored(row string) string {
	if strings.TrimSpace(theme.Strip(row)) != "" {
		return row
	}
	return blankAnchor + row
}

type rateFacts struct {
	tps   *float64
	apiMs *float64
	tpsAt float64
}

func (r rateFacts) storedTPS() *float64 {
	if r.tps == nil || *r.tps == 0 {
		return nil
	}
	v := roundTo2(*r.tps)
	return &v
}

func roundTo2(f float64) float64 { return float64(int64(f*100+0.5)) / 100 }

// measureRate works out the output rate and what to carry forward.
//
// The two fields do NOT measure the same thing, and that is the trick:
// total_output_tokens is what the LAST response produced - it resets every
// turn, it is not a running total - while total_api_duration_ms is the
// session's ACCUMULATED api time. The rate of the last response is the first
// over how much the second grew. Subtracting two consecutive
// total_output_tokens measures nothing: they are two different responses.
func measureRate(p *Payload, facts session.Facts, now float64) rateFacts {
	out := rateFacts{tps: facts.TPS, apiMs: facts.APIMs, tpsAt: facts.TPSAt}

	if p.OutTokens != nil && *p.OutTokens != 0 && p.APIMs != nil {
		if facts.APIMs != nil && *p.APIMs > *facts.APIMs {
			elapsed := (*p.APIMs - *facts.APIMs) / 1000.0
			// Under 300 ms of api time is not a measurement, it is noise
			// divided.
			if elapsed >= 0.3 && elapsed <= 600 {
				rate := *p.OutTokens / elapsed
				out.tps = &rate
				out.tpsAt = now
			}
		}
		api := *p.APIMs
		out.apiMs = &api
	}

	// Once the model has been quiet for a while the last rate says nothing.
	if now-out.tpsAt > 120 {
		out.tps = nil
	}
	return out
}

// Run renders one refresh.
func Run(stdin io.Reader, stdout io.Writer) error {
	now := time.Now()
	p := Parse(ReadStdin(stdin))

	columns := TermWidth()
	width := columns - RightPad()
	if width < minWidth {
		width = minWidth
	}
	showPet := EnvOn("STATUSLINE_PET") && width >= minWidthForPet
	leftWidth := width
	if showPet {
		leftWidth = width - (cardWidth + cardGap)
	}

	// The working tree, behind its cache. Out here rather than in Parse
	// because it is the one part of the git reading worth paying for at most
	// once every dirtyTTL; the branch itself came out of .git/HEAD for free.
	if p.Root != "" {
		p.Dirty = dirtyOf(p.Root, p.SessionID, now)
	}

	factsPath := session.PathFor(p.SessionID, "")
	facts := session.Load(factsPath)
	rate := measureRate(p, facts, float64(now.UnixNano())/1e9)

	// The sweep is a ReadDir of the whole of $TMPDIR, and it used to run on
	// every refresh that changed anything - which, while the model is
	// answering, is every single one of them: total_api_duration_ms grows
	// under it. What it collects is leftovers older than a DAY, so once per
	// session is as often as it can possibly need to be. T0 is only zero
	// before this session has written its first facts file.
	firstRefresh := facts.T0 == 0

	// A turn is a prompt, and the payload carries its id. Counted when the
	// prompt changes, not on every refresh.
	newTurn := p.PromptID != "" && facts.PromptID != p.PromptID

	band2 := work(p)
	band3 := quota(p)

	var card Card
	var creature *pet.State
	haveCard := false
	if showPet {
		card, creature = RenderCard(p, facts, rate, newTurn, columns >= BubbleMin, pet.Path(), now)
		haveCard = true
		if card.Facts != nil && factsPath != "" {
			session.Save(factsPath, *card.Facts)
			if firstRefresh {
				session.Sweep(now)
			}
		}
	}

	// Band 1 is built after the card, not before, because it borrows the
	// creature's colour: the pill and the context bar are painted whatever the
	// torso is painted. With the pet switched off there is no torso and band 1
	// falls back to the palette it always had.
	var accent *theme.Colour
	if haveCard {
		accent = &card.Body
	}
	band1 := engine(p, rate.tps, accent)

	var lines []string
	laidOut := false
	if haveCard {
		left := []string{
			assemble(band1, leftWidth),
			assemble(band2, leftWidth),
			assemble(band3, leftWidth),
			assemble(petBand(card, columns), leftWidth),
		}
		fits := true
		for _, row := range left {
			if theme.Width(row) > leftWidth {
				fits = false
				break
			}
		}
		if fits {
			laidOut = true
			// Band 4 drops the bubble before anything else when it runs short,
			// so whether the pet actually got a word in is only knowable once
			// the band is assembled - and only in the branch that is really
			// going to be printed. Clearing it before choosing the layout
			// robbed the full-width fallback of a bubble that would have fit.
			card.Bubble = spokenIn(left[3], card.Bubble)
			for i, row := range left {
				lines = append(lines, theme.PadRight(anchored(row), leftWidth)+
					strings.Repeat(" ", cardGap)+card.Rows[i])
			}
		}
	}
	if !laidOut {
		// Falling back to full-width bands re-assembles band 4 against a wider
		// budget, so the bubble gets a second chance and is re-checked below.
		lines = []string{
			assemble(band1, width),
			assemble(band2, width),
			assemble(band3, width),
		}
		if haveCard {
			band := assemble(petBand(card, columns), width)
			card.Bubble = spokenIn(band, card.Bubble)
			lines = append(lines, band)
		}
	}
	if haveCard {
		// The rung first: it is what keeps a broken streak from taking the
		// shape back down the tree, and it must not depend on there having
		// been a bubble to say so.
		RememberShape(card, creature, pet.Path())
		Spoke(card, creature, pet.Path(), now)
	}

	_, err := io.WriteString(stdout, NewFooter(width).Render(lines)+"\n")
	return err
}
