// Tests for the shared secrets-handler prelude (resolveAndValidateEnv) and
// the env-existence assertion (requireEnvExists). These helpers are touched
// by every secrets command that targets a single environment, so the cases
// here cover every branch: project-not-found, picker-cancel, picker-error,
// invalid-name, and the happy path.
package secrets

import (
	"errors"
	"strings"
	"testing"

	"github.com/xaaha/hulak/pkg/vault"
)

func TestResolveAndValidateEnv(t *testing.T) {
	pickerErr := errors.New("picker boom")

	tests := []struct {
		name             string
		envName          string
		pickerEnv        string
		pickerCancel     bool
		pickerErr        error
		wantResolved     string
		wantCancelled    bool
		wantErrContains  string
		wantPickerCalled bool
	}{
		{
			name:             "envName set bypasses picker, validates, passes through",
			envName:          "prod",
			wantResolved:     "prod",
			wantPickerCalled: false,
		},
		{
			name:             "empty envName triggers picker, returns env",
			envName:          "",
			pickerEnv:        "staging",
			wantResolved:     "staging",
			wantPickerCalled: true,
		},
		{
			name:             "picker cancel surfaces as cancelled=true with nil err",
			envName:          "",
			pickerCancel:     true,
			wantCancelled:    true,
			wantPickerCalled: true,
		},
		{
			name:             "picker error propagates",
			envName:          "",
			pickerErr:        pickerErr,
			wantErrContains:  "picker boom",
			wantPickerCalled: true,
		},
		{
			name:             "invalid env name fails validation",
			envName:          "bad/name",
			wantErrContains:  "environment",
			wantPickerCalled: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Tests run inside a real vault project so requireVaultProject passes.
			setupVaultProject(t)

			prev := envPicker
			t.Cleanup(func() { envPicker = prev })

			called := false
			envPicker = func() (string, bool, error) {
				called = true
				return tc.pickerEnv, tc.pickerCancel, tc.pickerErr
			}

			gotEnv, gotCancelled, gotErr := resolveAndValidateEnv(tc.envName)

			if called != tc.wantPickerCalled {
				t.Errorf("picker called = %v, want %v", called, tc.wantPickerCalled)
			}
			if gotEnv != tc.wantResolved {
				t.Errorf("env = %q, want %q", gotEnv, tc.wantResolved)
			}
			if gotCancelled != tc.wantCancelled {
				t.Errorf("cancelled = %v, want %v", gotCancelled, tc.wantCancelled)
			}
			if tc.wantErrContains == "" {
				if gotErr != nil {
					t.Errorf("unexpected error: %v", gotErr)
				}
			} else {
				if gotErr == nil || !strings.Contains(gotErr.Error(), tc.wantErrContains) {
					t.Errorf("err = %v, want substring %q", gotErr, tc.wantErrContains)
				}
			}
		})
	}
}

// TestResolveAndValidateEnvOutsideVaultProject covers the requireVaultProject
// branch — the prelude's job is to refuse before we ever ask for an env.
func TestResolveAndValidateEnvOutsideVaultProject(t *testing.T) {
	t.Cleanup(chdirTemp(t, t.TempDir()))

	prev := envPicker
	t.Cleanup(func() { envPicker = prev })
	envPicker = func() (string, bool, error) {
		t.Fatal("picker must not be called when the project check fails")
		return "", false, nil
	}

	_, _, err := resolveAndValidateEnv("prod")
	if err == nil {
		t.Fatal("expected error outside vault project, got nil")
	}
	if !strings.Contains(err.Error(), "no vault project") {
		t.Errorf("expected 'no vault project' in error, got %v", err)
	}
}

func TestRequireEnvExists(t *testing.T) {
	store := &vault.Store{Envs: map[string]vault.Env{
		"prod": {"API_KEY": "sk-xxx"},
	}}

	t.Run("returns env when present", func(t *testing.T) {
		env, err := requireEnvExists(store, "prod")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if env["API_KEY"] != "sk-xxx" {
			t.Errorf("got %v, want sk-xxx", env["API_KEY"])
		}
	})

	t.Run("errors when env missing", func(t *testing.T) {
		_, err := requireEnvExists(store, "missing")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), `environment "missing"`) {
			t.Errorf("error wording should quote env name, got: %v", err)
		}
	})

	t.Run("errors on empty store", func(t *testing.T) {
		empty := &vault.Store{Envs: map[string]vault.Env{}}
		_, err := requireEnvExists(empty, "anything")
		if err == nil {
			t.Fatal("expected error on empty store")
		}
	})
}
