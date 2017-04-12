package main

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Nerdmaster/terminal"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	var number = rand.Intn(10) + 1

	oldState, err := terminal.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	defer terminal.Restore(0, oldState)

	// Add a key logger callback
	var keyLogger []rune
	var p = terminal.NewPrompt(os.Stdin, os.Stdout, "Guess a number: ")
	p.AfterKeypress = func(e *terminal.KeyEvent) {
		keyLogger = append(keyLogger, e.Key)
	}

	fmt.Print("I'm thinking of a number from 1-10.  Try to guess it!\r\n")
	fmt.Print("(Type 'QUIT' at any time to exit)\r\n\r\n")

	for {
		var guess, err = p.ReadLine()
		if err != nil {
			fmt.Print("\r\nOh no, I got an error!\r\n")
			fmt.Printf("%s\r\n", err)
			break
		}

		if strings.ToLower(guess) == "quit" {
			fmt.Print("Quitter!\r\n")
			break
		}
		var g int
		g, err = strconv.Atoi(guess)
		if err != nil {
			fmt.Printf("Your entry, '%s', doesn't appear to be a valid number\r\n", guess)
			continue
		}

		if g < 1 || g > 10 {
			fmt.Print("Please enter a number in the range of 1 to 10, inclusive\r\n")
			continue
		}

		if g == number {
			fmt.Print("Correct!!\r\n")
			break
		}

		fmt.Printf("%d is WRONG!  Try again!\r\n", g)
	}

	fmt.Printf("Your entire list of keystrokes was %d runes long\r\n", len(keyLogger))
	fmt.Printf("The keystrokes were %#v\r\n", string(keyLogger))
}
