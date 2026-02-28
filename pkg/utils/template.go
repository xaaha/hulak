package utils

import (
	"os"
	"regexp"
	"slices"
	"strings"
)

var templateVarPattern = regexp.MustCompile(`\{\{\s*\.`)

// FileHasTemplateVars checks if a file contains Go template variable references
// (e.g., {{.token}}) that require environment variable resolution.
// It does NOT match {{getFile ...}} or {{getValueOf ...}} which work without env secrets.
func FileHasTemplateVars(filePath string) bool {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}
	return templateVarPattern.Match(content)
}

// MapHasEnvVars recursively checks if any string value in the map
// contains "{{." which indicates an env variable reference.
func MapHasEnvVars(data map[string]any) bool {
	for _, val := range data {
		if hasEnvVar(val) {
			return true
		}
	}
	return false
}

func hasEnvVar(val any) bool {
	switch v := val.(type) {
	case string:
		return strings.Contains(v, "{{.")
	case map[string]any:
		return MapHasEnvVars(v)
	case []any:
		if slices.ContainsFunc(v, hasEnvVar) {
			return true
		}
	}
	return false
}
