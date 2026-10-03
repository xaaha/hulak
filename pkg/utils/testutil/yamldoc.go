package testutil

import "fmt"

// SingleDocError is the whole message utils.ValidateSingleYAMLDoc returns for
// name when a second document starts at line.
func SingleDocError(name string, line int) string {
	return fmt.Sprintf(
		"%s: a request file must be a single YAML document, but a second document starts at line %d",
		name, line,
	)
}
