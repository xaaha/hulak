package tui

import (
	"fmt"
	"os"
	"time"
)

const spinnerDelay = 100 * time.Millisecond

func RunWithSpinnerAfter(message string, task func() (any, error)) (any, error) {
	stdinInfo, stdinErr := os.Stdin.Stat()
	stdoutInfo, stdoutErr := os.Stdout.Stat()
	if stdinErr != nil || stdoutErr != nil {
		return task()
	}

	stdinTTY := (stdinInfo.Mode() & os.ModeCharDevice) != 0
	stdoutTTY := (stdoutInfo.Mode() & os.ModeCharDevice) != 0
	if !stdinTTY || !stdoutTTY {
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

	frames := []rune{'|', '/', '-', '\\'}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	clearLine := "\r" + fmt.Sprintf("%*s", len(message)+2, "") + "\r"

	index := 0
	for {
		select {
		case completed := <-done:
			_, _ = fmt.Fprint(os.Stdout, clearLine)
			return completed.result, completed.err
		case <-ticker.C:
			_, _ = fmt.Fprintf(os.Stdout, "\r%c %s", frames[index], message)
			index = (index + 1) % len(frames)
		}
	}
}
