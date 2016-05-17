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

// visualLength returns the number of visible glyphs in a string
func visualLength(s string) int {
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

func NewPrompter(r io.Reader, w io.Writer, p string) *Prompter {
	return &Prompter{Reader: NewReader(r), Out: w, prompt: p, buf: bytes.Buffer{}, x: 1, y: 1}
}

func (p *Prompter) ReadLine() (string, error) {
	line, err := p.Reader.ReadLine()
	return line, err
}

// SetLocation changes the internal x and y coordinates.  If this is called
// while a ReadLine is in progress, you won't be happy.
func (p *Prompter) SetLocation(x, y int) {
	p.x = x+1
	p.inputX = p.x + visualLength(p.prompt)
	p.y = y+1
}

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

func (p *Prompter) printAt(x, y int, s string) {
	fmt.Fprintf(p.Out, "\x1b[%d;%dH%s", y, x, s)
}

func (p *Prompter) PrintPrompt() {
	p.printAt(p.x, p.y, p.prompt)
	p.pos = 0
}

func (p *Prompter) PrintLine() {
	p.printAt(p.inputX, p.y, p.line)
	p.pos = len(p.line)
}

func (p *Prompter) PrintCursorMovement() {
	p.printAt(p.inputX+p.pos, p.y, "")
}
