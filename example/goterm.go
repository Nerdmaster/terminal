package main

import (
	"os"
	"strings"
	"time"

	"github.com/Nerdmaster/terminal"
	"github.com/buger/goterm"
)

var done bool
var userInput string
var p *terminal.AbsPrompt
var cmdBox, sizeBox *goterm.Box

func main() {
	oldState, err := terminal.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	defer terminal.Restore(0, oldState)

	cmdBox = goterm.NewBox(90, 3, 0)
	sizeBox = goterm.NewBox(20, 10, 0)
	sizeBox.Write([]byte("I AM A REALLY COOL BOX"))

	p = terminal.NewAbsPrompt(os.Stdin, os.Stdout, "Command: ")
	p.SetLocation(3, 1)
	p.MaxLineLength = 70

	go readInput()
	go printOutput()

	for done == false {
		time.Sleep(time.Millisecond * 10)
	}

	goterm.Clear()
	goterm.Flush()
}

func readInput() {
	for {
		command, err := p.ReadLine()
		command = strings.ToLower(command)
		if command == "quit" {
			done = true
			return
		}
		if command == "bigger" {
			sizeBox.Width++
		}
		if command == "smaller" {
			sizeBox.Width--
		}
		if err != nil {
			done = true
			return
		}
	}
}

func printOutput() {
	goterm.Clear()
	goterm.Print(goterm.MoveTo(cmdBox.String(), 1, 1))
	goterm.Flush()

	for {
		// Redraw the goterm stuff which isn't static
		goterm.Print(goterm.MoveTo(sizeBox.String(), 1, 20))
		goterm.Flush()

		// Print any changes to user input since last tick
		p.WriteChangesNoCursor()
		p.PrintCursorMovement()

		time.Sleep(time.Millisecond * 50)
	}
}
