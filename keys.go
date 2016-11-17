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
	KeyAltLeft
	KeyAltRight
	KeyHome
	KeyEnd
	KeyPasteStart
	KeyPasteEnd
	KeyInsert
	KeyDelete
	KeyPgUp
	KeyPgDn
)

var pasteStart = []byte{KeyEscape, '[', '2', '0', '0', '~'}
var pasteEnd = []byte{KeyEscape, '[', '2', '0', '1', '~'}

// bytesToKey tries to parse a key sequence from b. If successful, it returns
// the key and the remainder of the input. Otherwise it returns utf8.RuneError.
func bytesToKey(b []byte, pasteActive bool) (rune, []byte) {
	if len(b) == 0 {
		return utf8.RuneError, nil
	}

	// Handle ctrl keys early (DecodeRune can do this, but it's a bit quicker to
	// handle this first (I'm assuming so, anyway, since the original
	// implementation did this first)
	if b[0] < KeyEscape {
		return rune(b[0]), b[1:]
	}

	if b[0] != KeyEscape {
		if !utf8.FullRune(b) {
			return utf8.RuneError, b
		}
		r, l := utf8.DecodeRune(b)
		return r, b[l:]
	}

	if pasteActive {
		if len(b) >= 6 && bytes.Equal(b[:6], pasteEnd) {
			return KeyPasteEnd, b[6:]
		}
		return keyUnknown(b)
	}

	if len(b) >= 3 && b[0] == KeyEscape && b[1] == '[' {
		switch b[2] {
		case 'A':
			return KeyUp, b[3:]
		case 'B':
			return KeyDown, b[3:]
		case 'C':
			return KeyRight, b[3:]
		case 'D':
			return KeyLeft, b[3:]
		case 'H':
			return KeyHome, b[3:]
		case 'F':
			return KeyEnd, b[3:]
		}

		if len(b) >= 4 && b[3] == '~' {
			switch b[3] {
			case '1':
				return KeyHome, b[4:]
			case '2':
				return KeyInsert, b[4:]
			case '3':
				return KeyDelete, b[4:]
			case '4':
				return KeyEnd, b[4:]
			case '5':
				return KeyPgUp, b[4:]
			case '6':
				return KeyPgDn, b[4:]
			}
		}
	}

	if len(b) >= 6 && b[0] == KeyEscape && b[1] == '[' && b[2] == '1' && b[3] == ';' && b[4] == '3' {
		switch b[5] {
		case 'C':
			return KeyAltRight, b[6:]
		case 'D':
			return KeyAltLeft, b[6:]
		}
	}

	if len(b) >= 6 && bytes.Equal(b[:6], pasteStart) {
		return KeyPasteStart, b[6:]
	}

	return keyUnknown(b)
}

// keyUnknown attempts to parse the unknown key and return the next part of the
// byte sequence.  If the key can't be figured out, it returns a RuneError.
func keyUnknown(b []byte) (rune, []byte) {
	for i, c := range b[0:] {
		// It's not clear how to find the end of a sequence without knowing them
		// all, but it seems that [a-zA-Z~] only appears at the end of a sequence
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '~' {
			return KeyUnknown, b[i+1:]
		}
	}

	return utf8.RuneError, b
}

func isPrintable(key rune) bool {
	isInSurrogateArea := key >= 0xd800 && key <= 0xdbff
	return key >= 32 && !isInSurrogateArea
}

