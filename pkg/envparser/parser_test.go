package envparser

import (
	"os"
	"testing"
)

func TestTrimQuotes(t *testing.T) {
	// it's a bit tricky to test since trim quotes trims the quotest the value from env file
	testCases := []struct {
		input  string
		output string
	}{
		{input: "", output: ""},
		{input: "test's value", output: "test's value"},
		{input: "userNam2", output: "userNam2"},
	}

	for _, tc := range testCases {
		resultStr := trimQuotes(tc.input)
		if resultStr != tc.output {
			t.Errorf(
				"Expected output does not match the result: \n%v \nvs \n%v",
				tc.output,
				resultStr,
			)
		}
	}
}

/*
Fix the the mocking part for these damn test
*/

// type mockUtility struct{}
//
// func (m *mockUtility) GetEnvFiles() ([]string, error) {
// 	return []string{"development.env", "production.env", "staging.env"}, nil
// }
//
// func TestSetEnvFunction(t *testing.T) {
// 	utility := mockUtility{}
//
// 	tests := []struct {
// 		prepareEnvFunc func(w *io.PipeWriter)
// 		name           string
// 		envVar         string
// 		expectedEnv    string
// 		flagArgs       []string
// 		expectedSkip   bool
// 		expectedError  bool
// 	}{
// 		{
// 			name:          "Default environment",
// 			envVar:        "",
// 			flagArgs:      []string{},
// 			expectedEnv:   "default",
// 			expectedSkip:  false,
// 			expectedError: false,
// 		},
// 		{
// 			name:          "Provided flag is captured",
// 			envVar:        "",
// 			flagArgs:      []string{"-env", "production"},
// 			expectedEnv:   "production",
// 			expectedSkip:  false,
// 			expectedError: false,
// 		},
// 		{
// 			name:          "Ignores second argument",
// 			envVar:        "",
// 			flagArgs:      []string{"-env", "production", "staging"},
// 			expectedEnv:   "production",
// 			expectedSkip:  false,
// 			expectedError: false,
// 		},
// 		{
// 			name:          "'env' file does not exist and creation is skipped",
// 			envVar:        "",
// 			flagArgs:      []string{"-env", "nonexistent"},
// 			expectedEnv:   "default",
// 			expectedSkip:  true,
// 			expectedError: false,
// 			prepareEnvFunc: func(w *io.PipeWriter) {
// 				_, err := w.Write([]byte("n\n"))
// 				if err != nil {
// 					t.Errorf("Error while running test for setEnvironment %v", err)
// 				}
// 				w.Close()
// 			},
// 		},
// 		{
// 			name:          "Environment variable is already set",
// 			envVar:        "staging",
// 			flagArgs:      []string{},
// 			expectedEnv:   "staging",
// 			expectedSkip:  false,
// 			expectedError: false,
// 		},
// 	}
//
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			// Prepare environment variable if necessary
// 			if tt.envVar != "" {
// 				os.Setenv(utils.EnvKey, tt.envVar)
// 			} else {
// 				os.Unsetenv(utils.EnvKey)
// 			}
//
// 			// Prepare function for mocking user input if needed
// 			_, w := io.Pipe()
// 			if tt.prepareEnvFunc != nil {
// 				go tt.prepareEnvFunc(w)
// 			}
//
// 			// Set flags
// 			flag.CommandLine = flag.NewFlagSet(tt.name, flag.ContinueOnError)
// 			os.Args = append([]string{"cmd"}, tt.flagArgs...)
// 			fileCreationSkipped, err := setEnvironment(utils.Utilities(utility))
//
// 			// Assert results
// 			if tt.expectedError {
// 				if err == nil {
// 					t.Errorf("expected an error but got nil")
// 				}
// 			} else {
// 				if err != nil {
// 					t.Errorf("expected no error but got %v", err)
// 				}
// 			}
//
// 			if fileCreationSkipped != tt.expectedSkip {
// 				t.Errorf(
// 					"expected fileCreationSkipped to be %v, got %v",
// 					tt.expectedSkip,
// 					fileCreationSkipped,
// 				)
// 			}
//
// 			if os.Getenv(utils.EnvKey) != tt.expectedEnv {
// 				t.Errorf("expected env to be %v, got %v", tt.expectedEnv, os.Getenv(utils.EnvKey))
// 			}
// 		})
// 	}
// }

// create a temporary file for tetsing
func createTempEnvFile(content string) (string, error) {
	file, err := os.CreateTemp("", "*.env")
	if err != nil {
		return "", err
	}
	defer file.Close()
	if _, err := file.Write([]byte(content)); err != nil {
		return "", err
	}
	return file.Name(), nil
}

func TestLoadEnvVars(t *testing.T) {
	content := `
# This is a comment
KEY1=value1
KEY2="value2"
KEY3='value3'
`

	filePath, err := createTempEnvFile(content)
	if err != nil {
		t.Fatalf("Failed to create temp env file: %v", err)
	}
	defer os.Remove(filePath)

	expected := map[string]string{
		"KEY1": "value1",
		"KEY2": "value2",
		"KEY3": "value3",
	}

	result, err := LoadEnvVars(filePath)
	if err != nil {
		t.Fatalf("LoadEnvVars returned error: %v", err)
	}

	if len(result) != len(expected) {
		t.Fatalf("Expected map length %d, got %d", len(expected), len(result))
	}

	for key, val := range expected {
		if result[key] != val {
			t.Errorf("Expected key %s to have value %s, got %s", key, val, result[key])
		}
	}
}

//
// func TestGenerateSecretsMap(t *testing.T) {
// 	// Mock the setEnvironment function
// 	originalSetEnvFunc := setEnvironment
// 	defer func() { setEnvironment = originalSetEnvFunc }()
// 	setEnvironment = func(utility utils.EnvUtility) (bool, error) {
// 		err := os.Setenv(utils.EnvKey, "custom")
// 		return false, err
// 	}
//
// 	globalContent := `
// GLOBAL_KEY1=global_value1
// GLOBAL_KEY2=global_value2
// `
// 	customContent := `
// CUSTOM_KEY1=custom_value1
// GLOBAL_KEY2=custom_value2
// `
//
// 	globalFilePath, err := createTempEnvFile(globalContent)
// 	if err != nil {
// 		t.Fatalf("Failed to create temp global env file: %v", err)
// 	}
// 	defer os.Remove(globalFilePath)
//
// 	customFilePath, err := createTempEnvFile(customContent)
// 	if err != nil {
// 		t.Fatalf("Failed to create temp custom env file: %v", err)
// 	}
// 	defer os.Remove(customFilePath)
//
// 	// Mock utils.CreateFilePath
// 	originalCreateFilePath := utils.CreateFilePath
// 	defer func() { utils.CreateFilePath = originalCreateFilePath }()
// 	utils.CreateFilePath = func(fileName string) (string, error) {
// 		if strings.Contains(fileName, "global.env") {
// 			return globalFilePath, nil
// 		}
// 		return customFilePath, nil
// 	}
//
// 	expected := map[string]string{
// 		"GLOBAL_KEY1": "global_value1",
// 		"GLOBAL_KEY2": "custom_value2",
// 		"CUSTOM_KEY1": "custom_value1",
// 	}
//
// 	result, err := GenerateSecretsMap()
// 	if err != nil {
// 		t.Fatalf("GenerateSecretsMap returned error: %v", err)
// 	}
//
// 	if len(result) != len(expected) {
// 		t.Fatalf("Expected map length %d, got %d", len(expected), len(result))
// 	}
//
// 	for key, val := range expected {
// 		if result[key] != val {
// 			t.Errorf("Expected key %s to have value %s, got %s", key, val, result[key])
// 		}
// 	}
// }
