package main

import (
	"fmt"
	"os"

	"github.com/Nerdmaster/terminal"
)

var keyText = map[rune]string{
	terminal.KeyCtrlA: "KeyCtrlA",
	terminal.KeyCtrlB: "KeyCtrlB",
	terminal.KeyCtrlC: "KeyCtrlC",
	terminal.KeyCtrlD: "KeyCtrlD",
	terminal.KeyCtrlE: "KeyCtrlE",
	terminal.KeyCtrlF: "KeyCtrlF",
	terminal.KeyCtrlG: "KeyCtrlG",
	terminal.KeyCtrlH: "KeyCtrlH",
	terminal.KeyCtrlI: "KeyCtrlI",
	terminal.KeyCtrlJ: "KeyCtrlJ",
	terminal.KeyCtrlK: "KeyCtrlK",
	terminal.KeyCtrlL: "KeyCtrlL",
	terminal.KeyCtrlN: "KeyCtrlN",
	terminal.KeyCtrlO: "KeyCtrlO",
	terminal.KeyCtrlP: "KeyCtrlP",
	terminal.KeyCtrlQ: "KeyCtrlQ",
	terminal.KeyCtrlR: "KeyCtrlR",
	terminal.KeyCtrlS: "KeyCtrlS",
	terminal.KeyCtrlT: "KeyCtrlT",
	terminal.KeyCtrlU: "KeyCtrlU",
	terminal.KeyCtrlV: "KeyCtrlV",
	terminal.KeyCtrlW: "KeyCtrlW",
	terminal.KeyCtrlX: "KeyCtrlX",
	terminal.KeyCtrlY: "KeyCtrlY",
	terminal.KeyCtrlZ: "KeyCtrlZ",
	terminal.KeyEscape: "KeyEscape",
	terminal.KeyEnter: "KeyEnter",
	terminal.KeyBackspace: "KeyBackspace",
	terminal.KeyUnknown: "KeyUnknown",
	terminal.KeyUp: "KeyUp",
	terminal.KeyDown: "KeyDown",
	terminal.KeyLeft: "KeyLeft",
	terminal.KeyRight: "KeyRight",
	terminal.KeyHome: "KeyHome",
	terminal.KeyEnd: "KeyEnd",
	terminal.KeyPasteStart: "KeyPasteStart",
	terminal.KeyPasteEnd: "KeyPasteEnd",
	terminal.KeyInsert: "KeyInsert",
	terminal.KeyDelete: "KeyDelete",
	terminal.KeyPgUp: "KeyPgUp",
	terminal.KeyPgDn: "KeyPgDn",
	terminal.KeyAlt: "KeyAlt",
	terminal.KeyAltUnknown: "KeyAltUnknown",
	terminal.KeyAltUp: "KeyAltUp",
	terminal.KeyAltDown: "KeyAltDown",
	terminal.KeyAltLeft: "KeyAltLeft",
	terminal.KeyAltRight: "KeyAltRight",
	terminal.KeyAltHome: "KeyAltHome",
	terminal.KeyAltEnd: "KeyAltEnd",
	terminal.KeyAltPasteStart: "KeyAltPasteStart",
	terminal.KeyAltPasteEnd: "KeyAltPasteEnd",
	terminal.KeyAltInsert: "KeyAltInsert",
	terminal.KeyAltDelete: "KeyAltDelete",
	terminal.KeyAltPgUp: "KeyAltPgUp",
	terminal.KeyAltPgDn: "KeyAltPgDn",
}

var done bool
var r *terminal.KeyReader

func printKey(kp terminal.Keypress) {
	if kp.Key == terminal.KeyCtrlC {
		fmt.Print("CTRL+C pressed; terminating\r\n")
		done = true
		return
	}

	var keyString = keyText[kp.Key]
	fmt.Printf("Key: %U [name: %s] [raw: %#v (%#v)] [size: %d]\r\n", kp.Key, keyString, string(kp.Raw), kp.Raw, kp.Size)
}

func main() {
	oldState, err := terminal.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	defer terminal.Restore(0, oldState)

	r = terminal.NewKeyReader(os.Stdin)
	readInput()
}

func readInput() {
	for !done {
		var kp, err = r.ReadKeypress()
		if err != nil {
			fmt.Printf("ERROR: %s", err)
			done = true
		}
		printKey(kp)
	}
}
