package terminal

import (
	"io"
	"time"
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
// stream, usually stdin or an ssh socket.  Stores raw bytes in a buffer so
// that if many keys are read at once, they can still be parsed individually.
type KeyReader struct {
	input io.Reader

	// If ForceParse is true, the reader won't wait for certain sequences to
	// finish, which allows for things like ESC or Alt-left-bracket to be
	// detected properly
	ForceParse bool

	// remainder contains the remainder of any partial key sequences after
	// a read. It aliases into inBuf.
	remainder []byte
	inBuf     [256]byte

	// firstRead tells us when a sequence started so we can properly "time out" a
	// previous sequence instead of keep adding to it indefinitely
	firstRead time.Time

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

// ReadKeypress reads the next key sequence, returning a Keypress object and
// possibly an error if the input stream can't be read for some reason.  This
// will block if the buffer has no more data, which would obviously require a
// direct Read call on the underlying io.Reader.
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

	var remLen = len(r.remainder)
	if r.midRune || remLen == 0 {
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

		// If we had previous data, but it's been long enough since the first read
		// in that sequence (>250ms), we force-parse the previous sequence and
		// return it.  We have a one-key "lag", but this allows things like Escape
		// + X to be handled properly and separately even without ForceParse.
		if remLen > 0 {
			if time.Since(r.firstRead) > time.Millisecond*250 {
				key, i, mod := ParseKey(r.remainder[:remLen], true)
				var kp = Keypress{Key: key, Size: i, Modifier: mod, Raw: r.remainder[:i]}
				r.offset = i
				return kp, nil
			}
		} else {
			// We can safely assume this is the first read
			r.firstRead = time.Now()
		}
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

func isPrintable(key rune) bool {
	isInSurrogateArea := key >= 0xd800 && key <= 0xdbff
	return key >= 32 && !isInSurrogateArea
}
