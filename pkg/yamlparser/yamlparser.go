// Package yamlparser does everything related to yaml file for hulak, including type translation
package yamlparser

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	yaml "github.com/goccy/go-yaml"
	"github.com/xaaha/hulak/pkg/actions"
	"github.com/xaaha/hulak/pkg/envparser"
	"github.com/xaaha/hulak/pkg/utils"
)

// From the yaml file, create a json file. But the json could have {{}} on it
// So, we need to, read the file, make sure those values are handled, then return the proper map

// replaceVarsWithValues walks the YAML tree and substitutes {{...}} template
// references with values from secretsMap. Errors carry a dotted path of the
// failing key (e.g. "body.urlencodedformdata.client_secret") so the user can
// jump straight to the offending YAML node — no need for a nested wrapper
// chain at each recursion level.
func replaceVarsWithValues(
	dict map[string]any,
	secretsMap map[string]any,
) (map[string]any, error) {
	return replaceVarsWithPrefix(dict, secretsMap, "")
}

// replaceVarsWithPrefix is the recursive helper. prefix is the dotted path
// of the parent map; the empty string at the root suppresses a leading dot.
func replaceVarsWithPrefix(
	dict map[string]any,
	secretsMap map[string]any,
	prefix string,
) (map[string]any, error) {
	changedMap := make(map[string]any)

	for key, val := range dict {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		switch valTyped := val.(type) {
		case map[string]any:
			nestedMap, err := replaceVarsWithPrefix(valTyped, secretsMap, fullKey)
			if err != nil {
				return nil, err
			}
			changedMap[key] = nestedMap

		case string:
			// OPTIMIZATION: Skip template parsing if no template syntax present
			if !strings.Contains(valTyped, "{{") {
				changedMap[key] = valTyped
				continue
			}
			finalChangedValue, err := envparser.SubstituteVariables(valTyped, secretsMap)
			if err != nil {
				return nil, fmt.Errorf("substituting %q: %w", fullKey, err)
			}
			changedMap[key] = finalChangedValue

		case map[string]string:
			innerMap := make(map[string]any)
			for k, v := range valTyped {
				if !strings.Contains(v, "{{") {
					innerMap[k] = v
					continue
				}
				finalChangedValue, err := envparser.SubstituteVariables(v, secretsMap)
				if err != nil {
					return nil, fmt.Errorf("substituting %q: %w", fullKey+"."+k, err)
				}
				innerMap[k] = finalChangedValue
			}
			changedMap[key] = innerMap

		default:
			changedMap[key] = val
		}
	}
	return changedMap, nil
}

// Reads YAML, validates if the file exists, is not empty, and changes keys to lowercase
func checkYamlFile(filepath string, secretsMap map[string]any) (*bytes.Buffer, error) {
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", filepath)
	}

	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("opening file %s: %w", filepath, err)
	}
	defer file.Close()

	fileInfo, _ := file.Stat()
	if fileInfo.Size() == 0 {
		return nil, fmt.Errorf("empty yaml file: %s", filepath)
	}

	var data map[string]any
	dec := yaml.NewDecoder(file)
	if err = dec.Decode(&data); err != nil {
		return nil, fmt.Errorf("decoding %s: %w", filepath, err)
	}

	// make yaml keys  case insensitive. method or Method or METHOD should all be the same
	data = utils.ConvertKeysToLowerCase(data)

	// parse all the values to with {{.key}} from .env folder
	parsedMap, err := replaceVarsWithValues(data, secretsMap)
	if err != nil {
		return nil, err
	}

	// translate the types, if acceptable
	parsedMap, err = translateType(data, parsedMap, secretsMap, actions.GetValueOf)
	if err != nil {
		// TODO(#180): replace with fmt.Errorf — ColorError injects \n + ANSI
		// codes that the runner has to strip before rendering.
		return nil, utils.ColorError(utils.ErrYAMLPostProcessing, err)
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	if err := enc.Encode(parsedMap); err != nil {
		return nil, fmt.Errorf("encoding %s: %w", filepath, err)
	}
	enc.Close()

	return &buf, nil
}

// FinalStructForAPI builds a final struct for the api call.
// Returns APICallFile struct, true if file is valid, and error
// It  checks the validity of all the fields in the yaml file meant for regular api call
func FinalStructForAPI(filePath string, secretsMap map[string]any) (APICallFile, bool, error) {
	buf, err := checkYamlFile(filePath, secretsMap)
	if err != nil {
		return APICallFile{}, false, err
	}

	var file APICallFile
	dec := yaml.NewDecoder(buf)
	if err := dec.Decode(&file); err != nil {
		return APICallFile{}, false, err
	}

	if valid, err := file.IsValid(filePath); !valid {
		return APICallFile{}, false, err
	}

	return file, true, nil
}

// FinalStructForOAuth2 checks the validity of all the fields in the yaml file meant for OAuth2.0.
// It returns AuthRequestBody struct
func FinalStructForOAuth2(
	filePath string,
	secretsMap map[string]any,
) (AuthRequestFile, error) {
	buf, err := checkYamlFile(filePath, secretsMap)
	if err != nil {
		return AuthRequestFile{}, utils.ColorError("Error after reading yaml file: %v", err)
	}

	var auth2Config AuthRequestFile
	dec := yaml.NewDecoder(buf)
	if err := dec.Decode(&auth2Config); err != nil {
		return AuthRequestFile{}, utils.ColorError("Error decoding data: %v", err)
	}

	if valid, err := auth2Config.IsValid(); !valid {
		return AuthRequestFile{}, utils.ColorError("Error on Auth2 Request Body %v", err)
	}
	return auth2Config, nil
}

// FinalStructForGraphQL builds a final struct for GraphQL requests.
// Returns APICallFile struct, true if valid, and error.
// It applies default method (POST) and headers (Content-Type: application/json).
// Unlike FinalStructForAPI, this does NOT require body/query since the TUI provides it.
func FinalStructForGraphQL(filePath string, secretsMap map[string]any) (APICallFile, bool, error) {
	buf, err := checkYamlFile(filePath, secretsMap)
	if err != nil {
		return APICallFile{}, false, err
	}

	var file APICallFile
	dec := yaml.NewDecoder(buf)
	if err := dec.Decode(&file); err != nil {
		return APICallFile{}, false, err
	}

	if valid, err := file.IsValidForGraphQL(filePath); !valid {
		return APICallFile{}, false, err
	}

	return file, true, nil
}
