package userflags

import (
	"fmt"
	"os"
)

// write logic to check if we have enough arguments with
// and use this function to return the flag struct that main can use
// if the os.Args's second argument is migrate then run subcommands

func UserFalgs() {
	if len(os.Args) < 2 {
		fmt.Println("expected 'subcommands' or 'flag'")
		os.Exit(1)
	}
}

// var (
// 	fp  *string
// 	env *string
// 	f   *string
// )
//
// // go's init func executes automatically, and registers the flags during package initialization
// func init() {
// 	env = flag.String("env", utils.DefaultEnvVal, "environment file to use during the call")
// 	fp = flag.String(
// 		"fp",
// 		"",
// 		"Relative (or absolute) file path (fp) of the request file from the environment directory",
// 	)
// 	f = flag.String(
// 		"f",
// 		"",
// 		"File name for making an api request. File name is case-insensitive",
// 	)
// }
//
// // FilePath returns the parsed value of the file path "fp" flag -fp
// func FilePath() string {
// 	return *fp
// }
//
// // File name, case insensitive, for the request -f
// func File() string {
// 	return *f
// }
//
// // defines the env for the call
// func Env() string {
// 	return *env
// }
