package main

import (
	"flag"
	"fmt"

	"github.com/xaaha/hulak/pkg/utils"
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
	utils.InitializeProject()

	// fmt.Println("Default Environment value:", os.Getenv("hulakEnv"))
}

/*
- Complete the SetDefaultEnv in the parser pkg
- Use the same function if the user gives in -env flag and argument
- Then put it in the initiallizer so that when I build the binary, it first
    - creates the env folders
    - creates the global.env file
    - should be able to read and parse the contents

Tests
Check SubstitueVariables and make sure the substitution is working as expected
Make sure no two items have the same key or is replaced by the later key/value pair
*/
