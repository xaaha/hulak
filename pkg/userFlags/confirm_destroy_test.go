package userflags

import (
	"errors"
	"strings"
	"testing"
)

// stubConfirm swaps confirmActionFn for the duration of t. The fake records
// the prompt it received so tests can assert on the user-facing wording, and
// returns whatever (ok, err) the caller pre-configured.
func stubConfirm(t *testing.T, ok bool, err error) *string {
	t.Helper()
	prev := confirmActionFn
	t.Cleanup(func() { confirmActionFn = prev })

	captured := ""
	confirmActionFn = func(prompt string) (bool, error) {
		captured = prompt
		return ok, err
	}
	return &captured
}

func TestConfirmDestroy(t *testing.T) {
	t.Run("force skips prompt and returns true", func(t *testing.T) {
		prompt := stubConfirm(t, false, errors.New("must not be called"))
		ok, err := confirmDestroy("keys in \"prod\"", 5, true /*force*/)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ok {
			t.Error("force=true should return true")
		}
		if *prompt != "" {
			t.Errorf("prompt should not have been called, got %q", *prompt)
		}
	})

	t.Run("count zero skips prompt and returns true", func(t *testing.T) {
		prompt := stubConfirm(t, false, errors.New("must not be called"))
		ok, err := confirmDestroy("keys in \"prod\"", 0, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ok {
			t.Error("count=0 should return true (nothing to destroy)")
		}
		if *prompt != "" {
			t.Errorf("prompt should not have been called, got %q", *prompt)
		}
	})

	t.Run("user accepts", func(t *testing.T) {
		prompt := stubConfirm(t, true, nil)
		ok, err := confirmDestroy("keys in \"prod\"", 3, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ok {
			t.Error("expected true on confirm")
		}
		if !strings.Contains(*prompt, "3 keys in \"prod\"") {
			t.Errorf("prompt did not include count+description; got %q", *prompt)
		}
		if !strings.Contains(*prompt, "Continue? [y/N]") {
			t.Errorf("prompt should ask Continue? [y/N]; got %q", *prompt)
		}
	})

	t.Run("user declines", func(t *testing.T) {
		stubConfirm(t, false, nil)
		ok, err := confirmDestroy("keys in \"prod\"", 3, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ok {
			t.Error("expected false on decline")
		}
	})

	t.Run("propagates prompt error", func(t *testing.T) {
		want := errors.New("scanner boom")
		stubConfirm(t, false, want)
		_, err := confirmDestroy("keys in \"prod\"", 1, false)
		if !errors.Is(err, want) {
			t.Errorf("got err = %v, want %v", err, want)
		}
	})

	t.Run("singular caller phrasing is respected", func(t *testing.T) {
		// The helper doesn't pluralize. Caller passes singular phrasing
		// when count == 1; this test makes that contract explicit.
		prompt := stubConfirm(t, true, nil)
		_, err := confirmDestroy("key in \"prod\"", 1, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(*prompt, "1 key in \"prod\"") {
			t.Errorf("expected singular phrasing, got %q", *prompt)
		}
	})
}
