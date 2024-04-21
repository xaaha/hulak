package main

import (
	"fmt"

	"github.com/xaaha/hulak/pkg/envparser"
)

func main() {
	// Initialize the project
	InitializeProject()

	// GenerateFinalEnvMap
	// envMap, err := envparser.GenerateFinalEnvMap()
	// if err != nil {
	// 	panic(err)
	// }
	// val, err := json.MarshalIndent(envMap, "", "  ")
	// if err != nil {
	// 	panic(err)
	// }
	// niceJson := string(val)
	// fmt.Println(niceJson)

	finalAns, err := envparser.SubstitueVariables("{{PORT}}:")
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
