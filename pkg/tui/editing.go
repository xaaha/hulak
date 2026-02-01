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
