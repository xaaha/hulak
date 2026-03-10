package tui

import (
	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
)

// CopiedMsg is sent after a clipboard write attempt.
// Err is nil on success.
type CopiedMsg struct{ Err error }

// CopyToClipboard returns a tea.Cmd that writes text to the system
// clipboard. The result is delivered as a CopiedMsg so the caller
// can show feedback or handle errors.
func CopyToClipboard(text string) tea.Cmd {
	return func() tea.Msg {
		return CopiedMsg{Err: clipboard.WriteAll(text)}
	}
}
