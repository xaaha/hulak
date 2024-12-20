package main

import (
	"sync"

	apicalls "github.com/xaaha/hulak/pkg/apiCalls"
	"github.com/xaaha/hulak/pkg/envparser"
	"github.com/xaaha/hulak/pkg/utils"
)

/*
InitializeProject() starts the project by creating envfolder and global.env file in it.
returns the envMap
TBC...
*/
func InitializeProject(env string) map[string]string {
	err := envparser.CreateDefaultEnvs(nil)
	if err != nil {
		panic(err)
	}
	envMap, err := envparser.GenerateSecretsMap(env)
	if err != nil {
		panic(err)
	}
	return envMap
}

func RunTasks(filePathList []string, envMap map[string]string) {
	var wg sync.WaitGroup

	// Run tasks concurrently
	for _, eachPath := range filePathList {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			apicalls.SendAndSaveApiRequest(utils.CopyEnvMap(envMap), path)
		}(eachPath)
	}

	wg.Wait()
}
