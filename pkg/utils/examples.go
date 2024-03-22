package utils

import (
	"flag"
	"fmt"
)

// example functions and brain storming

func HowToUseFlags() {
	/* Copied from https://gobyexample.com/command-line-flags
	go build -o hulak  cmd/hulak/main.go
	*/
	wordPtr := flag.String("word", "foo", "a string")
	numbPtr := flag.Int("numb", 42, "an int")
	forkPtr := flag.Bool("fork", false, "a bool")

	var svar string
	flag.StringVar(&svar, "svar", "bar", "a string var")

	flag.Parse()

	fmt.Println("word:", *wordPtr)
	fmt.Println("numb:", *numbPtr)
	fmt.Println("fork:", *forkPtr)
	fmt.Println("svar:", svar)
	fmt.Println("tail:", flag.Args())
	// these flags should be in the help docs
}
