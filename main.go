package main

import (
	apicalls "github.com/xaaha/hulak/pkg/apiCalls"
	fileReader "github.com/xaaha/hulak/pkg/yamlParser"
)

func main() {
	// testInitialization()
	apicalls.TestApiCalls()
	fileReader.ReadingYamlWithoutStruct()
}
