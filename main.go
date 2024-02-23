package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func parsingEnv(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// skip new line and comments
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}
		// remove the empty lines
		splitStr := strings.Split(line, "\n")
		fmt.Println(splitStr)
		// read each value on = strings.splitAfter =
	}
	// create a map so that when user calls it with {{key}} the value is returned
	return nil
}

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
	err := parsingEnv("./.env.global")
	if err != nil {
		panic(err)
	}
}
