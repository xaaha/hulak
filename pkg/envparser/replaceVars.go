// Package envparser contains environment parsing and functions around it
package envparser

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/xaaha/hulak/pkg/actions"
	"github.com/xaaha/hulak/pkg/utils"
)

// extractMissingKey parses a template error and extracts the missing key name
// Returns the key name if found, empty string otherwise
func extractMissingKey(err error) string {
	if err == nil {
		return ""
	}
	errMsg := err.Error()
	// Template errors for missing keys look like:
	// "template: template:1:2: executing "template" at <.keyName>: map has no entry for key "keyName""

	// Look for the pattern: map has no entry for key "keyName"
	if strings.Contains(errMsg, "map has no entry for key") {
		start := strings.Index(errMsg, `map has no entry for key "`)
		if start != -1 {
			start += len(`map has no entry for key "`)
			end := strings.Index(errMsg[start:], `"`)
			if end != -1 {
				return errMsg[start : start+end]
			}
		}
	}
	return ""
}

// formatMissingKeyError creates a user-friendly error message for missing template variables
func formatMissingKeyError(keyName string) error {
	env := os.Getenv(utils.EnvKey)
	if env == "" {
		env = utils.DefaultEnvVal
	}

	errMsg := fmt.Sprintf(
		`key "%s" not found in environment "%s"

Possible solutions:
  - Add "%s=<value>" to env/%s.env
  - Use a different environment: hulak -env <environment-name>`,
		keyName,
		env,
		keyName,
		env,
	)

	return fmt.Errorf("%s", errMsg)
}

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
		// Check if this is a missing key error and format it nicely
		if missingKey := extractMissingKey(err); missingKey != "" {
			return "", formatMissingKeyError(missingKey)
		}
		// For other template errors, return as-is
		return "", fmt.Errorf("template error: %w", err)
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
