package tui

import "strings"

// DeleteLastWord removes the last word from the string.
// Used for ctrl+w functionality in text inputs.
func DeleteLastWord(s string) string {
	if s == "" {
		return ""
	}
	// Trim trailing spaces first
	s = strings.TrimRight(s, " ")
	// Find last space
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
