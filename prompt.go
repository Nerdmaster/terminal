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

	// lastLine mirrors whatever was last printed to the console
	lastLine []rune

	// lastPos stores where the cursor was (relative to the beginning of the
	// user's input area) the last time text or cursor movement was printed
	lastPos int

	// AfterKeypress shadows the Reader variable of the same name to allow custom
	// keypress listeners even though Prompt has to listen in order to write output
	AfterKeypress func(event *KeyEvent)

	// moveBytes just holds onto the byte slice we use for cursor movement to
	// avoid every cursor move requesting tiny bits of memory
	moveBytes []byte
}

// NewPrompt returns a prompt which will read lines from r, write its
// prompt and current line to w, and use p as the prompt string.
func NewPrompt(r io.Reader, w io.Writer, p string) *Prompt {
	var prompt = &Prompt{Reader: NewReader(r), Out: w, moveBytes: make([]byte, 2, 16)}
	prompt.Reader.AfterKeypress = prompt.afterKeyPress
	prompt.SetPrompt(p)

	// Set up the constant moveBytes prefix
	prompt.moveBytes[0] = '\x1b'
	prompt.moveBytes[1] = '['

	return prompt
}

// ReadLine delegates to the reader's ReadLine function
func (p *Prompt) ReadLine() (string, error) {
	p.lastLine = p.lastLine[:0]
	p.lastPos = 0
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
	var index = runesDiffer(p.lastLine, e.Input.Line)
	if index >= 0 {
		p.moveCursor(index)
		var out = append([]rune{}, e.Input.Line[index:]...)
		for padding := len(p.lastLine) - len(e.Input.Line); padding > 0; padding-- {
			out = append(out, ' ')
		}
		p.lastPos += len(out)
		p.Out.Write([]byte(string(out)))
		p.lastLine = append(p.lastLine[:0], e.Input.Line...)
	}
	p.moveCursor(e.Input.Pos)
}

// moveCursor moves the cursor to the given x location (relative to the
// beginning of the user's input area)
func (p *Prompt) moveCursor(x int) {
	var dx = x - p.lastPos

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
	p.lastPos = x
}
