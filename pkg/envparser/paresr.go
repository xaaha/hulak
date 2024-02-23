package envparser

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func ParsingEnv(filePath string) error {
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

// handle ""
// make sure no two items have the same key
// be able to set a .env file as main so that,  {{}} is read as a variable declaration
// from either global or collection or env variable
