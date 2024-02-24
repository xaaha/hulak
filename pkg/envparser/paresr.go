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
		// skip empty line and comments
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}
		splitStr := strings.Split(line, "\n") // remove the empty lines
		secret := strings.Split(splitStr[0], "=")
		key := secret[0]
		val := secret[1]
		if strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"") {
			val = strings.Trim(val, "\"")
		}
		if strings.HasPrefix(val, "'") && strings.HasSuffix(val, "'") {
			val = strings.Trim(val, "'")
		}

		fmt.Println(key, val)
	}
	return nil
}

// create a map so that when user calls it with {{key}} the value is returned
// add make file
// handle ""
// make sure no two items have the same key
// be able to set a .env file as main so that,  {{}} is read as a variable
/*
- If the string includes {{}}, then get the name inside the curly brackets
- Find the name in the default current environment.
- Default is global only if user does not have a defined environment.
- Global > defined > Collection
*/
