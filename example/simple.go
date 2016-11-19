package main

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/Nerdmaster/terminal"
)

const CSI = "\x1b["
const ClearScreen = CSI + "2J" + CSI + ";H"

var done bool
var noise [][]rune
var nextNoise [][]rune
var userInput string
var t *terminal.Reader

var validRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890!@#$%^&*")

func onKeypress(e *terminal.KeyEvent) {
	// How can I ki amias without the  key?
	if e.Key == 'l' {
		e.IgnoreDefaultHandlers = true
	}

	if e.Key == terminal.KeyPgUp {
		e.Key = 'l'
	}

	if e.Key == terminal.KeyAltPgUp {
		e.Key = 'ģ'
	}

	if e.Key == terminal.KeyLeft {
		fmt.Print(ClearScreen)
		e.IgnoreDefaultHandlers = true
	}
}

func printAt(x, y int, output string) {
	fmt.Fprintf(os.Stdout, "%s%d;%dH%s", CSI, y, x, output)
}

func randomRune() rune {
	return validRunes[rand.Intn(len(validRunes))]
}

func initializeScreen() {
	// Clear everything
	fmt.Fprintf(os.Stdout, ClearScreen)

	// Print initial runes
	for y := 0; y < 10; y++ {
		printAt(1, y+1, string(noise[y]))
	}

	// Print prompt
	printAt(13, 4, "> ")
}

func setupNoise() {
	noise = make([][]rune, 10)
	for y := range noise {
		noise[y] = make([]rune, 100)
		for x := range noise[y] {
			if y == 3 && x > 10 && x < 90 {
				noise[y][x] = ' '
			} else {
				noise[y][x] = randomRune()
			}
		}
	}
	nextNoise = make([][]rune, 10)
	for y := range nextNoise {
		nextNoise[y] = make([]rune, 100)
		for x := range nextNoise[y] {
			nextNoise[y][x] = noise[y][x]
		}
	}
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	oldState, err := terminal.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	defer terminal.Restore(0, oldState)

	setupNoise()
	initializeScreen()

	t = terminal.NewReader(os.Stdin)
	t.MaxLineLength = 70
	t.OnKeypress = onKeypress
	go readInput()
	go printOutput()

	for done == false {
		x := rand.Intn(100)
		y := rand.Intn(10)
		nextNoise[y][x] = randomRune()
		time.Sleep(time.Millisecond * 10)
	}
}

func readInput() {
	for {
		command, err := t.ReadLine()
		if command == "quit" {
			done = true
			return
		}
		if err != nil {
			done = true
			return
		}
	}
}

func printOutput() {
	lastLine := ""
	lastPos := 0

	for {
		// Print any changes to noise since last tick
		for y := 0; y < 10; y++ {
			for x := 0; x < 100; x++ {
				if y == 3 && x > 10 && x < 90 {
					nextNoise[y][x] = noise[y][x]
				}
				if noise[y][x] != nextNoise[y][x] {
					printAt(x+1, y+1, string(nextNoise[y][x]))
					noise[y][x] = nextNoise[y][x]
				}
			}
		}

		// Print any changes to user input since last tick
		newLine, newPos := t.LinePos()

		if lastLine != newLine {
			toPrint := newLine
			if len(lastLine) > len(newLine) {
				toPrint += strings.Repeat(" ", len(lastLine) - len(newLine))
			}
			printAt(15, 4, toPrint)
		}

		if lastLine != newLine || lastPos != newPos {
			printAt(10, 20, strings.Repeat(" ", 100))
			printAt(1, 20, fmt.Sprintf("Current line: '%s'; position: %d", newLine, newPos))
			lastLine = newLine
			lastPos = newPos
		}

		printAt(15 + newPos, 4, "")

		time.Sleep(time.Millisecond * 50)
	}
}
