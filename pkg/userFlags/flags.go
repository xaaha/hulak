// Package userflags have everything related to user's flags & subcommands
package userflags

import (
	"flag"

	"github.com/xaaha/hulak/pkg/utils"
)

var (
	fp    *string
	env   *string
	f     *string
	debug *bool
)

// go's init func executes automatically, and registers the flags during package initialization
func init() {
	env = flag.String("env", utils.DefaultEnvVal, "environment file to use during the call")
	fp = flag.String(
		"fp",
		"",
		"Relative (or absolute) file path (fp) of the request file from the environment directory",
	)
	f = flag.String(
		"f",
		"",
		"File name for making an api request. File name is case-insensitive",
	)

	debug = flag.Bool(
		"debug",
		false,
		"enable debug mode to get the entire request, response, headers, and other info for the API call",
	)
}

// FilePath returns the parsed value of the file path "fp" flag -fp
func FilePath() string {
	return *fp
}

// File name, case insensitive, for the request -f
func File() string {
	return *f
}

// Env defines the env for the call, global by default
func Env() string {
	return *env
}

// Debug represents if the user wants the entire statement
func Debug() bool {
	return *debug
}
