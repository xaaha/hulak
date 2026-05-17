// Contains tests for hulak secrets set, get, and delete handlers.
package userflags

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/xaaha/hulak/pkg/vault"
)

func TestRunEnvSet_PositionalValue(t *testing.T) {
	setupVaultProject(t)

	if err := runEnvSet([]string{"API_KEY", "sk-123"}, "global", false); err != nil {
		t.Fatalf("runEnvSet: %v", err)
	}

	if got := readStoredValue(t, "global", "API_KEY"); got != "sk-123" {
		t.Errorf("stored value = %v, want %q", got, "sk-123")
	}
}

func TestRunEnvSet_StdinValue(t *testing.T) {
	setupVaultProject(t)

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	if _, err := w.WriteString("stdin-secret\n"); err != nil {
		t.Fatalf("write pipe: %v", err)
	}
	_ = w.Close()

	orig := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = orig })

	if err := runEnvSet([]string{"TOKEN"}, "global", true); err != nil {
		t.Fatalf("runEnvSet: %v", err)
	}

	if got := readStoredValue(t, "global", "TOKEN"); got != "stdin-secret" {
		t.Errorf(
			"stored value = %v, want %q (trailing newline should be trimmed)",
			got,
			"stdin-secret",
		)
	}
}

func TestRunEnvSet_CustomEnv(t *testing.T) {
	setupVaultProject(t)

	if err := runEnvSet([]string{"DB_URL", "postgres://x"}, "prod", false); err != nil {
		t.Fatalf("runEnvSet: %v", err)
	}

	if got := readStoredValue(t, "prod", "DB_URL"); got != "postgres://x" {
		t.Errorf("stored value = %v, want %q", got, "postgres://x")
	}
}

func TestRunEnvSet_Upsert(t *testing.T) {
	setupVaultProject(t)

	if err := runEnvSet([]string{"K", "v1"}, "global", false); err != nil {
		t.Fatalf("set v1: %v", err)
	}
	if err := runEnvSet([]string{"K", "v2"}, "global", false); err != nil {
		t.Fatalf("set v2: %v", err)
	}

	if got := readStoredValue(t, "global", "K"); got != "v2" {
		t.Errorf("stored value = %v, want %q (upsert failed)", got, "v2")
	}
}

func TestRunEnvSet_Errors(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		envName     string
		stdin       bool
		errContains string
	}{
		{"missing key", []string{}, "global", false, "missing required argument"},
		{"invalid env name (space)", []string{"K", "v"}, "bad name", false, "invalid"},
		{"invalid env name (empty)", []string{"K", "v"}, "", false, "empty"},
		{"invalid env name (reserved prefix)", []string{"K", "v"}, "_internal", false, "reserved"},
		{
			"too many positionals",
			[]string{"K", "hello", "world"},
			"global",
			false,
			"too many arguments",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setupVaultProject(t)
			err := runEnvSet(tc.args, tc.envName, tc.stdin)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.errContains) {
				t.Errorf("error %q should contain %q", err.Error(), tc.errContains)
			}
		})
	}
}

func TestRunEnvGet_PrintsString(t *testing.T) {
	setupVaultProject(t)
	if err := runEnvSet([]string{"FOO", "bar"}, "global", false); err != nil {
		t.Fatalf("seed: %v", err)
	}

	var runErr error
	out := captureStdout(t, func() {
		runErr = runEnvGet([]string{"FOO"}, "global")
	})
	if runErr != nil {
		t.Fatalf("runEnvGet: %v", runErr)
	}
	if out != "bar\n" {
		t.Errorf("stdout = %q, want %q (raw value + newline)", out, "bar\n")
	}
}

func TestRunEnvGet_PrintsNonString(t *testing.T) {
	setupVaultProject(t)

	// Seed non-string types via the store directly (CLI `set` always stores strings).
	if err := vault.WithStoreLock(func() error {
		ageKey, _ := vault.EnsureKeypair()
		store, _ := vault.DecryptStore(ageKey.Identity)
		store.SetKey("global", "PORT", json.Number("8000"))
		store.SetKey("global", "ENABLED", true)
		return vault.WriteStore(store, ageKey.Recipient)
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	tests := []struct {
		key  string
		want string
	}{
		{"PORT", "8000\n"},    // json.Number prints unquoted
		{"ENABLED", "true\n"}, // bool prints unquoted
	}
	for _, tc := range tests {
		t.Run(tc.key, func(t *testing.T) {
			var runErr error
			out := captureStdout(t, func() {
				runErr = runEnvGet([]string{tc.key}, "global")
			})
			if runErr != nil {
				t.Fatalf("runEnvGet: %v", runErr)
			}
			if out != tc.want {
				t.Errorf("stdout = %q, want %q", out, tc.want)
			}
		})
	}
}

func TestRunEnvGet_FromCustomEnv(t *testing.T) {
	setupVaultProject(t)
	if err := runEnvSet([]string{"DB_URL", "postgres"}, "prod", false); err != nil {
		t.Fatalf("seed: %v", err)
	}

	var runErr error
	out := captureStdout(t, func() {
		runErr = runEnvGet([]string{"DB_URL"}, "prod")
	})
	if runErr != nil {
		t.Fatalf("runEnvGet: %v", runErr)
	}
	if out != "postgres\n" {
		t.Errorf("stdout = %q, want %q", out, "postgres\n")
	}
}

func TestRunEnvGet_Errors(t *testing.T) {
	tests := []struct {
		name        string
		seedKey     string
		seedEnv     string
		args        []string
		envName     string
		errContains string
	}{
		{"missing key arg", "K", "global", []string{}, "global", "missing required argument"},
		{"too many args", "K", "global", []string{"A", "B"}, "global", "too many arguments"},
		{"invalid env name", "K", "global", []string{"K"}, "bad name", "invalid"},
		{
			"missing key in env",
			"K",
			"global",
			[]string{"NOPE"},
			"global",
			`key "NOPE" not found in environment "global"`,
		},
		{
			"missing env",
			"K",
			"global",
			[]string{"K"},
			"doesnotexist",
			`environment "doesnotexist" not found`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setupVaultProject(t)
			if err := runEnvSet([]string{tc.seedKey, "v"}, tc.seedEnv, false); err != nil {
				t.Fatalf("seed: %v", err)
			}

			err := runEnvGet(tc.args, tc.envName)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.errContains) {
				t.Errorf("error %q should contain %q", err.Error(), tc.errContains)
			}
		})
	}
}

func TestRunEnvDelete_RemovesKey(t *testing.T) {
	setupVaultProject(t)
	if err := runEnvSet([]string{"FOO", "bar"}, "global", false); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := runEnvDelete([]string{"FOO"}, "global"); err != nil {
		t.Fatalf("runEnvDelete: %v", err)
	}

	// Verify gone via get-style lookup.
	if err := runEnvGet([]string{"FOO"}, "global"); err == nil {
		t.Fatal("expected key to be gone after delete")
	}
}

func TestRunEnvDelete_LeavesOtherKeysAlone(t *testing.T) {
	setupVaultProject(t)
	if err := runEnvSet([]string{"FOO", "1"}, "global", false); err != nil {
		t.Fatalf("seed FOO: %v", err)
	}
	if err := runEnvSet([]string{"BAR", "2"}, "global", false); err != nil {
		t.Fatalf("seed BAR: %v", err)
	}

	if err := runEnvDelete([]string{"FOO"}, "global"); err != nil {
		t.Fatalf("runEnvDelete: %v", err)
	}

	if got := readStoredValue(t, "global", "BAR"); got != "2" {
		t.Errorf("BAR = %v, want %q (deleting FOO must not affect BAR)", got, "2")
	}
}

func TestRunEnvDelete_Errors(t *testing.T) {
	tests := []struct {
		name        string
		seedKey     string
		seedEnv     string
		args        []string
		envName     string
		errContains string
	}{
		{"missing key arg", "K", "global", []string{}, "global", "missing required argument"},
		{"too many args", "K", "global", []string{"A", "B"}, "global", "too many arguments"},
		{"invalid env name", "K", "global", []string{"K"}, "bad name", "invalid"},
		{
			"missing key in env",
			"K",
			"global",
			[]string{"NOPE"},
			"global",
			`key "NOPE" not found in environment "global"`,
		},
		{
			"missing env",
			"K",
			"global",
			[]string{"K"},
			"doesnotexist",
			`environment "doesnotexist" not found`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setupVaultProject(t)
			if err := runEnvSet([]string{tc.seedKey, "v"}, tc.seedEnv, false); err != nil {
				t.Fatalf("seed: %v", err)
			}

			err := runEnvDelete(tc.args, tc.envName)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.errContains) {
				t.Errorf("error %q should contain %q", err.Error(), tc.errContains)
			}
		})
	}
}

func TestRunEnvSet_LargeValueWarning(t *testing.T) {
	setupVaultProject(t)

	// Replace stderr with a pipe so we can capture the warning.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	origStderr := os.Stderr
	os.Stderr = w
	t.Cleanup(func() { os.Stderr = origStderr })

	big := strings.Repeat("a", MaxValueSizeWarnBytes+1024)
	done := make(chan string, 1)
	go func() {
		var b strings.Builder
		_, _ = io.Copy(&b, r)
		done <- b.String()
	}()

	if err := runEnvSet([]string{"BIG", big}, "global", false); err != nil {
		t.Fatalf("runEnvSet: %v", err)
	}
	_ = w.Close()
	stderr := <-done

	if !strings.Contains(stderr, "warning") || !strings.Contains(stderr, "KB") {
		t.Errorf("expected size warning on stderr, got %q", stderr)
	}
	// Value still written despite warning.
	if got := readStoredValue(t, "global", "BIG"); got != big {
		t.Error("large value should still be written")
	}
}
