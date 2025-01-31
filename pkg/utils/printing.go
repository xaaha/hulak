package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
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
	return fmt.Errorf("\n%sError: %s%s", Red, fullMsg, ColorReset)
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

// JSON.stringify equivalent for go
func MarshalToJSON(value interface{}) (interface{}, error) {
	switch val := value.(type) {
	case string, bool, int, float64:
		return val, nil
	case nil:
		return nil, nil
	default:
		if arr, ok := value.([]interface{}); ok {
			var jsonArray []string
			for _, item := range arr {
				jsonStr, err := json.Marshal(item)
				if err != nil {
					return "", err
				}
				jsonArray = append(jsonArray, string(jsonStr))
			}
			return fmt.Sprintf("[%s]", strings.Join(jsonArray, ",")), nil
		}
		jsonStr, err := json.Marshal(value)
		if err != nil {
			return "", err
		}
		return string(jsonStr), nil
	}
}
