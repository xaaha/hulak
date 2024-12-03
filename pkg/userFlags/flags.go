package userflags

import (
	"flag"

	"github.com/xaaha/hulak/pkg/utils"
)

var (
	fp  *string
	env *string
)

// go's init func executes automatically, and registers the flags during package initialization
func init() {
	fp = flag.String("fp", "", "relative yaml file path (fp) from env")
	env = flag.String("env", utils.DefaultEnvVal, "environment file to use during the call")
}

// FilePath returns the parsed value of the file path "fp" flag
func FilePath() string {
	return *fp
}

func Env() string {
	return *env
}
