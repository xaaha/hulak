package cliflags

import "flag"

// RegisterShow binds --show on fs to a bool with default false. Returns
// a pointer so the handler reads the parsed value after flag.Parse.
//
// Commands that print sensitive data (env values, request headers) should
// mask by default and require an explicit --show to reveal. Pattern mirrors
// RegisterOutput for -o/--out so every command opts in the same way.
func RegisterShow(fs *flag.FlagSet, usage string) *bool {
	var show bool
	fs.BoolVar(&show, "show", false, usage)
	return &show
}

// RegisterDryRun binds --dry-run on fs to a bool with default false.
// Returns a pointer so the handler reads the parsed value after flag.Parse.
//
// Kept as a helper so every command that opts into dry-run gets the same
// usage string and default. Mirrors RegisterShow.
func RegisterDryRun(fs *flag.FlagSet) *bool {
	var dryRun bool
	fs.BoolVar(&dryRun, "dry-run", false, "Print the built request and exit without sending it")
	return &dryRun
}
