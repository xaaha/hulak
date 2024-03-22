package main

import (
	"fmt"

	"github.com/xaaha/hulak/pkg/envparser"
)

func main() {
	// Initialize the project
	InitializeProject()
	// GenerateFinalEnvMap
	envMap, err := envparser.GenerateFinalEnvMap()
	if err != nil {
		panic(err)
	}
	fmt.Println(envMap)
	// fmt.Println("Default Environment value:", os.Getenv("hulakEnv"))
}

/*
Tests
- Fix the error when running make run-all
- Default Value is global.env
- Custom -env flag is used
- Complete & Final Map can be printed as json
- SubstitueVariables
 - SubstitueVariables and make sure the substitution is working as expected
- Implement Collection  Global > defined > Collection
- Find  a way to document the falg used
*/
