package main

import (
	"fmt"

	fileReader "github.com/xaaha/hulak/pkg/yamlParser"
)

func main() {
	envMap := InitializeProject()
	// testInitialization()
	// apicalls.TestApiCalls() // temp call.. replace with mock
	jsonString := fileReader.ReadYamlForHttpRequest("test_collection/user.yaml", envMap)
	fmt.Println(jsonString)
}
