package tui

import "strings"

// DeleteChar removes the last character from the string.
// Returns the string unchanged if empty.
func DeleteChar(s string) string {
	if len(s) == 0 {
		return s
	}
	return s[:len(s)-1]
}

// DeleteLastWord removes the last word from the string.
// Used for ctrl+w functionality in text inputs.
func DeleteLastWord(s string) string {
	if s == "" {
		return ""
	}
	s = strings.TrimRight(s, " ")
	lastSpace := strings.LastIndex(s, " ")
	if lastSpace == -1 {
		return ""
	}
	return s[:lastSpace+1]
}

// ClearLine returns an empty string.
// Used for ctrl+u functionality in text inputs.
func ClearLine() string {
	return ""
}

// AppendRunes appends runes to a string.
// Used for handling typed characters in text inputs.
func AppendRunes(s string, runes []rune) string {
	if len(runes) == 0 {
		return s
	}
	return s + string(runes)
}

// MoveCursorUp decrements cursor position, respecting lower bound of 0.
func MoveCursorUp(cursor int) int {
	if cursor > 0 {
		return cursor - 1
	}
	return cursor
}

// MoveCursorDown increments cursor position, respecting upper bound.
// maxIndex is the maximum valid index (typically len(items)-1).
func MoveCursorDown(cursor, maxIndex int) int {
	if cursor < maxIndex {
		return cursor + 1
	}
	return cursor
}

// ClampCursor ensures cursor is within valid bounds [0, maxIndex].
// Useful after filtering reduces the list size.
func ClampCursor(cursor, maxIndex int) int {
	if maxIndex < 0 {
		return 0
	}
	if cursor > maxIndex {
		return maxIndex
	}
	if cursor < 0 {
		return 0
	}
	return cursor
}

// --- Text cursor operations for input fields ---

// TextCursorLeft moves text cursor left by one position.
func TextCursorLeft(cursor int) int {
	if cursor > 0 {
		return cursor - 1
	}
	return cursor
}

// TextCursorRight moves text cursor right, bounded by text length.
func TextCursorRight(cursor, textLen int) int {
	if cursor < textLen {
		return cursor + 1
	}
	return cursor
}

// TextCursorStart moves cursor to start of text.
func TextCursorStart() int {
	return 0
}

// TextCursorEnd moves cursor to end of text.
func TextCursorEnd(textLen int) int {
	return textLen
}

// InsertAtCursor inserts runes at cursor position and returns new text and cursor.
func InsertAtCursor(text string, cursor int, runes []rune) (string, int) {
	if len(runes) == 0 {
		return text, cursor
	}
	// Clamp cursor to valid range
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(text) {
		cursor = len(text)
	}
	insert := string(runes)
	newText := text[:cursor] + insert + text[cursor:]
	return newText, cursor + len(insert)
}

// DeleteCharAtCursor deletes character before cursor (backspace behavior).
// Returns new text and new cursor position.
func DeleteCharAtCursor(text string, cursor int) (string, int) {
	if cursor <= 0 || len(text) == 0 {
		return text, cursor
	}
	newText := text[:cursor-1] + text[cursor:]
	return newText, cursor - 1
}

// DeleteWordAtCursor deletes word before cursor (ctrl+w behavior).
// Returns new text and new cursor position.
func DeleteWordAtCursor(text string, cursor int) (string, int) {
	if cursor <= 0 {
		return text, cursor
	}

	// Get text before cursor
	before := text[:cursor]
	after := text[cursor:]

	// Delete last word from before
	before = strings.TrimRight(before, " ")
	lastSpace := strings.LastIndex(before, " ")
	if lastSpace == -1 {
		// No space found, delete everything before cursor
		return after, 0
	}
	newBefore := before[:lastSpace+1]
	return newBefore + after, len(newBefore)
}

// ClearToStart clears text from start to cursor (ctrl+u behavior).
// Returns new text and new cursor position.
func ClearToStart(text string, cursor int) (string, int) {
	if cursor >= len(text) {
		return "", 0
	}
	return text[cursor:], 0
}

// RenderTextWithCursor returns text with cursor character at position.
func RenderTextWithCursor(text string, cursor int, cursorChar string) string {
	if cursor >= len(text) {
		return text + cursorChar
	}
	return text[:cursor] + cursorChar + text[cursor:]
}
