package main

import (
	"fmt"
	"sync"

	apicalls "github.com/xaaha/hulak/pkg/apiCalls"
	"github.com/xaaha/hulak/pkg/envparser"
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlParser"
)

/*
InitializeProject() starts the project by creating envfolder and global.env file in it.
returns the envMap
TBC...
*/
func InitializeProject(env string) map[string]interface{} {
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

func RunTasks(filePathList []string, secretsMap map[string]interface{}) {
	var wg sync.WaitGroup

	// Run tasks concurrently based on the kinds in yaml file
	for _, eachPath := range filePathList {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()

			// Parse the configuration for each file
			config := yamlParser.MustParseConfig(path, utils.CopyEnvMap(secretsMap))

			// Handle different kinds
			switch {
			case config.IsAuth():
				// For now, just print hello world for Auth kind
				fmt.Printf("hello world - processing Auth configuration: %s\n", path)

			case config.IsAPI():
				// Default API handling (existing functionality)
				apicalls.SendAndSaveApiRequest(utils.CopyEnvMap(secretsMap), path)

			default:
				// This shouldn't happen as invalid kinds are caught in MustParseConfig, but just in case...
				utils.PanicRedAndExit("Unsupported kind in file: %s", path)
			}
		}(eachPath)
	}

	wg.Wait()
}
