package utils

import (
	"fmt"
	"log"
	"os"
)

// Creates an error message that optionally includes an additional error.
// If an error is provided, it formats the message with the error appended.
// The returned error is colored for console output.
func ColorError(errMsg string, errs ...error) error {
	fullMsg := errMsg
	for _, err := range errs {
		if err != nil {
			fullMsg += ": " + err.Error()
		}
	}
	return fmt.Errorf("%sError: %s%s", Red, fullMsg, ColorReset)
}

// Success Message
func PrintGreen(msg string) {
	log.Printf("%s%s%s\n", Green, msg, ColorReset)
}

// Inform or Warn the user
func PrintWarning(msg string) {
	log.Printf("%s%s%s\n", Yellow, msg, ColorReset)
}

// Used mostly for errors
func PrintRed(msg string) {
	log.Printf("%s%s%s\n", Red, msg, ColorReset)
}

// Print message in Red and os.Exit(1)
func PanicRedAndExit(msg string, args ...any) {
	log.Printf("\n%s%s%s\n", Red, fmt.Sprintf(msg, args...), ColorReset)
	os.Exit(1)
}

/*
// assuming that ixiai is not a variable defined in the .env files
resolved1, err := envparser.SubstitueVariables("myNameIs={{.ixiai}}")
	if err != nil {
		utils.PrintError(err)
	}
	fmt.Println("Hello", resolved1)
*/
