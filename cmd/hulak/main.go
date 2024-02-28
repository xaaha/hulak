package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/xaaha/hulak/pkg/envparser"
	"github.com/xaaha/hulak/pkg/utils"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	filePath := filepath.Join(cwd, ".env.global")
	err = envparser.ParsingEnv(filePath)
	if err != nil {
		panic(err)
	}
	// fmt.Println(envparser.GetEnvVar("NAME"))
	resolved, err := envparser.SubstitueVariables("myNameIs={{NAME}}")
	if err != nil {
		panic(err)
	}
	fmt.Println(resolved)
	resolved1, err := envparser.SubstitueVariables("myNameIs={{ixiai}}")
	if err != nil {
		utils.PrintError(err)
	}
	fmt.Println("Hello", resolved1)
}
