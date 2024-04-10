package envparser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateDefaultEnvs(t *testing.T) {
	// Setup temporary directory as project root
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	tempDir, err := os.MkdirTemp("", "envTest")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Ensure we return to the original directory and remove tempDir when done
	defer func() {
		err := os.Chdir(oldDir) // return to the original directory
		if err != nil {
			t.Fatal(err)
		}
		os.RemoveAll(tempDir) // cleanup: remove the temporary directory
	}()

	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	envDirPath := filepath.Join(tempDir, "env")

	tests := []struct {
		name     string
		envName  *string
		expected string
	}{
		{"default env", nil, "global.env"},
		{"custom env", stringPointer("custom"), "custom.env"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := CreateDefaultEnvs(tc.envName)
			if err != nil {
				t.Errorf("Failed to create env files: %v", err)
			}
			envFilePath := filepath.Join(envDirPath, tc.expected)
			if _, err := os.Stat(envFilePath); os.IsNotExist(err) {
				t.Errorf("Expected file '%s' was not created", tc.expected)
			}
		})
	}
}

func stringPointer(s string) *string {
	return &s
}
