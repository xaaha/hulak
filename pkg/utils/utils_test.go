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
	var utility Utilities
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

	resultFiles, err := utility.GetEnvFiles()
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
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "Simple keys",
			input: map[string]interface{}{
				"KeyOne": "value1",
				"KeyTwo": "value2",
			},
			expected: map[string]interface{}{
				"keyone": "value1",
				"keytwo": "value2",
			},
		},
		{
			name: "Nested map",
			input: map[string]interface{}{
				"KeyOuter": map[string]interface{}{
					"KeyInner": "valueInner",
				},
				"AnotherKey": "valueAnother",
			},
			expected: map[string]interface{}{
				"keyouter": map[string]interface{}{
					"keyinner": "valueInner",
				},
				"anotherkey": "valueAnother",
			},
		},
		{
			name: "Mixed case and nested levels",
			input: map[string]interface{}{
				"MiXed": map[string]interface{}{
					"UPPer": "value",
					"loWer": map[string]interface{}{
						"INNerKey": "innerValue",
					},
				},
			},
			expected: map[string]interface{}{
				"mixed": map[string]interface{}{
					"upper": "value",
					"lower": map[string]interface{}{
						"innerkey": "innerValue",
					},
				},
			},
		},
		{
			name:     "Empty map",
			input:    map[string]interface{}{},
			expected: map[string]interface{}{},
		},
		{
			name: "Already lowercase keys",
			input: map[string]interface{}{
				"key": "value",
				"nested": map[string]interface{}{
					"innerkey": "innervalue",
				},
			},
			expected: map[string]interface{}{
				"key": "value",
				"nested": map[string]interface{}{
					"innerkey": "innervalue",
				},
			},
		},
		{
			name: "Keys with non-string values",
			input: map[string]interface{}{
				"BoolKey":  true,
				"IntKey":   123,
				"FloatKey": 12.34,
				"SliceKey": []interface{}{"item1", "item2"},
				"MapKey":   map[string]interface{}{"InnerKey": "innerValue"},
				"NilKey":   nil,
			},
			expected: map[string]interface{}{
				"boolkey":  true,
				"intkey":   123,
				"floatkey": 12.34,
				"slicekey": []interface{}{"item1", "item2"},
				"mapkey":   map[string]interface{}{"innerkey": "innerValue"},
				"nilkey":   nil,
			},
		},
		{
			name: "Nested empty map",
			input: map[string]interface{}{
				"OuterKey": map[string]interface{}{},
			},
			expected: map[string]interface{}{
				"outerkey": map[string]interface{}{},
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
