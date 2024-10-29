package main

import (
	apicalls "github.com/xaaha/hulak/pkg/apiCalls"
	fileReader "github.com/xaaha/hulak/pkg/hulak_yaml_reader"
)

func main() {
	// testInitialization()
	apicalls.TestApiCalls()
	fileReader.ReadingYamlWithoutStruct()
}
