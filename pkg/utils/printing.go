package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

// ColorError Creates an error message that optionally includes an additional error.
// If an error is provided, it formats the message with the error appended.
// The returned error is colored for console output.
func ColorError(errMsg string, errs ...error) error {
	var fullMsg strings.Builder
	fullMsg.WriteString(errMsg)
	for _, err := range errs {
		if err != nil {
			fullMsg.WriteString(": " + err.Error())
		}
	}
	return fmt.Errorf("\n%s%s%s", Red, fullMsg.String(), ColorReset)
}

// PrintGreen Prints Success Message
func PrintGreen(msg string) {
	fmt.Printf("%s%s%s\n", Green, msg, ColorReset)
}

// PrintWarning Inform or Warn the user
func PrintWarning(msg string) {
	fmt.Printf("%s%s%s\n", Yellow, msg, ColorReset)
}

// PrintRed is used mostly for errors
func PrintRed(msg string) {
	fmt.Printf("%s%s%s\n", Red, msg, ColorReset)
}

// PrintInfo prints the info for the user in blue
func PrintInfo(msg string) {
	fmt.Printf("%s%s%s\n", Blue, msg, ColorReset)
}

// PanicRedAndExit Print message in Red and os.Exit(1)
func PanicRedAndExit(msg string, args ...any) {
	fmt.Printf("\n%s%s%s\n", Red, fmt.Sprintf(msg, args...), ColorReset)
	os.Exit(1)
}

// MarshalToJSON is basically JSON.stringify equivalent for go
func MarshalToJSON(value any) (any, error) {
	switch val := value.(type) {
	case string, bool, int, float64:
		return val, nil
	case nil:
		return nil, nil
	default:
		if arr, ok := value.([]any); ok {
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

// CommandHelp holds a command and its description
type CommandHelp struct {
	Command     string
	Description string
}

// CommandHelp holds a command and its description
func WriteCommandHelp(commands []*CommandHelp) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)
	for _, cmd := range commands {
		if _, err := fmt.Fprintf(w, "  %s\t- %s\n", cmd.Command, cmd.Description); err != nil {
			return err
		}
	}
	return w.Flush()
}

// HelpfulError formats a multi-line error with a title, a section heading, and a
// bullet list, then wraps it via ColorError. Use for user-facing errors that
// should suggest remediation steps (e.g. "no env files found" → list of fixes).
func HelpfulError(title, heading string, bullets []string) error {
	var b strings.Builder
	fmt.Fprintln(&b, title)
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "%s:\n", heading)
	for _, item := range bullets {
		fmt.Fprintf(&b, "  - %s\n", item)
	}
	return ColorError(strings.TrimRight(b.String(), "\n"))
}
