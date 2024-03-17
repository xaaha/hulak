package main

import (
	"flag"
	"fmt"
)

func main() {
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

	// Initialize the project
	InitializeProject()

	// fmt.Println("Default Environment value:", os.Getenv("hulakEnv"))
}

/*
Tests
- Default Value is global.env
- Custom -env flag is used
- Complete & Final Map can be printed as json
- SubstitueVariables
 - SubstitueVariables and make sure the substitution is working as expected
- Implement Collection  Global > defined > Collection
- Find  a way to document the falg used
*/
