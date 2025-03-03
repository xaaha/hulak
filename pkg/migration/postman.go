package migration

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// hulak migrate "./globals.json"
// if the json has info object it's collection
// if the json has "name": "Globals",and  "_postman_variable_scope": "globals", then it's the env
// If name is empty ""  or name == "globals" then migrate things to global
// Otherwise a name in pm json file should create a new env file with the exact name if the env file does not exists
// If the name in json exists in the env folder there is no need to create it, just migrate
// Existing function to create folder and file for the environment
// If it's globals and  _postman_variable_scope has globals scope on it, then push it to the globals then push this into global.env
// Otherwise just create then same environment as the name
// values array struct

type EnvValues struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Enabled bool   `json:"enabled"`
}

// json struct for the entire env file
type Environment struct {
	Name   string      `json:"name"`
	Values []EnvValues `json:"values"`
}

// Reads the env.json postman file
func ReadPmEnvFile(filePath string) Environment {
	var env Environment
	jsonByteVal, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Println("error occured while opening the json file", err)
	}
	err = json.Unmarshal(jsonByteVal, &env)
	if err != nil {
		fmt.Println("error occured while unmarshalling the file", err)
	}
	return env
}

func MigrateEnv() {
	var message strings.Builder
	message.WriteString("hello = there\n")
	message.WriteString("foo = bar\n")
	content := message.String()
	// if !EnvValues.Enabled  then add #

	byteSlice := []byte(content)
	err := os.WriteFile("test.env", byteSlice, 0644)
	if err != nil {
		fmt.Println("error occured while writing file 'test.env'", err)
	}
}

func IsEnv() bool {
	// true if "values" exists, is an array with EnvValues match exist
	//  _postman_variable_scope exists in json

	return false
}

func IsCollection() bool {
	// true if the struct info.scehma, which is a url has the word collection on it,
	return false
}

/*
func main() {
	env := ReadPmEnvFile("./globals.json")
	fmt.Println("Key = ", env.Values[0].Key)
	fmt.Println("Value \u2713 =", env.Values[0].Value)
	MigrateEnv()
}
*/
