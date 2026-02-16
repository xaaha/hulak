// Package tui provides shared utilities for terminal UI components.
package tui

// Key binding constants - single source of truth for all TUI components.
// IMPORTANT: Never use single letter keys (like 'q') for quit/cancel
// as they conflict with text input and filtering.
const (
	// Navigation
	KeyUp    = "up"
	KeyDown  = "down"
	KeyLeft  = "left"
	KeyRight = "right"

	// Emacs-style navigation
	KeyCtrlP = "ctrl+p" // up
	KeyCtrlN = "ctrl+n" // down
	KeyCtrlF = "ctrl+f" // forward (right)
	KeyCtrlB = "ctrl+b" // backward (left)

	// Vim-style navigation (use only when text input is inactive)
	KeyJ = "j"
	KeyK = "k"

	// Actions
	KeyEnter  = "enter"
	KeyTab    = "tab"
	KeySpace  = "space"
	KeyQuit   = "ctrl+c" // Force quit - always works
	KeyCancel = "esc"    // Cancel/back - context aware

	// Text editing
	KeyBackspace = "backspace"
	KeyCtrlH     = "ctrl+h" // Alternative backspace
	KeyCtrlW     = "ctrl+w" // Delete word
	KeyCtrlU     = "ctrl+u" // Clear line
	KeyCtrlA     = "ctrl+a" // Beginning of line
	KeyCtrlE     = "ctrl+e" // End of line
	KeyCtrlK     = "ctrl+k" // Kill to end of line

	// Search/filter
	KeySlash = "/" // Start filter/search
	KeyHelp  = "?" // Show help
)
