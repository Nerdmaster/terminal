package terminal

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

// A Prompter is a wrapper around a Reader which will write a prompt, wait for
// a user's input, and return it.  It will print whatever needs to be printed
// on demand to an io.Writer.  The Prompter stores the Reader's prior state in
// order to avoid unnecessary writes.
type Prompter struct {
	*Reader
	prompt   string
	Out      io.Writer
	buf      bytes.Buffer
	x, y     int
	inputX   int
	line     string
	pos      int
	prompted bool
}

// VisualLength returns the number of visible glyphs in a string.  This can be
// useful for getting the length of a string which has ANSI color sequences,
// but it doesn't count "wide" glyphs differently than others, and it won't
// handle ANSI cursor commands; e.g., it ignores "\x1b[D" rather than knowing
// that the cursor position moved to the left.
func VisualLength(s string) int {
	runes := []rune(s)
	inEscapeSeq := false
	length := 0

	for _, r := range runes {
		switch {
		case inEscapeSeq:
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscapeSeq = false
			}
		case r == '\x1b':
			inEscapeSeq = true
		default:
			length++
		}
	}

	return length
}

// NewPrompter returns a prompter which will read lines from r, write its
// prompt and current line to w, and use p as the prompt string.
func NewPrompter(r io.Reader, w io.Writer, p string) *Prompter {
	return &Prompter{Reader: NewReader(r), Out: w, prompt: p, buf: bytes.Buffer{}, x: 1, y: 1}
}

// ReadLine delegates to the reader's ReadLine function
func (p *Prompter) ReadLine() (string, error) {
	line, err := p.Reader.ReadLine()
	return line, err
}

// SetPrompt changes the current prompt.  This shouldn't be called while a
// ReadLine is in progress.
func (p *Prompter) SetPrompt(s string) {
	p.prompt = s
	p.inputX = p.x + VisualLength(p.prompt)
}

// SetLocation changes the internal x and y coordinates.  If this is called
// while a ReadLine is in progress, you won't be happy.
func (p *Prompter) SetLocation(x, y int) {
	p.x = x + 1
	p.inputX = p.x + VisualLength(p.prompt)
	p.y = y + 1
}

// NeedWrite returns true if there are any pending changes to the line or
// cursor position
func (p *Prompter) NeedWrite() bool {
	line, pos := p.LinePos()
	return line != p.line || pos != p.pos
}

// WriteAll forces a write of the entire prompt
func (p *Prompter) WriteAll() {
	line, pos := p.LinePos()

	p.printAt(p.x, p.y, p.prompt+p.line)
	p.pos = len(p.line)

	if p.line != line {
		prevLine := p.line

		lpl := len(prevLine)
		ll := len(line)
		bigger := lpl - ll
		if bigger > 0 {
			fmt.Fprintf(p.Out, strings.Repeat(" ", bigger))
			p.pos += bigger
		}
	}

	if p.pos != pos {
		p.pos = pos
		p.PrintCursorMovement()
	}
}

// WriteChanges attempts to only write to the console when something has
// changed (line text or the cursor position).  It will also print the prompt
// if that hasn't yet been printed.
func (p *Prompter) WriteChanges() {
	line, pos := p.LinePos()

	if !p.prompted {
		p.PrintPrompt()
		p.prompted = true
	}

	if p.line != line {
		prevLine := p.line
		p.line = line
		p.PrintLine()

		lpl := len(prevLine)
		ll := len(line)
		bigger := lpl - ll
		if bigger > 0 {
			fmt.Fprintf(p.Out, strings.Repeat(" ", bigger))
			p.pos += bigger
		}
	}

	if p.pos != pos {
		p.pos = pos
		p.PrintCursorMovement()
	}
}

// WriteChangesNoCursor prints prompt and line if necessary, but doesn't
// reposition the cursor in order to allow a frequently-updating app to write
// the cursor change where it makes sense, regardless of changes to the user's
// input.
func (p *Prompter) WriteChangesNoCursor() {
	line, pos := p.LinePos()
	p.pos = pos

	if !p.prompted {
		p.PrintPrompt()
		p.prompted = true
	}

	if p.line != line {
		prevLine := p.line
		p.line = line
		p.PrintLine()

		lpl := len(prevLine)
		ll := len(line)
		bigger := lpl - ll
		if bigger > 0 {
			fmt.Fprintf(p.Out, strings.Repeat(" ", bigger))
			p.pos += bigger
		}
	}
}

// printAt hard-codes the ANSI escape sequence for moving to a given screen
// location, then prints a string
func (p *Prompter) printAt(x, y int, s string) {
	fmt.Fprintf(p.Out, "\x1b[%d;%dH%s", y, x, s)
}

// PrintPrompt moves to the x/y coordinates of the prompter and prints the
// prompt string
func (p *Prompter) PrintPrompt() {
	p.printAt(p.x, p.y, p.prompt)
	p.pos = 0
}

// PrintLine gets the current line and prints it to the screen just after the
// prompter location
func (p *Prompter) PrintLine() {
	p.line, _ = p.LinePos()
	p.printAt(p.inputX, p.y, p.line)
	p.pos = len(p.line)
}

// PrintCursorMovement sends the ANSI escape sequence for moving the cursor
func (p *Prompter) PrintCursorMovement() {
	p.pos = p.Pos()
	p.printAt(p.inputX+p.pos, p.y, "")
}
