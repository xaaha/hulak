package yamlParser

import (
	"reflect"
	"testing"
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
