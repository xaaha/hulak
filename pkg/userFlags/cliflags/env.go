// Package cliflags holds the reusable flag-registration helpers shared by
// every leaf command package. Commands import this package to wire common
// flags (--env, --out, --name, --show, --yes, --dry-run) with the same
// short/long alias conventions every time, rather than re-deriving the
// pair-binding pattern in each handler.
package cliflags

import "flag"

// RegisterEnv adds both --env and --environment aliases to a FlagSet,
// pointing to the same underlying variable, and returns a pointer so
// Run handlers can read the parsed value.
func RegisterEnv(fs *flag.FlagSet, defaultVal string, usage string) *string {
	var envVal string
	fs.StringVar(&envVal, "env", defaultVal, usage)
	fs.StringVar(&envVal, "environment", defaultVal, usage)
	return &envVal
}
