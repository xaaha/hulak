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
