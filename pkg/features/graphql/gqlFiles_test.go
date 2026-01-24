package graphql

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xaaha/hulak/pkg/utils"
)

// Test helper functions

// setupTestDirectory creates a temporary directory for tests
func setupTestDirectory(tb testing.TB) string {
	tb.Helper()
	return tb.TempDir()
}

// createGraphQLFile creates a test GraphQL YAML file with specified URL
func createGraphQLFile(tb testing.TB, dir, filename, url string) string {
	tb.Helper()

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
		tb.Fatalf("Failed to create test file: %v", err)
	}
	return filePath
}

// createYAMLFile creates a test YAML file with custom kind and optional URL
func createYAMLFile(tb testing.TB, dir, filename, kind, url string) string {
	tb.Helper()

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
		tb.Fatalf("Failed to create test file: %v", err)
	}
	return filePath
}

// Tests for peek functions

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

func TestPeekURLField(t *testing.T) {
	tempDir := setupTestDirectory(t)

	testCases := []struct {
		name        string
		content     string
		expectedURL string
		expectError bool
	}{
		{
			name:        "full_url",
			content:     "---\nkind: GraphQL\nurl: \"http://example.com\"\n",
			expectedURL: "http://example.com",
			expectError: false,
		},
		{
			name:        "template_url",
			content:     "---\nkind: GraphQL\nurl: \"{{.baseUrl}}\"\n",
			expectedURL: "{{.baseUrl}}",
			expectError: false,
		},
		{
			name:        "missing_url",
			content:     "---\nkind: GraphQL\nmethod: POST\n",
			expectedURL: "",
			expectError: false,
		},
		{
			name:        "empty_url",
			content:     "---\nkind: GraphQL\nurl: \"\"\n",
			expectedURL: "",
			expectError: false,
		},
		{
			name:        "url_with_whitespace",
			content:     "---\nkind: GraphQL\nurl: \"  http://test.com  \"\n",
			expectedURL: "http://test.com",
			expectError: false,
		},
		{
			name:        "complex_template_url",
			content:     "---\nkind: GraphQL\nurl: \"{{.endpoint}}/graphql\"\n",
			expectedURL: "{{.endpoint}}/graphql",
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

			result, err := peekURLField(filePath)

			if tc.expectError && err == nil {
				t.Errorf("Expected error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if result != tc.expectedURL {
				t.Errorf("Expected %q, got %q", tc.expectedURL, result)
			}
		})
	}

	// Test non-existent file
	t.Run("nonexistent_file", func(t *testing.T) {
		_, err := peekURLField("/nonexistent/file.yaml")
		if err == nil {
			t.Errorf("Expected error for non-existent file")
		}
	})
}

// Tests for FindGraphQLFiles

func TestFindGraphQLFiles_Success(t *testing.T) {
	tempDir := setupTestDirectory(t)

	// Create test files
	createGraphQLFile(t, tempDir, "valid1.yaml", "http://example.com")
	createGraphQLFile(t, tempDir, "valid2.yml", "http://test.com")
	createYAMLFile(t, tempDir, "api.yaml", "API", "http://api.com")
	createYAMLFile(t, tempDir, "auth.yaml", "Auth", "http://auth.com")

	urlToFile, err := FindGraphQLFiles(tempDir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(urlToFile) != 2 {
		t.Errorf("Expected 2 unique URLs, got %d", len(urlToFile))
	}

	// Verify the right URLs and files are included
	if _, exists := urlToFile["http://example.com"]; !exists {
		t.Errorf("Expected URL 'http://example.com' not found in results")
	}
	if _, exists := urlToFile["http://test.com"]; !exists {
		t.Errorf("Expected URL 'http://test.com' not found in results")
	}

	// Ensure non-GraphQL files are not included
	for _, filePath := range urlToFile {
		if strings.Contains(filePath, "api.yaml") || strings.Contains(filePath, "auth.yaml") {
			t.Errorf("Non-GraphQL file should not be included: %s", filePath)
		}
	}
}

func TestFindGraphQLFiles_DuplicateURLs(t *testing.T) {
	tempDir := setupTestDirectory(t)

	// Create multiple files with the same URL
	createGraphQLFile(t, tempDir, "file1.yaml", "http://example.com/graphql")
	createGraphQLFile(t, tempDir, "file2.yaml", "http://example.com/graphql")
	createGraphQLFile(t, tempDir, "file3.yaml", "http://different.com/graphql")

	urlToFile, err := FindGraphQLFiles(tempDir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should only have 2 unique URLs
	if len(urlToFile) != 2 {
		t.Errorf("Expected 2 unique URLs (duplicates removed), got %d", len(urlToFile))
	}

	// Verify the URLs exist
	if _, exists := urlToFile["http://example.com/graphql"]; !exists {
		t.Errorf("Expected URL 'http://example.com/graphql' not found")
	}
	if _, exists := urlToFile["http://different.com/graphql"]; !exists {
		t.Errorf("Expected URL 'http://different.com/graphql' not found")
	}
}

func TestFindGraphQLFiles_WithTemplateURLs(t *testing.T) {
	tempDir := setupTestDirectory(t)

	// Create files with various template URLs
	createGraphQLFile(t, tempDir, "template1.yaml", "{{.baseUrl}}")
	createGraphQLFile(t, tempDir, "template2.yaml", "{{.endpoint}}/graphql")
	createGraphQLFile(t, tempDir, "actual.yaml", "https://api.example.com/graphql")

	urlToFile, err := FindGraphQLFiles(tempDir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(urlToFile) != 3 {
		t.Errorf("Expected 3 files with template URLs, got %d", len(urlToFile))
	}

	// Verify template URLs are returned as-is (no substitution)
	if _, exists := urlToFile["{{.baseUrl}}"]; !exists {
		t.Errorf("Expected template URL '{{.baseUrl}}' in results")
	}
	if _, exists := urlToFile["{{.endpoint}}/graphql"]; !exists {
		t.Errorf("Expected template URL '{{.endpoint}}/graphql' in results")
	}
	if _, exists := urlToFile["https://api.example.com/graphql"]; !exists {
		t.Errorf("Expected full URL 'https://api.example.com/graphql' in results")
	}
}

func TestFindGraphQLFiles_WithoutURL(t *testing.T) {
	tempDir := setupTestDirectory(t)

	// Create GraphQL file without URL
	createGraphQLFile(t, tempDir, "no_url.yaml", "")

	urlToFile, err := FindGraphQLFiles(tempDir)

	if err == nil {
		t.Errorf("Expected error for GraphQL file without URL")
	}
	if urlToFile != nil {
		t.Errorf("Expected nil map, got %d URLs", len(urlToFile))
	}
}

func TestFindGraphQLFiles_EmptyURL(t *testing.T) {
	tempDir := setupTestDirectory(t)

	// Create file with empty URL
	content := "---\nkind: GraphQL\nurl: \"\"\nmethod: POST\n"
	filePath := filepath.Join(tempDir, "empty_url.yaml")
	err := os.WriteFile(filePath, []byte(content), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	urlToFile, err := FindGraphQLFiles(tempDir)

	if err == nil {
		t.Errorf("Expected error for GraphQL file with empty URL")
	}
	if urlToFile != nil {
		t.Errorf("Expected nil map, got %d URLs", len(urlToFile))
	}
}

func TestFindGraphQLFiles_NoGraphQLFiles(t *testing.T) {
	tempDir := setupTestDirectory(t)

	// Create only non-GraphQL files
	createYAMLFile(t, tempDir, "api.yaml", "API", "http://api.com")
	createYAMLFile(t, tempDir, "auth.yaml", "Auth", "http://auth.com")

	urlToFile, err := FindGraphQLFiles(tempDir)

	if err == nil {
		t.Errorf("Expected error when no GraphQL files found")
	}
	if !strings.Contains(err.Error(), "no files with 'kind: GraphQL'") {
		t.Errorf("Expected 'no files' error message, got: %v", err)
	}
	if urlToFile != nil {
		t.Errorf("Expected nil map, got %d URLs", len(urlToFile))
	}
}

func TestFindGraphQLFiles_OnlyResponseFiles(t *testing.T) {
	tempDir := setupTestDirectory(t)

	// Create only response files
	content := "---\nkind: GraphQL\nurl: http://example.com\nmethod: POST\n"
	responseFile := filepath.Join(tempDir, "test_response.json")
	err := os.WriteFile(responseFile, []byte(content), 0o644)
	if err != nil {
		t.Fatalf("Failed to create response file: %v", err)
	}

	urlToFile, err := FindGraphQLFiles(tempDir)

	if err == nil {
		t.Errorf("Expected error when only response files present")
	}
	if urlToFile != nil {
		t.Errorf("Expected nil map, got %d URLs", len(urlToFile))
	}
}

func TestFindGraphQLFiles_EmptyDirectory(t *testing.T) {
	tempDir := setupTestDirectory(t)

	urlToFile, err := FindGraphQLFiles(tempDir)

	if err == nil {
		t.Errorf("Expected error for empty directory")
	}
	if urlToFile != nil {
		t.Errorf("Expected nil map, got %d URLs", len(urlToFile))
	}
}

func TestFindGraphQLFiles_MalformedYAML(t *testing.T) {
	tempDir := setupTestDirectory(t)

	// Create valid file and a truly malformed file
	createGraphQLFile(t, tempDir, "valid.yaml", "http://example.com")

	// Create a file that's truly unparseable
	malformedPath := filepath.Join(tempDir, "malformed.yaml")
	malformedContent := "---\nthis is not: valid: yaml: at: all:\n  - [\n"
	err := os.WriteFile(malformedPath, []byte(malformedContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to create malformed file: %v", err)
	}

	urlToFile, err := FindGraphQLFiles(tempDir)
	// Should find at least the valid file, malformed should be skipped
	if err != nil {
		t.Fatalf("Expected no error (malformed files should be skipped), got: %v", err)
	}
	if len(urlToFile) < 1 {
		t.Errorf("Expected at least 1 valid file, got %d", len(urlToFile))
	}
}

func TestFindGraphQLFiles_NestedDirectories(t *testing.T) {
	tempDir := setupTestDirectory(t)

	// Create nested directory structure
	subDir := filepath.Join(tempDir, "subdir")
	err := os.Mkdir(subDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Create files in both directories with different URLs
	createGraphQLFile(t, tempDir, "root.yaml", "http://example.com")
	createGraphQLFile(t, subDir, "nested.yaml", "http://nested.com")

	urlToFile, err := FindGraphQLFiles(tempDir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(urlToFile) != 2 {
		t.Errorf("Expected 2 unique URLs from nested directories, got %d", len(urlToFile))
	}
}

// Tests for ValidateGraphQLFile

func TestValidateGraphQLFile_Valid(t *testing.T) {
	tempDir := setupTestDirectory(t)

	filePath := createGraphQLFile(t, tempDir, "valid.yaml", "http://example.com")

	url, isValid, err := ValidateGraphQLFile(filePath)
	if err != nil {
		t.Errorf("Expected no error for valid file, got: %v", err)
	}
	if !isValid {
		t.Errorf("Expected isValid=true, got false")
	}
	if url != "http://example.com" {
		t.Errorf("Expected URL 'http://example.com', got '%s'", url)
	}
}

func TestValidateGraphQLFile_MissingURL(t *testing.T) {
	tempDir := setupTestDirectory(t)

	filePath := createGraphQLFile(t, tempDir, "no_url.yaml", "")

	url, isValid, err := ValidateGraphQLFile(filePath)

	if err == nil {
		t.Errorf("Expected error for missing URL")
	}
	if isValid {
		t.Errorf("Expected isValid=false, got true")
	}
	if url != "" {
		t.Errorf("Expected empty URL string, got '%s'", url)
	}
	if !strings.Contains(err.Error(), "empty or missing 'url' field") {
		t.Errorf("Expected 'missing url' error message, got: %v", err)
	}
}

func TestValidateGraphQLFile_EmptyURL(t *testing.T) {
	tempDir := setupTestDirectory(t)

	content := "---\nkind: GraphQL\nurl: \"\"\nmethod: POST\n"
	filePath := filepath.Join(tempDir, "empty_url.yaml")
	err := os.WriteFile(filePath, []byte(content), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	url, isValid, err := ValidateGraphQLFile(filePath)

	if err == nil {
		t.Errorf("Expected error for empty URL")
	}
	if isValid {
		t.Errorf("Expected isValid=false, got true")
	}
	if url != "" {
		t.Errorf("Expected empty URL string, got '%s'", url)
	}
}

func TestValidateGraphQLFile_WrongKind(t *testing.T) {
	tempDir := setupTestDirectory(t)

	filePath := createYAMLFile(t, tempDir, "api.yaml", "API", "http://example.com")

	url, isValid, err := ValidateGraphQLFile(filePath)

	if err == nil {
		t.Errorf("Expected error for wrong kind")
	}
	if isValid {
		t.Errorf("Expected isValid=false, got true")
	}
	if url != "" {
		t.Errorf("Expected empty URL string, got '%s'", url)
	}
	if !strings.Contains(err.Error(), "does not have 'kind: GraphQL'") {
		t.Errorf("Expected 'wrong kind' error message, got: %v", err)
	}
}

func TestValidateGraphQLFile_FileNotFound(t *testing.T) {
	url, isValid, err := ValidateGraphQLFile("/nonexistent/file.yaml")

	if err == nil {
		t.Errorf("Expected error for non-existent file")
	}
	if isValid {
		t.Errorf("Expected isValid=false, got true")
	}
	if url != "" {
		t.Errorf("Expected empty URL string, got '%s'", url)
	}
	if !strings.Contains(err.Error(), "file not found") {
		t.Errorf("Expected 'file not found' error message, got: %v", err)
	}
}

func TestValidateGraphQLFile_CaseInsensitive(t *testing.T) {
	tempDir := setupTestDirectory(t)

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

			url, isValid, err := ValidateGraphQLFile(filePath)
			if err != nil {
				t.Errorf("Expected no error for kind '%s', got: %v", tc.kind, err)
			}
			if !isValid {
				t.Errorf("Expected isValid=true for kind '%s', got false", tc.kind)
			}
			if url != "http://example.com" {
				t.Errorf("Expected URL 'http://example.com', got '%s'", url)
			}
		})
	}
}

func TestValidateGraphQLFile_PathCleaning(t *testing.T) {
	tempDir := setupTestDirectory(t)

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
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			url, isValid, err := ValidateGraphQLFile(tc.path)
			if err != nil {
				t.Errorf("Expected no error for path '%s', got: %v", tc.path, err)
			}
			if !isValid {
				t.Errorf("Expected isValid=true for path '%s', got false", tc.path)
			}
			if url != "http://example.com" {
				t.Errorf("Expected URL 'http://example.com', got '%s'", url)
			}
		})
	}
}

func TestValidateGraphQLFile_TemplateURL(t *testing.T) {
	tempDir := setupTestDirectory(t)

	// Test various template URL formats
	testCases := []struct {
		name        string
		url         string
		expectValid bool
	}{
		{
			name:        "simple_template",
			url:         "{{.graphqlUrl}}",
			expectValid: true,
		},
		{
			name:        "template_with_path",
			url:         "{{.baseUrl}}/graphql",
			expectValid: true,
		},
		{
			name:        "full_url",
			url:         "http://example.com/graphql",
			expectValid: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filePath := createGraphQLFile(t, tempDir, tc.name+".yaml", tc.url)

			url, isValid, err := ValidateGraphQLFile(filePath)

			if tc.expectValid {
				if err != nil {
					t.Errorf("Expected no error for URL '%s', got: %v", tc.url, err)
				}
				if !isValid {
					t.Errorf("Expected isValid=true for URL '%s', got false", tc.url)
				}
				if url != tc.url {
					t.Errorf("Expected URL '%s', got '%s'", tc.url, url)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error for URL '%s', got none", tc.url)
				}
			}
		})
	}
}

// Integration test

func TestIntegration_RealWorldScenario(t *testing.T) {
	tempDir := setupTestDirectory(t)

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
		t.Fatalf("Failed to create env file: %v", err)
	}

	// Create API files
	createYAMLFile(t, apisDir, "users.yaml", "API", "http://api.com/users")
	createGraphQLFile(t, apisDir, "graphql.yaml", "http://gql.com/graphql")
	responseFile := filepath.Join(apisDir, "users_response.json")
	err = os.WriteFile(responseFile, []byte("{}"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create response file: %v", err)
	}

	// Create test files
	createGraphQLFile(t, testsDir, "test1.yaml", "http://test.com/graphql")
	testResponseFile := filepath.Join(testsDir, "test1_response.json")
	err = os.WriteFile(testResponseFile, []byte("{}"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test response file: %v", err)
	}

	// Create file in .git (should be skipped)
	createGraphQLFile(t, gitDir, "graphql.yaml", "http://git.com/graphql")

	// Test FindGraphQLFiles
	urlToFile, err := FindGraphQLFiles(tempDir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should find 2 GraphQL files (apis/graphql.yaml and tests/test1.yaml)
	// Should NOT find .git/graphql.yaml (git dir skipped by default)
	// Should NOT find response files
	if len(urlToFile) != 2 {
		t.Errorf("Expected 2 unique GraphQL URLs, got %d", len(urlToFile))
	}

	// Verify URLs
	if _, exists := urlToFile["http://gql.com/graphql"]; !exists {
		t.Errorf("Expected URL 'http://gql.com/graphql' not found")
	}
	if _, exists := urlToFile["http://test.com/graphql"]; !exists {
		t.Errorf("Expected URL 'http://test.com/graphql' not found")
	}

	// Verify each file is valid
	for url, filePath := range urlToFile {
		returnedURL, isValid, err := ValidateGraphQLFile(filePath)
		if err != nil {
			t.Errorf("File %s failed validation: %v", filePath, err)
		}
		if !isValid {
			t.Errorf("File %s should be valid", filePath)
		}
		if returnedURL != url {
			t.Errorf("URL mismatch: expected %s, got %s", url, returnedURL)
		}

		// Ensure .git files are not included
		if strings.Contains(filePath, ".git") {
			t.Errorf("Should not include files from .git directory: %s", filePath)
		}

		// Ensure response files are not included
		if strings.Contains(filePath, "_response") {
			t.Errorf("Should not include response files: %s", filePath)
		}
	}
}

// Tests for needsEnvResolution (from graphql.go)

func TestNeedsEnvResolution(t *testing.T) {
	testCases := []struct {
		name         string
		urlToFileMap map[string]string
		expected     bool
	}{
		{
			name: "no_templates",
			urlToFileMap: map[string]string{
				"http://example.com/graphql": "file1.yaml",
				"https://api.test.com/query": "file2.yaml",
			},
			expected: false,
		},
		{
			name: "dot_variable_template",
			urlToFileMap: map[string]string{
				"{{.baseUrl}}/graphql": "file1.yaml",
			},
			expected: true,
		},
		{
			name: "getValueOf_template",
			urlToFileMap: map[string]string{
				"{{getValueOf url config}}/graphql": "file1.yaml",
			},
			expected: true,
		},
		{
			name: "getFile_template",
			urlToFileMap: map[string]string{
				"{{getFile url.txt}}": "file1.yaml",
			},
			expected: true,
		},
		{
			name: "mixed_templates_and_urls",
			urlToFileMap: map[string]string{
				"http://example.com/graphql":          "file1.yaml",
				"{{.graphqlUrl}}":                     "file2.yaml",
				"{{getValueOf endpoint config.json}}": "file3.yaml",
			},
			expected: true,
		},
		{
			name: "partial_template_in_url",
			urlToFileMap: map[string]string{
				"https://{{.domain}}/graphql": "file1.yaml",
			},
			expected: true,
		},
		{
			name:         "empty_map",
			urlToFileMap: map[string]string{},
			expected:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := needsEnvResolution(tc.urlToFileMap)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

// Benchmark tests

func BenchmarkFindGraphQLFiles(b *testing.B) {
	// Setup: create directory with 100 files (10 GraphQL, 90 other)
	tempDir := b.TempDir()

	for i := range 10 {
		content := fmt.Sprintf("---\nkind: GraphQL\nurl: http://example.com/%d\nmethod: POST\n", i)
		filePath := filepath.Join(tempDir, fmt.Sprintf("graphql%d.yaml", i))
		err := os.WriteFile(filePath, []byte(content), 0o644)
		if err != nil {
			b.Fatalf("Failed to create test file: %v", err)
		}
	}

	for i := range 90 {
		content := fmt.Sprintf("---\nkind: API\nurl: http://example.com/%d\nmethod: POST\n", i)
		filePath := filepath.Join(tempDir, fmt.Sprintf("api%d.yaml", i))
		err := os.WriteFile(filePath, []byte(content), 0o644)
		if err != nil {
			b.Fatalf("Failed to create test file: %v", err)
		}
	}

	for b.Loop() {
		_, _ = FindGraphQLFiles(tempDir)
	}
}

func BenchmarkValidateGraphQLFile(b *testing.B) {
	// Setup: create single test file
	tempDir := b.TempDir()

	content := "---\nkind: GraphQL\nurl: http://example.com/graphql\nmethod: POST\n"
	filePath := filepath.Join(tempDir, "test.yaml")
	err := os.WriteFile(filePath, []byte(content), 0o644)
	if err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	for b.Loop() {
		_, _, _ = ValidateGraphQLFile(filePath)
	}
}

// Tests for concurrent resolution

func TestResolveTemplateURLsConcurrent_EmptyInput(t *testing.T) {
	emptyMap := make(map[string]string)
	secretsMap := make(map[string]any)

	summary, err := ResolveTemplateURLsConcurrent(emptyMap, secretsMap)
	if err != nil {
		t.Errorf("Expected no error for empty input, got: %v", err)
	}
	if summary.TotalFiles != 0 {
		t.Errorf("Expected TotalFiles=0, got %d", summary.TotalFiles)
	}
	if len(summary.Successful) != 0 {
		t.Errorf("Expected no successful results, got %d", len(summary.Successful))
	}
	if len(summary.Failed) != 0 {
		t.Errorf("Expected no failed results, got %d", len(summary.Failed))
	}
}

func TestResolveTemplateURLsConcurrent_NoTemplates(t *testing.T) {
	tempDir := setupTestDirectory(t)

	// Create files without templates
	createGraphQLFile(t, tempDir, "file1.yaml", "http://example.com/graphql")
	createGraphQLFile(t, tempDir, "file2.yaml", "https://api.test.com/query")

	urlToFileMap := map[string]string{
		"http://example.com/graphql": filepath.Join(tempDir, "file1.yaml"),
		"https://api.test.com/query": filepath.Join(tempDir, "file2.yaml"),
	}
	secretsMap := make(map[string]any)

	summary, err := ResolveTemplateURLsConcurrent(urlToFileMap, secretsMap)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if summary.TotalFiles != 2 {
		t.Errorf("Expected TotalFiles=2, got %d", summary.TotalFiles)
	}
	if len(summary.Successful) != 2 {
		t.Errorf("Expected 2 successful resolutions, got %d", len(summary.Successful))
	}
	if len(summary.Failed) != 0 {
		t.Errorf("Expected 0 failed resolutions, got %d", len(summary.Failed))
	}
	if summary.HasErrors() {
		t.Error("Expected HasErrors()=false")
	}

	// Verify resolved map
	resolvedMap := summary.GetResolvedMap()
	if len(resolvedMap) != 2 {
		t.Errorf("Expected 2 entries in resolved map, got %d", len(resolvedMap))
	}
}

func TestResolveTemplateURLsConcurrent_InvalidURLs(t *testing.T) {
	tempDir := setupTestDirectory(t)

	// Create file with invalid URL (no protocol)
	createGraphQLFile(t, tempDir, "invalid.yaml", "example.com/graphql")

	urlToFileMap := map[string]string{
		"example.com/graphql": filepath.Join(tempDir, "invalid.yaml"),
	}
	secretsMap := make(map[string]any)

	summary, err := ResolveTemplateURLsConcurrent(urlToFileMap, secretsMap)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(summary.Failed) != 1 {
		t.Errorf("Expected 1 failed resolution, got %d", len(summary.Failed))
	}
	if len(summary.Successful) != 0 {
		t.Errorf("Expected 0 successful resolutions, got %d", len(summary.Successful))
	}

	// Verify error message mentions invalid URL
	if !strings.Contains(summary.Failed[0].Error.Error(), "invalid URL") {
		t.Errorf("Expected error about invalid URL, got: %v", summary.Failed[0].Error)
	}
}

func TestResolutionSummary_Methods(t *testing.T) {
	t.Run("HasErrors_true", func(t *testing.T) {
		summary := &ResolutionSummary{
			Failed: []ResolutionResult{
				{RawURL: "test", Error: fmt.Errorf("test error")},
			},
		}
		if !summary.HasErrors() {
			t.Error("Expected HasErrors()=true when Failed is not empty")
		}
	})

	t.Run("HasErrors_false", func(t *testing.T) {
		summary := &ResolutionSummary{
			Successful: []ResolutionResult{
				{RawURL: "test", ResolvedURL: "http://test.com"},
			},
		}
		if summary.HasErrors() {
			t.Error("Expected HasErrors()=false when Failed is empty")
		}
	})

	t.Run("GetResolvedMap", func(t *testing.T) {
		summary := &ResolutionSummary{
			Successful: []ResolutionResult{
				{ResolvedURL: "http://example.com", FilePath: "/path/file1.yaml"},
				{ResolvedURL: "http://test.com", FilePath: "/path/file2.yaml"},
			},
		}
		resolved := summary.GetResolvedMap()
		if len(resolved) != 2 {
			t.Errorf("Expected 2 entries in map, got %d", len(resolved))
		}
		if resolved["http://example.com"] != "/path/file1.yaml" {
			t.Errorf("Expected correct mapping for example.com")
		}
	})

	t.Run("FormatErrors", func(t *testing.T) {
		summary := &ResolutionSummary{
			Failed: []ResolutionResult{
				{
					FilePath: "/path/file1.yaml",
					RawURL:   "{{.missing}}",
					Error:    fmt.Errorf("key not found"),
				},
			},
		}
		formatted := summary.FormatErrors()
		if !strings.Contains(formatted, "Failed to resolve") {
			t.Error("Expected format to mention 'Failed to resolve'")
		}
		if !strings.Contains(formatted, "file1.yaml") {
			t.Error("Expected format to include file path")
		}
		if !strings.Contains(formatted, "{{.missing}}") {
			t.Error("Expected format to include raw URL")
		}
	})

	t.Run("FormatErrors_empty", func(t *testing.T) {
		summary := &ResolutionSummary{}
		formatted := summary.FormatErrors()
		if formatted != "" {
			t.Errorf("Expected empty string, got: %s", formatted)
		}
	})
}

func TestCalculateWorkerCount(t *testing.T) {
	tests := []struct {
		name       string
		totalFiles int
		minWorkers int
		maxWorkers int
	}{
		{"zero_files", 0, 0, 0},
		{"one_file", 1, 1, 1},
		{"small_3files", 3, 3, 3},
		{"medium_10files", 10, 1, 20}, // Depends on CPU count
		{"large_50files", 50, 1, 20},  // Capped at 20
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			totalFiles := tt.totalFiles
			workers := utils.GetWorkers(&totalFiles)
			if workers < tt.minWorkers || workers > tt.maxWorkers {
				t.Errorf("Expected workers between %d and %d, got %d",
					tt.minWorkers, tt.maxWorkers, workers)
			}
		})
	}
}

func BenchmarkResolveTemplateURLsConcurrent(b *testing.B) {
	tempDir := b.TempDir()

	// Create 20 test files without templates for benchmarking
	urlToFileMap := make(map[string]string)
	for i := range 20 {
		url := fmt.Sprintf("http://example.com/%d", i)
		filename := fmt.Sprintf("file%d.yaml", i)
		createGraphQLFile(b, tempDir, filename, url)
		urlToFileMap[url] = filepath.Join(tempDir, filename)
	}

	secretsMap := make(map[string]any)

	b.ResetTimer()
	for b.Loop() {
		_, _ = ResolveTemplateURLsConcurrent(urlToFileMap, secretsMap)
	}
}
