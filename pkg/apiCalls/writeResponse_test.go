package apicalls

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEvalAndWriteRes(t *testing.T) {
	tempDir := t.TempDir() // Create a temporary directory for tests

	tests := []struct {
		name         string
		resBody      string
		expectedFile string
		expectedExt  string
	}{
		{
			name:         "JSON file creation",
			resBody:      `{"key": "value"}`,
			expectedFile: "test.json",
			expectedExt:  ".json",
		},
		{
			name:         "XML file creation",
			resBody:      `<root><key>value</key></root>`,
			expectedFile: "test.xml",
			expectedExt:  ".xml",
		},
		// {
		// 	name:         "HTML file creation",
		// 	resBody:      `<html><body>Hello</body></html>`,
		// 	expectedFile: "test.html",
		// 	expectedExt:  ".html",
		// },
		{
			name:         "Plain text file creation",
			resBody:      "This is plain text",
			expectedFile: "test.txt",
			expectedExt:  ".txt",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Define file path
			filePath := filepath.Join(tempDir, "test")

			// Call the function to evaluate and write the response
			evalAndWriteRes(tc.resBody, filePath)

			// Verify the file with the correct extension is created
			expectedPath := filepath.Join(tempDir, tc.expectedFile)
			if _, err := os.Stat(expectedPath); err != nil {
				t.Errorf("Expected file %s to be created, but it was not", expectedPath)
			}

			// Clean up after test
			_ = os.Remove(expectedPath)
		})
	}

	// t.Run("Invalid inputs should not create files", func(t *testing.T) {
	// 	evalAndWriteRes("", "") // Invalid input
	//
	// 	// Check if no files are created
	// 	createdFiles, err := os.ReadDir(tempDir)
	// 	if err != nil {
	// 		t.Fatalf("Failed to read directory: %v", err)
	// 	}
	// 	if len(createdFiles) != 0 {
	// 		t.Errorf("Expected no files to be created, but found %d files", len(createdFiles))
	// 	}
	// })
}
