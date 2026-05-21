package userflags

import "flag"

// registerShowFlag binds --show on fs to a bool with default false. Returns
// a pointer so the handler reads the parsed value after flag.Parse.
//
// Commands that print sensitive data (env values, request headers) should
// mask by default and require an explicit --show to reveal. Pattern mirrors
// registerOutputFlag for -o/--out so every command opts in the same way.
func registerShowFlag(fs *flag.FlagSet, usage string) *bool {
	var show bool
	fs.BoolVar(&show, "show", false, usage)
	return &show
}
