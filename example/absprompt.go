package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/Nerdmaster/terminal"
)

// CSI == Control Sequence Introducer, begins most ANSI commands
const CSI = "\x1b["
const ClearScreen = CSI + "2J" + CSI + ";H"

var done bool
var noise [][]rune
var nextNoise [][]rune
var userInput string
var p *terminal.AbsPrompt

var validRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890!@#$%^&*")

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
}

func setupNoise() {
	noise = make([][]rune, 10)
	for y := range noise {
		noise[y] = make([]rune, 100)
		for x := range noise[y] {
			if y == 3 && x > 9 && x < 90 {
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

	p = terminal.NewAbsPrompt(os.Stdin, os.Stdout, "> ")
	p.SetLocation(10, 3)
	p.MaxLineLength = 70
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
		command, err := p.ReadLine()
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
	for {
		// Print any changes to noise since last tick
		for y := 0; y < 10; y++ {
			for x := 0; x < 100; x++ {
				if y == 3 && x > 9 && x < 90 {
					nextNoise[y][x] = noise[y][x]
				}
				if noise[y][x] != nextNoise[y][x] {
					printAt(x+1, y+1, string(nextNoise[y][x]))
					noise[y][x] = nextNoise[y][x]
				}
			}
		}
		// Print any changes to user input since last tick
		p.WriteChangesNoCursor()
		p.PrintCursorMovement()

		time.Sleep(time.Millisecond * 50)
	}
}
