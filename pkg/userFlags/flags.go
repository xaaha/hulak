package userflags

import (
	"flag"

	"github.com/xaaha/hulak/pkg/utils"
)

var (
	fp  *string
	env *string
	f   *string
)

// go's init func executes automatically, and registers the flags during package initialization
func init() {
	env = flag.String("env", utils.DefaultEnvVal, "environment file to use during the call")
	fp = flag.String("fp", "", "file path (fp) of the request file")
	f = flag.String("f", "", "file name case insensitive")
}

// FilePath returns the parsed value of the file path "fp" flag
func FilePath() string {
	return *fp
}

// File name, case insensitive, for the request
func File() string {
	return *f
}

func Env() string {
	return *env
}
