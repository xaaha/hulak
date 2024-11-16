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
	}{
		{
			name:           "Valid variables with nested replacement",
			stringToChange: "this/is/a/{{.varName}}/with/{{.number}}/{{.xaaha}}",
			expectedOutput: "this/is/a/replacedValue/with/12345678/hero",
			expectedErr:    nil,
		},
		{
			name:           "String without variables",
			stringToChange: "a string without any curly braces",
			expectedOutput: "a string without any curly braces",
			expectedErr:    nil,
		},
		{
			name:           "Unresolved variable",
			stringToChange: "1234 comes before {{.naa}}",
			expectedOutput: "",
			expectedErr: errors.New(
				"map has no entry for key \"naa\"",
			),
		},
		{
			name:           "Empty string input",
			stringToChange: "",
			expectedOutput: "",
			expectedErr:    errors.New("input string is empty"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := SubstituteVariables(tc.stringToChange, varMap)

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
