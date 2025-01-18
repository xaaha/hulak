package yamlParser

import (
	"fmt"
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
		beforeMap map[string]interface{}
		expected  map[ActionType][]string
	}{
		{
			beforeMap: map[string]interface{}{
				"foo":   "bar",
				"miles": "{{.distance}}",
				"age":   "28",
				"person": map[string]interface{}{
					"name":   "Jane Doe",
					"age":    "{{.Age}}",
					"height": "{{getValueOf 'key1', 'path2'}}",
				},
				"users": []map[string]interface{}{
					{
						"person": map[string]interface{}{
							"name":   "Jane Doe",
							"age":    "{{.Age}}",
							"height": "{{getValueOf 'key2', 'path1'}}",
						},
					},
				},
			},
			expected: map[ActionType][]string{
				DotString:  {"miles", "person -> age", "users[0] -> person -> age"},
				GetValueOf: {"person -> height", "users[0] -> person -> height"},
			},
		},
		{
			beforeMap: map[string]interface{}{
				"foo":   "bar",
				"miles": "{{get}}",
				"age":   "28",
				"person": map[string]interface{}{
					"name":   "{{.jane}}",
					"age":    "{{.Age}}",
					"height": "{{getValueOf 'key1', 'path2'}}",
				},
				"users": []map[string]interface{}{
					{
						"person": map[string]interface{}{
							"name":   "Jane Doe",
							"age":    "{{.Age}}",
							"height": "{{getValueOf 'key2', 'path1'}}",
						},
					},
				},
			},
			expected: map[ActionType][]string{
				DotString:  {"person -> name", "person -> age", "users[0] -> person -> age"},
				GetValueOf: {"person -> height", "users[0] -> person -> height"},
			},
		},
		{
			beforeMap: map[string]interface{}{
				"foo":   "bar",
				"miles": "{{.get}}",
				"age":   "28",
				"person": map[string]interface{}{
					"name":   "{{.jane}}",
					"age":    "{{.Age}}",
					"height": "{{getValueOf 'key1', 'path2'}}",
				},
				"users": []map[string]interface{}{
					{
						"person": map[string]interface{}{
							"name":   "Jane Doe",
							"age":    "{{.Age}}",
							"height": "{{getValueOf 'key2', 'path1'}}",
						},
					},
				},
			},
			expected: map[ActionType][]string{
				DotString: {
					"miles",
					"person -> name",
					"person -> age",
					"users[0] -> person -> age",
				},
				GetValueOf: {"person -> height", "users[0] -> person -> height"},
			},
		},
	}
	for _, tc := range testCases {
		result := findPathFromMap(tc.beforeMap, "")
		if !compareMaps(tc.expected, result) {
			t.Errorf("FindPathFromMap error: expected %v got %v", tc.expected, result)
		}
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
			expectedOutput: []string{"Hello", "World", "Test'"},
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

func TestTranslateType(t *testing.T) {
	testCases := []struct {
		name        string
		before      map[string]interface{}
		after       map[string]interface{}
		secrets     map[string]interface{}
		getValueOf  interface{}
		modifiedMap map[string]interface{}
	}{
		{
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
			getValueOf: true,
			modifiedMap: map[string]interface{}{
				"foo":  22,
				"bar":  true,
				"baz":  "22.2292",
				"name": "Jane",
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resultMap, err := TranslateType(tt.before, tt.after, tt.secrets, tt.getValueOf)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(tt.modifiedMap, resultMap) {
				mod, _ := utils.MarshalToJSON(tt.modifiedMap)
				fmt.Println("Expected modifiedMap", mod)

				res, _ := utils.MarshalToJSON(resultMap)
				fmt.Println("Result Map", res)

				t.Errorf(
					"TranslateType error: \nExpected \n%v, \ngot \n%v",
					tt.modifiedMap,
					resultMap,
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

func compareMaps(expected, actual map[ActionType][]string) bool {
	if len(expected) != len(actual) {
		return false
	}
	for key, expectedValues := range expected {
		actualValues, exists := actual[key]
		if !exists {
			return false
		}
		if !compareStringSlices(expectedValues, actualValues) {
			return false
		}
	}
	return true
}

func compareStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	aMap := make(map[string]int)
	bMap := make(map[string]int)

	for _, val := range a {
		aMap[val]++
	}
	for _, val := range b {
		bMap[val]++
	}
	for key, count := range aMap {
		if bMap[key] != count {
			return false
		}
	}
	return true
}
