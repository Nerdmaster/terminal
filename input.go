package terminal

// Input just manages a very encapsulated version of a terminal line so it can
// be synchronized for read and write without the sprawling access this data
// had in the original terminal structure
type Input struct {
	Line []rune
	Pos  int
}

// Set overwrites Line and Pos with l and p, respectively
func (i *Input) Set(l []rune, p int) {
	i.Line = l
	i.Pos = p
}

// Clear erases the input line
func (i *Input) Clear() {
	i.Line = i.Line[:0]
	i.Pos = 0
}

// AddKeyToLine inserts the given key at the current position in the current
// line.
func (i *Input) AddKeyToLine(key rune) {
	if len(i.Line) == cap(i.Line) {
		newLine := make([]rune, len(i.Line), 2*(1+len(i.Line)))
		copy(newLine, i.Line)
		i.Line = newLine
	}
	i.Line = i.Line[:len(i.Line)+1]
	copy(i.Line[i.Pos+1:], i.Line[i.Pos:])
	i.Line[i.Pos] = key
	i.Pos++
}

// String just returns the Line runes as a single string
func (i *Input) String() string {
	return string(i.Line)
}

// Split returns everything to the left of the cursor and everything at and to
// the right of the cursor as two strings
func (i *Input) Split() (string, string) {
	return string(i.Line[:i.Pos]), string(i.Line[i.Pos:])
}

// EraseNPreviousChars deletes n characters from i.Line and updates i.Pos
func (i *Input) EraseNPreviousChars(n int) {
	if i.Pos == 0 || n == 0 {
		return
	}

	if i.Pos < n {
		n = i.Pos
	}
	i.Pos -= n

	copy(i.Line[i.Pos:], i.Line[n+i.Pos:])
	i.Line = i.Line[:len(i.Line)-n]
}

// DeleteLine removes all runes after the cursor position
func (i *Input) DeleteLine() {
	i.Line = i.Line[:i.Pos]
}

// DeleteRuneUnderCursor erases the character under the current position
func (i *Input) DeleteRuneUnderCursor() {
	if i.Pos < len(i.Line) {
		i.MoveRight()
		i.EraseNPreviousChars(1)
	}
}

// DeleteToBeginningOfLine removes everything behind the cursor
func (i *Input) DeleteToBeginningOfLine() {
	i.EraseNPreviousChars(i.Pos)
}

// CountToLeftWord returns then number of characters from the cursor to the
// start of the previous word
func (i *Input) CountToLeftWord() int {
	if i.Pos == 0 {
		return 0
	}

	pos := i.Pos - 1
	for pos > 0 {
		if i.Line[pos] != ' ' {
			break
		}
		pos--
	}
	for pos > 0 {
		if i.Line[pos] == ' ' {
			pos++
			break
		}
		pos--
	}

	return i.Pos - pos
}

// MoveToLeftWord moves pos to the first rune of the word to the left
func (i *Input) MoveToLeftWord() {
	i.Pos -= i.CountToLeftWord()
}

// CountToRightWord returns then number of characters from the cursor to the
// start of the next word
func (i *Input) CountToRightWord() int {
	pos := i.Pos
	for pos < len(i.Line) {
		if i.Line[pos] == ' ' {
			break
		}
		pos++
	}
	for pos < len(i.Line) {
		if i.Line[pos] != ' ' {
			break
		}
		pos++
	}
	return pos - i.Pos
}

// MoveToRightWord moves pos to the first rune of the word to the right
func (i *Input) MoveToRightWord() {
	i.Pos += i.CountToRightWord()
}

// MoveLeft moves pos one rune left
func (i *Input) MoveLeft() {
	if i.Pos == 0 {
		return
	}

	i.Pos--
}

// MoveRight moves pos one rune right
func (i *Input) MoveRight() {
	if i.Pos == len(i.Line) {
		return
	}

	i.Pos++
}

// MoveHome moves the cursor to the beginning of the line
func (i *Input) MoveHome() {
	i.Pos = 0
}

// MoveEnd puts the cursor at the end of the line
func (i *Input) MoveEnd() {
	i.Pos = len(i.Line)
}
