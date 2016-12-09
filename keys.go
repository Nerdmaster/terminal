package terminal

import (
	"bytes"
	"io"
	"unicode/utf8"
)

// Keypress contains the data which made up a key: our internal KeyXXX constant
// and the bytes which were parsed to get said constant
type Keypress struct {
	Key  rune
	Size int
	Raw  []byte
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

	rest := r.remainder

	// We must have bytes here; try to parse a key
	key, i := ParseKey(rest, r.ForceParse)

	// Rune errors combined with a zero-length character mean we've got a partial
	// rune; invalid bytes get treated by utf8.DecodeRune as a 1-byte RuneError
	if i == 0 && key == utf8.RuneError {
		r.midRune = true
	}

	// TODO: fix this; it eliminates a race condition, but allocates memory on
	// every single key read
	var kp = Keypress{Key: key, Size: i}
	kp.Raw = make([]byte, i)
	copy(kp.Raw, r.remainder[:i])

	// Move 'rest' forward
	rest = rest[i:]

	// If we still have bytes, adjust the buffers
	if len(rest) > 0 {
		n := copy(r.inBuf[:], rest)
		r.remainder = r.inBuf[:n]
	} else {
		r.remainder = nil
	}

	return kp, nil
}

// ParseKey tries to parse a key sequence from b. If successful, it returns the
// key and the length in bytes of that key. Otherwise it returns
// utf8.RuneError, 0.  If force is true, partial sequences will be returned
// with a best-effort approach to making them meaningful, rather than flagging
// the caller that there may be more bytes needed.  This is useful for
// gathering special keys like escape, which otherwise hold up the key reader
// waiting for the rest of a nonexistent sequence.
func ParseKey(b []byte, force bool) (rune, int) {
	var runeLen int
	var l = len(b)
	if l == 0 {
		return utf8.RuneError, 0
	}

	// Handle ctrl keys early (DecodeRune can do this, but it's a bit quicker to
	// handle this first (I'm assuming so, anyway, since the original
	// implementation did this first)
	if b[0] < KeyEscape {
		return rune(b[0]), 1
	}

	if b[0] != KeyEscape {
		if !utf8.FullRune(b) {
			if force {
				return utf8.RuneError, len(b)
			}
			return utf8.RuneError, 0
		}
		return utf8.DecodeRune(b)
	}

	// From the above test we know the first key is escape.  Everything else we
	// know how to handle is at least 3 bytes
	if l < 3 {
		if force {
			return utf8.RuneError, len(b)
		}
		return keyUnknown(b, force)
	}

	// Various alt keys, at least from tmux sessions, come through as 0x1b, 0x1b, ...
	var alt rune
	if b[1] == 0x1b {
		b = b[1:]
		l--
		runeLen = 1
		alt = KeyAlt
	}

	// If it wasn't a tmux alt key, it has to be escape followed by a left bracket
	if b[1] != '[' {
		return keyUnknown(b, force)
	}

	// Local terminal alt keys are sometimes longer sequences that come through
	// as "\x1b[1;3" + some alpha
	if l >= 6 && b[2] == '1' && b[3] == ';' && b[4] == '3' {
		b = append([]byte{0x1b, '['}, b[5:]...)
		l -= 3
		runeLen = 3
		alt = KeyAlt
	}

	// ...and sometimes they're "\x1b[", some num, ";3~"
	if l >= 6 && b[3] == ';' && b[4] == '3' && b[5] == '~' {
		b = append([]byte{0x1b, '[', b[2]}, b[6:]...)
		l -= 3
		runeLen = 3
		alt = KeyAlt
	}

	// Since the buffer may have been manipulated, we re-check that we have 3+
	// characters left
	if l < 3 {
		return keyUnknown(b, force)
	}

	// From here on, all known return values must be at least 3 characters
	runeLen += 3
	switch b[2] {
	case 'A':
		return KeyUp + alt, runeLen
	case 'B':
		return KeyDown + alt, runeLen
	case 'C':
		return KeyRight + alt, runeLen
	case 'D':
		return KeyLeft + alt, runeLen
	case 'H':
		return KeyHome + alt, runeLen
	case 'F':
		return KeyEnd + alt, runeLen
	}

	if l < 4 {
		return keyUnknown(b, force)
	}
	runeLen++

	// NOTE: these appear to be escape sequences I see in tmux, but some don't
	// actually seem to happen on a "direct" terminal!
	if b[3] == '~' {
		switch b[2] {
		case '1':
			return KeyHome + alt, runeLen
		case '2':
			return KeyInsert + alt, runeLen
		case '3':
			return KeyDelete + alt, runeLen
		case '4':
			return KeyEnd + alt, runeLen
		case '5':
			return KeyPgUp + alt, runeLen
		case '6':
			return KeyPgDn + alt, runeLen
		}
	}

	if l < 6 {
		return keyUnknown(b, force)
	}
	runeLen += 2

	if len(b) >= 6 && bytes.Equal(b[:6], pasteEnd) {
		return KeyPasteEnd, runeLen
	}

	if len(b) >= 6 && bytes.Equal(b[:6], pasteStart) {
		return KeyPasteStart, runeLen
	}

	return keyUnknown(b, force)
}

// keyUnknown attempts to parse the unknown key and return its size.  If the
// key can't be figured out, it returns a RuneError.
func keyUnknown(b []byte, force bool) (rune, int) {
	for i, c := range b[0:] {
		// It's not clear how to find the end of a sequence without knowing them
		// all, but it seems that [a-zA-Z~] only appears at the end of a sequence
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '~' {
			return KeyUnknown, i + 1
		}
	}

	if force {
		return utf8.RuneError, len(b)
	}

	return utf8.RuneError, 0
}

func isPrintable(key rune) bool {
	isInSurrogateArea := key >= 0xd800 && key <= 0xdbff
	return key >= 32 && !isInSurrogateArea
}
