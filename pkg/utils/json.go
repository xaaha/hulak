package utils

import (
	"encoding/json"
	"fmt"
)

// sample function on how I should handle the json responses
func PrintJson() {
	chunk := map[string]interface{}{
		"code":  "5000",
		"error": "Error",
		"a":     5,
		"b":     7,
	}
	val, _ := json.Marshal(chunk)
	fmt.Println(string(val))
}
