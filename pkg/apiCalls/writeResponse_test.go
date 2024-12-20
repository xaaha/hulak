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
		// not sure about html parser and test
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
			filePath := filepath.Join(tempDir, "test")

			_ = evalAndWriteRes(tc.resBody, filePath)

			expectedFileName := "test_response" + tc.expectedExt
			expectedPath := filepath.Join(tempDir, expectedFileName)
			if _, err := os.Stat(expectedPath); err != nil {
				t.Errorf("Expected file %s to be created, but it was not", expectedPath)
			}

			_ = os.Remove(expectedPath)
		})
	}

	t.Run("Invalid inputs should not create files", func(t *testing.T) {
		err := evalAndWriteRes("", "")
		if err == nil {
			t.Fatal("Expected Error but did not get it")
		}
	})
}
