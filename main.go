// Package main initializes the project and runs the query
package main

import (
	userflags "github.com/xaaha/hulak/pkg/userFlags"
	"github.com/xaaha/hulak/pkg/utils"
)

func main() {
	flags, err := userflags.ParseFlagsSubcmds()
	if err != nil {
		utils.PanicRedAndExit("%v", err)
	}

	env := flags.Env
	fp := flags.FilePath
	fileName := flags.File
	debug := flags.Debug
	// dir := flags.Dir
	// dirseq := flags.Dirseq
	envMap := InitializeProject(env)
	filePathList, err := userflags.GenerateFilePathList(fileName, fp)
	if err != nil {
		utils.PanicRedAndExit("%v", err)
	}

	if userflags.HasFlag() {
		RunTasks(filePathList, envMap, debug)
	}
}
