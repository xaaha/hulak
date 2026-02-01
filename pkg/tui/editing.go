package tui

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
