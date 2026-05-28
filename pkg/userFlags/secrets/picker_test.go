package secrets

import (
	"errors"
	"testing"
)

// TestResolveEnv covers the three branches of resolveEnv: bypass when envName
// is already set, picker-success, picker-cancel, and picker-error. The cancel
// case is the regression guard for the contract that callers rely on to do a
// clean exit on Esc — without it, callers fall through to ValidateEnvName and
// the user sees a misleading "environment name cannot be empty".
func TestResolveEnv(t *testing.T) {
	pickerErr := errors.New("picker boom")

	tests := []struct {
		name          string
		envName       string
		pickerEnv     string
		pickerCancel  bool
		pickerErr     error
		wantResolved  string
		wantCancelled bool
		wantErr       error
		// pickerCalled asserts whether the picker was invoked. envName != ""
		// must bypass it; envName == "" must call it exactly once.
		wantPickerCalled bool
	}{
		{
			name:             "envName set bypasses picker",
			envName:          "prod",
			wantResolved:     "prod",
			wantPickerCalled: false,
		},
		{
			name:             "empty envName + picker returns env",
			envName:          "",
			pickerEnv:        "staging",
			wantResolved:     "staging",
			wantPickerCalled: true,
		},
		{
			name:             "empty envName + picker cancelled",
			envName:          "",
			pickerCancel:     true,
			wantCancelled:    true,
			wantPickerCalled: true,
		},
		{
			name:             "empty envName + picker errors",
			envName:          "",
			pickerErr:        pickerErr,
			wantErr:          pickerErr,
			wantPickerCalled: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prev := envPicker
			t.Cleanup(func() { envPicker = prev })

			called := false
			envPicker = func() (string, bool, error) {
				called = true
				return tc.pickerEnv, tc.pickerCancel, tc.pickerErr
			}

			gotEnv, gotCancelled, gotErr := resolveEnv(tc.envName)

			if called != tc.wantPickerCalled {
				t.Errorf("picker called = %v, want %v", called, tc.wantPickerCalled)
			}
			if gotEnv != tc.wantResolved {
				t.Errorf("env = %q, want %q", gotEnv, tc.wantResolved)
			}
			if gotCancelled != tc.wantCancelled {
				t.Errorf("cancelled = %v, want %v", gotCancelled, tc.wantCancelled)
			}
			if !errors.Is(gotErr, tc.wantErr) {
				t.Errorf("err = %v, want %v", gotErr, tc.wantErr)
			}
		})
	}
}
