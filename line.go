package terminal

// Line manages a very encapsulated version of a terminal line's state
type Line struct {
	Text []rune
	Pos  int
}

// Set overwrites Line and Pos with t and p, respectively
func (l *Line) Set(t []rune, p int) {
	l.Text = t
	l.Pos = p
}

// Clear erases the input line
func (l *Line) Clear() {
	l.Text = l.Text[:0]
	l.Pos = 0
}

// AddKeyToLine inserts the given key at the current position in the current
// line.
func (l *Line) AddKeyToLine(key rune) {
	if len(l.Text) == cap(l.Text) {
		newLine := make([]rune, len(l.Text), 2*(1+len(l.Text)))
		copy(newLine, l.Text)
		l.Text = newLine
	}
	l.Text = l.Text[:len(l.Text)+1]
	copy(l.Text[l.Pos+1:], l.Text[l.Pos:])
	l.Text[l.Pos] = key
	l.Pos++
}

// String just returns the Line runes as a single string
func (l *Line) String() string {
	return string(l.Text)
}

// Split returns everything to the left of the cursor and everything at and to
// the right of the cursor as two strings
func (l *Line) Split() (string, string) {
	return string(l.Text[:l.Pos]), string(l.Text[l.Pos:])
}

// EraseNPreviousChars deletes n characters from l.Text and updates l.Pos
func (l *Line) EraseNPreviousChars(n int) {
	if l.Pos == 0 || n == 0 {
		return
	}

	if l.Pos < n {
		n = l.Pos
	}
	l.Pos -= n

	copy(l.Text[l.Pos:], l.Text[n+l.Pos:])
	l.Text = l.Text[:len(l.Text)-n]
}

// DeleteLine removes all runes after the cursor position
func (l *Line) DeleteLine() {
	l.Text = l.Text[:l.Pos]
}

// DeleteRuneUnderCursor erases the character under the current position
func (l *Line) DeleteRuneUnderCursor() {
	if l.Pos < len(l.Text) {
		l.MoveRight()
		l.EraseNPreviousChars(1)
	}
}

// DeleteToBeginningOfLine removes everything behind the cursor
func (l *Line) DeleteToBeginningOfLine() {
	l.EraseNPreviousChars(l.Pos)
}

// CountToLeftWord returns then number of characters from the cursor to the
// start of the previous word
func (l *Line) CountToLeftWord() int {
	if l.Pos == 0 {
		return 0
	}

	pos := l.Pos - 1
	for pos > 0 {
		if l.Text[pos] != ' ' {
			break
		}
		pos--
	}
	for pos > 0 {
		if l.Text[pos] == ' ' {
			pos++
			break
		}
		pos--
	}

	return l.Pos - pos
}

// MoveToLeftWord moves pos to the first rune of the word to the left
func (l *Line) MoveToLeftWord() {
	l.Pos -= l.CountToLeftWord()
}

// CountToRightWord returns then number of characters from the cursor to the
// start of the next word
func (l *Line) CountToRightWord() int {
	pos := l.Pos
	for pos < len(l.Text) {
		if l.Text[pos] == ' ' {
			break
		}
		pos++
	}
	for pos < len(l.Text) {
		if l.Text[pos] != ' ' {
			break
		}
		pos++
	}
	return pos - l.Pos
}

// MoveToRightWord moves pos to the first rune of the word to the right
func (l *Line) MoveToRightWord() {
	l.Pos += l.CountToRightWord()
}

// MoveLeft moves pos one rune left
func (l *Line) MoveLeft() {
	if l.Pos == 0 {
		return
	}

	l.Pos--
}

// MoveRight moves pos one rune right
func (l *Line) MoveRight() {
	if l.Pos == len(l.Text) {
		return
	}

	l.Pos++
}

// MoveHome moves the cursor to the beginning of the line
func (l *Line) MoveHome() {
	l.Pos = 0
}

// MoveEnd puts the cursor at the end of the line
func (l *Line) MoveEnd() {
	l.Pos = len(l.Text)
}
