package main

import (
	"flag"
	"sync"

	apicalls "github.com/xaaha/hulak/pkg/apiCalls"
	userflags "github.com/xaaha/hulak/pkg/userFlags"
	"github.com/xaaha/hulak/pkg/utils"
)

func main() {
	// parse all the necessary flags
	flag.Parse()

	env := userflags.Env()
	fp := userflags.FilePath()
	// file := userflags.File()

	// create envMap
	envMap := InitializeProject(env)

	// filePathList, err := utils.ListMatchingFiles(file)
	// if err != nil {
	// 	utils.PanicRedAndExit(err.Error())
	// }

	// if the fp is present use that
	// if both -fp and -f are provided use the fp and ignore file
	// if only the filePathList is present then number of tasks equal to len(filePathList) and run job concurrently
	// if nothing is panic
	// if len(filePathList) > 0 {
	// }

	var wg sync.WaitGroup

	// var numTasks int
	// if len(fp) > 0 {
	// 	numTasks = len(fp)
	// } else {
	// 	numTasks = len(filePathList)
	// }
	// Define tasks

	// Run tasks concurrently
	wg.Add(1)
	go func(sendRequest func(map[string]string, string), env map[string]string, filePath string) {
		defer wg.Done()
		sendRequest(utils.CopyEnvMap(env), filePath)
	}(apicalls.SendApiRequest, envMap, fp)

	wg.Wait()
}

/*
// Define the shared task function
	task := apicalls.SendApiRequest

	// Run tasks concurrently
	for i := 0; i < numTasks; i++ {
		wg.Add(1)
		go func(env map[string]string, filePath string) {
			defer wg.Done()
			task(utils.CopyEnvMap(env), filePath)
		}(envMap, fp) // Pass the parameters
	}

	wg.Wait()
*/
