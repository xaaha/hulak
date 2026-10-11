//go:build unix

package testutil

import (
	"os"
	"syscall"
	"testing"
	"time"
)

// WithinFIFOTimeout fails t when call stays blocked on a FIFO, writing EOF into fifos until it returns.
func WithinFIFOTimeout(t *testing.T, call func() error, fifos ...string) error {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- call() }()
	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
	}
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	giveUp := time.After(5 * time.Second)
	for {
		select {
		case <-done:
			t.Fatal("the call blocked on a FIFO")
			return nil
		case <-giveUp:
			t.Fatal("the call blocked on a FIFO and could not be unblocked")
			return nil
		case <-tick.C:
			for _, fifo := range fifos {
				if f, err := os.OpenFile(fifo, os.O_WRONLY|syscall.O_NONBLOCK, 0); err == nil {
					_ = f.Close()
				}
			}
		}
	}
}
