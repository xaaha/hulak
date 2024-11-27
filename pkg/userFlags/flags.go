package userflags

import "flag"

var fp *string

// go's init func executes automatically, and registers the flags during package initialization
func init() {
	fp = flag.String("fp", "", "relative yaml file path (fp) from env")
}

// FilePath returns the parsed value of the "fp" flag
func FilePath() string {
	return *fp
}
