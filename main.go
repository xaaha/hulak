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
	fileName := userflags.File()

	// create envMap
	envMap := InitializeProject(env)

	filePathList, err := userflags.GenerateFilePathList(fileName, fp)
	if err != nil {
		utils.PanicRedAndExit("main.go %v", err)
	}

	var wg sync.WaitGroup

	// Run tasks concurrently
	for _, eachPath := range filePathList {
		wg.Add(1)
		go func() {
			defer wg.Done()
			apicalls.SendApiRequest(utils.CopyEnvMap(envMap), eachPath)
		}()
	}

	// wait for all go routines to complete
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
