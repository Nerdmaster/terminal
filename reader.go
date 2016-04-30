// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package terminal

import (
	"bytes"
	"io"
	"sync"
	"unicode/utf8"
)

const DefaultMaxLineLength = 4096

// Reader contains the state for running a VT100 terminal that is capable of
// reading lines of input.  It is similar to the golang crypto/ssh/terminal
// package except that it doesn't write, leaving that to the caller.  The idea
// is to store what the user is typing, and where the cursor should be, while
// letting something else decide what to draw and where on the screen to draw
// it.  This separation enables more complex applications where there's other
// real-time data being rendered at the same time as the input line.
type Reader struct {
	// AutoCompleteCallback, if non-null, is called for each keypress with
	// the full input line and the current position of the cursor (in
	// bytes, as an index into |line|). If it returns ok=false, the key
	// press is processed normally. Otherwise it returns a replacement line
	// and the new cursor position.
	AutoCompleteCallback func(line string, pos int, key rune) (newLine string, newPos int, ok bool)

	c io.Reader
	m sync.RWMutex

	// NoHistory is on when we don't want to preserve history, such as when a
	// password is being entered
	NoHistory bool

	// MaxLineLength tells us when to stop accepting input (other than things
	// like allowing up/down/left/right and other control keys)
	MaxLineLength int

	// input is the current line being entered, and the cursor position
	input *Input

	// pasteActive is true iff there is a bracketed paste operation in
	// progress.
	pasteActive bool

	// remainder contains the remainder of any partial key sequences after
	// a read. It aliases into inBuf.
	remainder []byte
	inBuf     [256]byte

	// history contains previously entered commands so that they can be
	// accessed with the up and down keys.
	history stRingBuffer
	// historyIndex stores the currently accessed history entry, where zero
	// means the immediately previous entry.
	historyIndex int
	// When navigating up and down the history it's possible to return to
	// the incomplete, initial line. That value is stored in
	// historyPending.
	historyPending string
}

// NewReader runs a terminal reader on the given io.Reader. If the Reader is a
// local terminal, that terminal must first have been put into raw mode.
func NewReader(c io.Reader) *Reader {
	return &Reader{
		c:             c,
		MaxLineLength: DefaultMaxLineLength,
		historyIndex:  -1,
		input:         &Input{},
	}
}

const (
	keyCtrlD     = 4
	keyCtrlU     = 21
	keyEnter     = '\r'
	keyEscape    = 27
	keyBackspace = 127
	keyUnknown   = 0xd800 /* UTF-16 surrogate area */ + iota
	keyUp
	keyDown
	keyLeft
	keyRight
	keyAltLeft
	keyAltRight
	keyHome
	keyEnd
	keyDeleteWord
	keyDeleteLine
	keyClearScreen
	keyPasteStart
	keyPasteEnd
)

var pasteStart = []byte{keyEscape, '[', '2', '0', '0', '~'}
var pasteEnd = []byte{keyEscape, '[', '2', '0', '1', '~'}

// bytesToKey tries to parse a key sequence from b. If successful, it returns
// the key and the remainder of the input. Otherwise it returns utf8.RuneError.
func bytesToKey(b []byte, pasteActive bool) (rune, []byte) {
	if len(b) == 0 {
		return utf8.RuneError, nil
	}

	if !pasteActive {
		switch b[0] {
		case 1: // ^A
			return keyHome, b[1:]
		case 5: // ^E
			return keyEnd, b[1:]
		case 8: // ^H
			return keyBackspace, b[1:]
		case 11: // ^K
			return keyDeleteLine, b[1:]
		case 12: // ^L
			return keyClearScreen, b[1:]
		case 23: // ^W
			return keyDeleteWord, b[1:]
		}
	}

	if b[0] != keyEscape {
		if !utf8.FullRune(b) {
			return utf8.RuneError, b
		}
		r, l := utf8.DecodeRune(b)
		return r, b[l:]
	}

	if !pasteActive && len(b) >= 3 && b[0] == keyEscape && b[1] == '[' {
		switch b[2] {
		case 'A':
			return keyUp, b[3:]
		case 'B':
			return keyDown, b[3:]
		case 'C':
			return keyRight, b[3:]
		case 'D':
			return keyLeft, b[3:]
		case 'H':
			return keyHome, b[3:]
		case 'F':
			return keyEnd, b[3:]
		}
	}

	if !pasteActive && len(b) >= 6 && b[0] == keyEscape && b[1] == '[' && b[2] == '1' && b[3] == ';' && b[4] == '3' {
		switch b[5] {
		case 'C':
			return keyAltRight, b[6:]
		case 'D':
			return keyAltLeft, b[6:]
		}
	}

	if !pasteActive && len(b) >= 6 && bytes.Equal(b[:6], pasteStart) {
		return keyPasteStart, b[6:]
	}

	if pasteActive && len(b) >= 6 && bytes.Equal(b[:6], pasteEnd) {
		return keyPasteEnd, b[6:]
	}

	// If we get here then we have a key that we don't recognise, or a
	// partial sequence. It's not clear how one should find the end of a
	// sequence without knowing them all, but it seems that [a-zA-Z~] only
	// appears at the end of a sequence.
	for i, c := range b[0:] {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '~' {
			return keyUnknown, b[i+1:]
		}
	}

	return utf8.RuneError, b
}

func isPrintable(key rune) bool {
	isInSurrogateArea := key >= 0xd800 && key <= 0xdbff
	return key >= 32 && !isInSurrogateArea
}

// handleKey processes the given key and, optionally, returns a line of text
// that the user has entered.
func (t *Reader) handleKey(key rune) (line string, ok bool) {
	t.m.Lock()
	defer t.m.Unlock()

	i := t.input
	if t.pasteActive && key != keyEnter {
		i.AddKeyToLine(key)
		return
	}

	switch key {
	case keyBackspace:
		i.EraseNPreviousChars(1)
	case keyAltLeft:
		i.MoveToLeftWord()
	case keyAltRight:
		i.MoveToRightWord()
	case keyLeft:
		i.MoveLeft()
	case keyRight:
		i.MoveRight()
	case keyHome:
		i.MoveHome()
	case keyEnd:
		i.MoveEnd()
	case keyUp:
		ok := t.fetchPreviousHistory()
		if !ok {
			return "", false
		}
	case keyDown:
		t.fetchNextHistory()
	case keyEnter:
		line = i.String()
		ok = true
		i.Clear()
	case keyDeleteWord:
		i.EraseNPreviousChars(i.CountToLeftWord())
	case keyDeleteLine:
		i.DeleteLine()
	case keyCtrlD:
		// (The EOF case is handled in ReadLine)
		i.DeleteRuneUnderCursor()
	case keyCtrlU:
		i.DeleteToBeginningOfLine()
	case keyClearScreen:
		// TODO: implement a callback for this
	default:
		if t.AutoCompleteCallback != nil {
			prefix, suffix := i.Split()
			newLine, newPos, completeOk := t.AutoCompleteCallback(prefix+suffix, len(prefix), key)

			if completeOk {
				i.Set([]rune(newLine), utf8.RuneCount([]byte(newLine)[:newPos]))
				return
			}
		}
		if !isPrintable(key) {
			return
		}
		if len(i.Line) == t.MaxLineLength {
			return
		}
		i.AddKeyToLine(key)
	}
	return
}

// ReadPassword temporarily reads a password without saving to history
func (t *Reader) ReadPassword() (line string, err error) {
	oldNoHistory := t.NoHistory
	t.NoHistory = true
	line, err = t.ReadLine()
	t.NoHistory = oldNoHistory
	return
}

// ReadLine returns a line of input from the terminal.
func (t *Reader) ReadLine() (line string, err error) {
	t.m.RLock()
	lineLen := len(t.input.Line)
	t.m.RUnlock()

	lineIsPasted := t.pasteActive

	for {
		rest := t.remainder
		lineOk := false
		for !lineOk {
			var key rune
			key, rest = bytesToKey(rest, t.pasteActive)
			if key == utf8.RuneError {
				break
			}
			if !t.pasteActive {
				if key == keyCtrlD {
					if lineLen == 0 {
						return "", io.EOF
					}
				}
				if key == keyPasteStart {
					t.pasteActive = true
					if lineLen == 0 {
						lineIsPasted = true
					}
					continue
				}
			} else if key == keyPasteEnd {
				t.pasteActive = false
				continue
			}
			if !t.pasteActive {
				lineIsPasted = false
			}
			line, lineOk = t.handleKey(key)
		}
		if len(rest) > 0 {
			n := copy(t.inBuf[:], rest)
			t.remainder = t.inBuf[:n]
		} else {
			t.remainder = nil
		}

		if lineOk {
			if !t.NoHistory {
				t.historyIndex = -1
				t.history.Add(line)
			}
			if lineIsPasted {
				err = ErrPasteIndicator
			}
			return
		}

		// t.remainder is a slice at the beginning of t.inBuf
		// containing a partial key sequence
		readBuf := t.inBuf[len(t.remainder):]
		var n int

		n, err = t.c.Read(readBuf)

		if err != nil {
			return
		}

		t.remainder = t.inBuf[:n+len(t.remainder)]
	}

	panic("unreachable") // for Go 1.0.
}

// LinePos returns the current input line and cursor position
func (t *Reader) LinePos() (string, int) {
	t.m.RLock()
	defer t.m.RUnlock()
	return t.input.String(), t.input.Pos
}

// fetchPreviousHistory sets the input line to the previous entry in our history
func (t *Reader) fetchPreviousHistory() bool {
	// lock has to be held here
	if t.NoHistory {
		return false
	}

	entry, ok := t.history.NthPreviousEntry(t.historyIndex + 1)
	if !ok {
		return false
	}
	if t.historyIndex == -1 {
		t.historyPending = string(t.input.Line)
	}
	t.historyIndex++
	runes := []rune(entry)
	t.input.Set(runes, len(runes))
	return true
}

// fetchNextHistory sets the input line to the next entry in our history
func (t *Reader) fetchNextHistory() {
	// lock has to be held here
	if t.NoHistory {
		return
	}

	switch t.historyIndex {
	case -1:
		return
	case 0:
		runes := []rune(t.historyPending)
		t.input.Set(runes, len(runes))
		t.historyIndex--
	default:
		entry, ok := t.history.NthPreviousEntry(t.historyIndex - 1)
		if ok {
			t.historyIndex--
			runes := []rune(entry)
			t.input.Set(runes, len(runes))
		}
	}
}

type pasteIndicatorError struct{}

func (pasteIndicatorError) Error() string {
	return "terminal: ErrPasteIndicator not correctly handled"
}

// ErrPasteIndicator may be returned from ReadLine as the error, in addition
// to valid line data. It indicates that bracketed paste mode is enabled and
// that the returned line consists only of pasted data. Programs may wish to
// interpret pasted data more literally than typed data.
var ErrPasteIndicator = pasteIndicatorError{}

// stRingBuffer is a ring buffer of strings.
type stRingBuffer struct {
	// entries contains max elements.
	entries []string
	max     int
	// head contains the index of the element most recently added to the ring.
	head int
	// size contains the number of elements in the ring.
	size int
}

func (s *stRingBuffer) Add(a string) {
	if s.entries == nil {
		const defaultNumEntries = 100
		s.entries = make([]string, defaultNumEntries)
		s.max = defaultNumEntries
	}

	s.head = (s.head + 1) % s.max
	s.entries[s.head] = a
	if s.size < s.max {
		s.size++
	}
}

// NthPreviousEntry returns the value passed to the nth previous call to Add.
// If n is zero then the immediately prior value is returned, if one, then the
// next most recent, and so on. If such an element doesn't exist then ok is
// false.
func (s *stRingBuffer) NthPreviousEntry(n int) (value string, ok bool) {
	if n >= s.size {
		return "", false
	}
	index := s.head - n
	if index < 0 {
		index += s.max
	}
	return s.entries[index], true
}
