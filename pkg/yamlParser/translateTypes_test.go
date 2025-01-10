package yamlParser

import "testing"

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
