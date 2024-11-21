package main

import (
	fileReader "github.com/xaaha/hulak/pkg/yamlParser"
)

func main() {
	envMap := InitializeProject()
	// testInitialization()
	// apicalls.TestApiCalls() // temp call.. replace with mock
	fileReader.ReadYamlForHttpRequest("test_collection/user.yaml", envMap)
}
