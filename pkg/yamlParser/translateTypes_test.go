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
		{input: "", expected: Action{}},
		{input: "{{.Value}}", expected: Action{Type: DotString, DotString: "Value"}},
		{
			input:    `{{getValueOf "key" "value"}}`,
			expected: Action{Type: GetValueOf, GetValueOf: []string{"getValueOf", "key", "value"}},
		},
	}

	for _, tc := range testCases {
		action := delimiterLogic(tc.input)
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
		if !reflect.DeepEqual(action.GetValueOf, tc.expected.GetValueOf) {
			t.Errorf(
				"expected getValueOf to be '%v' but got '%v'",
				tc.expected.GetValueOf,
				action.GetValueOf,
			)
		}
	}
}
