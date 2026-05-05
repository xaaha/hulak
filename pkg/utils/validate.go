package utils

import (
	"fmt"
	"regexp"
)

// MaxEnvNameLen is the upper bound on environment-name length.
// Chosen to fit comfortably in a single line of human output, leave headroom for
// JSON keys, and stay well under filesystem path limits.
const MaxEnvNameLen = 64

// envNamePattern matches valid hulak secretsironment names: ASCII letters, digits,
// underscore, and hyphen. No spaces, no dots, no slashes, no shell metacharacters.
var envNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// ValidateEnvName reports whether name is a syntactically valid environment
// identifier. Used by every CLI site that accepts an --env value, by migration
// when sanitizing legacy filenames, and by the vault store on read.
//
// Names starting with `_` are reserved for the store's internal metadata
// (e.g. _version). User-defined env names must not start with an underscore.
func ValidateEnvName(name string) error {
	if name == "" {
		return fmt.Errorf("environment name cannot be empty")
	}
	if len(name) > MaxEnvNameLen {
		return fmt.Errorf(
			"environment name %q is too long (%d chars, max %d)",
			name, len(name), MaxEnvNameLen,
		)
	}
	if name[0] == '_' {
		return fmt.Errorf(
			"environment name %q is invalid: names starting with '_' are reserved for internal use",
			name,
		)
	}
	if !envNamePattern.MatchString(name) {
		return fmt.Errorf(
			"environment name %q is invalid: only letters, digits, underscore, and hyphen are allowed",
			name,
		)
	}
	return nil
}
