// Package envparser contains environment parsing and functions around it
package envparser

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/xaaha/hulak/pkg/actions"
)

func replaceVariables(
	strToChange string,
	secretsMap map[string]any,
) (string, error) {
	if len(strToChange) == 0 {
		return "", nil
	}

	funcMap := template.FuncMap{
		"getValueOf": func(key, fileName string) any {
			return actions.GetValueOf(key, fileName)
		},
		"getFile": func(fileName string) (string, error) {
			return actions.GetFile(fileName)
		},
	}

	tmpl, err := template.New("template").
		Funcs(funcMap).
		Option("missingkey=error").
		Parse(strToChange)
	if err != nil {
		return "", err
	}
	var result bytes.Buffer
	err = tmpl.Execute(&result, secretsMap)
	if err != nil {
		return "", err
	}
	return result.String(), nil
}

// Iterates over a map of secret values (secretsMap), resolving any string values
// containing template variables using the replaceVariables function.
// It ensures that non-string values (e.g., booleans, integers) are preserved and validates against unsupported types.
// Returns a new map with resolved values or an error if any resolution fails.
func prepareMap(secretsMap map[string]any) (map[string]any, error) {
	updatedMap := make(map[string]any)
	for key, val := range secretsMap {
		switch v := val.(type) {
		case string:
			changedValue, err := replaceVariables(v, secretsMap)
			if err != nil {
				return nil, err
			}
			updatedMap[key] = changedValue
		case bool, int, float64, nil:
			updatedMap[key] = v
		default:
			return nil, fmt.Errorf("unsupported type for key '%s': %T", key, val)
		}
	}
	return updatedMap, nil
}

// SubstituteVariables Substitutes template variables in a given string strToChange using the secretsMap.
// It first prepares the map by resolving all nested variables using prepareMap
// and then applies replaceVariables to the input string.
// Returns the final substituted string or an error if any step fails.
func SubstituteVariables(
	strToChange string,
	secretsMap map[string]any,
) (any, error) {
	finalMap, err := prepareMap(secretsMap)
	if err != nil {
		return nil, err
	}

	result, err := replaceVariables(strToChange, finalMap)
	if err != nil {
		return nil, err
	}

	return result, nil
}
