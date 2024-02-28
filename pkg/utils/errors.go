package utils

import (
	"fmt"
	"os"
)

const (
	colorRed   = "\033[31m"
	colorReset = "\033[0m"
)

// Prints the error message in red color and exits the program
func PrintError(err error) {
	fmt.Printf("%sError: %s%s\n", colorRed, err, colorReset)
	os.Exit(1)
}
