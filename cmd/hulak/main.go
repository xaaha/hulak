package main

import (
	"encoding/json"
	"fmt"

	"github.com/xaaha/hulak/pkg/envparser"
)

func main() {
	// Initialize the project
	InitializeProject()

	envMap, err := envparser.GenerateSecretsMap()
	if err != nil {
		panic(err)
	}
	// print entire json
	niceJson, _ := json.MarshalIndent(envMap, "", "  ")
	fmt.Println(string(niceJson))
	finalAns, err := envparser.SubstitueVariables("env{{PORT}}", envMap)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(finalAns)
	// fmt.Println("Default Environment value:", os.Getenv("hulakEnv"))
}

/*
Tests
- Complete & Final Map can be printed as json
- SubstitueVariables
 - SubstitueVariables and make sure the substitution is working as expected
- Find  a way to document the falg used
*/
