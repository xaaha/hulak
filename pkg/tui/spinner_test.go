package tui

import (
	"errors"
	"testing"
	"time"
)

func TestRunWithSpinnerAfterFastTask(t *testing.T) {
	result, err := RunWithSpinnerAfter("Loading...", func() (any, error) {
		return "done", nil
	})

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result != "done" {
		t.Fatalf("expected result done, got %v", result)
	}
}

func TestRunWithSpinnerAfterSlowTask(t *testing.T) {
	result, err := RunWithSpinnerAfter("Loading...", func() (any, error) {
		time.Sleep(200 * time.Millisecond)
		return "done", nil
	})

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result != "done" {
		t.Fatalf("expected result done, got %v", result)
	}
}

func TestRunWithSpinnerAfterReturnsTaskError(t *testing.T) {
	wantErr := errors.New("boom")
	_, err := RunWithSpinnerAfter("Loading...", func() (any, error) {
		return nil, wantErr
	})

	if err == nil {
		t.Fatal("expected an error")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

// TestRunWithSpinnerOnStderr verifies the stderr variant runs the task and
// returns its result. The actual spinner frames cannot be inspected here
// because the test harness runs without a TTY on stderr, so the wrapper
// falls through to the task directly. That fallthrough is itself the
// behavior we want to guarantee for piped/CI invocations of `hulak run`.
func TestRunWithSpinnerOnStderr(t *testing.T) {
	result, err := RunWithSpinnerOnStderr("Running...", func() (any, error) {
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result != "ok" {
		t.Fatalf("expected result ok, got %v", result)
	}
}

func TestRunWithSpinnerOnStderrSlowTask(t *testing.T) {
	result, err := RunWithSpinnerOnStderr("Running...", func() (any, error) {
		time.Sleep(200 * time.Millisecond)
		return 42, nil
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result != 42 {
		t.Fatalf("expected result 42, got %v", result)
	}
}

func TestRunWithSpinnerOnStderrPropagatesError(t *testing.T) {
	wantErr := errors.New("api boom")
	_, err := RunWithSpinnerOnStderr("Running...", func() (any, error) {
		return nil, wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}
