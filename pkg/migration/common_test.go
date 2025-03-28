package migration

import "testing"

func TestAddDotToTemplate(t *testing.T) {
	// Store current time for the version comment
	currentTime := "2025-03-23 16:16:58"
	user := "xaaha"

	t.Logf("// Unit tests created at %s by %s", currentTime, user)

	// Define test cases
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "String without pattern",
			input:    "str",
			expected: "str",
		},
		{
			name:     "Pattern without dot",
			input:    "{{value}}",
			expected: "{{.value}}",
		},
		{
			name:     "Pattern already has dot",
			input:    "{{.anyV}}",
			expected: "{{.anyV}}",
		},
		{
			name:     "Multiple patterns in one string",
			input:    "First {{value}} and second {{another}}",
			expected: "First {{.value}} and second {{.another}}",
		},
		{
			name:     "Mixed patterns with and without dots",
			input:    "{{value}} with {{.existing}}",
			expected: "{{.value}} with {{.existing}}",
		},
		{
			name:     "Multiple occurrences of same pattern",
			input:    "{{value}} {{value}} {{value}}",
			expected: "{{.value}} {{.value}} {{.value}}",
		},
		{
			name:     "Pattern with different content",
			input:    "{{someValue}}",
			expected: "{{.someValue}}",
		},
		{
			name:     "Pattern within text",
			input:    "This is a {{value}} in a sentence",
			expected: "This is a {{.value}} in a sentence",
		},
		{
			name:     "Complex case with multiple patterns",
			input:    "Template with {{first}}, {{.second}}, and {{third}}",
			expected: "Template with {{.first}}, {{.second}}, and {{.third}}",
		},
		{
			name:     "Pattern with numbers",
			input:    "{{value123}}",
			expected: "{{.value123}}",
		},
		{
			name:     "Pattern with underscore",
			input:    "{{value_name}}",
			expected: "{{.value_name}}",
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := addDotToTemplate(tc.input)
			if result != tc.expected {
				t.Errorf("Test case '%s' failed:\nInput: %q\nExpected: %q\nGot: %q",
					tc.name, tc.input, tc.expected, result)
			}
		})
	}
}

func BenchmarkAddDotToTemplate(b *testing.B) {
	input := "Template with {{first}}, {{.second}}, and {{third}}"

	for b.Loop() {
		addDotToTemplate(input)
	}
}
