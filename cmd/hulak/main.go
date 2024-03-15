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
// something like this
func FinalEnvMap() error {
	err := SetEnvironment()
	if err != nil {
		return fmt.Errorf("error while setting environment: %v", err)
	}

	envVal, ok := os.LookupEnv("ENV_KEY") // Make sure to use the correct envKey variable
	if !ok {
		return fmt.Errorf("error while looking up environment variable")
	}

	envFileName := envVal + ".env"

	// Always start by loading global environment variables
	globalEnvFileName := "global.env"
	globalFilePath, err := utils.CreateFilePath(globalEnvFileName)
	if err != nil {
		return fmt.Errorf("error during creating %v: %v", globalEnvFileName, err)
	}

	// Load global environment variables
	err = LoadEnvVars(globalFilePath)
	if err != nil {
		return fmt.Errorf("error while loading %v: %v", globalFilePath, err)
	}

	// If envFileName is not "global.env", load the user-provided environment variables
	if envFileName != globalEnvFileName {
		completeFilePath, err := utils.CreateFilePath(envFileName)
		if err != nil {
			return fmt.Errorf("error during creating %v: %v", envFileName, err)
		}

		// Load user-provided environment variables
		err = LoadEnvVars(completeFilePath)
		if err != nil {
			return fmt.Errorf("error while loading %v: %v", completeFilePath, err)
		}
	}

	return nil
}
- Comment in the FinalEnvMap with code above
Tests
Check SubstitueVariables and make sure the substitution is working as expected
Make sure no two items have the same key or is replaced by the later key/value pair
*/
