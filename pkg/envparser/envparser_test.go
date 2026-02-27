package envparser

import (
	"os"
	"testing"
)

func TestTrimQuotes(t *testing.T) {
	testCases := []struct {
		output     any
		input      string
		wasTrimmed bool
	}{
		{input: "", output: "", wasTrimmed: false},
		{input: `"test's value"`, output: "test's value", wasTrimmed: true},
		{input: `"userNam2"`, output: "userNam2", wasTrimmed: true},
		{input: `22`, output: `22`, wasTrimmed: false},
		{input: `"false"`, output: `false`, wasTrimmed: true},
		{input: `199.289`, output: `199.289`, wasTrimmed: false},
		{input: `"199.289"`, output: `199.289`, wasTrimmed: true},
	}

	for _, tc := range testCases {
		resultStr, wasTrimmed := trimQuotes(tc.input)
		if resultStr != tc.output {
			t.Errorf(
				"Expected output does not match the result: \n%v \nvs \n%v",
				tc.output,
				resultStr,
			)
		}
		if tc.wasTrimmed != wasTrimmed {
			t.Errorf("Expected wasTrimmed to be %t but got %t", tc.wasTrimmed, wasTrimmed)
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
	if _, err := file.WriteString(content); err != nil {
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
