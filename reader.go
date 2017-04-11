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

// DefaultMaxLineLength is the default MaxLineLength for a Reader; once a line
// reaches this length, the Reader no longer accepts input which would increase
// the line
const DefaultMaxLineLength = 4096

// KeyEvent is used for OnKeypress handlers to get the key and modify handler
// state when the custom handler needs default handlers to be bypassed
type KeyEvent struct {
	Keypress
	Input                 *Input
	IgnoreDefaultHandlers bool
}

// Reader contains the state for running a VT100 terminal that is capable of
// reading lines of input.  It is similar to the golang crypto/ssh/terminal
// package except that it doesn't write, leaving that to the caller.  The idea
// is to store what the user is typing, and where the cursor should be, while
// letting something else decide what to draw and where on the screen to draw
// it.  This separation enables more complex applications where there's other
// real-time data being rendered at the same time as the input line.
type Reader struct {
	// OnKeypress, if non-null, is called for each keypress with the key and
	// input line sent in
	OnKeypress func(event *KeyEvent)

	// AfterKeypress, if non-nil, is called after each keypress has been
	// processed.  event should be considered read-only, as any changes will be
	// ignored since the key has already been processed.
	AfterKeypress func(event *KeyEvent)

	keyReader *KeyReader
	m         sync.RWMutex

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
func NewReader(r io.Reader) *Reader {
	return &Reader{
		keyReader:     NewKeyReader(r),
		MaxLineLength: DefaultMaxLineLength,
		historyIndex:  -1,
		input:         &Input{},
	}
}

// handleKeypress processes the given keypress data and, optionally, returns a
// line of text that the user has entered.
func (r *Reader) handleKeypress(kp Keypress) (line string, ok bool) {
	r.m.Lock()
	defer r.m.Unlock()

	var e = &KeyEvent{Keypress: kp, Input: r.input}
	if r.OnKeypress != nil {
		r.OnKeypress(e)
		if e.IgnoreDefaultHandlers {
			return
		}
		kp.Key = e.Key
	}

	line, ok = r.processKeypress(kp)

	if r.AfterKeypress != nil {
		r.AfterKeypress(e)
	}
	return
}

// processKeypress applies all non-overrideable logic needed for various
// keypresses to have their desired effects
func (r *Reader) processKeypress(kp Keypress) (line string, ok bool) {
	var key = kp.Key
	var i = r.input
	if r.pasteActive && key != KeyEnter {
		i.AddKeyToLine(key)
		return
	}

	if kp.Modifier == ModAlt {
		switch key {
		case KeyLeft:
			i.MoveToLeftWord()
		case KeyRight:
			i.MoveToRightWord()
		}
	}

	if kp.Modifier != ModNone {
		return
	}

	switch key {
	case KeyBackspace, KeyCtrlH:
		i.EraseNPreviousChars(1)
	case KeyLeft:
		i.MoveLeft()
	case KeyRight:
		i.MoveRight()
	case KeyHome, KeyCtrlA:
		i.MoveHome()
	case KeyEnd, KeyCtrlE:
		i.MoveEnd()
	case KeyUp:
		fetched := r.fetchPreviousHistory()
		if !fetched {
			return "", false
		}
	case KeyDown:
		r.fetchNextHistory()
	case KeyEnter:
		line = i.String()
		ok = true
		i.Clear()
	case KeyCtrlW:
		i.EraseNPreviousChars(i.CountToLeftWord())
	case KeyCtrlK:
		i.DeleteLine()
	case KeyCtrlD, KeyDelete:
		// (The EOF case is handled in ReadLine)
		i.DeleteRuneUnderCursor()
	case KeyCtrlU:
		i.DeleteToBeginningOfLine()
	default:
		if !isPrintable(key) {
			return
		}
		if len(i.Line) == r.MaxLineLength {
			return
		}
		i.AddKeyToLine(key)
	}
	return
}

// ReadPassword temporarily reads a password without saving to history
func (r *Reader) ReadPassword() (line string, err error) {
	oldNoHistory := r.NoHistory
	r.NoHistory = true
	line, err = r.ReadLine()
	r.NoHistory = oldNoHistory
	return
}

// ReadLine returns a line of input from the terminal.
func (r *Reader) ReadLine() (line string, err error) {
	lineIsPasted := r.pasteActive

	for {
		lineOk := false
		for !lineOk {
			var kp Keypress
			kp, err = r.keyReader.ReadKeypress()
			if err != nil {
				return
			}

			key := kp.Key
			if key == utf8.RuneError {
				break
			}

			r.m.RLock()
			lineLen := len(r.input.Line)
			r.m.RUnlock()

			if !r.pasteActive {
				if key == KeyCtrlD {
					if lineLen == 0 {
						return "", io.EOF
					}
				}
				if key == KeyPasteStart {
					r.pasteActive = true
					if lineLen == 0 {
						lineIsPasted = true
					}
					continue
				}
			} else if key == KeyPasteEnd {
				r.pasteActive = false
				continue
			}
			if !r.pasteActive {
				lineIsPasted = false
			}
			line, lineOk = r.handleKeypress(kp)
		}

		if lineOk {
			if !r.NoHistory {
				r.historyIndex = -1
				r.history.Add(line)
			}
			if lineIsPasted {
				err = ErrPasteIndicator
			}
			return
		}
	}
}

// LinePos returns the current input line and cursor position
func (r *Reader) LinePos() (string, int) {
	r.m.RLock()
	defer r.m.RUnlock()
	return r.input.String(), r.input.Pos
}

// Pos returns the position of the cursor
func (r *Reader) Pos() int {
	r.m.RLock()
	defer r.m.RUnlock()
	return r.input.Pos
}

// fetchPreviousHistory sets the input line to the previous entry in our history
func (r *Reader) fetchPreviousHistory() bool {
	// lock has to be held here
	if r.NoHistory {
		return false
	}

	entry, ok := r.history.NthPreviousEntry(r.historyIndex + 1)
	if !ok {
		return false
	}
	if r.historyIndex == -1 {
		r.historyPending = string(r.input.Line)
	}
	r.historyIndex++
	runes := []rune(entry)
	r.input.Set(runes, len(runes))
	return true
}

// fetchNextHistory sets the input line to the next entry in our history
func (r *Reader) fetchNextHistory() {
	// lock has to be held here
	if r.NoHistory {
		return
	}

	switch r.historyIndex {
	case -1:
		return
	case 0:
		runes := []rune(r.historyPending)
		r.input.Set(runes, len(runes))
		r.historyIndex--
	default:
		entry, ok := r.history.NthPreviousEntry(r.historyIndex - 1)
		if ok {
			r.historyIndex--
			runes := []rune(entry)
			r.input.Set(runes, len(runes))
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
