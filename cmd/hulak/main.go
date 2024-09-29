package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/xaaha/hulak/pkg/envparser"
	ymlReader "github.com/xaaha/hulak/pkg/hulak_yaml_reader"
)

func testInitialization() {
	// Initialize the project
	InitializeProject()

	envMap, err := envparser.GenerateSecretsMap()
	if err != nil {
		panic(err)
	}

	// print entire json
	niceJson, _ := json.MarshalIndent(envMap, "", "  ")
	fmt.Println(string(niceJson))

	// how to substitute variable
	finalAns, err := envparser.SubstitueVariables("env{{PORT}}", envMap)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(finalAns)

	fmt.Println("Default Environment value:", os.Getenv("hulakEnv"))
}

func main() {
	testInitialization() // this is not working. fix it
	ymlReader.ReadingYamlWithStruct()
}
