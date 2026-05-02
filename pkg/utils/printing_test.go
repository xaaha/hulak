package utils

import (
	"os"
	"testing"
)

func TestConfirmAction(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		// Affirmative inputs
		{"lowercase y", "y\n", true},
		{"lowercase yes", "yes\n", true},
		{"uppercase Y", "Y\n", true},
		{"uppercase YES", "YES\n", true},
		{"mixed case Yes", "Yes\n", true},
		// Negative / other inputs
		{"lowercase n", "n\n", false},
		{"lowercase no", "no\n", false},
		{"empty line", "\n", false},
		{"arbitrary text", "anything\n", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, w, _ := os.Pipe()
			if _, err := w.WriteString(tt.input); err != nil {
				t.Fatalf("failed to write to pipe: %v", err)
			}
			w.Close()

			orig := os.Stdin
			os.Stdin = r
			t.Cleanup(func() { os.Stdin = orig })

			got, err := ConfirmAction("Continue? [y/N] ")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("ConfirmAction(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
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
