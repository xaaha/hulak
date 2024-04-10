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
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)
	defer os.RemoveAll(tempDir)

	// Test Cases
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
			envDirPath := filepath.Join(tempDir, "env")
			envFilePath := filepath.Join(envDirPath, tc.expected)
			if _, err := os.Stat(envFilePath); os.IsNotExist(err) {
				t.Errorf("Expected file '%s' was not created", tc.expected)
			}
		})
	}

	// TODO: Cleanup && FIX THIS
	defer func() {
		os.RemoveAll(envDirPath) // Remove the test 'env' directory
	}()
}

func stringPointer(s string) *string {
	return &s
}
