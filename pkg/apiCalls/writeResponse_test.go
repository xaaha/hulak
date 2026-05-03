package apicalls

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

			if err := evalAndWriteRes(tc.resBody, filePath); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

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

// TestEvalAndWriteRes_ReadOnlyDir is a regression for #205 — write failures
// used to be silently swallowed, leaving the user with a "success" outcome
// and no response file on disk. Skipped on Windows because chmod 0o555 on a
// directory does not block writes there the way it does on POSIX.
func TestEvalAndWriteRes_ReadOnlyDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("read-only dir semantics differ on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root bypasses dir mode bits")
	}

	tempDir := t.TempDir()
	if err := os.Chmod(tempDir, 0o555); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(tempDir, 0o755) })

	err := evalAndWriteRes(`{"ok":true}`, filepath.Join(tempDir, "req"))
	if err == nil {
		t.Fatal("expected error writing to read-only dir, got nil")
	}
	if !strings.Contains(err.Error(), "saving response") {
		t.Errorf("expected error to wrap with 'saving response', got %v", err)
	}
}
