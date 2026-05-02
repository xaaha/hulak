package utils

import (
	"os"
	"testing"
)

func TestConfirmAction_Yes(t *testing.T) {
	for _, input := range []string{"y\n", "yes\n", "Y\n", "YES\n", "Yes\n"} {
		r, w, _ := os.Pipe()
		if _, err := w.WriteString(input); err != nil {
			t.Fatalf("input %q: failed to write to pipe: %v", input, err)
		}
		w.Close()

		orig := os.Stdin
		os.Stdin = r
		t.Cleanup(func() { os.Stdin = orig })

		got, err := ConfirmAction("Continue? [y/N] ")
		if err != nil {
			t.Fatalf("input %q: unexpected error: %v", input, err)
		}
		if !got {
			t.Errorf("input %q: want true, got false", input)
		}
	}
}

func TestConfirmAction_No(t *testing.T) {
	for _, input := range []string{"n\n", "no\n", "\n", "anything\n"} {
		r, w, _ := os.Pipe()
		if _, err := w.WriteString(input); err != nil {
			t.Fatalf("input %q: failed to write to pipe: %v", input, err)
		}
		w.Close()

		orig := os.Stdin
		os.Stdin = r
		t.Cleanup(func() { os.Stdin = orig })

		got, err := ConfirmAction("Continue? [y/N] ")
		if err != nil {
			t.Fatalf("input %q: unexpected error: %v", input, err)
		}
		if got {
			t.Errorf("input %q: want false, got true", input)
		}
	}
}

func TestConfirmAction_EOF(t *testing.T) {
	r, w, _ := os.Pipe()
	w.Close()

	orig := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = orig })

	got, err := ConfirmAction("Continue? [y/N] ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Error("EOF stdin should return false")
	}
}
