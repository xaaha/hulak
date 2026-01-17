// Package tui provides shared utilities for terminal UI components.
package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// Key binding constants - single source of truth for all TUI components.
// IMPORTANT: Never use single letter keys (like 'q') for quit/cancel
// as they conflict with text input and filtering.
const (
	KeyQuit   = "ctrl+c" // Force quit - always works
	KeyCancel = "esc"    // Cancel/back - context aware
	KeyEnter  = "enter"  // Confirm/select
	KeyUp     = "up"
	KeyDown   = "down"
	KeyLeft   = "left"
	KeyRight  = "right"
	KeyTab    = "tab"
	KeySpace  = "space"
	KeySlash  = "/" // Start filter/search
	KeyHelp   = "?" // Show help
)

// CommonKeyMap provides standard key bindings for TUI components
type CommonKeyMap struct {
	Quit   key.Binding
	Cancel key.Binding
	Enter  key.Binding
	Up     key.Binding
	Down   key.Binding
	Help   key.Binding
}

// NewCommonKeyMap creates a new CommonKeyMap with default bindings
func NewCommonKeyMap() CommonKeyMap {
	return CommonKeyMap{
		Quit: key.NewBinding(
			key.WithKeys(KeyQuit),
			key.WithHelp("ctrl+c", "quit"),
		),
		Cancel: key.NewBinding(
			key.WithKeys(KeyCancel),
			key.WithHelp("esc", "cancel"),
		),
		Enter: key.NewBinding(
			key.WithKeys(KeyEnter),
			key.WithHelp("enter", "select"),
		),
		Up: key.NewBinding(
			key.WithKeys(KeyUp, "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys(KeyDown, "j"),
			key.WithHelp("↓/j", "down"),
		),
		Help: key.NewBinding(
			key.WithKeys(KeyHelp),
			key.WithHelp("?", "help"),
		),
	}
}

// IsQuitKey checks if the key message is the quit key (ctrl+c)
func IsQuitKey(msg tea.KeyMsg) bool {
	return msg.String() == KeyQuit
}

// IsCancelKey checks if the key message is the cancel key (esc)
func IsCancelKey(msg tea.KeyMsg) bool {
	return msg.String() == KeyCancel
}

// IsConfirmKey checks if the key message is the confirm key (enter)
func IsConfirmKey(msg tea.KeyMsg) bool {
	return msg.String() == KeyEnter
}

// IsNavigationKey checks if the key is for navigation (up/down/j/k)
func IsNavigationKey(msg tea.KeyMsg) bool {
	k := msg.String()
	return k == KeyUp || k == KeyDown || k == "j" || k == "k"
}

// HandleQuitCancel is a helper that handles quit and cancel keys.
// Returns (shouldQuit, wasCancelled).
// Use this in Update() methods for consistent quit/cancel behavior.
//
// Example:
//
//	case tea.KeyMsg:
//	    if quit, cancelled := tui.HandleQuitCancel(msg); quit {
//	        m.Cancelled = cancelled
//	        return m, tea.Quit
//	    }
func HandleQuitCancel(msg tea.KeyMsg) (shouldQuit bool, wasCancelled bool) {
	if IsQuitKey(msg) {
		return true, true
	}
	return false, false
}

// HandleQuitCancelWithFilter handles quit/cancel with filter mode awareness.
// When inFilterMode is true, esc won't cancel (allows exiting filter first).
// Returns (shouldQuit, wasCancelled).
//
// Example:
//
//	case tea.KeyMsg:
//	    if quit, cancelled := tui.HandleQuitCancelWithFilter(msg, m.list.SettingFilter()); quit {
//	        m.Cancelled = cancelled
//	        return m, tea.Quit
//	    }
func HandleQuitCancelWithFilter(msg tea.KeyMsg, inFilterMode bool) (shouldQuit bool, wasCancelled bool) {
	// ctrl+c always quits
	if IsQuitKey(msg) {
		return true, true
	}
	// esc only cancels when not in filter mode
	if IsCancelKey(msg) && !inFilterMode {
		return true, true
	}
	return false, false
}
