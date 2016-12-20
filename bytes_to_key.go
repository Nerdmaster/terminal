package terminal

import (
	"bytes"
	"unicode/utf8"
)

// ParseKey tries to parse a key sequence from b. If successful, it returns the
// key and the length in bytes of that key. Otherwise it returns
// utf8.RuneError, 0.  If force is true, partial sequences will be returned
// with a best-effort approach to making them meaningful, rather than flagging
// the caller that there may be more bytes needed.  This is useful for
// gathering special keys like escape, which otherwise hold up the key reader
// waiting for the rest of a nonexistent sequence.
func ParseKey(b []byte, force bool) (r rune, rl int, mod KeyModifier) {
	var l = len(b)
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
		rl = 1
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
		rl = 3
		mod = ModAlt
	}

	// ...and sometimes they're "\x1b[", some num, ";3~"
	if l >= 6 && b[3] == ';' && b[4] == '3' && b[5] == '~' {
		b = append([]byte{0x1b, '[', b[2]}, b[5:]...)
		l -= 2
		rl = 2
		mod = ModAlt
	}

	// Since the buffer may have been manipulated, we re-check that we have 3+
	// characters left
	if l < 3 {
		return keyUnknown(b, force, mod)
	}

	// From here on, all known return values must be at least 3 characters
	rl += 3
	switch b[2] {
	case 'A':
		return KeyUp, rl, mod
	case 'B':
		return KeyDown, rl, mod
	case 'C':
		return KeyRight, rl, mod
	case 'D':
		return KeyLeft, rl, mod
	case 'H':
		return KeyHome, rl, mod
	case 'F':
		return KeyEnd, rl, mod
	}

	if l < 4 {
		return keyUnknown(b, force, mod)
	}
	rl++

	// NOTE: these appear to be escape sequences I see in tmux, but some don't
	// actually seem to happen on a "direct" terminal!
	if b[3] == '~' {
		switch b[2] {
		case '1':
			return KeyHome, rl, mod
		case '2':
			return KeyInsert, rl, mod
		case '3':
			return KeyDelete, rl, mod
		case '4':
			return KeyEnd, rl, mod
		case '5':
			return KeyPgUp, rl, mod
		case '6':
			return KeyPgDn, rl, mod
		}
	}

	if l < 6 {
		return keyUnknown(b, force, mod)
	}
	rl += 2

	if len(b) >= 6 && bytes.Equal(b[:6], pasteEnd) {
		return KeyPasteEnd, rl, mod
	}

	if len(b) >= 6 && bytes.Equal(b[:6], pasteStart) {
		return KeyPasteStart, rl, mod
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

