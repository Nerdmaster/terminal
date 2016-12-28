// Portions Copyright 2011- The Go Authors. All rights reserved.
// Portions Copyright 2016- Jeremy Echols. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package terminal

import (
	"io"
	"sync"
	"unicode/utf8"
)

// CRLF is the byte sequence all terminals use for moving to the beginning of
// the next line
var CRLF = []byte("\r\n")

// DT contains the state for running a *very* basic terminal which
// operates effectively like an old-school telnet connection: no ANSI, no
// special keys, no history preservation, etc.  My hope here is that this is a
// more blind-user-friendly key reader.
type DT struct {
	keyReader *KeyReader
	w         io.Writer
	outBuffer []byte

	sync.RWMutex

	// Echo is on by default; set it to false for things like password prompts
	Echo bool

	// input is the current line being entered
	input []rune
}

// Dumb runs a dumb terminal reader on the given io.Reader. If the terminal is
// local, it must first have been put into raw mode.
func Dumb(r io.Reader, w io.Writer) *DT {
	return &DT{keyReader: NewKeyReader(r), w: w, Echo: true}
}

// queue prepares bytes for printing
func (dt *DT) queue(b []byte) {
	if dt.Echo {
		dt.outBuffer = b
	}
}

// handleKeypress processes the given keypress data and, optionally, returns a
// line of text that the user has entered.
func (dt *DT) handleKeypress(kp Keypress) (line string, ok bool) {
	dt.Lock()
	defer dt.Unlock()

	key := kp.Key
	switch key {
	case KeyBackspace, KeyCtrlH:
		if len(dt.input) == 0 {
			return
		}
		dt.input = dt.input[:len(dt.input)-1]
		dt.queue([]byte("\x08 \x08"))
	case KeyEnter:
		line = string(dt.input)
		ok = true
		dt.input = dt.input[:0]
		dt.queue(CRLF)
	default:
		if !isPrintable(key) {
			return
		}
		dt.input = append(dt.input, key)
		dt.queue(kp.Raw)
	}
	return
}

// flushOut attempts to write to the terminal.  Output errors aren't something
// we can easily handle here, so we ignore them rather than panic.
func (dt *DT) flushOut() {
	if len(dt.outBuffer) == 0 {
		return
	}

	dt.w.Write(dt.outBuffer)
	dt.outBuffer = nil
}

// ReadLine returns a line of input from the terminal
func (dt *DT) ReadLine() (line string, err error) {
	for {
		lineOk := false
		for !lineOk {
			var kp Keypress
			kp, err = dt.keyReader.ReadKeypress()
			if err != nil {
				return
			}

			key := kp.Key
			if key == utf8.RuneError {
				break
			}

			line, lineOk = dt.handleKeypress(kp)
			dt.flushOut()
		}

		if lineOk {
			return
		}
	}
}

// Line returns the current input line as a string; this can be useful for
// interrupting a user's input for critical messages and then re-entering the
// previous prompt and text.  I don't know if doing this is accessible, but
// I've seen apps that do it.
func (dt *DT) Line() string {
	dt.RLock()
	defer dt.RUnlock()
	return string(dt.input)
}
