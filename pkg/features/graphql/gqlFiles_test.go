package graphql

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test helper functions

// setupTestDirectory creates a temporary directory for tests
func setupTestDirectory(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

// createGraphQLFile creates a test GraphQL YAML file with specified URL
func createGraphQLFile(t *testing.T, dir, filename, url string) string {
	t.Helper()

	var content string
	if url == "" {
		content = "---\nkind: GraphQL\nmethod: POST\n"
	} else {
		// Quote the URL value to ensure valid YAML
		content = fmt.Sprintf("---\nkind: GraphQL\nurl: \"%s\"\nmethod: POST\n", url)
	}

	filePath := filepath.Join(dir, filename)
	err := os.WriteFile(filePath, []byte(content), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	return filePath
}

// createYAMLFile creates a test YAML file with custom kind and optional URL
func createYAMLFile(t *testing.T, dir, filename, kind, url string) string {
	t.Helper()

	var content string
	if url == "" {
		content = fmt.Sprintf("---\nkind: %s\nmethod: POST\n", kind)
	} else {
		// Quote the URL value to ensure valid YAML
		content = fmt.Sprintf("---\nkind: %s\nurl: \"%s\"\nmethod: POST\n", kind, url)
	}

	filePath := filepath.Join(dir, filename)
	err := os.WriteFile(filePath, []byte(content), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	return filePath
}

// Tests for FindGraphQLFiles

func TestFindGraphQLFiles_Success(t *testing.T) {
	tempDir := setupTestDirectory(t)
	secretsMap := make(map[string]any)

	// Create test files
	createGraphQLFile(t, tempDir, "valid1.yaml", "http://example.com")
	createGraphQLFile(t, tempDir, "valid2.yml", "{{.baseUrl}}")
	createYAMLFile(t, tempDir, "api.yaml", "API", "http://api.com")
	createYAMLFile(t, tempDir, "auth.yaml", "Auth", "http://auth.com")

	files, err := FindGraphQLFiles(tempDir, secretsMap)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}

	// Verify the right files are included
	foundValid1 := false
	foundValid2 := false
	for _, f := range files {
		if strings.Contains(f, "valid1.yaml") {
			foundValid1 = true
		}
		if strings.Contains(f, "valid2.yml") {
			foundValid2 = true
		}
		// Ensure non-GraphQL files are not included
		if strings.Contains(f, "api.yaml") || strings.Contains(f, "auth.yaml") {
			t.Errorf("Non-GraphQL file should not be included: %s", f)
		}
	}

	if !foundValid1 || !foundValid2 {
		t.Errorf(
			"Expected GraphQL files not found. Found valid1: %v, valid2: %v",
			foundValid1,
			foundValid2,
		)
	}
}

func TestFindGraphQLFiles_WithTemplateURLs(t *testing.T) {
	tempDir := setupTestDirectory(t)
	secretsMap := make(map[string]any)

	// Create files with various template URLs
	createGraphQLFile(t, tempDir, "template1.yaml", "{{.baseUrl}}")
	createGraphQLFile(t, tempDir, "template2.yaml", "{{.endpoint}}/graphql")
	createGraphQLFile(t, tempDir, "actual.yaml", "https://api.example.com/graphql")

	files, err := FindGraphQLFiles(tempDir, secretsMap)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(files) != 3 {
		t.Errorf("Expected 3 files with template URLs, got %d", len(files))
	}
}

func TestFindGraphQLFiles_WithoutURL(t *testing.T) {
	tempDir := setupTestDirectory(t)
	secretsMap := make(map[string]any)

	// Create GraphQL file without URL
	createGraphQLFile(t, tempDir, "no_url.yaml", "")

	files, err := FindGraphQLFiles(tempDir, secretsMap)

	if err == nil {
		t.Errorf("Expected error for GraphQL file without URL")
	}
	if files != nil {
		t.Errorf("Expected nil files, got %d files", len(files))
	}
}

func TestFindGraphQLFiles_EmptyURL(t *testing.T) {
	tempDir := setupTestDirectory(t)
	secretsMap := make(map[string]any)

	// Create file with empty URL
	content := "---\nkind: GraphQL\nurl: \"\"\nmethod: POST\n"
	filePath := filepath.Join(tempDir, "empty_url.yaml")
	err := os.WriteFile(filePath, []byte(content), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	files, err := FindGraphQLFiles(tempDir, secretsMap)

	if err == nil {
		t.Errorf("Expected error for GraphQL file with empty URL")
	}
	if files != nil {
		t.Errorf("Expected nil files, got %d files", len(files))
	}
}

func TestFindGraphQLFiles_NoGraphQLFiles(t *testing.T) {
	tempDir := setupTestDirectory(t)
	secretsMap := make(map[string]any)

	// Create only non-GraphQL files
	createYAMLFile(t, tempDir, "api.yaml", "API", "http://api.com")
	createYAMLFile(t, tempDir, "auth.yaml", "Auth", "http://auth.com")

	files, err := FindGraphQLFiles(tempDir, secretsMap)

	if err == nil {
		t.Errorf("Expected error when no GraphQL files found")
	}
	if !strings.Contains(err.Error(), "no files with 'kind: GraphQL'") {
		t.Errorf("Expected 'no files' error message, got: %v", err)
	}
	if files != nil {
		t.Errorf("Expected nil files, got %d files", len(files))
	}
}

func TestFindGraphQLFiles_OnlyResponseFiles(t *testing.T) {
	tempDir := setupTestDirectory(t)
	secretsMap := make(map[string]any)

	// Create only response files
	content := "---\nkind: GraphQL\nurl: http://example.com\n"
	responseFile := filepath.Join(tempDir, "test_response.json")
	err := os.WriteFile(responseFile, []byte(content), 0o644)
	if err != nil {
		t.Fatalf("Failed to create response file: %v", err)
	}

	files, err := FindGraphQLFiles(tempDir, secretsMap)

	if err == nil {
		t.Errorf("Expected error when only response files present")
	}
	if files != nil {
		t.Errorf("Expected nil files, got %d files", len(files))
	}
}

func TestFindGraphQLFiles_EmptyDirectory(t *testing.T) {
	tempDir := setupTestDirectory(t)
	secretsMap := make(map[string]any)

	files, err := FindGraphQLFiles(tempDir, secretsMap)

	if err == nil {
		t.Errorf("Expected error for empty directory")
	}
	if files != nil {
		t.Errorf("Expected nil files, got %d files", len(files))
	}
}

func TestFindGraphQLFiles_MalformedYAML(t *testing.T) {
	tempDir := setupTestDirectory(t)
	secretsMap := make(map[string]any)

	// Create valid file and a truly malformed file
	createGraphQLFile(t, tempDir, "valid.yaml", "http://example.com")

	// Create a file that's truly unparseable
	malformedPath := filepath.Join(tempDir, "malformed.yaml")
	malformedContent := "---\nthis is not: valid: yaml: at: all:\n  - [\n"
	err := os.WriteFile(malformedPath, []byte(malformedContent), 0o644)
	if err != nil {
		t.Fatalf("error on writing file in TestFindGraphQLFiles_MalformedYAML()")
	}

	files, err := FindGraphQLFiles(tempDir, secretsMap)
	// Should find at least the valid file, malformed should be skipped
	if err != nil {
		t.Fatalf("Expected no error (malformed files should be skipped), got: %v", err)
	}
	if len(files) < 1 {
		t.Errorf("Expected at least 1 valid file, got %d", len(files))
	}
}

func TestFindGraphQLFiles_NestedDirectories(t *testing.T) {
	tempDir := setupTestDirectory(t)
	secretsMap := make(map[string]any)

	// Create nested directory structure
	subDir := filepath.Join(tempDir, "subdir")
	err := os.Mkdir(subDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Create files in both directories
	createGraphQLFile(t, tempDir, "root.yaml", "http://example.com")
	createGraphQLFile(t, subDir, "nested.yaml", "{{.baseUrl}}")

	files, err := FindGraphQLFiles(tempDir, secretsMap)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("Expected 2 files from nested directories, got %d", len(files))
	}
}

// Tests for ValidateGraphQLFile

func TestValidateGraphQLFile_Valid(t *testing.T) {
	tempDir := setupTestDirectory(t)
	secretsMap := make(map[string]any)

	filePath := createGraphQLFile(t, tempDir, "valid.yaml", "http://example.com")

	isValid, err := ValidateGraphQLFile(filePath, secretsMap)
	if err != nil {
		t.Errorf("Expected no error for valid file, got: %v", err)
	}
	if !isValid {
		t.Errorf("Expected isValid=true, got false")
	}
}

func TestValidateGraphQLFile_TemplateURL(t *testing.T) {
	tempDir := setupTestDirectory(t)
	secretsMap := make(map[string]any)

	filePath := createGraphQLFile(t, tempDir, "template.yaml", "{{.baseUrl}}")

	isValid, err := ValidateGraphQLFile(filePath, secretsMap)
	if err != nil {
		t.Errorf("Expected no error for template URL, got: %v", err)
	}
	if !isValid {
		t.Errorf("Expected isValid=true for template URL, got false")
	}
}

func TestValidateGraphQLFile_MissingURL(t *testing.T) {
	tempDir := setupTestDirectory(t)
	secretsMap := make(map[string]any)

	filePath := createGraphQLFile(t, tempDir, "no_url.yaml", "")

	isValid, err := ValidateGraphQLFile(filePath, secretsMap)

	if err == nil {
		t.Errorf("Expected error for missing URL")
	}
	if isValid {
		t.Errorf("Expected isValid=false, got true")
	}
	if !strings.Contains(err.Error(), "missing required 'url' field") {
		t.Errorf("Expected 'missing url' error message, got: %v", err)
	}
}

func TestValidateGraphQLFile_EmptyURL(t *testing.T) {
	tempDir := setupTestDirectory(t)
	secretsMap := make(map[string]any)

	content := "---\nkind: GraphQL\nurl: \"\"\nmethod: POST\n"
	filePath := filepath.Join(tempDir, "empty_url.yaml")
	err := os.WriteFile(filePath, []byte(content), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	isValid, err := ValidateGraphQLFile(filePath, secretsMap)

	if err == nil {
		t.Errorf("Expected error for empty URL")
	}
	if isValid {
		t.Errorf("Expected isValid=false, got true")
	}
}

func TestValidateGraphQLFile_WrongKind(t *testing.T) {
	tempDir := setupTestDirectory(t)
	secretsMap := make(map[string]any)

	filePath := createYAMLFile(t, tempDir, "api.yaml", "API", "http://example.com")

	isValid, err := ValidateGraphQLFile(filePath, secretsMap)

	if err == nil {
		t.Errorf("Expected error for wrong kind")
	}
	if isValid {
		t.Errorf("Expected isValid=false, got true")
	}
	if !strings.Contains(err.Error(), "does not have 'kind: GraphQL'") {
		t.Errorf("Expected 'wrong kind' error message, got: %v", err)
	}
}

func TestValidateGraphQLFile_FileNotFound(t *testing.T) {
	secretsMap := make(map[string]any)

	isValid, err := ValidateGraphQLFile("/nonexistent/file.yaml", secretsMap)

	if err == nil {
		t.Errorf("Expected error for non-existent file")
	}
	if isValid {
		t.Errorf("Expected isValid=false, got true")
	}
	if !strings.Contains(err.Error(), "file not found") {
		t.Errorf("Expected 'file not found' error message, got: %v", err)
	}
}

func TestValidateGraphQLFile_CaseInsensitive(t *testing.T) {
	tempDir := setupTestDirectory(t)
	secretsMap := make(map[string]any)

	testCases := []struct {
		name string
		kind string
	}{
		{"lowercase", "graphql"},
		{"uppercase", "GRAPHQL"},
		{"mixed", "GraphQL"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			content := fmt.Sprintf(
				"---\nkind: %s\nurl: http://example.com\nmethod: POST\n",
				tc.kind,
			)
			filePath := filepath.Join(tempDir, fmt.Sprintf("%s.yaml", tc.name))
			err := os.WriteFile(filePath, []byte(content), 0o644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			isValid, err := ValidateGraphQLFile(filePath, secretsMap)
			if err != nil {
				t.Errorf("Expected no error for kind '%s', got: %v", tc.kind, err)
			}
			if !isValid {
				t.Errorf("Expected isValid=true for kind '%s', got false", tc.kind)
			}
		})
	}
}

func TestValidateGraphQLFile_PathCleaning(t *testing.T) {
	tempDir := setupTestDirectory(t)
	secretsMap := make(map[string]any)

	// Create a file
	createGraphQLFile(t, tempDir, "test.yaml", "http://example.com")

	testCases := []struct {
		name string
		path string
	}{
		{"direct", filepath.Join(tempDir, "test.yaml")},
		{"with_dot", filepath.Join(tempDir, ".", "test.yaml")},
		{"with_redundant", filepath.Join(tempDir, "subdir", "..", "test.yaml")},
	}

	// Create subdir for path testing
	subDir := filepath.Join(tempDir, "subdir")
	err := os.Mkdir(subDir, 0o755)
	if err != nil {
		t.Fatalf("error on making dir TestValidateGraphQLFile_PathCleaning")
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			isValid, err := ValidateGraphQLFile(tc.path, secretsMap)
			if err != nil {
				t.Errorf("Expected no error for path '%s', got: %v", tc.path, err)
			}
			if !isValid {
				t.Errorf("Expected isValid=true for path '%s', got false", tc.path)
			}
		})
	}
}

// NOTE: TestValidateGraphQLFile_MalformedYAML cannot be tested in the current architecture
// because yamlparser.ParseConfig calls utils.PanicRedAndExit on malformed YAML,
// which exits the process. This is existing behavior in the codebase.
// The FindGraphQLFiles function handles this gracefully by skipping unparseable files.

// Tests for hasValidURLField

func TestHasValidURLField_EdgeCases(t *testing.T) {
	tempDir := setupTestDirectory(t)

	testCases := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "valid_http_url",
			content:  "---\nkind: GraphQL\nurl: http://example.com\n",
			expected: true,
		},
		{
			name:     "valid_template",
			content:  "---\nkind: GraphQL\nurl: \"{{.baseUrl}}\"\n",
			expected: true,
		},
		{
			name:     "url_with_whitespace",
			content:  "---\nkind: GraphQL\nurl: \"  http://test.com  \"\n",
			expected: true,
		},
		{
			name:     "empty_string",
			content:  "---\nkind: GraphQL\nurl: \"\"\n",
			expected: false,
		},
		{
			name:     "only_whitespace",
			content:  "---\nkind: GraphQL\nurl: \"   \"\n",
			expected: false,
		},
		{
			name:     "missing_url",
			content:  "---\nkind: GraphQL\nmethod: POST\n",
			expected: false,
		},
		{
			name:     "url_null",
			content:  "---\nkind: GraphQL\nurl: null\n",
			expected: false,
		},
		{
			name:     "url_number",
			content:  "---\nkind: GraphQL\nurl: 12345\n",
			expected: false,
		},
		{
			name:     "url_boolean",
			content:  "---\nkind: GraphQL\nurl: true\n",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, fmt.Sprintf("%s.yaml", tc.name))
			err := os.WriteFile(filePath, []byte(tc.content), 0o644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			result := hasValidURLField(filePath)

			if result != tc.expected {
				t.Errorf("Expected %v for case '%s', got %v", tc.expected, tc.name, result)
			}
		})
	}
}

// Tests for peekKindField

func TestPeekKindField(t *testing.T) {
	tempDir := setupTestDirectory(t)

	testCases := []struct {
		name        string
		content     string
		expectedStr string
		expectError bool
	}{
		{
			name:        "valid_graphql_kind",
			content:     "---\nkind: GraphQL\nurl: \"http://test.com\"\n",
			expectedStr: "graphql",
			expectError: false,
		},
		{
			name:        "valid_api_kind",
			content:     "---\nkind: API\nurl: \"http://test.com\"\n",
			expectedStr: "api",
			expectError: false,
		},
		{
			name:        "missing_kind_field",
			content:     "---\nurl: \"http://test.com\"\nmethod: POST\n",
			expectedStr: "",
			expectError: false,
		},
		{
			name:        "malformed_yaml",
			content:     "---\nkind: GraphQL\nbody: {\n  unclosed: bracket\n",
			expectedStr: "",
			expectError: true,
		},
		{
			name:        "case_insensitive",
			content:     "---\nKIND: GraphQL\nurl: \"http://test.com\"\n",
			expectedStr: "graphql",
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, tc.name+".yaml")
			err := os.WriteFile(filePath, []byte(tc.content), 0o644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			result, err := peekKindField(filePath)

			if tc.expectError && err == nil {
				t.Errorf("Expected error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if result != tc.expectedStr {
				t.Errorf("Expected %q, got %q", tc.expectedStr, result)
			}
		})
	}

	// Test non-existent file
	t.Run("nonexistent_file", func(t *testing.T) {
		_, err := peekKindField("/nonexistent/file.yaml")
		if err == nil {
			t.Errorf("Expected error for non-existent file")
		}
	})
}

// Integration test

func TestIntegration_RealWorldScenario(t *testing.T) {
	tempDir := setupTestDirectory(t)
	secretsMap := make(map[string]any)

	// Create project structure
	envDir := filepath.Join(tempDir, "env")
	apisDir := filepath.Join(tempDir, "apis")
	testsDir := filepath.Join(tempDir, "tests")
	gitDir := filepath.Join(tempDir, ".git")

	for _, dir := range []string{envDir, apisDir, testsDir, gitDir} {
		err := os.MkdirAll(dir, 0o755)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
	}

	// Create env file
	envFile := filepath.Join(envDir, "global.env")
	err := os.WriteFile(envFile, []byte("baseUrl=http://example.com\n"), 0o644)
	if err != nil {
		t.Fatalf("error on writing file in TestFindGraphQLFiles_MalformedYAML()")
	}

	// Create API files
	createYAMLFile(t, apisDir, "users.yaml", "API", "http://api.com/users")
	createGraphQLFile(t, apisDir, "graphql.yaml", "{{.baseUrl}}/graphql")
	responseFile := filepath.Join(apisDir, "users_response.json")
	err = os.WriteFile(responseFile, []byte("{}"), 0o644)
	if err != nil {
		t.Fatalf("error write file")
	}

	// Create test files
	createGraphQLFile(t, testsDir, "test1.yaml", "http://test.com/graphql")
	testResponseFile := filepath.Join(testsDir, "test1_response.json")
	err = os.WriteFile(testResponseFile, []byte("{}"), 0o644)
	if err != nil {
		t.Fatalf("error write file")
	}

	// Create file in .git (should be skipped)
	createGraphQLFile(t, gitDir, "graphql.yaml", "http://git.com/graphql")

	// Test FindGraphQLFiles
	files, err := FindGraphQLFiles(tempDir, secretsMap)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should find 2 GraphQL files (apis/graphql.yaml and tests/test1.yaml)
	// Should NOT find .git/graphql.yaml (git dir skipped by default)
	// Should NOT find response files
	if len(files) != 2 {
		t.Errorf("Expected 2 GraphQL files, got %d", len(files))
	}

	// Verify each file is valid
	for _, f := range files {
		isValid, err := ValidateGraphQLFile(f, secretsMap)
		if err != nil {
			t.Errorf("File %s failed validation: %v", f, err)
		}
		if !isValid {
			t.Errorf("File %s should be valid", f)
		}

		// Ensure .git files are not included
		if strings.Contains(f, ".git") {
			t.Errorf("Should not include files from .git directory: %s", f)
		}

		// Ensure response files are not included
		if strings.Contains(f, "_response") {
			t.Errorf("Should not include response files: %s", f)
		}
	}
}

// Benchmark tests

func BenchmarkFindGraphQLFiles(b *testing.B) {
	// Setup: create directory with 100 files (10 GraphQL, 90 other)
	tempDir := b.TempDir()
	secretsMap := make(map[string]any)

	for i := range 10 {
		content := fmt.Sprintf("---\nkind: GraphQL\nurl: http://example.com/%d\nmethod: POST\n", i)
		filePath := filepath.Join(tempDir, fmt.Sprintf("graphql%d.yaml", i))
		err := os.WriteFile(filePath, []byte(content), 0o644)
		if err != nil {
			b.Fatalf("error write file")
		}
	}

	for i := range 90 {
		content := fmt.Sprintf("---\nkind: API\nurl: http://example.com/%d\nmethod: POST\n", i)
		filePath := filepath.Join(tempDir, fmt.Sprintf("api%d.yaml", i))
		err := os.WriteFile(filePath, []byte(content), 0o644)
		if err != nil {
			b.Fatalf("error on write file")
		}

	}

	for b.Loop() {
		_, _ = FindGraphQLFiles(tempDir, secretsMap)
	}
}

func BenchmarkValidateGraphQLFile(b *testing.B) {
	// Setup: create single test file
	tempDir := b.TempDir()
	secretsMap := make(map[string]any)

	content := "---\nkind: GraphQL\nurl: http://example.com/graphql\nmethod: POST\n"
	filePath := filepath.Join(tempDir, "test.yaml")
	err := os.WriteFile(filePath, []byte(content), 0o644)
	if err != nil {
		b.Fatalf("error on wirte file")
	}

	for b.Loop() {
		_, _ = ValidateGraphQLFile(filePath, secretsMap)
	}
}
