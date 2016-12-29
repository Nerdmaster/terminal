package main

import (
	"fmt"
	"io"
	"os"

	"github.com/Nerdmaster/terminal"
)

var keyText = map[rune]string{
	terminal.KeyCtrlA:        "KeyCtrlA",
	terminal.KeyCtrlB:        "KeyCtrlB",
	terminal.KeyCtrlC:        "KeyCtrlC",
	terminal.KeyCtrlD:        "KeyCtrlD",
	terminal.KeyCtrlE:        "KeyCtrlE",
	terminal.KeyCtrlF:        "KeyCtrlF",
	terminal.KeyCtrlG:        "KeyCtrlG",
	terminal.KeyCtrlH:        "KeyCtrlH",
	terminal.KeyCtrlI:        "KeyCtrlI",
	terminal.KeyCtrlJ:        "KeyCtrlJ",
	terminal.KeyCtrlK:        "KeyCtrlK",
	terminal.KeyCtrlL:        "KeyCtrlL",
	terminal.KeyCtrlN:        "KeyCtrlN",
	terminal.KeyCtrlO:        "KeyCtrlO",
	terminal.KeyCtrlP:        "KeyCtrlP",
	terminal.KeyCtrlQ:        "KeyCtrlQ",
	terminal.KeyCtrlR:        "KeyCtrlR",
	terminal.KeyCtrlS:        "KeyCtrlS",
	terminal.KeyCtrlT:        "KeyCtrlT",
	terminal.KeyCtrlU:        "KeyCtrlU",
	terminal.KeyCtrlV:        "KeyCtrlV",
	terminal.KeyCtrlW:        "KeyCtrlW",
	terminal.KeyCtrlX:        "KeyCtrlX",
	terminal.KeyCtrlY:        "KeyCtrlY",
	terminal.KeyCtrlZ:        "KeyCtrlZ",
	terminal.KeyEscape:       "KeyEscape",
	terminal.KeyLeftBracket:  "KeyLeftBracket",
	terminal.KeyRightBracket: "KeyRightBracket",
	terminal.KeyEnter:        "KeyEnter",
	terminal.KeyBackspace:    "KeyBackspace",
	terminal.KeyUnknown:      "KeyUnknown",
	terminal.KeyUp:           "KeyUp",
	terminal.KeyDown:         "KeyDown",
	terminal.KeyLeft:         "KeyLeft",
	terminal.KeyRight:        "KeyRight",
	terminal.KeyHome:         "KeyHome",
	terminal.KeyEnd:          "KeyEnd",
	terminal.KeyPasteStart:   "KeyPasteStart",
	terminal.KeyPasteEnd:     "KeyPasteEnd",
	terminal.KeyInsert:       "KeyInsert",
	terminal.KeyDelete:       "KeyDelete",
	terminal.KeyPgUp:         "KeyPgUp",
	terminal.KeyPgDn:         "KeyPgDn",
	terminal.KeyPause:        "KeyPause",
	terminal.KeyF1:           "KeyF1",
	terminal.KeyF2:           "KeyF2",
	terminal.KeyF3:           "KeyF3",
	terminal.KeyF4:           "KeyF4",
	terminal.KeyF5:           "KeyF5",
	terminal.KeyF6:           "KeyF6",
	terminal.KeyF7:           "KeyF7",
	terminal.KeyF8:           "KeyF8",
	terminal.KeyF9:           "KeyF9",
	terminal.KeyF10:          "KeyF10",
	terminal.KeyF11:          "KeyF11",
	terminal.KeyF12:          "KeyF12",
}

var done bool
var r *terminal.KeyReader

func printKey(kp terminal.Keypress) {
	if kp.Key == terminal.KeyCtrlF {
		r.ForceParse = !r.ForceParse
		fmt.Printf("  [ForceParse: %#v]\r\n", r.ForceParse)
	}

	if kp.Key == terminal.KeyCtrlC {
		fmt.Print("CTRL+C pressed; terminating\r\n")
		done = true
		return
	}

	var keyString = keyText[kp.Key]
	fmt.Printf("Key: %U [name: %s] [mod: %s] [raw: %#v (%#v)] [size: %d]\r\n",
		kp.Key, keyString, kp.Modifier.String(), string(kp.Raw), kp.Raw, kp.Size)
}

func main() {
	// It's possible this will error if one does `echo "foo" | bin/keyreport`, so
	// in order to test more interesting scenarios, we let errors through.
	oldState, err := terminal.MakeRaw(int(os.Stdin.Fd()))
	if err == nil {
		defer terminal.Restore(int(os.Stdin.Fd()), oldState)
	}

	r = terminal.NewKeyReader(os.Stdin)
	readInput()
}

func readInput() {
	for !done {
		var kp, err = r.ReadKeypress()
		if err != nil {
			if err == io.EOF {
				fmt.Println("EOF encountered; exiting")
				return
			}
			fmt.Printf("ERROR: %s", err)
			done = true
		}
		printKey(kp)
	}
}
