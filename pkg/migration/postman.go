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
// If it's globals and  _postman_variable_scope has globals scope on it, then push it to the globals then push this into global.env
// If the name in json exists in the env folder there is no need to create it, just migrate
// Existing function to create folder and file for the environment
// Otherwise just create then same environment as the name
// values array struct
// Collection:
// for collection: if the collection has variables and the variables is coming from the variable below add it to the globals
// but what if there is already a variable that exists with the same name in global? Because,

type KeyValuePair struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type EnvValues struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Enabled bool   `json:"enabled"`
}

// Postman's Environment Json file
type Environment struct {
	Name   string      `json:"name"`
	Values []EnvValues `json:"values"`
	Scope  string      `json:"_postman_variable_scope"`
}

// collectionInfo object
type collectionInfo struct {
	PostmanId      string `json:"_postman_id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	Schema         string `json:"schema"`
	CollectionLink string `json:"_collection_link"`
}

// type collectionItemBody struct {
// 	Mode string `json:"mode"`
// }

type itemRawUrl struct {
	Raw string `json:"raw"`
}

type collectionItemRequest struct {
	Method string         `json:"method"`
	Header []KeyValuePair `json:"header"`
	Url    itemRawUrl     `json:"url"`
}

type collectionItem struct {
	Name    string                `json:"name"`
	Request collectionItemRequest `json:"request"`
	// dis-regard event and response []
}

// postman 2.1 collection
type Collection struct {
	Info     collectionInfo `json:"info"`
	Item     collectionItem `json:"item"`
	Variable []KeyValuePair `json:"variable"`
}

// Reads the json file and returns the jsonString
func ReadPmFile(filePath string) map[string]any {
	var jsonStrFile map[string]any
	jsonByteVal, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Println("error occured while opening the json file", err)
		return nil
	}

	err = json.Unmarshal(jsonByteVal, &jsonStrFile)
	if err != nil {
		fmt.Println("error occured while unmarshalling the file", err)
		return nil
	}

	return jsonStrFile
}

func MigrateEnv(env Environment) {
	var message strings.Builder
	for _, eachVarItem := range env.Values {
		keyVal := fmt.Sprintf("%s = %s\n", eachVarItem.Key, eachVarItem.Value)
		if !eachVarItem.Enabled {
			keyVal = fmt.Sprintf("# %s = %s\n", eachVarItem.Key, eachVarItem.Value)
		}
		message.WriteString(keyVal)
	}
	content := message.String()

	byteSlice := []byte(content)

	// TODO-1: Use existing function to write file in the env/global.env || env/staging.env
	err := os.WriteFile("test.env", byteSlice, 0644)
	if err != nil {
		fmt.Println("error occured while writing file 'test.env'", err)
	}
}

func MigrateCollection(collection Collection) {
}

func CompleteMigration(filePath []string) {
	// loop over the path array
	// for each path, read the pm file with ReadPmFile function
	// ReadPmFile function returns jsonString. Use the string, to find out what the json it is
	// Then marshall the jsonString to the appropriate struct, either Environment or  Collection
	// If the jsonString is envFile, migrateEnv
	// If the jsonString is collection, migrateCollection
}

// returns true if, the jsonString has "values" and "_postman_variable_scope"
func IsEnv(jsonString map[string]any) bool {
	_, valuesExists := jsonString["values"]
	_, pmVarScopeExists := jsonString["_postman_variable_scope"]

	return valuesExists && pmVarScopeExists
}

// returns true, if the jsonString has "info" and "item"
func IsCollection(jsonString map[string]any) bool {
	_, infoExists := jsonString["info"]
	_, itemExists := jsonString["item"]

	return infoExists && itemExists
}
