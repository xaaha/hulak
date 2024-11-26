package main

import (
	"fmt"
	"io"

	apicalls "github.com/xaaha/hulak/pkg/apiCalls"
	fileReader "github.com/xaaha/hulak/pkg/yamlParser"
)

func main() {
	envMap := InitializeProject()
	// testInitialization()
	// apicalls.TestApiCalls() // temp call.. replace with mock
	jsonString := fileReader.ReadYamlForHttpRequest("test_collection/user.yaml", envMap)
	fmt.Println(jsonString)
	apiInfo, err := apicalls.CombineAndCall(jsonString)
	if err != nil {
		fmt.Errorf("Error occoured in main %v", err)
	}
	rdr := apiInfo.Body
	if rdr != nil {
		pritnThis, _ := io.ReadAll(rdr)
		fmt.Println(string(pritnThis))
	}
	fmt.Println("Headers", apiInfo.Headers)
}
