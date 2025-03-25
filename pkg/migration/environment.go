package migration

// postman environment
import (
	"fmt"
	"os"
	"strings"

	"github.com/xaaha/hulak/pkg/envparser"
	"github.com/xaaha/hulak/pkg/utils"
)

// EnvValues represents a single environment variable in a Postman environment
type EnvValues struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Enabled bool   `json:"enabled"`
}

// Environment represents a Postman environment JSON file structure
type Environment struct {
	Name   string      `json:"name"`
	Values []EnvValues `json:"values"`
	Scope  string      `json:"_postman_variable_scope"`
}

// IsEnv determines if the JSON contains a Postman environment
func IsEnv(jsonString map[string]any) bool {
	_, valuesExists := jsonString["values"]
	_, pmVarScopeExists := jsonString["_postman_variable_scope"]
	return valuesExists && pmVarScopeExists
}

// PrepareEnvStruct converts JSON to Environment struct
func PrepareEnvStruct(jsonStr map[string]any) (Environment, error) {
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

// MigrateEnv migrates a Postman JSON environment file to `key = value` pairs inside the env dir
func MigrateEnv(env Environment, comment ...string) error {
	var message strings.Builder
	if len(comment) == 0 {
		comment[0] = "\n### Postman Env Migration ###\n"
	}
	message.WriteString(comment[0])
	for _, eachVarItem := range env.Values {
		key := sanitizeKey(eachVarItem.Key)
		keyVal := fmt.Sprintf("%s = %s\n", key, eachVarItem.Value)
		if !eachVarItem.Enabled || eachVarItem.Value == "" {
			keyVal = fmt.Sprintf("# %s = %s\n", key, eachVarItem.Value)
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
		return utils.ColorError("error creating env directory or file: %w", err)
	}

	// append the content
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, utils.FilePer)
	if err != nil {
		return utils.ColorError("error opening file: %w", err)
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		return utils.ColorError("error writing to file: %w", err)
	}

	utils.PrintGreen("\nEnvironment migration successful!")
	return nil
}
