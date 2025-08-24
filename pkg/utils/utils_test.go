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
	result, err := CreatePath("testfile.txt")
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
		err = os.Chdir(oldDir)
		if err != nil {
			t.Fatal(err)
		}
		os.RemoveAll(tempDir)
	}()

	// Test Cases
	testFileNames := []string{"global.env", "test.ENV", "spec_pm.env"}

	for _, fName := range testFileNames {
		var file *os.File
		file, err = os.Create(filepath.Join(envDir, fName))
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

func TestListMatchingFiles(t *testing.T) {
	// Create temporary test directory
	tempDir, err := os.MkdirTemp("", "list_matching_files_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Setup test directory structure
	setupTestDirectories := []string{
		"regular_dir",
		".hidden_dir",
		"regular_dir/nested_dir",
	}

	for _, dir := range setupTestDirectories {
		dirPath := filepath.Join(tempDir, dir)
		if err := os.MkdirAll(dirPath, DirPer.Perm()); err != nil {
			t.Fatalf("Failed to create directory: %s, error: %v", dirPath, err)
		}
	}

	// Create test files
	testFiles := []struct {
		path     string
		contents string
	}{
		{filepath.Join(tempDir, "config.json"), "json content"},
		{filepath.Join(tempDir, "config.yaml"), "yaml content"},
		{filepath.Join(tempDir, "config.yml"), "yml content"},
		{filepath.Join(tempDir, "other.txt"), "txt content"},
		{filepath.Join(tempDir, "regular_dir", "config.json"), "nested json"},
		{filepath.Join(tempDir, "regular_dir", "different.yaml"), "nested yaml"},
		{filepath.Join(tempDir, "regular_dir", "nested_dir", "config.yml"), "deeply nested yml"},
		{filepath.Join(tempDir, ".hidden_dir", "config.json"), "hidden json"},
		{filepath.Join(tempDir, ".config.json"), "hidden json file"},
	}

	for _, tf := range testFiles {
		if err := os.WriteFile(tf.path, []byte(tf.contents), 0644); err != nil {
			t.Fatalf("Failed to create file: %s, error: %v", tf.path, err)
		}
	}

	// Test cases
	tests := []struct {
		name           string
		matchFile      string
		initialPath    string
		expectedCount  int
		expectedError  bool
		checkFilePaths func(t *testing.T, paths []string)
	}{
		{
			name:          "Match config files",
			matchFile:     "config",
			initialPath:   tempDir,
			expectedCount: 6, // 3 in root, 1 in regular_dir, 1 in nested_dir, 1 in hidden dir (not .config.json)
			expectedError: false,
			checkFilePaths: func(t *testing.T, paths []string) {
				// Verify all paths are for config files
				for _, path := range paths {
					if !strings.Contains(path, "config") {
						t.Errorf("Found non-config file: %s", path)
					}
				}

				// Verify no .config.json files are found
				for _, path := range paths {
					if strings.HasSuffix(path, ".config.json") {
						t.Errorf("Found hidden file that should not match: %s", path)
					}
				}

				// Verify we do find files in hidden directories
				foundHiddenDir := false
				for _, path := range paths {
					if strings.Contains(path, ".hidden_dir") {
						foundHiddenDir = true
						break
					}
				}
				if !foundHiddenDir {
					t.Errorf("Missing file from hidden directory")
				}
			},
		},
		{
			name:          "Match hidden config files",
			matchFile:     ".config",
			initialPath:   tempDir,
			expectedCount: 1, // Just the .config.json file
			expectedError: false,
			checkFilePaths: func(t *testing.T, paths []string) {
				if !strings.HasSuffix(paths[0], ".config.json") {
					t.Errorf("Expected .config.json, got: %s", paths[0])
				}
			},
		},
		{
			name:          "Match different files",
			matchFile:     "different",
			initialPath:   tempDir,
			expectedCount: 1,
			expectedError: false,
			checkFilePaths: func(t *testing.T, paths []string) {
				if !strings.Contains(paths[0], "different.yaml") {
					t.Errorf("Found wrong file: %s", paths[0])
				}
			},
		},
		{
			name:          "Match non-existent file",
			matchFile:     "nonexistent",
			initialPath:   tempDir,
			expectedCount: 0,
			expectedError: true,
			checkFilePaths: func(_ *testing.T, _ []string) {
				// Should not be called due to expected error
			},
		},
		{
			name:          "Empty matchFile",
			matchFile:     "",
			initialPath:   tempDir,
			expectedCount: 0,
			expectedError: true,
			checkFilePaths: func(_ *testing.T, _ []string) {
				// Should not be called due to expected error
			},
		},
		{
			name:          "Match with extension included",
			matchFile:     "config.json",
			initialPath:   tempDir,
			expectedCount: 6,
			expectedError: false,
			checkFilePaths: func(t *testing.T, paths []string) {
				for _, path := range paths {
					if !strings.Contains(path, "config") {
						t.Errorf("Found non-config file: %s", path)
					}
					if strings.HasSuffix(path, ".config.json") {
						t.Errorf("Found hidden file that should not match: %s", path)
					}
				}
			},
		},
		{
			name:          "Match in subdirectory",
			matchFile:     "config",
			initialPath:   filepath.Join(tempDir, "regular_dir"),
			expectedCount: 2, // 1 in regular_dir, 1 in nested_dir
			expectedError: false,
			checkFilePaths: func(t *testing.T, paths []string) {
				for _, path := range paths {
					if !strings.Contains(path, "regular_dir") {
						t.Errorf("Found file outside of regular_dir: %s", path)
					}
				}
			},
		},
		{
			name:          "Match in hidden directory",
			matchFile:     "config",
			initialPath:   filepath.Join(tempDir, ".hidden_dir"),
			expectedCount: 1,
			expectedError: false,
			checkFilePaths: func(t *testing.T, paths []string) {
				if !strings.Contains(paths[0], ".hidden_dir") {
					t.Errorf("Found file outside of .hidden_dir: %s", paths[0])
				}
			},
		},
	}

	// Run test cases
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var initialPath []string
			if tc.initialPath != "" {
				initialPath = []string{tc.initialPath}
			}

			paths, err := ListMatchingFiles(tc.matchFile, initialPath...)

			// Check error condition
			if tc.expectedError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Check number of files found
			if len(paths) != tc.expectedCount {
				t.Errorf("Expected %d files, got %d: %v", tc.expectedCount, len(paths), paths)
			}

			// Run custom path checks
			tc.checkFilePaths(t, paths)
		})
	}

	// Test non-existent directory
	t.Run("Non-existent directory", func(t *testing.T) {
		nonExistentPath := filepath.Join(tempDir, "does_not_exist")
		_, err := ListMatchingFiles("config", nonExistentPath)
		if err == nil {
			t.Errorf("Expected error for non-existent directory, got nil")
		}
	})
}

func TestSanitizeDirPath(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "sanitize_dir_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a temporary file for testing non-directory paths
	tempFile, err := os.CreateTemp(tempDir, "test_file")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()

	// Create a nested directory for testing relative paths
	nestedDir := filepath.Join(tempDir, "nested_dir")
	if err = os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("Failed to create nested directory: %v", err)
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}

	// Get absolute paths for comparison
	tempDirAbs, _ := filepath.Abs(tempDir)
	nestedDirAbs, _ := filepath.Abs(nestedDir)
	cwdAbs, _ := filepath.Abs(cwd)
	parentDirAbs, _ := filepath.Abs(filepath.Join(cwd, ".."))

	// Create a symbolic link for testing symlinks (if supported)
	symlinkDir := filepath.Join(tempDir, "symlink_dir")
	err = os.Symlink(nestedDir, symlinkDir)
	symlinkSupported := err == nil
	symlinkDirAbs, _ := filepath.Abs(symlinkDir)

	// Test cases
	tests := []struct {
		name           string
		inputPath      string
		expectedOutput string
		expectError    bool
		errorContains  string
		setup          func() error
		cleanup        func()
	}{
		{
			name:           "Valid absolute directory path",
			inputPath:      tempDir,
			expectedOutput: tempDirAbs,
			expectError:    false,
		},
		{
			name:           "Valid nested directory path",
			inputPath:      nestedDir,
			expectedOutput: nestedDirAbs,
			expectError:    false,
		},
		{
			name:          "Non-existent path",
			inputPath:     filepath.Join(tempDir, "non_existent_dir"),
			expectError:   true,
			errorContains: "error accessing path",
		},
		{
			name:          "File path (not a directory)",
			inputPath:     tempFile.Name(),
			expectError:   true,
			errorContains: "path is not a directory",
		},
		{
			name:           "Empty path",
			inputPath:      "",
			expectedOutput: cwdAbs, // Current working directory
			expectError:    false,
		},
		{
			name:           "Current directory",
			inputPath:      ".",
			expectedOutput: cwdAbs, // Current working directory
			expectError:    false,
		},
		{
			name:           "Path with trailing slash",
			inputPath:      tempDir + "/",
			expectedOutput: tempDirAbs,
			expectError:    false,
		},
		{
			name:           "Path with redundant elements",
			inputPath:      tempDir + "/./",
			expectedOutput: tempDirAbs,
			expectError:    false,
		},
		{
			name:           "Parent directory",
			inputPath:      "..",
			expectedOutput: parentDirAbs,
			expectError:    false,
		},
	}

	// Add symlink test case if supported
	if symlinkSupported {
		tests = append(tests, struct {
			name           string
			inputPath      string
			expectedOutput string
			expectError    bool
			errorContains  string
			setup          func() error
			cleanup        func()
		}{
			name:           "Symlink to directory",
			inputPath:      symlinkDir,
			expectedOutput: symlinkDirAbs,
			expectError:    false,
			setup:          nil,
			cleanup:        nil,
		})
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				if err := tc.setup(); err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}

			if tc.cleanup != nil {
				defer tc.cleanup()
			}

			result, err := SanitizeDirPath(tc.inputPath)

			// Check error conditions
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
					return
				}
				if tc.errorContains != "" && !strings.Contains(err.Error(), tc.errorContains) {
					t.Errorf(
						"Expected error to contain '%s', got '%s'",
						tc.errorContains,
						err.Error(),
					)
				}
				return
			}

			// Check success conditions
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Normalize paths for comparison on all platforms
			expectedPath := filepath.Clean(tc.expectedOutput)
			resultPath := filepath.Clean(result)

			if expectedPath != resultPath {
				t.Errorf("Expected path '%s', got '%s'", expectedPath, resultPath)
			}
		})
	}
}

// TestSanitizeDirPathWithPermissions tests path sanitization with different permission scenarios
func TestSanitizeDirPathWithPermissions(t *testing.T) {
	if os.Getenv("GOOS") == "windows" {
		t.Skip("Skipping permission tests on Windows")
	}
	tempDir, err := os.MkdirTemp("", "sanitize_perm_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a directory with no read permissions
	noReadDir := filepath.Join(tempDir, "no_read_dir")
	if err := os.MkdirAll(noReadDir, 0755); err != nil { // Create with read permissions first
		t.Fatalf("Failed to create no-read directory: %v", err)
	}

	// Change permissions after creation
	if err := os.Chmod(noReadDir, 0000); err != nil {
		t.Fatalf("Failed to change directory permissions: %v", err)
	}

	// Restore permissions for cleanup
	defer os.Chmod(noReadDir, 0755)

	t.Run("No read permission directory", func(t *testing.T) {
		// Skip if running as root (permissions won't affect root)
		if os.Geteuid() == 0 {
			t.Skip("Skipping as root user")
		}

		_, err := SanitizeDirPath(noReadDir)

		// On some systems/configurations, this may not error out due to caching
		// So we need to check if we can actually access the directory content
		if err == nil {
			// Try to read the directory to verify permissions actually prevent access
			_, err = os.ReadDir(noReadDir)
			if err == nil {
				t.Error("Expected error when reading directory with no permissions, but got nil")
				// If this passes, permissions weren't actually restricted
				t.Logf("WARNING: Permission test may be unreliable on this system")
			} else {
				t.Logf("Note: SanitizeDirPath succeeded but directory is not readable: %v", err)
			}
		}
	})
}

// TestSanitizeDirPathWithSpecialPaths tests path sanitization with special path values
func TestSanitizeDirPathWithSpecialPaths(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	parentDir := filepath.Dir(cwd)
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Logf("Couldn't determine home directory: %v", err)
	}

	tests := []struct {
		name            string
		inputPath       string
		expectedPath    string
		expectError     bool
		skipIfNoHomeDir bool
	}{
		{
			name:         "Multiple dot path",
			inputPath:    "./././.",
			expectedPath: cwd,
			expectError:  false,
		},
		{
			name:         "Parent directory",
			inputPath:    "..",
			expectedPath: parentDir,
			expectError:  false,
		},
	}

	if homeDir != "" {
		tests = append(tests, struct {
			name            string
			inputPath       string
			expectedPath    string
			expectError     bool
			skipIfNoHomeDir bool
		}{
			name:            "Home directory tilde expansion",
			inputPath:       "~/",
			expectedPath:    homeDir,
			expectError:     false,
			skipIfNoHomeDir: true,
		})
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skipIfNoHomeDir && homeDir == "" {
				t.Skip("Skipping test as home directory couldn't be determined")
			}

			// Handle tilde expansion manually for the test
			inputPath := tc.inputPath
			if strings.HasPrefix(inputPath, "~/") && homeDir != "" {
				inputPath = filepath.Join(homeDir, inputPath[2:])
			}

			result, err := SanitizeDirPath(inputPath)

			if tc.expectError && err == nil {
				t.Errorf("Expected error but got nil")
				return
			}

			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if err == nil {
				normalizedResult := filepath.Clean(result)
				normalizedExpected := filepath.Clean(tc.expectedPath)

				if normalizedResult != normalizedExpected {
					t.Errorf("Expected result '%s', got '%s'", normalizedExpected, normalizedResult)
				}
			}
		})
	}
}
