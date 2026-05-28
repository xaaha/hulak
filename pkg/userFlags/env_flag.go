// Centralizes the --env / --environment flag pair used by every secrets
// subcommand that targets a single environment. Keeping the helper in its
// own file (next to the other *_flag.go helpers) means new commands can
// reuse the canonical long/short alias wiring without re-implementing it.
package userflags

import "flag"

// registerEnvFlag adds both --env and --environment aliases to a FlagSet,
// pointing to the same underlying variable, and returns a pointer so
// Run handlers can read the parsed value.
func registerEnvFlag(fs *flag.FlagSet, defaultVal string, usage string) *string {
	var envVal string
	fs.StringVar(&envVal, "env", defaultVal, usage)
	fs.StringVar(&envVal, "environment", defaultVal, usage)
	return &envVal
}
