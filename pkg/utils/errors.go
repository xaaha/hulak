package utils

import (
	"fmt"
	"os"
)

// Prints the error message in red color and exits the program
func PrintError(err error) {
	fmt.Printf("%sError: %s%s\n", Red, err, colorReset)
	os.Exit(1)
}

func PrintGreen(msg string) {
	fmt.Printf("%s%s%s\n", Green, msg, colorReset)
}

func PrintWarning(msg string) {
	fmt.Printf("%s%s%s\n", Yellow, msg, colorReset)
}

/*
// assuming that ixiai is not a variable defined in the .env files
resolved1, err := envparser.SubstitueVariables("myNameIs={{ixiai}}")
	if err != nil {
		utils.PrintError(err)
	}
	fmt.Println("Hello", resolved1)
*/
