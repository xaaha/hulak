package yamlParser

import (
	"reflect"
	"strings"
	"testing"

	"github.com/xaaha/hulak/pkg/utils"
)

func TestStringHasDelimiter(t *testing.T) {
	testCases := []struct {
		number   int
		input    string
		expected bool
		content  string
	}{
		{1, "{{ valid }}", true, "valid"},
		{2, "{{   valid2}}", true, "valid2"},
		{3, "{{valid }}", true, "valid"},
		{4, "{{valid}}", true, "valid"},
		{5, "{{}}", false, ""},
		{6, "{{     }}", false, ""},
		{7, "No delimiters here", false, ""},
		{8, "{{valid}}", true, "valid"},
		{9, "{{valid}}", true, "valid"},
		{10, "{{ .valid}}", true, ".valid"},
		{11, "{{.valid }}", true, ".valid"},
		{12, "{}", false, ""},
		{13, "{{{valid}}}", false, ""},
		{14, "this {{valid}}", false, ""},
		{15, "this {{}} is invalid", false, ""},
		{16, "{{getValueOf 'foo' 'bar'}}", true, "getValueOf 'foo' 'bar'"},
		{17, "{{getValueOf 'foo' 'bar'}}", true, "getValueOf 'foo' 'bar'"},
		{18, `{{getValueOf "foo" "bar"}}`, true, `getValueOf "foo" "bar"`},
		{19, `{{ getValueOf "foo" "bar" }}`, true, `getValueOf "foo" "bar"`},
		{20, `{{getValueOf "foo" 'bar' }}`, true, `getValueOf "foo" 'bar'`},
	}

	// Run the tests
	for _, tc := range testCases {
		result, resultContent := stringHasDelimiter(tc.input)
		if result != tc.expected {
			t.Errorf(
				"On %d: stringHasDelimiter(%q) = %v; want %v",
				tc.number,
				tc.input,
				result,
				tc.expected,
			)
		}
		if resultContent != tc.content {
			t.Errorf(
				"On %d: stringHasDelimiter error: expected %v but got %v",
				tc.number,
				tc.content,
				resultContent,
			)
			if len(tc.content) != len(resultContent) {
				t.Errorf(
					"length of expected content %d, but got %d",
					len(tc.content),
					len(resultContent),
				)
			}

		}
	}
}

func TestDelimiterLogic(t *testing.T) {
	testCases := []struct {
		input    string
		expected Action
	}{
		{input: "", expected: Action{Type: Invalid}},
		{input: "{{.Value}}", expected: Action{Type: DotString, DotString: "Value"}},
		{
			input:    `{{getValueOf "key" "value"}}`,
			expected: Action{Type: GetValueOf, GetValueOf: []string{"getValueOf", "key", "value"}},
		},
		{
			input:    `{{getvalueof "key" "value"}}`,
			expected: Action{Type: Invalid, GetValueOf: []string{}},
		},
		{
			input:    `{{getValueOf key "value}}`,
			expected: Action{Type: GetValueOf, GetValueOf: []string{"getValueOf", "key", "value"}},
		},
		{
			input:    `{{getValueOf}}`,
			expected: Action{Type: Invalid, GetValueOf: []string{"getValueOf", "key", "value"}},
		},
	}

	for _, tc := range testCases {
		action := delimiterLogicAndCleanup(tc.input)
		if action.Type != tc.expected.Type {
			t.Errorf("expected type to be %v but got %v", tc.expected.Type, action.Type)
		}
		if action.DotString != tc.expected.DotString {
			t.Errorf(
				"expected DotString to be '%v' but got '%v'",
				tc.expected.DotString,
				action.DotString,
			)
		}
		if tc.expected.Type == GetValueOf && action.Type == GetValueOf &&
			!reflect.DeepEqual(action.GetValueOf, tc.expected.GetValueOf) {
			t.Errorf(
				"expected getValueOf to be '%v' but got '%v'",
				tc.expected.GetValueOf,
				action.GetValueOf,
			)
		}
	}
}

func TestFindPathFromMap(t *testing.T) {
	testCases := []struct {
		name      string
		beforeMap map[string]interface{}
		expected  Path
	}{
		{
			name: "Basic nested map with array",
			beforeMap: map[string]interface{}{
				"foo":   "bar",
				"miles": "{{.distance}}",
				"age":   "28",
				"person": map[string]interface{}{
					"name":   "Jane Doe",
					"age":    "{{.Age}}",
					"height": "{{getValueOf 'key1' 'path2'}}",
				},
				"users": []map[string]interface{}{
					{
						"person": map[string]interface{}{
							"name":   "Jane Doe",
							"age":    "{{.Age}}",
							"height": "{{getValueOf 'key2' 'path1'}}",
						},
					},
				},
			},
			expected: Path{
				DotStrings: []EachDotStringAction{
					{Path: "miles", KeyName: "distance"},
					{Path: "person -> age", KeyName: "Age"},
					{Path: "users[0] -> person -> age", KeyName: "Age"},
				},
				GetValueOfs: []EachGetValueofAction{
					{Path: "person -> height", KeyName: "key1", FileName: "path2"},
					{Path: "users[0] -> person -> height", KeyName: "key2", FileName: "path1"},
				},
			},
		},
		{
			name: "Map with empty array element",
			beforeMap: map[string]interface{}{
				"foo":   "bar",
				"miles": "{{get}}",
				"age":   "28",
				"person": map[string]interface{}{
					"name":   "{{.jane}}",
					"age":    "{{.Age}}",
					"height": "{{getValueOf 'key1' 'path2'}}",
				},
				"users": []map[string]interface{}{
					{},
					{
						"person": map[string]interface{}{
							"name":   "Jane Doe",
							"age":    "{{.age}}",
							"height": "{{getValueOf 'key2' 'path1'}}",
						},
					},
				},
			},
			expected: Path{
				DotStrings: []EachDotStringAction{
					{Path: "person -> name", KeyName: "jane"},
					{Path: "person -> age", KeyName: "Age"},
					{Path: "users[1] -> person -> age", KeyName: "age"},
				},
				GetValueOfs: []EachGetValueofAction{
					{Path: "person -> height", KeyName: "key1", FileName: "path2"},
					{Path: "users[1] -> person -> height", KeyName: "key2", FileName: "path1"},
				},
			},
		},
		{
			name: "Complex nested structure with interface array",
			beforeMap: map[string]interface{}{
				"settings": map[string]interface{}{
					"users": []interface{}{
						map[string]interface{}{
							"id":       "{{.userId}}",
							"isActive": "{{getValueOf 'isActive' '/'}}",
						},
					},
					"config": map[string]interface{}{
						"maxCount": "{{.maxCount}}",
						"enabled":  "{{.enabled}}",
					},
				},
			},
			expected: Path{
				DotStrings: []EachDotStringAction{
					{Path: "settings -> users[0] -> id", KeyName: "userId"},
					{Path: "settings -> config -> maxCount", KeyName: "maxCount"},
					{Path: "settings -> config -> enabled", KeyName: "enabled"},
				},
				GetValueOfs: []EachGetValueofAction{
					{Path: "settings -> users[0] -> isActive", KeyName: "isActive", FileName: "/"},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := findPathFromMap(tc.beforeMap, "")
			if !comparePaths(tc.expected, result) {
				expected, _ := utils.MarshalToJSON(tc.expected)
				res, _ := utils.MarshalToJSON(result)
				t.Errorf("\nTest case: %s\nFindPathFromMap error: \nexpected \n%+v, \ngot \n%+v",
					tc.name, expected, res)
			}
		})
	}
}

func TestCleanStrings(t *testing.T) {
	tests := []struct {
		name           string
		input          []string
		expectedOutput []string
	}{
		{
			name:           "Basic replacement",
			input:          []string{`"test"`, "`example`"},
			expectedOutput: []string{"test", "example"},
		},
		{
			name:           "Replace Single quotes",
			input:          []string{`'test'`, "'example'"},
			expectedOutput: []string{"test", "example"},
		},
		{
			name:           "No replacement needed",
			input:          []string{"hello", "world"},
			expectedOutput: []string{"hello", "world"},
		},
		{
			name:           "Empty string",
			input:          []string{""},
			expectedOutput: []string{""},
		},
		{
			name:           "Multiple replacements in one string",
			input:          []string{`"He"llo"`, "`Te`st`", `"Mu"l`},
			expectedOutput: []string{"Hello", "Test", "Mul"},
		},
		{
			name:           "Mixed content",
			input:          []string{`He"llo`, "Wor`ld", `Tes"t'`},
			expectedOutput: []string{"Hello", "World", "Test"},
		},
		{
			name:           "Special characters and whitespace",
			input:          []string{"` ", `" "`, `"hello`},
			expectedOutput: []string{" ", " ", "hello"},
		},
		{
			name:           "Large input",
			input:          []string{`"a"`, "`b`", `"c"`, "`d`"},
			expectedOutput: []string{"a", "b", "c", "d"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := cleanStrings(tt.input)
			if !equal(output, tt.expectedOutput) {
				t.Errorf("got %v, want %v", output, tt.expectedOutput)
			}
		})
	}
}

func TestParsePath(t *testing.T) {
	testCases := []struct {
		name   string
		input  string
		output []interface{}
		error  string
	}{
		{
			name:   "empty string",
			input:  "",
			output: []interface{}{},
			error:  "path should not be empty",
		},
		{
			name:   "single key",
			input:  "key",
			output: []interface{}{"key"},
			error:  "",
		},
		{
			name:   "multiple keys with no arrays",
			input:  "key1 -> key2 -> key3",
			output: []interface{}{"key1", "key2", "key3"},
			error:  "",
		},
		{
			name:   "array index in path",
			input:  "users[0] -> name",
			output: []interface{}{"users", 0, "name"},
			error:  "",
		},
		{
			name:   "complex path with arrays",
			input:  "users[2] -> address -> city",
			output: []interface{}{"users", 2, "address", "city"},
			error:  "",
		},
		{
			name:   "leading and trailing whitespace",
			input:  "  key1  ->   key2 ->   key3  ",
			output: []interface{}{"key1", "key2", "key3"},
			error:  "",
		},
		{
			name:   "empty key in path",
			input:  "key1 ->  -> key3",
			output: nil,
			error:  "Invalid format: empty key at position 2",
		},
		{
			name:   "array key with invalid format",
			input:  "users[invalid] -> name",
			output: []interface{}{"users[invalid]", "name"},
			error:  "",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parsePath(tt.input)
			if err != nil && !strings.Contains(err.Error(), tt.error) {
				t.Errorf(
					"error message do not match on parsePath. \nExpected \n%s \nbut got %s",
					tt.error,
					err.Error(),
				)
			}
			// cannot deep equal empty slices
			if len(tt.output) != len(result) && !reflect.DeepEqual(tt.output, result) {
				t.Errorf(
					"result does not match the expected output: expected %v, got %v",
					tt.output,
					result,
				)
			}
		})
	}
}

func TestSetValueOnAfterMap(t *testing.T) {
	testCases := []struct {
		name        string
		afterMap    map[string]interface{}
		path        []interface{}
		replaceWith interface{}
		expected    map[string]interface{}
	}{
		{
			name:        "Replace String with Int",
			afterMap:    map[string]interface{}{"age": "30"},
			path:        []interface{}{"age"},
			replaceWith: 30,
			expected:    map[string]interface{}{"age": 30},
		},
		{
			name:        "Replace String with bool",
			afterMap:    map[string]interface{}{"deploy": "false"},
			path:        []interface{}{"deploy"},
			replaceWith: false,
			expected:    map[string]interface{}{"deploy": false},
		},
		{
			name:        "Replace String with float",
			afterMap:    map[string]interface{}{"height": "300.12"},
			path:        []interface{}{"height"},
			replaceWith: 300.12,
			expected:    map[string]interface{}{"height": 300.12},
		},
		{
			name:        "Don't replace string if values don't match",
			afterMap:    map[string]interface{}{"height": "300.12"},
			path:        []interface{}{"height"},
			replaceWith: 300, // since 300 does not match with "300.12" when converted to string
			expected:    map[string]interface{}{"height": "300.12"},
		},
		{
			name:        "Don't replace string if values mismatch",
			afterMap:    map[string]interface{}{"height": "300.12"},
			path:        []interface{}{"height"},
			replaceWith: false,
			expected:    map[string]interface{}{"height": "300.12"},
		},
		{
			name: "Complex Map",
			afterMap: map[string]interface{}{
				"person": map[string]interface{}{
					"contact": map[string]interface{}{
						"phone": "2222222222",
					},
				},
			},
			path:        []interface{}{"person", "contact", "phone"},
			replaceWith: 2222222222,
			expected: map[string]interface{}{
				"person": map[string]interface{}{
					"contact": map[string]interface{}{
						"phone": 2222222222,
					},
				},
			},
		},
		{
			name: "Complex Map with incorrect info should return the same map",
			afterMap: map[string]interface{}{
				"person": map[string]interface{}{
					"contact": map[string]interface{}{
						"phone": "2222222222",
					},
				},
			},
			path:        []interface{}{"person", "contact", "phone"},
			replaceWith: 2333939,
			expected: map[string]interface{}{
				"person": map[string]interface{}{
					"contact": map[string]interface{}{
						"phone": "2222222222",
					},
				},
			},
		},
		{
			name: "Complex Map with Array",
			afterMap: map[string]interface{}{
				"person": []interface{}{
					map[string]interface{}{
						"contact": map[string]interface{}{
							"phone": "2222222222",
						},
					},
					map[string]interface{}{
						"contact": map[string]interface{}{
							"phone": "3333333333",
						},
					},
				},
			},
			path:        []interface{}{"person", 1, "contact", "phone"},
			replaceWith: 3333333333,
			expected: map[string]interface{}{
				"person": []interface{}{
					map[string]interface{}{
						"contact": map[string]interface{}{
							"phone": "2222222222",
						},
					},
					map[string]interface{}{
						"contact": map[string]interface{}{
							"phone": 3333333333,
						},
					},
				},
			},
		},
	}

	for _, tt := range testCases {
		result := setValueOnAfterMap(tt.path, tt.afterMap, tt.replaceWith)
		if !reflect.DeepEqual(result, tt.expected) {
			expectedStr, _ := utils.MarshalToJSON(tt.expected)
			resStr, _ := utils.MarshalToJSON(result)
			t.Errorf(
				"error in '%s' expected the map \n'%v' \n but got \n'%v'",
				tt.name,
				expectedStr,
				resStr,
			)
		}
	}
}

func TestTranslateType(t *testing.T) {
	// Mock implementation of getValueOf function
	getValueOfMock := func(key, fileName string) interface{} {
		mockValues := map[string]interface{}{
			"bar":         true,
			"height":      300.2,
			"isActive":    false,
			"count":       42,
			"temperature": 98.6,
			"multiplier":  1.5,
			"status":      "active",
			"nullValue":   nil,
			"emptyString": "",
		}
		return mockValues[key]
	}
	testCases := []struct {
		name        string
		before      map[string]interface{}
		after       map[string]interface{}
		secrets     map[string]interface{}
		getValueOf  func(key string, fileName string) interface{}
		modifiedMap map[string]interface{}
		wantErr     bool
		errMsg      string
	}{
		{
			name: "Basic Type Conversion",
			before: map[string]interface{}{
				"foo":  "{{.foo}}",
				"bar":  "{{getValueOf 'bar' '/'}}",
				"baz":  "{{.baz}}",
				"name": "Jane",
			},
			after: map[string]interface{}{
				"foo":  "22",      // should be converted to int
				"bar":  "true",    // should be converted to bool
				"baz":  "22.2292", // should remain string,
				"name": "Jane",
			},
			secrets: map[string]interface{}{
				"foo": 22,
				"baz": "22.2292",
			},
			getValueOf: getValueOfMock,
			modifiedMap: map[string]interface{}{
				"foo":  22,
				"bar":  true,
				"baz":  "22.2292",
				"name": "Jane",
			},
			wantErr: false,
			errMsg:  "",
		},
		{
			name: "One Nested Map",
			before: map[string]interface{}{
				"foo": "{{.fii}}",
				"baz": "{{.baz}}",
				"person": map[string]interface{}{
					"age":    "{{.age}}",
					"height": "{{getValueOf 'height' '/'}}",
				},
			},
			after: map[string]interface{}{
				"foo": "22",
				"baz": "22.2292",
				"person": map[string]interface{}{
					"age":    "39",
					"height": "300.2",
				},
			},
			secrets: map[string]interface{}{
				"fii": 22,
				"baz": "22.2292",
				"age": 39,
			},
			getValueOf: getValueOfMock,
			modifiedMap: map[string]interface{}{
				"foo": 22,
				"baz": "22.2292",
				"person": map[string]interface{}{
					"age":    39,
					"height": 300.2,
				},
			},
			wantErr: false,
			errMsg:  "",
		},
		{
			name: "Nested structure with arrays",
			before: map[string]interface{}{
				"settings": map[string]interface{}{
					"users": []interface{}{
						map[string]interface{}{
							"id":       "{{.userId}}",
							"isActive": "{{getValueOf 'isActive' '/'}}",
						},
					},
					"config": map[string]interface{}{
						"maxCount": "{{.maxCount}}",
						"enabled":  "{{.enabled}}",
					},
				},
			},
			after: map[string]interface{}{
				"settings": map[string]interface{}{
					"users": []interface{}{
						map[string]interface{}{
							"id":       "123",
							"isActive": "false",
						},
					},
					"config": map[string]interface{}{
						"maxCount": "100",
						"enabled":  "true",
					},
				},
			},
			modifiedMap: map[string]interface{}{
				"settings": map[string]interface{}{
					"users": []interface{}{
						map[string]interface{}{
							"id":       123,
							"isActive": false,
						},
					},
					"config": map[string]interface{}{
						"maxCount": 100,
						"enabled":  true,
					},
				},
			},
			secrets: map[string]interface{}{
				"userId":   123,
				"maxCount": 100,
				"enabled":  true,
			},
			getValueOf: getValueOfMock,
		},
		{
			name: "Multiple type conversions",
			before: map[string]interface{}{
				"metrics": map[string]interface{}{
					"count":       "{{getValueOf 'count' '/'}}",
					"temperature": "{{getValueOf 'temperature' '/'}}",
					"multiplier":  "{{getValueOf 'multiplier' '/'}}",
					"status":      "{{getValueOf 'status' '/'}}",
				},
			},
			after: map[string]interface{}{
				"metrics": map[string]interface{}{
					"count":       "42",
					"temperature": "98.6",
					"multiplier":  "1.5",
					"status":      "active",
				},
			},
			secrets:    map[string]interface{}{},
			getValueOf: getValueOfMock,
			modifiedMap: map[string]interface{}{
				"metrics": map[string]interface{}{
					"count":       42,
					"temperature": 98.6,
					"multiplier":  1.5,
					"status":      "active",
				},
			},
		},
		{
			name: "Empty and null values",
			before: map[string]interface{}{
				"empty":    "{{getValueOf 'emptyString' '/'}}",
				"nullVal":  "{{getValueOf 'nullValue' '/'}}",
				"missing":  "{{.missingKey}}",
				"preserve": "",
			},
			after: map[string]interface{}{
				"empty":    "",
				"nullVal":  "null",
				"missing":  "",
				"preserve": "",
			},
			secrets: map[string]interface{}{
				"missingKey": nil,
			},
			getValueOf: getValueOfMock,
			modifiedMap: map[string]interface{}{
				"empty":    "",
				"nullVal":  "null",
				"missing":  "",
				"preserve": "",
			},
		},
		{
			name: "Nested structures with arrays",
			before: map[string]interface{}{
				"settings": map[string]interface{}{
					"users": []interface{}{ // Use []interface{} for dynamic lists
						map[string]interface{}{
							"id":       "{{.userId}}",
							"isActive": "{{getValueOf 'isActive' '/'}}",
						},
					},
					"config": map[string]interface{}{
						"maxCount": "{{.maxCount}}",
						"enabled":  "{{.enabled}}",
					},
				},
			},
			after: map[string]interface{}{
				"settings": map[string]interface{}{
					"users": []interface{}{ // Use []interface{} for consistency
						map[string]interface{}{
							"id":       "123",
							"isActive": "false",
						},
					},
					"config": map[string]interface{}{
						"maxCount": "100",
						"enabled":  "true",
					},
				},
			},
			secrets: map[string]interface{}{
				"userId":   123,
				"maxCount": 100,
				"enabled":  false,
			},
			getValueOf: getValueOfMock,
			modifiedMap: map[string]interface{}{
				"settings": map[string]interface{}{
					"users": []interface{}{
						map[string]interface{}{
							"id":       123,
							"isActive": false,
						},
					},
					"config": map[string]interface{}{
						"maxCount": 100,
						"enabled":  "true", // since the secret enabled value does not match,
					},
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resultMap, err := translateType(tt.before, tt.after, tt.secrets, tt.getValueOf)

			// Error case handling
			if tt.wantErr {
				if err == nil {
					t.Errorf(
						"TranslateType() expected error containing %q, got no error",
						tt.errMsg,
					)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("TranslateType() error = %v, want error containing %q", err, tt.errMsg)
				}
				return
			}

			// Success case handling
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(tt.modifiedMap, resultMap) {
				mod, _ := utils.MarshalToJSON(tt.modifiedMap)
				res, _ := utils.MarshalToJSON(resultMap)
				t.Errorf(
					"TranslateType error:\nExpected:\n%s\nGot:\n%s",
					mod,
					res,
				)
			}
		})
	}
}

// Helper function for slice comparison
func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Helper function for FindPathFromMap
func comparePaths(expected, actual Path) bool {
	// If the slices have different lengths, they cannot be equal
	if len(expected.DotStrings) != len(actual.DotStrings) ||
		len(expected.GetValueOfs) != len(actual.GetValueOfs) {
		return false
	}

	// Compare DotStrings slices without considering the order
	if !compareUnorderedDotStrings(expected.DotStrings, actual.DotStrings) {
		return false
	}

	// Compare GetValueOfs slices without considering the order
	if !compareUnorderedGetValueOfs(expected.GetValueOfs, actual.GetValueOfs) {
		return false
	}

	return true
}

func compareUnorderedDotStrings(expected, actual []EachDotStringAction) bool {
	// If the slices have different lengths, they cannot be equal
	if len(expected) != len(actual) {
		return false
	}

	// Create maps to count occurrences of each struct
	expectedCounts := make(map[string]int)
	actualCounts := make(map[string]int)

	// Count occurrences in the expected slice
	for _, v := range expected {
		// Create a unique key for the struct based on its fields
		structKey := v.Path + v.KeyName
		expectedCounts[structKey]++
	}

	// Count occurrences in the actual slice
	for _, v := range actual {
		// Create a unique key for the struct based on its fields
		structKey := v.Path + v.KeyName
		actualCounts[structKey]++
	}

	// Compare the counts
	for key, count := range expectedCounts {
		if actualCounts[key] != count {
			return false
		}
	}

	return true
}

func compareUnorderedGetValueOfs(expected, actual []EachGetValueofAction) bool {
	// If the slices have different lengths, they cannot be equal
	if len(expected) != len(actual) {
		return false
	}

	// Create maps to count occurrences of each struct
	expectedCounts := make(map[string]int)
	actualCounts := make(map[string]int)

	// Count occurrences in the expected slice
	for _, v := range expected {
		// Create a unique key for the struct based on its fields
		structKey := v.Path + v.KeyName + v.FileName
		expectedCounts[structKey]++
	}

	// Count occurrences in the actual slice
	for _, v := range actual {
		// Create a unique key for the struct based on its fields
		structKey := v.Path + v.KeyName + v.FileName
		actualCounts[structKey]++
	}

	// Compare the counts
	for key, count := range expectedCounts {
		if actualCounts[key] != count {
			return false
		}
	}

	return true
}
