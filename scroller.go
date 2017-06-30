package terminal

import "log"

// ScrollBy is the default value a scroller scrolls by when the cursor would
// otherwise be outside the input area
const ScrollBy = 10

// A Scroller is a Line filter for taking the internal Line's state and giving
// an output widget what should be drawn to the screen
type Scroller struct {
	// InputWidth should be set to the terminal width or smaller.  If this is
	// equal to or larger than MaxWidth, no scrolling will occur
	InputWidth int

	// MaxLineLength should be set to the maximum number of runes to allow in the
	// scrolling input.  This should be set to the underlying Reader's
	// MaxLineLength or less, otherwise the Reader will block further input.
	MaxLineLength int

	// ScrollOffset is set to the number of characters which are "off-screen" to
	// the left of the input area; the input line displays just the characters
	// which are after this offset.  This should typically not be adjusted
	// manually, but it may make sense to allow scrolling the input via a
	// keyboard shortcut that doesn't alter the line or cursor position.
	ScrollOffset int

	// LeftOverflow and RightOverflow are used to signify that the input is
	// scrolling left or right.  They both default to the UTF ellipsis character,
	// but can be overridden as needed.  If set to '', no overflow character will
	// be displayed when scrolling
	LeftOverflow, RightOverflow rune

	// ScrollBy is the number of runes we "shift" when the cursor would otherwise
	// leave the printable area; defaults to the ScrollBy package constant
	ScrollBy int

	// nextOutput is built as we determine what needs printing so we aren't
	// allocating memory all the time
	nextOutput []rune
}

// NewScroller creates a simple scroller instance with no limits.
// MaxLineLength and InputWidth must be set before any of the scrolling logic
// will kick in.  Whatever uses this is responsible for setting InputWidth and
// MaxLineLength to appropriate values.
func NewScroller() *Scroller {
	return &Scroller{
		InputWidth:    -1,
		MaxLineLength: -1,
		ScrollBy:      ScrollBy,
		LeftOverflow:  '…',
		RightOverflow: '…',
	}
}

// Reset is called when a new line is being evaluated
func (s *Scroller) Reset() {
	s.ScrollOffset = 0
}

// Filter looks at the Input's line and our scroll properties to figure out
// if we should scroll, and what should be drawn in the input area
func (s *Scroller) Filter(l *Line) ([]rune, int) {
	if s.InputWidth < 1 || s.MaxLineLength < 1 {
		return l.Text, l.Pos
	}

	// Check for new cursor location being off-screen
	var cursorLoc = l.Pos - s.ScrollOffset
	var lineLen = len(l.Text)

	// Too far left
	for cursorLoc <= 0 && s.ScrollOffset > 0 {
		s.ScrollOffset -= s.ScrollBy
		cursorLoc += s.ScrollBy
	}
	if s.ScrollOffset < 0 {
		s.ScrollOffset = 0
	}

	// Too far right
	var maxScroll = s.MaxLineLength - s.InputWidth
	log.Println(s.InputWidth-1)
	log.Println(s.ScrollOffset)
	log.Println(maxScroll)
	for cursorLoc >= s.InputWidth-1 && s.ScrollOffset < maxScroll {
		log.Println("scrolling: too far right...")
		s.ScrollOffset += s.ScrollBy
		cursorLoc -= s.ScrollBy
	}
	if s.ScrollOffset >= maxScroll {
		s.ScrollOffset = maxScroll
	}

	// Figure out what we need to output next by pulling just the parts of the
	// input runes that will be visible
	var end = s.ScrollOffset + s.InputWidth
	if end > lineLen {
		end = lineLen
	}
	s.nextOutput = append(s.nextOutput[:0], l.Text[s.ScrollOffset:end]...)
	if s.ScrollOffset > 0 && s.LeftOverflow != 0 {
		s.nextOutput[0] = s.LeftOverflow
	}
	if s.InputWidth+s.ScrollOffset < lineLen && s.RightOverflow != 0 {
		s.nextOutput[len(s.nextOutput)-1] = s.RightOverflow
	}

	return s.nextOutput, cursorLoc
}
