package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/xaaha/hulak/pkg/envparser"
)

/*
InitializeProject() starts the project by creating envfolder and global file in it.
TBC...
*/
func InitializeProject() {
	err := envparser.CreateDefaultEnvs(nil)
	if err != nil {
		panic(err)
	}
}

/*
Initialize the project and how to substiture a variable.
This function is just for my dumb brain that forgets how to do simple stuff in a program I wrote
*/
func testInitialization() {
	InitializeProject()

	envMap, err := envparser.GenerateSecretsMap()
	if err != nil {
		panic(err)
	}

	// print entire json
	niceJson, _ := json.MarshalIndent(envMap, "", "  ")
	fmt.Println(string(niceJson))

	// how to substitute variable
	finalAns, err := envparser.SubstituteVariables("env{{.PORT}}", envMap)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(finalAns)

	fmt.Println("Default Environment value:", os.Getenv("hulakEnv"))
}
