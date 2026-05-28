package cliflags

import "flag"

// RegisterYes binds --yes and -y to one bool variable on fs and returns
// a pointer so Run handlers can read the parsed value. Use with
// confirmDestroy: pass *yes as the force argument to skip the prompt for
// scripts and CI.
func RegisterYes(fs *flag.FlagSet, usage string) *bool {
	var yes bool
	fs.BoolVar(&yes, "yes", false, usage)
	fs.BoolVar(&yes, "y", false, usage)
	return &yes
}
