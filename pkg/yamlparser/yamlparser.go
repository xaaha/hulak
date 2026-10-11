// Package yamlparser does everything related to yaml file for hulak, including type translation
package yamlparser

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	yaml "github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
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
	currentFile string,
) (map[string]any, error) {
	return replaceVarsWithPrefix(dict, secretsMap, "", currentFile)
}

// replaceVarsWithPrefix is the recursive helper. prefix is the dotted path
// of the parent map; the empty string at the root suppresses a leading dot.
// currentFile anchors the getFile "*" sibling shorthand.
func replaceVarsWithPrefix(
	dict map[string]any,
	secretsMap map[string]any,
	prefix string,
	currentFile string,
) (map[string]any, error) {
	changedMap := make(map[string]any)

	for key, val := range dict {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		switch valTyped := val.(type) {
		case map[string]any:
			nestedMap, err := replaceVarsWithPrefix(valTyped, secretsMap, fullKey, currentFile)
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
			finalChangedValue, err := envparser.SubstituteVariables(valTyped, secretsMap, currentFile)
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
				finalChangedValue, err := envparser.SubstituteVariables(v, secretsMap, currentFile)
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
		return nil, fmt.Errorf("file does not exist %s: %w", filepath, err)
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

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("reading file %s: %w", filepath, err)
	}

	if err := utils.ValidateSingleYAMLDoc(filepath, content); err != nil {
		return nil, err
	}

	var data map[string]any
	dec := yaml.NewDecoder(bytes.NewReader(content))
	if err = dec.Decode(&data); err != nil {
		return nil, fmt.Errorf("decoding %s: %w", filepath, err)
	}

	// make yaml keys  case insensitive. method or Method or METHOD should all be the same
	data = utils.ConvertKeysToLowerCase(data)

	// parse all the values to with {{.key}} from .env folder
	parsedMap, err := replaceVarsWithValues(data, secretsMap, filepath)
	if err != nil {
		return nil, err
	}

	// translate the types, if acceptable
	parsedMap, err = translateType(data, parsedMap, secretsMap, actions.GetValueOf)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", utils.ErrYAMLPostProcessing, err)
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf, yaml.JSON())
	if err := enc.Encode(parsedMap); err != nil {
		return nil, fmt.Errorf("encoding %s: %w", filepath, err)
	}
	enc.Close()

	return &buf, nil
}

// decodeSubstituted never shows the substituted request: its snippet holds resolved secrets and its position is not the file.
func decodeSubstituted[T any](filePath string, buf *bytes.Buffer) (T, error) {
	var out T
	err := yaml.NewDecoder(buf).Decode(&out)
	if err == nil {
		return out, nil
	}
	var raw T
	if rawErr := rawDecodeError(filePath, &raw); rawErr != nil {
		return out, fmt.Errorf("%s: %s", filePath, yaml.FormatError(rawErr, false, true))
	}
	var yamlErr yaml.Error
	if errors.As(err, &yamlErr) {
		return out, fmt.Errorf("%s: %s", filePath, yamlErr.GetMessage())
	}
	return out, fmt.Errorf("%s: %w", filePath, err)
}

// rawDecodeError decodes the file as written, keys lowercased, and returns nil unless that decode fails.
func rawDecodeError(filePath string, out any) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}
	file, err := parser.ParseBytes(content, 0)
	if err != nil {
		return nil
	}
	for _, doc := range file.Docs {
		if doc.Body == nil {
			continue
		}
		if _, ok := doc.Body.(*ast.DirectiveNode); ok {
			continue
		}
		ast.Walk(lowerKeys{}, doc.Body)
		return yaml.NodeToValue(doc.Body, out)
	}
	return nil
}

type lowerKeys struct{}

func (lowerKeys) Visit(node ast.Node) ast.Visitor {
	if mv, ok := node.(*ast.MappingValueNode); ok {
		if key, ok := mv.Key.(*ast.StringNode); ok {
			key.Value = strings.ToLower(key.Value)
		}
	}
	return lowerKeys{}
}

// FinalStructForAPI builds a final struct for the api call.
// Returns APICallFile struct, true if file is valid, and error
// It  checks the validity of all the fields in the yaml file meant for regular api call
func FinalStructForAPI(filePath string, secretsMap map[string]any) (APICallFile, bool, error) {
	buf, err := checkYamlFile(filePath, secretsMap)
	if err != nil {
		return APICallFile{}, false, err
	}

	file, err := decodeSubstituted[APICallFile](filePath, buf)
	if err != nil {
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
		return AuthRequestFile{}, fmt.Errorf("error after reading yaml file: %w", err)
	}

	auth2Config, err := decodeSubstituted[AuthRequestFile](filePath, buf)
	if err != nil {
		return AuthRequestFile{}, fmt.Errorf("error decoding data: %w", err)
	}

	if valid, err := auth2Config.IsValid(); !valid {
		return AuthRequestFile{}, fmt.Errorf("error on auth2 request body: %w", err)
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

	file, err := decodeSubstituted[APICallFile](filePath, buf)
	if err != nil {
		return APICallFile{}, false, err
	}

	if valid, err := file.IsValidForGraphQL(filePath); !valid {
		return APICallFile{}, false, err
	}

	return file, true, nil
}
