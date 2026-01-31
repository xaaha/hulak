package tui

import "testing"

func TestDeleteLastWord(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"single word", "hello", ""},
		{"two words", "hello world", "hello "},
		{"three words", "one two three", "one two "},
		{"trailing space", "hello world ", "hello "},
		{"multiple spaces", "hello  world", "hello  "},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := DeleteLastWord(tc.input)
			if result != tc.expected {
				t.Errorf("DeleteLastWord(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestClearLine(t *testing.T) {
	result := ClearLine()
	if result != "" {
		t.Errorf("ClearLine() = %q, want empty string", result)
	}
}
