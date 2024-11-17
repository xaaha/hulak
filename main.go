package main

import (
	fileReader "github.com/xaaha/hulak/pkg/yamlParser"
)

func main() {
	// InitializeProject()
	// testInitialization()
	// apicalls.TestApiCalls() // temp call.. replace with mock
	fileReader.ReadYamlForHttpRequest("test_collection/user.yaml", map[string]string{})
}
