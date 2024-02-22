package main

import (
	"encoding/json"
	"fmt"
)

func main() {
	chunk := map[string]interface{}{
		"code":  "5000",
		"error": "Error",
		"a":     5,
		"b":     7,
	}
	val, _ := json.Marshal(chunk)
	fmt.Println(string(val))
}
