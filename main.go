package main

import (
	fileReader "github.com/xaaha/hulak/pkg/yamlParser"
)

func main() {
	// testInitialization()
	// apicalls.TestApiCalls() // temp call. replace with mock
	fileReader.ReadingYamlWithStruct()
}
