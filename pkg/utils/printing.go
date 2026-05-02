package utils

import (
	"bufio"
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

// Stderr-routed printers: use these for diagnostics, status, warnings, and
// errors during commands whose stdout must stay clean. Stdout is reserved
// for actual program output (e.g. `hulak env get` captured via $(...) must
// return only the value). Each function colors ONLY the leading prefix —
// the message body stays plain text so it remains readable when redirected
// to a non-color terminal or a log file.

// PrintWarningStderr writes a "warning: <msg>" line to stderr.
// Prefix in yellow, message plain.
func PrintWarningStderr(msg string) {
	fmt.Fprintf(os.Stderr, "%swarning:%s %s\n", Yellow, ColorReset, msg)
}

// PrintErrorStderr writes an "error: <msg>" line to stderr.
// Prefix in red, message plain. Use for user-facing errors that have
// already been handled (don't print one of these AND return an error —
// pick one).
func PrintErrorStderr(msg string) {
	fmt.Fprintf(os.Stderr, "%serror:%s %s\n", Red, ColorReset, msg)
}

// PrintSuccessStderr writes a "✓ <msg>" line to stderr. Use for status
// confirmations on commands whose stdout must stay clean (env set, env
// delete, env edit). Checkmark in green, message plain.
func PrintSuccessStderr(msg string) {
	fmt.Fprintf(os.Stderr, "%s%s%s %s\n", Green, CheckMark, ColorReset, msg)
}

// PrintInfoStderr writes a plain informational line to stderr — no color,
// no prefix. Use for progress hints and "FYI" output that shouldn't
// pollute stdout but doesn't warrant warning/error decoration either.
func PrintInfoStderr(msg string) {
	fmt.Fprintln(os.Stderr, msg)
}

// PrintRed is used mostly for errors
func PrintRed(msg string) {
	fmt.Printf("%s%s%s\n", Red, msg, ColorReset)
}

// PrintInfo prints the info for the user in blue
func PrintInfo(msg string) {
	fmt.Printf("%s%s%s\n", Blue, msg, ColorReset)
}

// PanicRedAndExit prints a fatal error to STDERR (red prefix, plain message)
// and exits with code 1. Use sparingly — only for errors that must terminate
// the process and cannot reasonably be returned. Library code should return
// errors instead and let main() decide whether to call this.
func PanicRedAndExit(msg string, args ...any) {
	fmt.Fprintf(os.Stderr, "\n%serror:%s %s\n", Red, ColorReset, fmt.Sprintf(msg, args...))
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

// ConfirmAction prints prompt to stderr and reads a single line from stdin.
// Returns true for "y" or "yes" (case-insensitive), false otherwise.
// Returns false (not error) on EOF or non-interactive stdin.
func ConfirmAction(prompt string) (bool, error) {
	fmt.Fprint(os.Stderr, prompt)
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return false, scanner.Err()
	}
	answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
	return answer == "y" || answer == "yes", nil
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
