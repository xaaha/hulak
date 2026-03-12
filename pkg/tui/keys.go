// Package tui provides shared utilities for terminal UI components.
package tui

// Common Key binding and other constants
// single source of truth for all TUI components.
const (
	// Navigation
	KeyUp    = "up"
	KeyDown  = "down"
	KeyCtrlP = "ctrl+p" // up
	KeyCtrlN = "ctrl+n" // down
	KeyLeft  = "left"
	KeyRight = "right"

	// Vim-style navigation (use only when text input is inactive)
	KeyJ             = "j"
	KeyK             = "k"
	KeyH             = "h"
	KeyL             = "l"
	KeyG             = "g"      // first press of gg (go to top)
	KeyShiftG        = "G"      // go to bottom
	KeyYank          = "ctrl+y" // single 'y' key suffers when the cursor is in TextInput
	KeyRefresh       = "ctrl+r"
	KeySend          = "ctrl+u"
	KeySave          = "ctrl+s"
	KeySaveQuery     = "ctrl+q"
	KeyCreateRequest = "ctrl+x"
	KeySlash         = "/" // vim-style search trigger
	KeyAt            = "@" // reopen or hide the most recent notification

	// Actions
	KeyEnter    = "enter"
	KeyTab      = "tab"
	KeySpace    = " "
	KeyShiftTab = "shift+tab" // Reverse tab - navigate backward
	KeyQuit     = "ctrl+c"    // Force quit - always works
	KeyCancel   = "esc"       // Cancel/back - context aware

	// Rendering
	listPadding = "   " // 3-space indent for non-selected list items
)
