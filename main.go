package main

import (
	"fmt"

	apicalls "github.com/xaaha/hulak/pkg/apiCalls"
	fileReader "github.com/xaaha/hulak/pkg/yamlParser"
)

func main() {
	envMap := InitializeProject()
	// apicalls.TestApiCalls() // temp call.. replace with mock
	jsonString := fileReader.ReadYamlForHttpRequest("test_collection/form_data.yaml", envMap)
	apiInfo, err := apicalls.CombineAndCall(jsonString)
	if err != nil {
		_ = fmt.Errorf("Error occoured in main %v", err)
	}
	resp := apicalls.StandardCall(apiInfo)
	fmt.Println(resp)
}
