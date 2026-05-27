//go:build !windows

// Helpers for opening the controlling terminal directly, so a TUI can render
// even when stdin or stdout is being used as a data channel (e.g. a pipe
// feeding `--stdin`). Without this, bubbletea's default behavior of reading
// from os.Stdin would consume the piped input meant for the command, and the
// picker would render nowhere useful.
package tui

import "os"

// OpenControllingTerminal opens /dev/tty for read+write so a TUI can run
// independently of stdin/stdout redirection. Returns the file (caller must
// Close) or an error if no controlling terminal is reachable — e.g. a
// detached process, a daemon, or a true CI job without a PTY.
func OpenControllingTerminal() (*os.File, error) {
	return os.OpenFile("/dev/tty", os.O_RDWR, 0)
}
