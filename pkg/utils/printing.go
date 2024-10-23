package utils

import (
	"fmt"
)

// ColorError creates an error message that optionally includes an additional error.
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
	fmt.Printf("%s%s%s\n", Green, msg, ColorReset)
}

// Inform or Warn the user
func PrintWarning(msg string) {
	fmt.Printf("%s%s%s\n", Yellow, msg, ColorReset)
}

/*
// assuming that ixiai is not a variable defined in the .env files
resolved1, err := envparser.SubstitueVariables("myNameIs={{ixiai}}")
	if err != nil {
		utils.PrintError(err)
	}
	fmt.Println("Hello", resolved1)
*/
