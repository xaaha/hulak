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
	KeyJ = "j"
	KeyK = "k"

	// Actions
	KeyEnter  = "enter"
	KeyTab    = "tab"
	KeySpace  = " "
	KeyQuit   = "ctrl+c" // Force quit - always works
	KeyCancel = "esc"    // Cancel/back - context aware

	// Rendering
	listPadding = "   " // 3-space indent for non-selected list items

	// Required
	Asterisk = "*"

	Connector = "└─"
)
