package tui

import "github.com/charmbracelet/bubbles/viewport"

// DefaultScrollMargin is the number of lines kept visible above/below the
// cursor when scrolling inside a viewport-backed list.
const DefaultScrollMargin = 3

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

// SyncViewport updates vp's content and adjusts its YOffset so that
// cursorLine stays visible with scrollMargin lines of padding.
func SyncViewport(vp *viewport.Model, content string, cursorLine, scrollMargin int) {
	vp.SetContent(content)
	h := vp.Height
	if cursorLine < vp.YOffset {
		vp.SetYOffset(max(0, cursorLine-1))
	} else if cursorLine+scrollMargin >= vp.YOffset+h {
		vp.SetYOffset(cursorLine - h + 1 + scrollMargin)
	}
}
