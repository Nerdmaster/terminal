package terminal

import (
	"bytes"
	"io"
	"unicode/utf8"
)

// KeyModifier tells us what modifiers were pressed at the same time as a
// normal key, such as CTRL, Alt, Meta, etc.
type KeyModifier int

// KeyModifier values.  We don't include Shift in here because terminals don't
// include shift for a great deal of keys that can exist; e.g., there is no
// "SHIFT + PgUp".  Similarly, CTRL doesn't make sense as a modifier in
// terminals.  CTRL+A is just ASCII character 1, whereas there is no CTRL+1,
// and CTRL+Up is its own totally separate sequence from Up.  So CTRL keys are
// just defined on an as-needed basis.
const (
	ModNone KeyModifier = 0
	ModAlt              = 1
	ModMeta             = 2
)

func (m KeyModifier) String() string {
	if m&ModAlt != 0 {
		if m&ModMeta != 0 {
			return "Meta+Alt"
		}
		return "Alt"
	}
	if m&ModMeta != 0 {
		return "Meta"
	}
	return "None"
}

// Keypress contains the data which made up a key: our internal KeyXXX constant
// and the bytes which were parsed to get said constant.  If the raw bytes need
// to be held for any reason, they should be copied, not stored as-is, since
// what's in here is a simple slice into the raw buffer.
type Keypress struct {
	Key      rune
	Modifier KeyModifier
	Size     int
	Raw      []byte
}

// KeyReader is the low-level type for reading raw keypresses from a given io
// stream, usually stdin or an ssh socket
type KeyReader struct {
	input  io.Reader

	// If ForceParse is true, the reader won't wait for certain sequences to
	// finish, which allows for things like ESC or Alt-left-bracket to be
	// detected properly
	ForceParse bool

	// remainder contains the remainder of any partial key sequences after
	// a read. It aliases into inBuf.
	remainder []byte
	inBuf     [256]byte

	// offset stores the number of bytes in inBuf to skip next time a keypress is
	// read, allowing us to guarantee inBuf (and thus a Keypress's Raw bytes)
	// stays the same after returning.
	offset int

	// midRune is true when we believe we have a partial rune and need to read
	// more bytes
	midRune bool
}

// NewKeyReader returns a simple KeyReader set to read from i
func NewKeyReader(i io.Reader) *KeyReader {
	return &KeyReader{input: i}
}

// ReadKeypress reads the next key sequence, returning a Keypress object and possibly
// an error if the input stream can't be read for some reason.  This will block
// only if the "remainder" buffer has no more data, which would obviously
// require a read.
func (r *KeyReader) ReadKeypress() (Keypress, error) {
	// Unshift from inBuf if we have an offset from a prior read
	if r.offset > 0 {
		var rest = r.remainder[r.offset:]
		if len(rest) > 0 {
			var n = copy(r.inBuf[:], rest)
			r.remainder = r.inBuf[:n]
		} else {
			r.remainder = nil
		}

		r.offset = 0
	}

	if r.midRune || len(r.remainder) == 0 {
		// r.remainder is a slice at the beginning of r.inBuf
		// containing a partial key sequence
		readBuf := r.inBuf[len(r.remainder):]

		n, err := r.input.Read(readBuf)
		if err != nil {
			return Keypress{}, err
		}

		// After a read, we assume we are not mid-rune, and we adjust remainder to
		// include what was just read
		r.midRune = false
		r.remainder = r.inBuf[:n+len(r.remainder)]
	}

	// We must have bytes here; try to parse a key
	key, i, mod := ParseKey(r.remainder, r.ForceParse)

	// Rune errors combined with a zero-length character mean we've got a partial
	// rune; invalid bytes get treated by utf8.DecodeRune as a 1-byte RuneError
	if i == 0 && key == utf8.RuneError {
		r.midRune = true
	}

	var kp = Keypress{Key: key, Size: i, Modifier: mod, Raw: r.remainder[:i]}

	// Store new offset so we can adjust the buffer next loop
	r.offset = i

	return kp, nil
}

// ParseKey tries to parse a key sequence from b. If successful, it returns the
// key and the length in bytes of that key. Otherwise it returns
// utf8.RuneError, 0.  If force is true, partial sequences will be returned
// with a best-effort approach to making them meaningful, rather than flagging
// the caller that there may be more bytes needed.  This is useful for
// gathering special keys like escape, which otherwise hold up the key reader
// waiting for the rest of a nonexistent sequence.
func ParseKey(b []byte, force bool) (rune, int, KeyModifier) {
	var runeLen int
	var l = len(b)
	var mod KeyModifier
	if l == 0 {
		return utf8.RuneError, 0, mod
	}

	// Handle ctrl keys early (DecodeRune can do this, but it's a bit quicker to
	// handle this first (I'm assuming so, anyway, since the original
	// implementation did this first)
	if b[0] < KeyEscape {
		return rune(b[0]), 1, mod
	}

	if b[0] != KeyEscape {
		if !utf8.FullRune(b) {
			if force {
				return utf8.RuneError, len(b), mod
			}
			return utf8.RuneError, 0, mod
		}
		var r rune
		r, l = utf8.DecodeRune(b)
		return r, l, mod
	}

	// From the above test we know the first key is escape.  If that's all we
	// have, we are *probably* missing some bytes... but maybe not.
	if l == 1 {
		if force {
			return KeyEscape, 1, mod
		}
		return keyUnknown(b, force, mod)
	}

	// Check for alt+valid rune
	if b[1] != '[' && b[1] != 0x1b && utf8.FullRune(b[1:]) {
		var r, l = utf8.DecodeRune(b[1:])
		return r, l + 1, ModAlt
	}

	// If length is exactly 2, and we have '[', that can be alt-left-bracket or
	// an unfinished sequence
	if l == 2 && b[1] == '[' {
		if force {
			return KeyLeftBracket, 2, ModAlt
		}
		return keyUnknown(b, force, mod)
	}

	// Everything else we know how to handle is at least 3 bytes
	if l < 3 {
		if force {
			return utf8.RuneError, len(b), mod
		}
		return keyUnknown(b, force, mod)
	}

	// Various alt keys, at least from tmux sessions, come through as 0x1b, 0x1b, ...
	if b[1] == 0x1b {
		b = b[1:]
		l--
		runeLen = 1
		mod = ModAlt
	}

	// If it wasn't a tmux alt key, it has to be escape followed by a left bracket
	if b[1] != '[' {
		return keyUnknown(b, force, mod)
	}

	// Local terminal alt keys are sometimes longer sequences that come through
	// as "\x1b[1;3" + some alpha
	if l >= 6 && b[2] == '1' && b[3] == ';' && b[4] == '3' {
		b = append([]byte{0x1b, '['}, b[5:]...)
		l -= 3
		runeLen = 3
		mod = ModAlt
	}

	// ...and sometimes they're "\x1b[", some num, ";3~"
	if l >= 6 && b[3] == ';' && b[4] == '3' && b[5] == '~' {
		b = append([]byte{0x1b, '[', b[2]}, b[5:]...)
		l -= 2
		runeLen = 2
		mod = ModAlt
	}

	// Since the buffer may have been manipulated, we re-check that we have 3+
	// characters left
	if l < 3 {
		return keyUnknown(b, force, mod)
	}

	// From here on, all known return values must be at least 3 characters
	runeLen += 3
	switch b[2] {
	case 'A':
		return KeyUp, runeLen, mod
	case 'B':
		return KeyDown, runeLen, mod
	case 'C':
		return KeyRight, runeLen, mod
	case 'D':
		return KeyLeft, runeLen, mod
	case 'H':
		return KeyHome, runeLen, mod
	case 'F':
		return KeyEnd, runeLen, mod
	}

	if l < 4 {
		return keyUnknown(b, force, mod)
	}
	runeLen++

	// NOTE: these appear to be escape sequences I see in tmux, but some don't
	// actually seem to happen on a "direct" terminal!
	if b[3] == '~' {
		switch b[2] {
		case '1':
			return KeyHome, runeLen, mod
		case '2':
			return KeyInsert, runeLen, mod
		case '3':
			return KeyDelete, runeLen, mod
		case '4':
			return KeyEnd, runeLen, mod
		case '5':
			return KeyPgUp, runeLen, mod
		case '6':
			return KeyPgDn, runeLen, mod
		}
	}

	if l < 6 {
		return keyUnknown(b, force, mod)
	}
	runeLen += 2

	if len(b) >= 6 && bytes.Equal(b[:6], pasteEnd) {
		return KeyPasteEnd, runeLen, mod
	}

	if len(b) >= 6 && bytes.Equal(b[:6], pasteStart) {
		return KeyPasteStart, runeLen, mod
	}

	return keyUnknown(b, force, mod)
}

// keyUnknown attempts to parse the unknown key and return its size.  If the
// key can't be figured out, it returns a RuneError.
func keyUnknown(b []byte, force bool, mod KeyModifier) (rune, int, KeyModifier) {
	for i, c := range b[0:] {
		// It's not clear how to find the end of a sequence without knowing them
		// all, but it seems that [a-zA-Z~] only appears at the end of a sequence
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '~' {
			return KeyUnknown, i + 1, mod
		}
	}

	if force {
		return utf8.RuneError, len(b), mod
	}

	return utf8.RuneError, 0, mod
}

func isPrintable(key rune) bool {
	isInSurrogateArea := key >= 0xd800 && key <= 0xdbff
	return key >= 32 && !isInSurrogateArea
}
