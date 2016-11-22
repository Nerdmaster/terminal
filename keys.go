package terminal

import (
	"bytes"
	"unicode/utf8"
)

const (
	KeyCtrlA = 1 + iota
	KeyCtrlB
	KeyCtrlC
	KeyCtrlD
	KeyCtrlE
	KeyCtrlF
	KeyCtrlG
	KeyCtrlH
	KeyCtrlI
	KeyCtrlJ
	KeyCtrlK
	KeyCtrlL
	KeyCtrlM
	KeyCtrlN
	KeyCtrlO
	KeyCtrlP
	KeyCtrlQ
	KeyCtrlR
	KeyCtrlS
	KeyCtrlT
	KeyCtrlU
	KeyCtrlV
	KeyCtrlW
	KeyCtrlX
	KeyCtrlY
	KeyCtrlZ
	KeyEscape
	KeyEnter     = '\r'
	KeyBackspace = 127
	KeyUnknown   = 0xd800 /* UTF-16 surrogate area */ + iota
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeyHome
	KeyEnd
	KeyPasteStart
	KeyPasteEnd
	KeyInsert
	KeyDelete
	KeyPgUp
	KeyPgDn

	KeyAlt           = 0x0100
	KeyAltUnknown    = KeyAlt + KeyUnknown
	KeyAltUp         = KeyAlt + KeyUp
	KeyAltDown       = KeyAlt + KeyDown
	KeyAltLeft       = KeyAlt + KeyLeft
	KeyAltRight      = KeyAlt + KeyRight
	KeyAltHome       = KeyAlt + KeyHome
	KeyAltEnd        = KeyAlt + KeyEnd
	KeyAltPasteStart = KeyAlt + KeyPasteStart
	KeyAltPasteEnd   = KeyAlt + KeyPasteEnd
	KeyAltInsert     = KeyAlt + KeyInsert
	KeyAltDelete     = KeyAlt + KeyDelete
	KeyAltPgUp       = KeyAlt + KeyPgUp
	KeyAltPgDn       = KeyAlt + KeyPgDn
)

var pasteStart = []byte{KeyEscape, '[', '2', '0', '0', '~'}
var pasteEnd = []byte{KeyEscape, '[', '2', '0', '1', '~'}

// ParseKey tries to parse a key sequence from b. If successful, it returns the
// key and the length in bytes of that key. Otherwise it returns utf8.RuneError, 0.
func ParseKey(b []byte) (rune, int) {
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
			return utf8.RuneError, 0
		}
		return utf8.DecodeRune(b)
	}

	// From the above test we know the first key is escape.  Everything else we
	// know how to handle is at least 3 bytes
	if l < 3 {
		return keyUnknown(b)
	}

	// Alt keys, at least from tmux sessions, come through as 0x1b, 0x1b, ...
	var alt rune
	if b[1] == 0x1b {
		b = b[1:]
		l--
		runeLen = 1
		alt = KeyAlt
	}

	// If it wasn't a tmux alt key, it has to be escape followed by a left bracket
	if b[1] != '[' {
		return keyUnknown(b)
	}

	// Local terminal alt keys seem to be longer sequences that come through as
	// 0x1b, "[1;3", ...
	if l >= 6 && b[2] == '1' && b[3] == ';' && b[4] == '3' {
		b = append([]byte{0x1b, '['}, b[5:]...)
		l -= 2
		runeLen = 2
		alt = KeyAlt
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
		return keyUnknown(b)
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
		return keyUnknown(b)
	}
	runeLen += 2

	if len(b) >= 6 && bytes.Equal(b[:6], pasteEnd) {
		return KeyPasteEnd, runeLen
	}

	if len(b) >= 6 && bytes.Equal(b[:6], pasteStart) {
		return KeyPasteStart, runeLen
	}

	return keyUnknown(b)
}

// keyUnknown attempts to parse the unknown key and return its size.  If the
// key can't be figured out, it returns a RuneError.
func keyUnknown(b []byte) (rune, int) {
	for i, c := range b[0:] {
		// It's not clear how to find the end of a sequence without knowing them
		// all, but it seems that [a-zA-Z~] only appears at the end of a sequence
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '~' {
			return KeyUnknown, i + 1
		}
	}

	return utf8.RuneError, 0
}

func isPrintable(key rune) bool {
	isInSurrogateArea := key >= 0xd800 && key <= 0xdbff
	return key >= 32 && !isInSurrogateArea
}
