package terminal

// Giant list of key constants.  Everything above KeyUnknown matches an actual
// ASCII key value.  After that, we have various pseudo-keys in order to
// represent complex byte sequences that correspond to keys like Page up, Right
// arrow, etc.
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

