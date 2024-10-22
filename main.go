package main

import (
	"fmt"

	apicalls "github.com/xaaha/hulak/pkg/apiCalls"
	ymlReader "github.com/xaaha/hulak/pkg/hulak_yaml_reader"
)

func main() {
	// testInitialization()
	//	ymlReader.ReadingYamlWithStruct()
	// apicalls.Get()
	ymlReader.ReadingYamlWithoutStruct()
	fmt.Println("---")
	apicalls.CallUrlEncodedForm()

	// apicalls.GetAuthToken()
}
