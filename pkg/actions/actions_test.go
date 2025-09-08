package actions

import (
	"os"
	"path/filepath"
	"testing"
)

func Test_processValueOf(t *testing.T) {
	// Create temporary test files
	tmpDir, err := os.MkdirTemp("", "hulak-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test JSON file with an object at root
	objectJSON := `{
		"name": "xaaha",
		"age": 30,
		"nested": {
			"key": "value",
			"array": [1, 2, 3]
		}
	}`
	objectFilePath := filepath.Join(tmpDir, "object.json")
	if err := os.WriteFile(objectFilePath, []byte(objectJSON), 0644); err != nil {
		t.Fatalf("Failed to write object test file: %v", err)
	}

	// Create a test JSON file with an array at root
	arrayJSON := `[
		{
			"access_token": "pratik",
			"refresh_token": "",
			"id_token": "",
			"scope": "",
			"expires_in": 86400,
			"token_type": "Bearer"
		},
		{
			"access_token": "thapa",
			"refresh_token": "",
			"id_token": "",
			"scope": "",
			"expires_in": 81691643,
			"token_type": "Bearer"
		}
	]`
	arrayFilePath := filepath.Join(tmpDir, "array.json")
	if err := os.WriteFile(arrayFilePath, []byte(arrayJSON), 0644); err != nil {
		t.Fatalf("Failed to write array test file: %v", err)
	}

	// Create an invalid JSON file for testing error cases
	invalidJSON := `{ "invalid": json }`
	invalidFilePath := filepath.Join(tmpDir, "invalid.json")
	if err := os.WriteFile(invalidFilePath, []byte(invalidJSON), 0644); err != nil {
		t.Fatalf("Failed to write invalid test file: %v", err)
	}

	tests := []struct {
		name     string // description of this test case
		key      string
		fileName string
		want     any
	}{
		// Object JSON tests
		{
			name:     "Get direct property from object",
			key:      "name",
			fileName: objectFilePath,
			want:     "xaaha",
		},
		{
			name:     "Get nested property from object",
			key:      "nested.key",
			fileName: objectFilePath,
			want:     "value",
		},
		{
			name:     "Get array element from nested property in object",
			key:      "nested.array[1]",
			fileName: objectFilePath,
			want:     float64(2), // JSON numbers are decoded as float64
		},
		{
			name:     "Get nonexistent property from object",
			key:      "nonexistent",
			fileName: objectFilePath,
			want:     "",
		},

		// Array JSON tests
		{
			name:     "Get property from array element",
			key:      "[0].access_token",
			fileName: arrayFilePath,
			want:     "pratik",
		},
		{
			name:     "Get property from second array element",
			key:      "[1].access_token",
			fileName: arrayFilePath,
			want:     "thapa",
		},
		{
			name:     "Get expires_in from second array element",
			key:      "[1].expires_in",
			fileName: arrayFilePath,
			want:     float64(81691643), // JSON numbers are decoded as float64
		},
		{
			name:     "Use invalid syntax for array (missing brackets)",
			key:      "0.access_token",
			fileName: arrayFilePath,
			want:     "",
		},
		{
			name:     "Access out of bounds array index",
			key:      "[2].access_token",
			fileName: arrayFilePath,
			want:     "",
		},

		// Error cases
		{
			name:     "Empty key",
			key:      "",
			fileName: objectFilePath,
			want:     "",
		},
		{
			name:     "Empty filename",
			key:      "name",
			fileName: "",
			want:     "",
		},
		{
			name:     "Nonexistent file",
			key:      "name",
			fileName: filepath.Join(tmpDir, "nonexistent.json"),
			want:     "",
		},
		{
			name:     "Invalid JSON file",
			key:      "invalid",
			fileName: invalidFilePath,
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := processValueOf(tt.key, tt.fileName)

			// Compare results
			if got != tt.want {
				t.Errorf("processValueOf() = %v, want %v", got, tt.want)
			}
		})
	}
}
