package terminal_test

import (
	"fmt"
	"os"
	"strings"

	"github.com/Nerdmaster/terminal"
)

// Example of a very basic use of the Prompt type, which is probably the
// simplest type available
func Example() {
	// Put terminal in raw mode; this is almost always going to be required for
	// local terminals, but not necessary when connecting to an ssh terminal
	oldState, err := terminal.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	defer terminal.Restore(0, oldState)

	// This is a simple example, so we just use Stdin and Stdout; but any reader
	// and writer will work.  Note that the prompt can contain ANSI colors.
	var p = terminal.NewPrompt(os.Stdin, os.Stdout, "\x1b[34;1mCommand\x1b[0m: ")

	// Make the input scroll at 40 characters
	p.InputWidth = 40

	// Loop forever until we get an error (typically EOF from user pressing
	// CTRL+D) or the "quit" command is entered.  We echo each command unless the
	// user turns echoing off.
	var echo = true
	for {
		var cmd, err = p.ReadLine()
		if err != nil {
			fmt.Printf("%s\r\n", err)
			break
		}

		if strings.ToLower(cmd) == "quit" {
			fmt.Print("Quitter!\r\n")
			break
		}

		if strings.ToLower(cmd) == "echo on" {
			echo = true
		}
		if strings.ToLower(cmd) == "echo off" {
			echo = false
		}

		if echo {
			fmt.Printf("%#v\r\n", cmd)
		}
	}
}
