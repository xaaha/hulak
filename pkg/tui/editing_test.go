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

// Text cursor tests

func TestTextCursorLeft(t *testing.T) {
	tests := []struct {
		name     string
		cursor   int
		expected int
	}{
		{"from middle", 5, 4},
		{"from 1", 1, 0},
		{"from 0 stays", 0, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := TextCursorLeft(tc.cursor)
			if result != tc.expected {
				t.Errorf("TextCursorLeft(%d) = %d, want %d", tc.cursor, result, tc.expected)
			}
		})
	}
}

func TestTextCursorRight(t *testing.T) {
	tests := []struct {
		name     string
		cursor   int
		textLen  int
		expected int
	}{
		{"from middle", 2, 5, 3},
		{"from 0", 0, 5, 1},
		{"at end stays", 5, 5, 5},
		{"empty text", 0, 0, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := TextCursorRight(tc.cursor, tc.textLen)
			if result != tc.expected {
				t.Errorf("TextCursorRight(%d, %d) = %d, want %d", tc.cursor, tc.textLen, result, tc.expected)
			}
		})
	}
}

func TestTextCursorStart(t *testing.T) {
	result := TextCursorStart()
	if result != 0 {
		t.Errorf("TextCursorStart() = %d, want 0", result)
	}
}

func TestTextCursorEnd(t *testing.T) {
	tests := []struct {
		textLen  int
		expected int
	}{
		{0, 0},
		{5, 5},
		{10, 10},
	}

	for _, tc := range tests {
		result := TextCursorEnd(tc.textLen)
		if result != tc.expected {
			t.Errorf("TextCursorEnd(%d) = %d, want %d", tc.textLen, result, tc.expected)
		}
	}
}

func TestInsertAtCursor(t *testing.T) {
	tests := []struct {
		name           string
		text           string
		cursor         int
		runes          []rune
		expectedText   string
		expectedCursor int
	}{
		{"insert at start", "hello", 0, []rune{'x'}, "xhello", 1},
		{"insert at middle", "hello", 2, []rune{'x'}, "hexllo", 3},
		{"insert at end", "hello", 5, []rune{'x'}, "hellox", 6},
		{"insert multiple", "hi", 1, []rune{'a', 'b'}, "habi", 3},
		{"empty runes", "hello", 2, []rune{}, "hello", 2},
		{"empty text", "", 0, []rune{'x'}, "x", 1},
		{"cursor beyond text", "", 10, []rune{'x'}, "x", 1},
		{"negative cursor", "hello", -5, []rune{'x'}, "xhello", 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			text, cursor := InsertAtCursor(tc.text, tc.cursor, tc.runes)
			if text != tc.expectedText {
				t.Errorf("InsertAtCursor text = %q, want %q", text, tc.expectedText)
			}
			if cursor != tc.expectedCursor {
				t.Errorf("InsertAtCursor cursor = %d, want %d", cursor, tc.expectedCursor)
			}
		})
	}
}

func TestDeleteCharAtCursor(t *testing.T) {
	tests := []struct {
		name           string
		text           string
		cursor         int
		expectedText   string
		expectedCursor int
	}{
		{"delete at end", "hello", 5, "hell", 4},
		{"delete in middle", "hello", 3, "helo", 2},
		{"delete at 1", "hello", 1, "ello", 0},
		{"cursor at 0", "hello", 0, "hello", 0},
		{"empty text", "", 0, "", 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			text, cursor := DeleteCharAtCursor(tc.text, tc.cursor)
			if text != tc.expectedText {
				t.Errorf("DeleteCharAtCursor text = %q, want %q", text, tc.expectedText)
			}
			if cursor != tc.expectedCursor {
				t.Errorf("DeleteCharAtCursor cursor = %d, want %d", cursor, tc.expectedCursor)
			}
		})
	}
}

func TestDeleteWordAtCursor(t *testing.T) {
	tests := []struct {
		name           string
		text           string
		cursor         int
		expectedText   string
		expectedCursor int
	}{
		{"delete last word", "hello world", 11, "hello ", 6},
		{"delete only word", "hello", 5, "", 0},
		{"cursor at 0", "hello", 0, "hello", 0},
		{"cursor in middle of word", "hello world", 8, "hello rld", 6},
		{"trailing spaces", "hello world  ", 13, "hello ", 6},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			text, cursor := DeleteWordAtCursor(tc.text, tc.cursor)
			if text != tc.expectedText {
				t.Errorf("DeleteWordAtCursor text = %q, want %q", text, tc.expectedText)
			}
			if cursor != tc.expectedCursor {
				t.Errorf("DeleteWordAtCursor cursor = %d, want %d", cursor, tc.expectedCursor)
			}
		})
	}
}

func TestClearToStart(t *testing.T) {
	tests := []struct {
		name           string
		text           string
		cursor         int
		expectedText   string
		expectedCursor int
	}{
		{"clear from middle", "hello world", 6, "world", 0},
		{"cursor at start", "hello", 0, "hello", 0},
		{"cursor at end", "hello", 5, "", 0},
		{"cursor beyond end", "hello", 10, "", 0},
		{"empty text", "", 0, "", 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			text, cursor := ClearToStart(tc.text, tc.cursor)
			if text != tc.expectedText {
				t.Errorf("ClearToStart text = %q, want %q", text, tc.expectedText)
			}
			if cursor != tc.expectedCursor {
				t.Errorf("ClearToStart cursor = %d, want %d", cursor, tc.expectedCursor)
			}
		})
	}
}

func TestRenderTextWithCursor(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		cursor     int
		cursorChar string
		expected   string
	}{
		{"cursor at start", "hello", 0, "|", "|hello"},
		{"cursor in middle", "hello", 2, "|", "he|llo"},
		{"cursor at end", "hello", 5, "|", "hello|"},
		{"cursor beyond end", "hello", 10, "|", "hello|"},
		{"empty text", "", 0, "|", "|"},
		{"block cursor", "hi", 1, "█", "h█i"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := RenderTextWithCursor(tc.text, tc.cursor, tc.cursorChar)
			if result != tc.expected {
				t.Errorf("RenderTextWithCursor = %q, want %q", result, tc.expected)
			}
		})
	}
}
