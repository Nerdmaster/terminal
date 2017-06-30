package terminal

import (
	"io"
	"strconv"
)

// A Prompt is a wrapper around a Reader which will write a prompt, wait for
// a user's input, and return it.  It will print whatever needs to be printed
// on demand to an io.Writer.  The Prompt stores the Reader's prior state in
// order to avoid unnecessary writes.
type Prompt struct {
	*Reader
	prompt []byte
	Out    io.Writer

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

	// moveBytes just holds onto the byte slice we use for cursor movement to
	// avoid every cursor move requesting tiny bits of memory
	moveBytes []byte

	// Scroller processes the pending output to figure out if scrolling is
	// necessary and what should be printed if so
	Scroller *Scroller
}

// NewPrompt returns a prompt which will read lines from r, write its
// prompt and current line to w, and use p as the prompt string.
func NewPrompt(r io.Reader, w io.Writer, p string) *Prompt {
	var prompt = &Prompt{
		Reader:    NewReader(r),
		Out:       w,
		moveBytes: make([]byte, 2, 16),
	}

	prompt.Scroller = NewScroller()

	prompt.Reader.AfterKeypress = prompt.afterKeyPress
	prompt.SetPrompt(p)

	// Set up the constant moveBytes prefix
	prompt.moveBytes[0] = '\x1b'
	prompt.moveBytes[1] = '['

	return prompt
}

// ReadLine delegates to the reader's ReadLine function
func (p *Prompt) ReadLine() (string, error) {
	p.lastOutput = p.lastOutput[:0]
	p.lastCurPos = 0
	p.Scroller.Reset()

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
	var out, curPos = p.Scroller.Filter(e.Line)
	p.nextOutput = append(p.nextOutput[:0], out...)

	// Pad output if it's shorter than last output
	var outputLen = len(p.nextOutput)
	for outputLen < len(p.lastOutput) {
		p.nextOutput = append(p.nextOutput, ' ')
		outputLen++
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
	p.moveCursor(curPos)
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
