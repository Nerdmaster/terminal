package terminal

import (
	"io"
	"strconv"
)

// ScrollBy is the default number of runes we scroll left or right when
// scrolling a Prompt's input
const ScrollBy = 10

// A Prompt is a wrapper around a Reader which will write a prompt, wait for
// a user's input, and return it.  It will print whatever needs to be printed
// on demand to an io.Writer.  The Prompt stores the Reader's prior state in
// order to avoid unnecessary writes.
type Prompt struct {
	*Reader
	prompt []byte
	Out    io.Writer

	// LeftOverflow and RightOverflow are used to signify that the input is
	// scrolling left or right.  They both default to the UTF ellipsis character,
	// but can be overridden as needed.  If set to '', no overflow character will
	// be displayed when scrolling
	LeftOverflow, RightOverflow rune

	// lastOutput mirrors whatever was last printed to the console
	lastOutput []rune

	// nextOutput is built as we determine what needs printing, and then whatever
	// parts have changed from lastOutput to nextOutput are printed
	nextOutput []rune

	// lastCurPos stores the previous physical cursor position on the screen.
	// This is a screen position relative to the user's input, not the location
	// within the full string
	lastCurPos int

	// AfterKeypress shadows the Reader variable of the same name to allow custom
	// keypress listeners even though Prompt has to listen in order to write output
	AfterKeypress func(event *KeyEvent)

	// InputWidth should be set to the maximum size of the input area.  If this
	// is less than the MaxLineLength, the input area will scroll left and right
	// to allow for longer text input than the screen might otherwise allow.
	InputWidth int

	// ScrollOffset is set to the number of characters which are "off-screen";
	// the input line displays just the characters which are after this offset.
	// This should typically not be adjusted manually, but it may make sense to
	// allow scrolling the input via a keyboard shortcut that doesn't alter the
	// line or cursor position.
	ScrollOffset int

	// ScrollBy is the number of runes we "shift" when the cursor would otherwise
	// leave the printable area; defaults to the ScrollBy package constant
	ScrollBy int

	// moveBytes just holds onto the byte slice we use for cursor movement to
	// avoid every cursor move requesting tiny bits of memory
	moveBytes []byte
}

// NewPrompt returns a prompt which will read lines from r, write its
// prompt and current line to w, and use p as the prompt string.
func NewPrompt(r io.Reader, w io.Writer, p string) *Prompt {
	var prompt = &Prompt{
		Reader:        NewReader(r),
		Out:           w,
		moveBytes:     make([]byte, 2, 16),
		ScrollBy:      ScrollBy,
		LeftOverflow:  '…',
		RightOverflow: '…',
	}
	prompt.Reader.AfterKeypress = prompt.afterKeyPress
	prompt.InputWidth = prompt.Reader.MaxLineLength
	prompt.SetPrompt(p)

	// Set up the constant moveBytes prefix
	prompt.moveBytes[0] = '\x1b'
	prompt.moveBytes[1] = '['

	return prompt
}

// ReadLine delegates to the reader's ReadLine function
func (p *Prompt) ReadLine() (string, error) {
	p.ScrollOffset = 0
	p.lastOutput = p.lastOutput[:0]
	p.lastCurPos = 0
	p.Out.Write(p.prompt)
	line, err := p.Reader.ReadLine()
	p.Out.Write(CRLF)

	return line, err
}

// SetPrompt changes the current prompt
func (p *Prompt) SetPrompt(s string) {
	p.prompt = []byte(s)
}

// afterKeyPress calls Prompt's key handler to draw changes, then the user-
// defined callback if present
func (p *Prompt) afterKeyPress(e *KeyEvent) {
	// We never write changes when enter is pressed, because the line has been
	// cleared by the Reader, and is about to be returned
	if e.Key != KeyEnter {
		p.writeChanges(e)
	}
	if p.AfterKeypress != nil {
		p.AfterKeypress(e)
	}
}

// writeChanges checks for differences in whatever was previously written to
// the console and the new line, attempting to draw the smallest amount of data
// to get things back in sync
func (p *Prompt) writeChanges(e *KeyEvent) {
	// Check for new cursor location being off-screen
	var cursorLoc = e.Input.Pos - p.ScrollOffset
	var lineLen = len(e.Input.Line)

	// Too far left
	for cursorLoc <= 0 && p.ScrollOffset > 0 {
		p.ScrollOffset -= p.ScrollBy
		cursorLoc += p.ScrollBy
	}
	if p.ScrollOffset < 0 {
		p.ScrollOffset = 0
	}

	// Too far right
	var maxScroll = p.MaxLineLength - p.InputWidth
	for cursorLoc >= p.InputWidth-1 && p.ScrollOffset < maxScroll {
		p.ScrollOffset += p.ScrollBy
		cursorLoc -= p.ScrollBy
	}
	if p.ScrollOffset >= maxScroll {
		p.ScrollOffset = maxScroll
	}

	// Figure out what we need to output next by pulling just the parts of the
	// input runes that will be visible
	var end = p.ScrollOffset + p.InputWidth
	if end > lineLen {
		end = lineLen
	}
	p.nextOutput = append(p.nextOutput[:0], e.Input.Line[p.ScrollOffset:end]...)
	var outputLen = end - p.ScrollOffset
	for outputLen < len(p.lastOutput) {
		p.nextOutput = append(p.nextOutput, ' ')
		outputLen++
	}
	if p.ScrollOffset > 0 && p.LeftOverflow != 0 {
		p.nextOutput[0] = p.LeftOverflow
	}
	if p.InputWidth+p.ScrollOffset < lineLen && p.RightOverflow != 0 {
		p.nextOutput[len(p.nextOutput)-1] = p.RightOverflow
	}

	// Compare last output with what we need to print next so we only redraw
	// starting from where they differ
	var index = runesDiffer(p.lastOutput, p.nextOutput)
	if index >= 0 {
		p.moveCursor(index)
		var out = p.nextOutput[index:]
		p.lastCurPos += len(out)
		p.Out.Write([]byte(string(out)))
		p.lastOutput = append(p.lastOutput[:0], p.nextOutput...)
	}

	// Make sure that after all the redrawing, the cursor gets back to where it should be
	if e.Input.Pos - p.ScrollOffset == p.InputWidth {
		e.Input.Pos--
	}
	p.moveCursor(e.Input.Pos - p.ScrollOffset)
}

// moveCursor moves the cursor to the given x location (relative to the
// beginning of the user's input area)
func (p *Prompt) moveCursor(x int) {
	var dx = x - p.lastCurPos

	if dx == 0 {
		return
	}

	var seq []byte = p.moveBytes[:2]

	var last byte
	if dx > 0 {
		last = 'C'
	} else {
		dx = -dx
		last = 'D'
	}

	// For the most common cases, let's make this simpler
	if dx == 1 {
		seq = append(seq, last)
	} else if dx < 10 {
		seq = append(seq, '0'+byte(dx), last)
	} else {
		var dxString = strconv.Itoa(dx)
		seq = append(seq, []byte(dxString)...)
		seq = append(seq, last)
	}
	p.Out.Write(seq)
	p.lastCurPos = x
}
