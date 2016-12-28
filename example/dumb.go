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

	var dt = terminal.Dumb(os.Stdin, os.Stdout)

	fmt.Print("I'm thinking of a number from 1-10.  Try to guess it!\r\n")
	fmt.Print("(Type 'QUIT' at any time to exit)\r\n\r\n")
	for {
		fmt.Print("Guess a number: ")
		guess, err := dt.ReadLine()
		if strings.ToLower(guess) == "quit" {
			fmt.Print("Quitter!\r\n")
			break
		}
		if err != nil {
			fmt.Print("Oh no, I got an error!\r\n")
			fmt.Printf("%s\r\n", err)
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
}
