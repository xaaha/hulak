package main

import (
	"encoding/json"
	"fmt"

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
	err := envparser.ParsingEnv("../../.env.global")
	if err != nil {
		panic(err)
	}
}
