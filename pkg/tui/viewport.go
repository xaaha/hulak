package tui

import "github.com/charmbracelet/bubbles/viewport"

// DefaultScrollMargin is the number of lines kept visible above/below the
// cursor when scrolling inside a viewport-backed list.
const DefaultScrollMargin = 3

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
