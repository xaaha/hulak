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
