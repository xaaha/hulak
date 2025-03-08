package migration

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/xaaha/hulak/pkg/envparser"
	"github.com/xaaha/hulak/pkg/utils"
)

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
func readPmFile(filePath string) map[string]any {
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

// takes the jsonString  and convets to Environment struct
func prepareEnvStruct(jsonStr map[string]any) (Environment, error) {
	var env Environment

	// extract name first
	if name, ok := jsonStr["name"].(string); ok {
		env.Name = name
	} else {
		return env, utils.ColorError("name field is missing or not a string")
	}

	if values, ok := jsonStr["values"].([]any); ok {
		for _, v := range values {
			if valueMap, ok := v.(map[string]any); ok {
				var envValue EnvValues

				if key, ok := valueMap["key"].(string); ok {
					envValue.Key = key
				} else {
					return env, utils.ColorError("key field is missing or not a string in EnvValues")
				}

				if value, ok := valueMap["value"].(string); ok {
					envValue.Value = value
				} else {
					return env, utils.ColorError("value field is missing or not a string in EnvValues")
				}

				if enabled, ok := valueMap["enabled"].(bool); ok {
					envValue.Enabled = enabled
				} else {
					return env, utils.ColorError("enabled field is missing or not a boolean in EnvValues")
				}

				env.Values = append(env.Values, envValue)
			} else {
				return env, utils.ColorError("value is not a valid map for EnvValues")
			}
		}
	} else {
		return env, utils.ColorError("values field is missing or not an array")
	}

	if scope, ok := jsonStr["_postman_variable_scope"].(string); ok {
		env.Scope = scope
	} else {
		return env, utils.ColorError("scope field is missing or not a string")
	}

	return env, nil
}

// migrates a postman json environment file to `key = value` pair inside the env dir
func migrateEnv(env Environment) {
	var message strings.Builder
	message.WriteString("\n### Postman Env Migration ###\n")
	for _, eachVarItem := range env.Values {
		keyVal := fmt.Sprintf("%s = %s\n", eachVarItem.Key, eachVarItem.Value)
		if !eachVarItem.Enabled {
			keyVal = fmt.Sprintf("# %s = %s\n", eachVarItem.Key, eachVarItem.Value)
		}
		message.WriteString(keyVal)
	}
	content := message.String()

	// env fileName
	var envFileName string
	lowerCased := strings.ToLower(env.Name)
	if env.Name == "" || lowerCased == "globals" {
		envFileName = utils.DefaultEnvVal
	} else {
		envFileName = env.Name
	}

	filePath, err := envparser.CreateEnvDirAndFiles(envFileName)
	if err != nil {
		utils.PrintRed("error occured on env migration: " + err.Error())
		return
	}

	// append the content
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		utils.PrintRed("Error opening or creating file: " + err.Error())
		return
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}

	utils.PrintGreen("\nEnvironment migration successful!")
}

func MigrateCollection(collection Collection) {
}

func CompleteMigration(filePath []string) {
	for _, path := range filePath {
		jsonStr := readPmFile(path)

		if isEnv(jsonStr) {
			env, err := prepareEnvStruct(jsonStr)
			if err != nil {
				fmt.Println("error occurred while converting to Environment:", err)
				return
			}
			migrateEnv(env)
		}
		if isCollection(jsonStr) {
			fmt.Println("Collection Migration Coming Soon")
		}
	}
}

// returns true if, the jsonString has "values" and "_postman_variable_scope"
func isEnv(jsonString map[string]any) bool {
	_, valuesExists := jsonString["values"]
	_, pmVarScopeExists := jsonString["_postman_variable_scope"]

	return valuesExists && pmVarScopeExists
}

// returns true, if the jsonString has "info" and "item"
func isCollection(jsonString map[string]any) bool {
	_, infoExists := jsonString["info"]
	_, itemExists := jsonString["item"]

	return infoExists && itemExists
}
