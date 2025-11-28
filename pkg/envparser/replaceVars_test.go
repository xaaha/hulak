package envparser

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestHandleEnvVarValue(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		expectedOutput string
		setOSEnvFunc   func()
		cleanupFunc    func()
	}{
		{
			name:           "Value starts with $ and env var exists",
			input:          "$EXISTING_ENV_VAR",
			expectedOutput: "env_value",
			setOSEnvFunc: func() {
				_ = os.Setenv("EXISTING_ENV_VAR", "env_value")
			},
			cleanupFunc: func() {
				_ = os.Unsetenv("EXISTING_ENV_VAR")
			},
		},
		{
			name:           "Value starts with $ and env var does not exist",
			input:          "$NON_EXISTENT_ENV_VAR",
			expectedOutput: "",
			setOSEnvFunc:   func() {},
			cleanupFunc:    func() {},
		},
		{
			name:           "Value a standard string",
			input:          "regular_value",
			expectedOutput: "regular_value",
			setOSEnvFunc:   func() {},
			cleanupFunc:    func() {},
		},
		{
			name:           "Value is just $",
			input:          "$",
			expectedOutput: "",
			setOSEnvFunc:   func() {},
			cleanupFunc:    func() {},
		},
		{
			name:           "Value is empty string",
			input:          "",
			expectedOutput: "",
			setOSEnvFunc:   func() {},
			cleanupFunc:    func() {},
		},
		{
			name:           "Value a plaintext with $ inside",
			input:          "value_with_$inside",
			expectedOutput: "value_with_$inside",
			setOSEnvFunc:   func() {},
			cleanupFunc:    func() {},
		},
		{
			name:           "Value is only an env var name without $",
			input:          "EXISTING_ENV_VAR",
			expectedOutput: "EXISTING_ENV_VAR",
			setOSEnvFunc: func() {
				_ = os.Setenv("EXISTING_ENV_VAR", "env_value")
			},
			cleanupFunc: func() {
				_ = os.Unsetenv("EXISTING_ENV_VAR")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setOSEnvFunc()
			output := handleEnvVarValue(tc.input)
			if output != tc.expectedOutput {
				t.Errorf("Expected '%v', got '%v'", tc.expectedOutput, output)
			}
			tc.cleanupFunc()
		})
	}
}

func TestSubstituteVariables(t *testing.T) {
	varMap := map[string]any{
		"varName":       "replacedValue",
		"secondName":    "John",
		"thirdName":     "Doe",
		"anotherNumber": 5678, // int type
		"xaaha":         "hero",
		"number":        "1234{{.anotherNumber}}",
		"truthyValue":   true,  // bool type
		"floatValue":    12.34, // float64 type
	}

	testCases := []struct {
		expectedErr    error
		expectedOutput any
		varMap         map[string]any
		name           string
		stringToChange string
	}{
		{
			name:           "Valid variables with nested replacement",
			stringToChange: "this/is/a/{{.varName}}/with/{{.number}}/{{.xaaha}}",
			expectedOutput: "this/is/a/replacedValue/with/12345678/hero",
			expectedErr:    nil,
			varMap:         varMap,
		},
		{
			name:           "Variable with bool value",
			stringToChange: "This is {{.truthyValue}}",
			expectedOutput: "This is true",
			expectedErr:    nil,
			varMap:         varMap,
		},
		{
			name:           "Variable with float value",
			stringToChange: "Float is {{.floatValue}}",
			expectedOutput: "Float is 12.34",
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
			expectedOutput: nil,
			expectedErr: errors.New(
				"map has no entry for key \"naa\"",
			),
			varMap: varMap,
		},
		{
			name:           "Empty string input",
			stringToChange: "",
			expectedOutput: "",
			expectedErr:    nil,
			varMap:         varMap,
		},
		{
			name:           "Empty map and empty string",
			stringToChange: "",
			expectedOutput: "",
			expectedErr:    nil,
			varMap:         map[string]any{},
		},
		{
			name:           "Empty map with regular string",
			stringToChange: "just a normal string",
			expectedOutput: "just a normal string",
			expectedErr:    nil,
			varMap:         map[string]any{},
		},
		{
			name:           "Empty map with unresolved template",
			stringToChange: "this string has {{.unresolvedKey}}",
			expectedOutput: nil,
			expectedErr: errors.New(
				"map has no entry for key \"unresolvedKey\"",
			),
			varMap: map[string]any{},
		},
		{
			name:           "Empty map with multiple templates",
			stringToChange: "{{.varName}} is missing, so is {{.secondName}}",
			expectedOutput: nil,
			expectedErr: errors.New(
				"map has no entry for key \"varName\"",
			),
			varMap: map[string]any{},
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
