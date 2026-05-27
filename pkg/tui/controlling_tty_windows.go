//go:build windows

package tui

import (
	"errors"
	"os"
)

// OpenControllingTerminal is not implemented on Windows. The console fallback
// (CONIN$/CONOUT$) behaves differently enough from /dev/tty that wiring it
// into bubbletea risks a hang in CI. Returning an error keeps callers on
// the pre-existing "stdin is not a terminal" guard path on Windows.
func OpenControllingTerminal() (*os.File, error) {
	return nil, errors.New("controlling terminal opener not implemented on windows")
}
