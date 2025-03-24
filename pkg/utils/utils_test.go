package utils

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCreateFilePath(t *testing.T) {
	// Test case with a known relative file path
	expected, _ := os.Getwd()
	expected = filepath.Join(expected, "testfile.txt")
	result, err := CreateFilePath("testfile.txt")
	if err != nil {
		t.Errorf("CreateFilePath returned an error: %v", err)
	}
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestGetEnvFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "envTest")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	envDir := filepath.Join(tempDir, "env")
	err = os.Mkdir(envDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	// Cleanup temp dir
	defer func() {
		err := os.Chdir(oldDir)
		if err != nil {
			t.Fatal(err)
		}
		os.RemoveAll(tempDir)
	}()

	// Test Cases
	testFileNames := []string{"global.env", "test.ENV", "spec_pm.env"}

	for _, fName := range testFileNames {
		file, err := os.Create(filepath.Join(envDir, fName))
		if err != nil {
			t.Fatalf("Failed to Createfile :%v", err)
		}
		file.Close()
	}

	// Test Case: Directories should not be included:
	err = os.Mkdir(filepath.Join(envDir, "shouldNotAppear"), 0755)
	if err != nil {
		t.Fatalf("Temporary directory could not be created: %v", err)
	}

	// change the cwd for test
	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Could not change the temp dir: %v", err)
	}

	resultFiles, err := GetEnvFiles()
	if err != nil {
		t.Fatalf("Error while running GetEnvFiles(): %v", err)
	}
	expectedFiles := []string{"global.env", "test.env", "spec_pm.env"}
	if len(resultFiles) != len(expectedFiles) {
		t.Errorf(
			"Expected %d files, got %d files",
			len(expectedFiles),
			len(resultFiles),
		)
	}
	/*
		order is not guranteed with resultFiles so need to create map
		and check if the file exists
	*/
	expectedFilesMap := make(map[string]bool)
	for _, file := range expectedFiles {
		expectedFilesMap[file] = true
	}

	for _, file := range resultFiles {
		if !expectedFilesMap[strings.ToLower(file)] {
			t.Errorf("Unexpected file %s returned", file)
		}
	}
}

// TestToLowercaseMap tests the ToLowercaseMap function with various cases
func TestToLowercaseMap(t *testing.T) {
	testCases := []struct {
		input    map[string]any
		expected map[string]any
		name     string
	}{
		{
			name: "Simple keys",
			input: map[string]any{
				"KeyOne": "value1",
				"KeyTwo": "value2",
			},
			expected: map[string]any{
				"keyone": "value1",
				"keytwo": "value2",
			},
		},
		{
			name: "Nested map",
			input: map[string]any{
				"KeyOuter": map[string]any{
					"KeyInner": "valueInner",
				},
				"AnotherKey": "valueAnother",
			},
			expected: map[string]any{
				"keyouter": map[string]any{
					"keyinner": "valueInner",
				},
				"anotherkey": "valueAnother",
			},
		},
		{
			name: "Mixed case and nested levels",
			input: map[string]any{
				"MiXed": map[string]any{
					"UPPer": "value",
					"loWer": map[string]any{
						"INNerKey": "innerValue",
					},
				},
			},
			expected: map[string]any{
				"mixed": map[string]any{
					"upper": "value",
					"lower": map[string]any{
						"innerkey": "innerValue",
					},
				},
			},
		},
		{
			name:     "Empty map",
			input:    map[string]any{},
			expected: map[string]any{},
		},
		{
			name: "Already lowercase keys",
			input: map[string]any{
				"key": "value",
				"nested": map[string]any{
					"innerkey": "innervalue",
				},
			},
			expected: map[string]any{
				"key": "value",
				"nested": map[string]any{
					"innerkey": "innervalue",
				},
			},
		},
		{
			name: "Keys with non-string values",
			input: map[string]any{
				"BoolKey":  true,
				"IntKey":   123,
				"FloatKey": 12.34,
				"SliceKey": []any{"item1", "item2"},
				"MapKey":   map[string]any{"InnerKey": "innerValue"},
				"NilKey":   nil,
			},
			expected: map[string]any{
				"boolkey":  true,
				"intkey":   123,
				"floatkey": 12.34,
				"slicekey": []any{"item1", "item2"},
				"mapkey":   map[string]any{"innerkey": "innerValue"},
				"nilkey":   nil,
			},
		},
		{
			name: "Nested empty map",
			input: map[string]any{
				"OuterKey": map[string]any{},
			},
			expected: map[string]any{
				"outerkey": map[string]any{},
			},
		},
	}

	// Iterate over each test case
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ConvertKeysToLowerCase(tc.input)
			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("Test %s failed. Expected %v, got %v", tc.name, tc.expected, result)
			}
		})
	}
}

func TestCreateDir(t *testing.T) {
	t.Run("DirDoesNotExist_CreatesDirectory", func(t *testing.T) {
		// Use t.TempDir() as a temporary base directory.
		baseDir := t.TempDir()
		newDirPath := filepath.Join(baseDir, "newdir")

		// Ensure newDirPath doesn't exist.
		if _, err := os.Stat(newDirPath); err == nil {
			t.Fatalf("Directory %s should not exist", newDirPath)
		}

		// Call CreateDir; expect it to succeed.
		if err := CreateDir(newDirPath); err != nil {
			t.Fatalf("CreateDir returned an unexpected error: %v", err)
		}

		// Verify that the directory now exists and has the expected permissions.
		info, err := os.Stat(newDirPath)
		if err != nil {
			t.Fatalf("Expected directory %s to exist, got error: %v", newDirPath, err)
		}
		if !info.IsDir() {
			t.Fatalf("%s is not a directory", newDirPath)
		}

		if info.Mode().Perm() != DirPer {
			t.Errorf("Expected permissions %o, got %o", DirPer, info.Mode().Perm())
		}
	})

	t.Run("AlreadyExists_NoError", func(t *testing.T) {
		// Use t.TempDir() for an already existing directory.
		existingDir := t.TempDir()

		// Call CreateDir on the existing directory; should not error.
		if err := CreateDir(existingDir); err != nil {
			t.Fatalf("CreateDir returned an error when directory already exists: %v", err)
		}
	})

	t.Run("PathExistsButIsFile_ReturnsError", func(t *testing.T) {
		// Use t.TempDir() as a base directory.
		baseDir := t.TempDir()
		filePath := filepath.Join(baseDir, "afile")

		// Create a file at filePath.
		f, err := os.Create(filePath)
		if err != nil {
			t.Fatalf("Error creating file: %v", err)
		}
		f.Close()

		// Verify that filePath exists and is a file.
		info, err := os.Stat(filePath)
		if err != nil {
			t.Fatalf("Expected file %s to exist, got error: %v", filePath, err)
		}
		if info.IsDir() {
			t.Fatalf("%s should be a file", filePath)
		}

		// Call CreateDir with the file path; expect an error.
		err = CreateDir(filePath)
		if err == nil {
			t.Errorf("Expected error when CreateDir is called on a file path, got nil")
		}
	})
}

func TestCreateFile(t *testing.T) {
	// Helper function to check if a path exists and is a regular file
	checkFileExists := func(t *testing.T, path string) bool {
		t.Helper()
		info, err := os.Stat(path)
		if err != nil {
			return false
		}
		return info.Mode().IsRegular()
	}

	t.Run("CreateNewFile_Success", func(t *testing.T) {
		// Use t.TempDir() for a clean test environment
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "newfile.txt")

		err := CreateFile(filePath)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if !checkFileExists(t, filePath) {
			t.Errorf("Expected file %s to exist", filePath)
		}
	})

	t.Run("CreateFile_AlreadyExists", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "existing.txt")

		// Create the file first
		f, err := os.Create(filePath)
		if err != nil {
			t.Fatalf("Setup failed: %v", err)
		}
		f.Close()

		// Try to create the same file
		err = CreateFile(filePath)
		if err != nil {
			t.Errorf("Expected no error for existing file, got %v", err)
		}

		// Verify file still exists
		if !checkFileExists(t, filePath) {
			t.Errorf("Expected file %s to still exist", filePath)
		}
	})

	t.Run("CreateFile_InNonexistentDirectory", func(t *testing.T) {
		tempDir := t.TempDir()
		nonExistentDir := filepath.Join(tempDir, "nonexistent")
		filePath := filepath.Join(nonExistentDir, "file.txt")

		err := CreateFile(filePath)
		if err == nil {
			t.Error("Expected error when creating file in nonexistent directory, got nil")
		}
	})

	t.Run("CreateFile_NoPermission", func(t *testing.T) {
		if os.Getuid() == 0 { // Skip if running as root
			t.Skip("Test skipped when running as root")
		}

		tempDir := t.TempDir()
		// Remove all permissions from the directory
		err := os.Chmod(tempDir, 0o000)
		if err != nil {
			t.Fatalf("Failed to change directory permissions: %v", err)
		}
		defer os.Chmod(tempDir, 0o700) // Restore permissions for cleanup

		filePath := filepath.Join(tempDir, "noperm.txt")
		err = CreateFile(filePath)
		if err == nil {
			t.Error("Expected error when creating file without permissions, got nil")
		}
	})

	t.Run("CreateFile_PathIsDirectory", func(t *testing.T) {
		tempDir := t.TempDir()
		subDir := filepath.Join(tempDir, "subdir")

		// Create a directory
		err := os.Mkdir(subDir, 0o755)
		if err != nil {
			t.Fatalf("Setup failed: %v", err)
		}

		// Try to create a file with the same path as directory
		err = CreateFile(subDir)
		if err == nil {
			t.Error("Expected error when creating file at directory path, got nil")
		}
	})
}
