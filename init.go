// Package main initializes the project and runs the query
package main

import (
	"sync"

	apicalls "github.com/xaaha/hulak/pkg/apiCalls"
	"github.com/xaaha/hulak/pkg/envparser"
	"github.com/xaaha/hulak/pkg/features"
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

/*
InitializeProject starts the project by creating envfolder and global.env file in it.
*/
func InitializeProject(env string) map[string]any {
	if err := envparser.CreateDefaultEnvs(nil); err != nil {
		utils.PanicRedAndExit("%v", err)
	}
	envMap, err := envparser.GenerateSecretsMap(env)
	if err != nil {
		panic(err)
	}
	return envMap
}

// RunTasks manages the go tasks
func RunTasks(filePathList []string, secretsMap map[string]any, debug bool) {
	var wg sync.WaitGroup

	// Run tasks concurrently based on the kinds in yaml file
	for _, eachPath := range filePathList {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			// Parse the configuration for each file
			config := yamlparser.MustParseConfig(path, utils.CopyEnvMap(secretsMap))

			// Handle different kinds based on the yaml 'kind' we get.
			switch {
			case config.IsAuth():
				if err := features.SendAPIRequestForAuth2(utils.CopyEnvMap(secretsMap), path, debug); err != nil {
					utils.PrintRed(err.Error())
				}

			case config.IsAPI():
				if err := apicalls.SendAndSaveAPIRequest(utils.CopyEnvMap(secretsMap), path, debug); err != nil {
					utils.PrintRed(err.Error())
				}

			default:
				// This shouldn't happen as invalid kinds are caught in MustParseConfig, but just in case...
				utils.PanicRedAndExit("Unsupported kind in file: %s", path)
			}
		}(eachPath)
	}

	wg.Wait()
}
