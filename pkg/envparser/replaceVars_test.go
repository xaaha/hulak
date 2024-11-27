package envparser

import (
	"errors"
	"strings"
	"testing"
)

func TestSubstituteVariables(t *testing.T) {
	varMap := map[string]string{
		"varName":       "replacedValue",
		"secondName":    "anju",
		"thirdName":     "pratik",
		"anotherNumber": "5678",
		"xaaha":         "hero",
		"number":        "1234{{.anotherNumber}}",
	}

	testCases := []struct {
		name           string
		stringToChange string
		expectedOutput string
		expectedErr    error
		varMap         map[string]string
	}{
		{
			name:           "Valid variables with nested replacement",
			stringToChange: "this/is/a/{{.varName}}/with/{{.number}}/{{.xaaha}}",
			expectedOutput: "this/is/a/replacedValue/with/12345678/hero",
			expectedErr:    nil,
			varMap:         varMap,
		},
		{
			name:           "String without variables",
			stringToChange: "a string without any curly braces",
			expectedOutput: "a string without any curly braces",
			expectedErr:    nil,
			varMap:         varMap,
		},
		{
			name:           "Unresolved variable",
			stringToChange: "1234 comes before {{.naa}}",
			expectedOutput: "",
			expectedErr: errors.New(
				"map has no entry for key \"naa\"",
			),
			varMap: varMap,
		},
		{
			name:           "Empty string input",
			stringToChange: "",
			expectedOutput: "",
			expectedErr:    errors.New("input string is empty"),
			varMap:         varMap,
		},
		{
			name:           "Empty map and empty string",
			stringToChange: "",
			expectedOutput: "",
			expectedErr:    errors.New("input string is empty"),
			varMap:         map[string]string{},
		},
		{
			name:           "Empty map with regular string",
			stringToChange: "just a normal string",
			expectedOutput: "just a normal string",
			expectedErr:    nil,
			varMap:         map[string]string{},
		},
		{
			name:           "Empty map with unresolved template",
			stringToChange: "this string has {{.unresolvedKey}}",
			expectedOutput: "",
			expectedErr: errors.New(
				"map has no entry for key \"unresolvedKey\"",
			),
			varMap: map[string]string{},
		},
		{
			name:           "Empty map with multiple templates",
			stringToChange: "{{.varName}} is missing, so is {{.secondName}}",
			expectedOutput: "",
			expectedErr: errors.New(
				"map has no entry for key \"varName\"",
			),
			varMap: map[string]string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := SubstituteVariables(tc.stringToChange, tc.varMap)

			// Compare output
			if output != tc.expectedOutput {
				t.Errorf("Output mismatch: expected %v, got %v", tc.expectedOutput, output)
			}

			// Compare errors
			if (err == nil && tc.expectedErr == nil) ||
				(err != nil && tc.expectedErr != nil && strings.Contains(err.Error(), tc.expectedErr.Error())) {
				// Errors match; no action needed
			} else {
				t.Errorf("Error mismatch: expected %v, got %v", tc.expectedErr, err)
			}
		})
	}
}
