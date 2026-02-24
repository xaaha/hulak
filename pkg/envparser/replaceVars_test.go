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
				"key \"naa\" not found",
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
				"key \"unresolvedKey\" not found",
			),
			varMap: map[string]any{},
		},
		{
			name:           "Empty map with multiple templates",
			stringToChange: "{{.varName}} is missing, so is {{.secondName}}",
			expectedOutput: nil,
			expectedErr: errors.New(
				"key \"varName\" not found",
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
			if (err == nil) != (tc.expectedErr == nil) ||
				(err != nil && tc.expectedErr != nil && !strings.Contains(err.Error(), tc.expectedErr.Error())) {
				t.Errorf("Error mismatch: expected %v, got %v", tc.expectedErr, err)
			}
		})
	}
}

func TestExtractMissingKey(t *testing.T) {
	testCases := []struct {
		name        string
		err         error
		expectedKey string
	}{
		{
			name:        "Standard missing key error",
			err:         errors.New(`template: template:1:2: executing "template" at <.graphqlUrl>: map has no entry for key "graphqlUrl"`),
			expectedKey: "graphqlUrl",
		},
		{
			name:        "Missing key with different name",
			err:         errors.New(`template: template:1:2: executing "template" at <.apiKey>: map has no entry for key "apiKey"`),
			expectedKey: "apiKey",
		},
		{
			name:        "Non-missing key error",
			err:         errors.New("some other template error"),
			expectedKey: "",
		},
		{
			name:        "Nil error",
			err:         nil,
			expectedKey: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractMissingKey(tc.err)
			if result != tc.expectedKey {
				t.Errorf("Expected key '%s', got '%s'", tc.expectedKey, result)
			}
		})
	}
}

func TestFormatMissingKeyError(t *testing.T) {
	// Set up test environment
	originalEnv := os.Getenv("hulakEnv")
	defer func() {
		if originalEnv != "" {
			os.Setenv("hulakEnv", originalEnv)
		} else {
			os.Unsetenv("hulakEnv")
		}
	}()

	testCases := []struct {
		name          string
		keyName       string
		envValue      string
		expectedInMsg []string
	}{
		{
			name:     "Missing key in global environment",
			keyName:  "graphqlUrl",
			envValue: "global",
			expectedInMsg: []string{
				`key "graphqlUrl" not found`,
				`environment "global"`,
				`Add "graphqlUrl=<value>" to env/global.env`,
			},
		},
		{
			name:     "Missing key in custom environment",
			keyName:  "apiKey",
			envValue: "prod",
			expectedInMsg: []string{
				`key "apiKey" not found`,
				`environment "prod"`,
				`Add "apiKey=<value>" to env/prod.env`,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			os.Setenv("hulakEnv", tc.envValue)

			err := formatMissingKeyError(tc.keyName)
			if err == nil {
				t.Fatal("Expected error, got nil")
			}

			errMsg := err.Error()
			for _, expectedStr := range tc.expectedInMsg {
				if !strings.Contains(errMsg, expectedStr) {
					t.Errorf("Expected error message to contain '%s', got:\n%s", expectedStr, errMsg)
				}
			}
		})
	}
}

func TestSubstituteVariablesWithImprovedErrors(t *testing.T) {
	testCases := []struct {
		name             string
		stringToChange   string
		varMap           map[string]any
		env              string
		expectedErrParts []string
	}{
		{
			name:           "Missing key shows helpful error",
			stringToChange: "{{.missingKey}}",
			varMap:         map[string]any{},
			env:            "global",
			expectedErrParts: []string{
				`key "missingKey" not found`,
				`environment "global"`,
				`Add "missingKey=<value>"`,
			},
		},
		{
			name:           "Missing key in prod environment",
			stringToChange: "https://api.example.com/{{.endpoint}}",
			varMap:         map[string]any{},
			env:            "prod",
			expectedErrParts: []string{
				`key "endpoint" not found`,
				`environment "prod"`,
				`Add "endpoint=<value>" to env/prod.env`,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set environment
			if tc.env != "" {
				os.Setenv("hulakEnv", tc.env)
				defer os.Unsetenv("hulakEnv")
			}

			_, err := SubstituteVariables(tc.stringToChange, tc.varMap)

			if err == nil {
				t.Fatal("Expected error, got nil")
			}

			errMsg := err.Error()
			for _, expectedPart := range tc.expectedErrParts {
				if !strings.Contains(errMsg, expectedPart) {
					t.Errorf("Expected error to contain '%s', got:\n%s", expectedPart, errMsg)
				}
			}
		})
	}
}
