package tui

import (
	"fmt"
	"io"
	"os"
	"time"
)

const spinnerDelay = 100 * time.Millisecond

var SpinnerFrames = []rune{'|', '/', '-', '\\'}

// RunWithSpinnerAfter wraps a long-running task with a stdout spinner. The
// spinner only appears when both stdin and stdout are TTYs. It is shown after
// spinnerDelay, so quick tasks finish silently.
func RunWithSpinnerAfter(message string, task func() (any, error)) (any, error) {
	return runWithSpinner(os.Stdout, os.Stdout, message, task)
}

// RunWithSpinnerOnStderr is the stderr variant. Use this when the task's
// stdout carries the result the user wants to capture (e.g. a piped response
// body) and the spinner needs to live on the side channel instead.
func RunWithSpinnerOnStderr(message string, task func() (any, error)) (any, error) {
	return runWithSpinner(os.Stderr, os.Stderr, message, task)
}

// runWithSpinner does the work. ttyProbe is the file used for the isatty
// check; out is where the spinner frames are drawn. They are usually the same
// file but kept separate so callers can probe one channel and draw on another
// if needed.
func runWithSpinner(
	ttyProbe *os.File,
	out io.Writer,
	message string,
	task func() (any, error),
) (any, error) {
	if !isInteractiveTerminal(ttyProbe) {
		return task()
	}

	type taskResult struct {
		result any
		err    error
	}

	done := make(chan taskResult, 1)
	go func() {
		result, err := task()
		done <- taskResult{result: result, err: err}
	}()

	timer := time.NewTimer(spinnerDelay)
	defer timer.Stop()

	select {
	case completed := <-done:
		return completed.result, completed.err
	case <-timer.C:
	}

	frames := SpinnerFrames
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	clearLine := "\r" + fmt.Sprintf("%*s", len(message)+2, "") + "\r"

	index := 0
	for {
		select {
		case completed := <-done:
			_, _ = fmt.Fprint(out, clearLine)
			return completed.result, completed.err
		case <-ticker.C:
			_, _ = fmt.Fprintf(out, "\r%c %s", frames[index], message)
			index = (index + 1) % len(frames)
		}
	}
}

// isInteractiveTerminal reports whether f is attached to a TTY. Stdin is
// checked alongside the target so that piped input (cron jobs, CI) suppresses
// the spinner even when the output side happens to be a TTY.
func isInteractiveTerminal(f *os.File) bool {
	stdinInfo, stdinErr := os.Stdin.Stat()
	fInfo, fErr := f.Stat()
	if stdinErr != nil || fErr != nil {
		return false
	}
	if stdinInfo.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	if fInfo.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	return true
}
