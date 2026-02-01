package tui

import "testing"

func TestDeleteChar(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"single char", "a", ""},
		{"multiple chars", "hello", "hell"},
		{"unicode", "héllo", "héll"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := DeleteChar(tc.input)
			if result != tc.expected {
				t.Errorf("DeleteChar(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

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

func TestAppendRunes(t *testing.T) {
	tests := []struct {
		name     string
		str      string
		runes    []rune
		expected string
	}{
		{"empty both", "", nil, ""},
		{"empty runes", "hello", nil, "hello"},
		{"empty runes slice", "hello", []rune{}, "hello"},
		{"append single", "hello", []rune{'!'}, "hello!"},
		{"append multiple", "hi", []rune{'!', '?'}, "hi!?"},
		{"to empty string", "", []rune{'a', 'b'}, "ab"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := AppendRunes(tc.str, tc.runes)
			if result != tc.expected {
				t.Errorf("AppendRunes(%q, %v) = %q, want %q", tc.str, tc.runes, result, tc.expected)
			}
		})
	}
}

func TestMoveCursorUp(t *testing.T) {
	tests := []struct {
		name     string
		cursor   int
		expected int
	}{
		{"from middle", 5, 4},
		{"from 1", 1, 0},
		{"from 0 stays", 0, 0},
		{"negative stays", -1, -1}, // edge case, shouldn't happen but safe
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := MoveCursorUp(tc.cursor)
			if result != tc.expected {
				t.Errorf("MoveCursorUp(%d) = %d, want %d", tc.cursor, result, tc.expected)
			}
		})
	}
}

func TestMoveCursorDown(t *testing.T) {
	tests := []struct {
		name     string
		cursor   int
		maxIndex int
		expected int
	}{
		{"from middle", 2, 5, 3},
		{"from 0", 0, 5, 1},
		{"at max stays", 5, 5, 5},
		{"beyond max stays", 6, 5, 6}, // edge case
		{"single item", 0, 0, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := MoveCursorDown(tc.cursor, tc.maxIndex)
			if result != tc.expected {
				t.Errorf("MoveCursorDown(%d, %d) = %d, want %d", tc.cursor, tc.maxIndex, result, tc.expected)
			}
		})
	}
}

func TestClampCursor(t *testing.T) {
	tests := []struct {
		name     string
		cursor   int
		maxIndex int
		expected int
	}{
		{"within bounds", 2, 5, 2},
		{"at lower bound", 0, 5, 0},
		{"at upper bound", 5, 5, 5},
		{"below lower", -1, 5, 0},
		{"above upper", 10, 5, 5},
		{"empty list", 5, -1, 0},
		{"single item valid", 0, 0, 0},
		{"single item above", 1, 0, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ClampCursor(tc.cursor, tc.maxIndex)
			if result != tc.expected {
				t.Errorf("ClampCursor(%d, %d) = %d, want %d", tc.cursor, tc.maxIndex, result, tc.expected)
			}
		})
	}
}
