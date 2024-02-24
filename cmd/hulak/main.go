package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/xaaha/hulak/pkg/envparser"
)

func printJson() {
	chunk := map[string]interface{}{
		"code":  "5000",
		"error": "Error",
		"a":     5,
		"b":     7,
	}
	val, _ := json.Marshal(chunk)
	fmt.Println(string(val))
}

func main() {
	printJson()
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	filePath := filepath.Join(cwd, ".env.global")
	err = envparser.ParsingEnv(filePath)
	if err != nil {
		panic(err)
	}
}
