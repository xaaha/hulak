package main

import (
	userflags "github.com/xaaha/hulak/pkg/userFlags"
	"github.com/xaaha/hulak/pkg/utils"
)

func main() {
	flags, err := userflags.ParseFlagsSubcmds()
	if err != nil {
		utils.PanicRedAndExit("main.go %v", err)
	}
	env := flags.Env
	fp := flags.FilePath
	fileName := flags.File

	// create envMap
	envMap := InitializeProject(env)

	filePathList, err := userflags.GenerateFilePathList(fileName, fp)
	if err != nil {
		utils.PanicRedAndExit("main.go %v", err)
	}

	if userflags.HasFlag() {
		RunTasks(filePathList, envMap)
	}
}
