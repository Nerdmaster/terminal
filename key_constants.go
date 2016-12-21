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
	KeyLeftBracket  = '['
	KeyRightBracket = ']'
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
)

var pasteStart = []byte{KeyEscape, '[', '2', '0', '0', '~'}
var pasteEnd = []byte{KeyEscape, '[', '2', '0', '1', '~'}

