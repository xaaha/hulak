// Centralizes the --yes / -y skip-confirm flag used by destructive commands.
// Kept in its own file (next to the other *_flag.go helpers) so new
// destructive commands can opt in with a single helper call instead of
// duplicating the BoolVar pair.
package userflags

import "flag"

// registerYesFlag binds --yes and -y to one bool variable on fs and returns
// a pointer so Run handlers can read the parsed value. Use with
// confirmDestroy: pass *yes as the force argument to skip the prompt for
// scripts and CI.
func registerYesFlag(fs *flag.FlagSet, usage string) *bool {
	var yes bool
	fs.BoolVar(&yes, "yes", false, usage)
	fs.BoolVar(&yes, "y", false, usage)
	return &yes
}
