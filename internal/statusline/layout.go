package statusline

import (
	"os"
	"strconv"
	"strings"

	"github.com/kyros-software/claude-code-themes/internal/theme"
)

// How the four bands are packed into the columns we actually have. Each band
// drops its lowest-priority elements rather than wrap, because wrapping knocks
// the prompt box out of square.

var (
	sep  = theme.Fg(theme.Rule) + "│" + theme.Reset
	sepX = " " + sep + " "
)

const (
	// BubbleMin is the design's floor: below this many columns the speech
	// bubble drops itself. Measured against the terminal's columns, not the
	// usable width, as the design says.
	BubbleMin = 100

	cardWidth = 9
	cardGap   = 2

	defaultRightPad = 6
	minWidth        = 20
	fallbackCols    = 80
)

// EnvOn reads a switch that defaults to on.
func EnvOn(name string) bool {
	v := strings.ToLower(os.Getenv(name))
	return v != "0" && v != "off" && v != "no"
}

// TermWidth is not in the payload, so it comes from COLUMNS. Claude Code trims
// the line a few characters before that value, hence the right pad.
func TermWidth() int {
	if cols := os.Getenv("COLUMNS"); cols != "" {
		if n, err := strconv.Atoi(cols); err == nil && n > 0 {
			return n
		}
	}
	if n := ttyWidth(); n > 0 {
		return n
	}
	return fallbackCols
}

// RightPad is the margin kept clear on the right.
func RightPad() int {
	if v := os.Getenv("STATUSLINE_RIGHT_PAD"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			if n < 1 {
				return 1
			}
			return n
		}
	}
	return defaultRightPad
}

// segment is one element of a band. Priority is the order it gets dropped in:
// higher goes first. paint/plain let a truncated element keep its colour.
type segment struct {
	priority int
	text     string
	width    int
	sep      string
	sepWidth int
	paint    string
	plain    string
	hasPlain bool
}

func seg(priority int, text string) segment {
	return segment{priority: priority, text: text, width: theme.Width(text),
		sep: sepX, sepWidth: theme.Width(sepX)}
}

func (s segment) withSep(sp string) segment {
	s.sep, s.sepWidth = sp, theme.Width(sp)
	return s
}

func (s segment) truncatable(paint, plain string) segment {
	s.paint, s.plain, s.hasPlain = paint, plain, true
	return s
}

func assemble(segments []segment, width int) string {
	items := make([]segment, 0, len(segments))
	items = append(items, segments...)

	total := func(its []segment) int {
		sum := 0
		for i, it := range its {
			sum += it.width
			if i > 0 {
				sum += it.sepWidth
			}
		}
		return sum
	}

	for len(items) > 1 && total(items) > width {
		worst := 0
		for i := range items {
			// Ties go to the later element, matching the design's order.
			if items[i].priority >= items[worst].priority {
				worst = i
			}
		}
		items = append(items[:worst], items[worst+1:]...)
	}

	if len(items) == 1 && items[0].width > width {
		it := items[0]
		keep := width - 1
		if keep < 1 {
			keep = 1
		}
		if it.hasPlain {
			text := theme.Truncate(it.plain, keep) + "…"
			it.text = it.paint + text + theme.Reset
		} else {
			it.text = theme.Truncate(theme.Strip(it.text), keep) + "…"
		}
		it.width = theme.Width(it.text)
		items = []segment{it}
	}

	var b strings.Builder
	for i, it := range items {
		if i > 0 {
			b.WriteString(it.sep)
		}
		b.WriteString(it.text)
	}
	return b.String()
}

// Footer paints the status line as a footer, not one more band: a background
// one tone above black and a hairline on top say it belongs to the window, not
// to the conversation. The rule costs a row; STATUSLINE_RULE=0 gives it back.
type Footer struct {
	Width      int
	Background bool
	Rule       bool
	bg         string
}

// NewFooter reads the two switches once.
func NewFooter(width int) *Footer {
	f := &Footer{Width: width,
		Background: EnvOn("STATUSLINE_BACKGROUND"),
		Rule:       EnvOn("STATUSLINE_RULE")}
	if f.Background {
		f.bg = theme.Bgc(theme.Bg)
	}
	return f
}

func (f *Footer) paint(line string) string {
	if !f.Background {
		return line
	}
	gap := f.Width - theme.Width(line)
	if gap < 0 {
		gap = 0
	}
	// A full reset takes the footer background with it, so it is swapped for
	// "drop the bold, back to the default foreground" and the background is
	// re-asserted - which survives even the model pill, which brings its own.
	return f.bg + strings.ReplaceAll(line, theme.Reset, theme.Soft+f.bg) +
		strings.Repeat(" ", gap) + theme.Reset
}

// Render assembles the final block of lines.
func (f *Footer) Render(lines []string) string {
	out := make([]string, 0, len(lines)+1)
	if f.Rule {
		out = append(out, theme.Fg(theme.Border)+strings.Repeat("─", f.Width)+theme.Reset)
	}
	out = append(out, lines...)
	for i, line := range out {
		out[i] = f.paint(line)
	}
	return strings.Join(out, "\n")
}
