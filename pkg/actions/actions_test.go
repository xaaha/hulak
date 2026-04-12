package actions

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xaaha/hulak/pkg/utils"
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
	if err := os.WriteFile(objectFilePath, []byte(objectJSON), 0600); err != nil {
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
	if err := os.WriteFile(arrayFilePath, []byte(arrayJSON), 0600); err != nil {
		t.Fatalf("Failed to write array test file: %v", err)
	}

	// Create an invalid JSON file for testing error cases
	invalidJSON := `{ "invalid": json }`
	invalidFilePath := filepath.Join(tmpDir, "invalid.json")
	if err := os.WriteFile(invalidFilePath, []byte(invalidJSON), 0600); err != nil {
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
			want:     2,
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
			want:     81691643,
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

func TestBasicAuth(t *testing.T) {
	tests := []struct {
		name     string
		username string
		password string
		want     string
	}{
		{
			name:     "standard credentials",
			username: "admin",
			password: "secret123",
			want:     "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:secret123")),
		},
		{
			name:     "empty username",
			username: "",
			password: "secret",
			want:     "Basic " + base64.StdEncoding.EncodeToString([]byte(":secret")),
		},
		{
			name:     "empty password",
			username: "admin",
			password: "",
			want:     "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:")),
		},
		{
			name:     "both empty",
			username: "",
			password: "",
			want:     "Basic " + base64.StdEncoding.EncodeToString([]byte(":")),
		},
		{
			name:     "special characters in password",
			username: "user@domain.com",
			password: "p@ss:w0rd/with=special+chars",
			want:     "Basic " + base64.StdEncoding.EncodeToString([]byte("user@domain.com:p@ss:w0rd/with=special+chars")),
		},
		{
			name:     "unicode characters",
			username: "usuario",
			password: "contraseña",
			want:     "Basic " + base64.StdEncoding.EncodeToString([]byte("usuario:contraseña")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BasicAuth(tt.username, tt.password)
			if got != tt.want {
				t.Errorf("BasicAuth() = %q, want %q", got, tt.want)
			}
			if !strings.HasPrefix(got, "Basic ") {
				t.Errorf("BasicAuth() should start with 'Basic ', got %q", got)
			}
		})
	}
}

// setupHulakProject creates a temp directory with env/ to simulate a hulak project,
// changes into it, and returns a cleanup function that restores the original cwd.
func setupHulakProject(t *testing.T) string {
	t.Helper()

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}

	tmpDir := t.TempDir()
	// Resolve symlinks (macOS /var -> /private/var) so paths are consistent
	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatalf("failed to resolve symlinks: %v", err)
	}

	if err := os.Mkdir(filepath.Join(tmpDir, utils.EnvironmentFolder), utils.DirPer); err != nil {
		t.Fatalf("failed to create env dir: %v", err)
	}

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir to temp dir: %v", err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Fatal(err)
		}
	})

	return tmpDir
}

func TestGetFile(t *testing.T) {
	projectDir := setupHulakProject(t)

	// Create test files within the project
	testContent := "hello world\nline two\n"
	testFile := filepath.Join(projectDir, "testfile.txt")
	if err := os.WriteFile(testFile, []byte(testContent), utils.FilePer); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	subDir := filepath.Join(projectDir, "subdir")
	if err := os.Mkdir(subDir, utils.DirPer); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}
	nestedFile := filepath.Join(subDir, "nested.txt")
	nestedContent := "nested content"
	if err := os.WriteFile(nestedFile, []byte(nestedContent), utils.FilePer); err != nil {
		t.Fatalf("failed to write nested file: %v", err)
	}

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty path returns error",
			input:   "",
			wantErr: true,
			errMsg:  "file path cannot be empty",
		},
		{
			name:  "reads file with absolute path",
			input: testFile,
			want:  testContent,
		},
		{
			name:  "reads file with relative path",
			input: "testfile.txt",
			want:  testContent,
		},
		{
			name:  "reads nested file with relative path",
			input: "subdir/nested.txt",
			want:  nestedContent,
		},
		{
			name:    "rejects path outside project root",
			input:   "/etc/hosts",
			wantErr: true,
			errMsg:  "access denied",
		},
		{
			name:    "rejects directory path",
			input:   "subdir",
			wantErr: true,
			errMsg:  "is a directory",
		},
		{
			name:    "nonexistent file returns error",
			input:   "does_not_exist.txt",
			wantErr: true,
			errMsg:  "file does not exist",
		},
		{
			name:  "reads file with absolute path in nested dir",
			input: nestedFile,
			want:  nestedContent,
		},
		{
			name:    "path traversal outside project is rejected",
			input:   "../../../etc/passwd",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetFile(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("GetFile(%q) expected error, got nil", tt.input)
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("GetFile(%q) error = %q, want it to contain %q", tt.input, err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Errorf("GetFile(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("GetFile(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestGetFile_NotHulakProject(t *testing.T) {
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}

	tmpDir := t.TempDir()
	// No env/ directory — not a hulak project
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Fatal(err)
		}
	}()

	_, err = GetFile("somefile.txt")
	if err == nil {
		t.Error("GetFile() expected error outside hulak project, got nil")
	}
	if !strings.Contains(err.Error(), "not a hulak project") {
		t.Errorf("GetFile() error = %q, want it to contain 'not a hulak project'", err.Error())
	}
}

func TestGetFile_PreservesFormatting(t *testing.T) {
	projectDir := setupHulakProject(t)

	content := "{\n  \"key\": \"value\",\n  \"nested\": {\n    \"arr\": [1, 2, 3]\n  }\n}\n"
	filePath := filepath.Join(projectDir, "formatted.json")
	if err := os.WriteFile(filePath, []byte(content), utils.FilePer); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	got, err := GetFile("formatted.json")
	if err != nil {
		t.Fatalf("GetFile() unexpected error: %v", err)
	}
	if got != content {
		t.Errorf("GetFile() did not preserve formatting.\ngot:\n%s\nwant:\n%s", got, content)
	}
}
