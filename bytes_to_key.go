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
	// Default to a rune error, since we use that in so many situations
	r = utf8.RuneError

	var l = len(b)
	if l == 0 {
		return
	}

	// Function keys F1-F4 are an even more ultra-super-special-case, because
	// they can get detected as alt+letter otherwise.  ARGH.
	if l > 2 && b[0] == 0x1b && b[1] == 'O' {
		var b2 = b[2]
		if l > 3 && b[2] == '1' {
			rl++
			b2 = b[3]
			mod |= ModMeta
		}
		switch b2 {
		case 'P':
			return KeyF1, rl + 3, mod
		case 'Q':
			return KeyF2, rl + 3, mod
		case 'R':
			return KeyF3, rl + 3, mod
		case 'S':
			return KeyF4, rl + 3, mod
		}
	}

	// Ultra-super-special-case handling for meta key
	if l > 3 && b[0] == 0x18 && b[1] == '@' && b[2] == 's' {
		b = b[3:]
		l -= 3
		rl += 3
		mod |= ModMeta
	}

	// Super-special-case handling for alt+esc and alt+left-bracket: these two
	// sequences are often just prefixes of other sequences, so when force is
	// true, if we have these and nothing else, we return immediately
	if l == 2 && force && b[0] == 0x1b {
		if b[1] == 0x1b {
			return KeyEscape, rl + 2, mod | ModAlt
		}
		if b[1] == '[' {
			return KeyLeftBracket, rl + 2, mod | ModAlt
		}
	}

	// Special case: some alt keys are "0x1b..." and need to be detected early
	if l > 1 && b[0] == 0x1b && b[1] != '[' {
		b = b[1:]
		l--
		rl++
		mod |= ModAlt
	}

	// Handle ctrl keys next.  DecodeRune can do this, but it's a bit quicker to
	// handle this first (I'm assuming so, anyway, since the original
	// implementation did this first)
	if b[0] < KeyEscape {
		return rune(b[0]), rl + 1, mod
	}

	if b[0] != KeyEscape {
		if !utf8.FullRune(b) {
			if force {
				rl += len(b)
				return
			}
			return
		}
		var r, nrl = utf8.DecodeRune(b)
		return r, rl + nrl, mod
	}

	// From the above test we know the first key is escape.  If that's all we
	// have, we are *probably* missing some bytes... but maybe not.
	if l == 1 {
		if force {
			return KeyEscape, rl + 1, mod
		}
		return keyUnknown(b, rl, force, mod)
	}

	// Everything else we know how to handle is at least 3 bytes
	if l < 3 {
		if force {
			rl += len(b)
			return
		}
		return keyUnknown(b, rl, force, mod)
	}

	// All sequences we know how to handle from here on start with "\x1b["
	if b[1] != '[' {
		return keyUnknown(b, rl, force, mod)
	}

	// Local terminal alt keys are sometimes longer sequences that come through
	// as "\x1b[1;3" + some alpha
	if l >= 6 && b[2] == '1' && b[3] == ';' && b[4] == '3' {
		b = append([]byte{0x1b, '['}, b[5:]...)
		l -= 3
		rl += 3
		mod |= ModAlt
	}

	// ...and sometimes they're "\x1b[", some num, ";3~"
	if l >= 6 && b[3] == ';' && b[4] == '3' && b[5] == '~' {
		b = append([]byte{0x1b, '[', b[2]}, b[5:]...)
		l -= 2
		rl += 2
		mod |= ModAlt
	}

	// Since the buffer may have been manipulated, we re-check that we have 3+
	// characters left
	if l < 3 {
		return keyUnknown(b, rl, force, mod)
	}

	// From here on, all known return values must be at least 3 characters
	switch b[2] {
	case 'A':
		return KeyUp, rl + 3, mod
	case 'B':
		return KeyDown, rl + 3, mod
	case 'C':
		return KeyRight, rl + 3, mod
	case 'D':
		return KeyLeft, rl + 3, mod
	case 'H':
		return KeyHome, rl + 3, mod
	case 'F':
		return KeyEnd, rl + 3, mod
	case 'P':
		return KeyPause, rl + 3, mod
	}

	if l < 4 {
		return keyUnknown(b, rl, force, mod)
	}

	// NOTE: these appear to be escape sequences I see in tmux, but some don't
	// actually seem to happen on a "direct" terminal!
	if b[3] == '~' {
		switch b[2] {
		case '1':
			return KeyHome, rl + 4, mod
		case '2':
			return KeyInsert, rl + 4, mod
		case '3':
			return KeyDelete, rl + 4, mod
		case '4':
			return KeyEnd, rl + 4, mod
		case '5':
			return KeyPgUp, rl + 4, mod
		case '6':
			return KeyPgDn, rl + 4, mod
		}
	}

	// "Raw terminal" function keys (VMWare non-gui debian)
	if b[2] == '[' {
		switch b[3] {
		case 'A':
			return KeyF1, rl + 4, mod
		case 'B':
			return KeyF2, rl + 4, mod
		case 'C':
			return KeyF3, rl + 4, mod
		case 'D':
			return KeyF4, rl + 4, mod
		case 'E':
			return KeyF5, rl + 4, mod
		}
	}

	if l < 5 {
		return keyUnknown(b, rl, force, mod)
	}

	// Meta + Function keys can be handled with a tiny bit of magic
	if len(b) > 6 && b[4] == ';' && b[5] == '1' && b[6] == '~' {
		b = append(b[:4], b[6:]...)
		l -= 2
		rl += 2
		mod |= ModMeta
	}

	// More function keys: these are shared across terminal and non-terminal
	// *except* F5, which is only seen this way when in a "non-raw" situation,
	// and F1-F4, which are only seen with these codes when sshed in from PuTTY
	if b[4] == '~' {
		switch b[2] {
		case '1':
			switch b[3] {
			case '1':
				return KeyF1, rl + 5, mod
			case '2':
				return KeyF2, rl + 5, mod
			case '3':
				return KeyF3, rl + 5, mod
			case '4':
				return KeyF4, rl + 5, mod
			case '5':
				return KeyF5, rl + 5, mod
			case '7':
				return KeyF6, rl + 5, mod
			case '8':
				return KeyF7, rl + 5, mod
			case '9':
				return KeyF8, rl + 5, mod
			}
		case '2':
			switch b[3] {
			case '0':
				return KeyF9, rl + 5, mod
			case '1':
				return KeyF10, rl + 5, mod
			case '3':
				return KeyF11, rl + 5, mod
			case '4':
				return KeyF12, rl + 5, mod
			}
		}
	}

	if l < 6 {
		return keyUnknown(b, rl, force, mod)
	}

	if len(b) >= 6 && bytes.Equal(b[:6], pasteEnd) {
		return KeyPasteEnd, rl + 6, mod
	}

	if len(b) >= 6 && bytes.Equal(b[:6], pasteStart) {
		return KeyPasteStart, rl + 6, mod
	}

	return keyUnknown(b, rl, force, mod)
}

// keyUnknown attempts to parse the unknown key and return its size.  If the
// key can't be figured out, it returns a RuneError.
func keyUnknown(b []byte, rl int, force bool, mod KeyModifier) (rune, int, KeyModifier) {
	// This is a hack, and it's guaranteed to not work in quite a few situations,
	// but there's really not much to be done when our buffer starts getting too
	// big.  Instead of trying to really make this awesome, we just throw away
	// the first character and call it an error.
	if len(b) > 8 && !force {
		return utf8.RuneError, 1, ModNone
	}

	for i, c := range b[0:] {
		// It's not clear how to find the end of a sequence without knowing them
		// all, but it seems that [a-zA-Z~] only appears at the end of a sequence
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '~' {
			return KeyUnknown, rl + i + 1, mod
		}
	}

	if force {
		return utf8.RuneError, rl + len(b), mod
	}

	return utf8.RuneError, 0, ModNone
}
