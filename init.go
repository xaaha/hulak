package main

import "github.com/xaaha/hulak/pkg/envparser"

/*
InitializeProject() starts the project by creating envfolder and global file in it.
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
