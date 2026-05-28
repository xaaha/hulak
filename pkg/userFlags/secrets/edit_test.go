// Contains tests for hulak secrets edit handler.
package secrets

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xaaha/hulak/pkg/utils"
)

// fakeEditor writes a small shell script that, when executed against $1,
// replaces its contents with body. Returns the script path (set as $EDITOR).
// Skipped on Windows.
func fakeEditor(t *testing.T, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fakeEditor uses /bin/sh; skip on windows")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "fake_editor.sh")
	content := "#!/bin/sh\ncat > \"$1\" <<'__EOF__'\n" + body + "\n__EOF__\n"
	if err := os.WriteFile(script, []byte(content), 0o700); err != nil { //nolint:gosec // G306 test script needs +x
		t.Fatalf("write fake editor: %v", err)
	}
	return script
}

func TestRunEnvEdit_NoChange(t *testing.T) {
	setupVaultProject(t)
	if err := runEnvSet([]string{"FOO", "bar"}, "global", false, ""); err != nil {
		t.Fatalf("seed: %v", err)
	}

	t.Setenv("EDITOR", "cat") // reads file, exits 0, doesn't modify
	if err := runEnvEdit(nil, "global"); err != nil {
		t.Fatalf("runEnvEdit: %v", err)
	}

	// FOO still present and unchanged.
	if got := readStoredValue(t, "global", "FOO"); got != "bar" {
		t.Errorf("FOO = %v, want %q (no-change path should not rewrite)", got, "bar")
	}
}

func TestRunEnvEdit_EditorFailsNoWrite(t *testing.T) {
	setupVaultProject(t)
	if err := runEnvSet([]string{"FOO", "bar"}, "global", false, ""); err != nil {
		t.Fatalf("seed: %v", err)
	}

	t.Setenv("EDITOR", "false") // exits 1 immediately
	err := runEnvEdit(nil, "global")
	if err == nil {
		t.Fatal("expected editor failure to surface as error")
	}
	if !strings.Contains(err.Error(), "editor failed") {
		t.Errorf("error %q should mention editor failure", err.Error())
	}
	// Store untouched.
	if got := readStoredValue(t, "global", "FOO"); got != "bar" {
		t.Errorf("FOO = %v, want %q (editor failure must not rewrite)", got, "bar")
	}
}

func TestRunEnvEdit_SaveValidJSON(t *testing.T) {
	setupVaultProject(t)
	if err := runEnvSet([]string{"OLD", "v"}, "staging", false, ""); err != nil {
		t.Fatalf("seed: %v", err)
	}

	t.Setenv("EDITOR", fakeEditor(t, `{"NEW_KEY":"new_value"}`))
	if err := runEnvEdit(nil, "staging"); err != nil {
		t.Fatalf("runEnvEdit: %v", err)
	}

	// OLD is gone (edit replaces the env wholesale), NEW_KEY is present.
	if got := readStoredValue(t, "staging", "NEW_KEY"); got != "new_value" {
		t.Errorf("NEW_KEY = %v, want %q", got, "new_value")
	}
}

func TestRunEnvEdit_InvalidJSONPreservesStore(t *testing.T) {
	setupVaultProject(t)
	if err := runEnvSet([]string{"FOO", "bar"}, "global", false, ""); err != nil {
		t.Fatalf("seed: %v", err)
	}

	t.Setenv("EDITOR", fakeEditor(t, "not json {"))
	err := runEnvEdit(nil, "global")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "invalid JSON") {
		t.Errorf("error %q should mention invalid JSON", err.Error())
	}
	// Store untouched.
	if got := readStoredValue(t, "global", "FOO"); got != "bar" {
		t.Errorf("FOO = %v, want %q (invalid edit must not rewrite)", got, "bar")
	}
}

func TestRunEnvEdit_CreatesEnvOnFirstEdit(t *testing.T) {
	setupVaultProject(t)

	// staging doesn't exist yet — edit should treat it as {} and let the
	// user populate it.
	t.Setenv("EDITOR", fakeEditor(t, `{"FRESH":"value"}`))
	if err := runEnvEdit(nil, "staging"); err != nil {
		t.Fatalf("runEnvEdit: %v", err)
	}

	if got := readStoredValue(t, "staging", "FRESH"); got != "value" {
		t.Errorf("FRESH = %v, want %q (edit should create absent env)", got, "value")
	}
}

func TestRunEnvEdit_TempFileCleanedUp(t *testing.T) {
	projectDir := setupVaultProject(t)

	t.Setenv("EDITOR", "cat")
	if err := runEnvEdit(nil, "global"); err != nil {
		t.Fatalf("runEnvEdit: %v", err)
	}

	// .hulak/ should not contain a leftover edit-<env>.json after a clean run.
	leftover := filepath.Join(projectDir, utils.HiddenProjectName, "edit-global.json")
	if _, err := os.Stat(leftover); !os.IsNotExist(err) {
		t.Errorf("temp file %q was not cleaned up", leftover)
	}
}

func TestRunEnvEdit_Errors(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		envName     string
		errContains string
	}{
		{"too many args", []string{"unexpected"}, "global", "too many arguments"},
		{"invalid env name", nil, "bad name", "invalid"},
		{"reserved env name", nil, "_internal", "reserved"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setupVaultProject(t)
			err := runEnvEdit(tc.args, tc.envName)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.errContains) {
				t.Errorf("error %q should contain %q", err.Error(), tc.errContains)
			}
		})
	}
}

// TestRunEnvEdit_NoDefaultGlobal documents that edit refuses to silently
// pick "global" — when --env is omitted (envName=="") the picker is invoked.
// We stub envPicker to avoid opening a real TUI in test runs, which would
// hang waiting for keypress.
func TestRunEnvEdit_NoDefaultGlobal(t *testing.T) {
	setupVaultProject(t)
	if err := runEnvSet([]string{"FOO", "bar"}, "global", false, ""); err != nil {
		t.Fatalf("seed: %v", err)
	}

	prevPicker := envPicker
	t.Cleanup(func() { envPicker = prevPicker })
	pickerCalled := false
	envPicker = func() (string, bool, error) {
		pickerCalled = true
		return "", false, errors.New("picker disabled in tests")
	}

	t.Setenv("EDITOR", fakeEditor(t, `{"GOT_EDITED":"yes"}`))
	err := runEnvEdit(nil, "") // empty envName → picker
	if err == nil {
		t.Fatal("expected stubbed picker error to propagate, got nil")
	}
	if !pickerCalled {
		t.Error("envPicker was not invoked; edit may have silently defaulted to global")
	}
	// Critical: global must NOT have been edited. If we'd defaulted to
	// "global" instead of prompting, FOO would be gone.
	if got := readStoredValue(t, "global", "FOO"); got != "bar" {
		t.Errorf(
			"global was edited despite empty --env (got FOO=%v); edit must prompt, not default",
			got,
		)
	}
}

// TestRunEnvEdit_WholesaleReplacesEnv pins down the destructive semantic
// documented in the help text: keys present before the edit but absent from
// the saved JSON are removed. A future refactor that "merges" edits with
// existing keys instead of replacing them would silently retain deleted
// secrets — exactly the behavior we promise NOT to have.
func TestRunEnvEdit_WholesaleReplacesEnv(t *testing.T) {
	setupVaultProject(t)
	for _, kv := range [][2]string{
		{"KEEP", "1"}, {"REMOVE_ME", "2"}, {"ALSO_REMOVE", "3"},
	} {
		if err := runEnvSet([]string{kv[0], kv[1]}, "global", false, ""); err != nil {
			t.Fatalf("seed %s: %v", kv[0], err)
		}
	}

	// Save only KEEP and a NEW key.
	t.Setenv("EDITOR", fakeEditor(t, `{"KEEP":"1","NEW":"4"}`))
	if err := runEnvEdit(nil, "global"); err != nil {
		t.Fatalf("runEnvEdit: %v", err)
	}

	// KEEP and NEW present.
	if got := readStoredValue(t, "global", "KEEP"); got != "1" {
		t.Errorf("KEEP = %v, want %q", got, "1")
	}
	if got := readStoredValue(t, "global", "NEW"); got != "4" {
		t.Errorf("NEW = %v, want %q", got, "4")
	}
	// REMOVE_ME / ALSO_REMOVE must be gone — get must error.
	if err := runEnvGet([]string{"REMOVE_ME"}, "global"); err == nil {
		t.Error("REMOVE_ME should be deleted by wholesale replacement")
	}
	if err := runEnvGet([]string{"ALSO_REMOVE"}, "global"); err == nil {
		t.Error("ALSO_REMOVE should be deleted by wholesale replacement")
	}
}

// TestRunEnvEdit_LeavesOtherEnvsUntouched is the second half of the wholesale
// guarantee: replacing one env must not affect any other env in the store.
func TestRunEnvEdit_LeavesOtherEnvsUntouched(t *testing.T) {
	setupVaultProject(t)
	if err := runEnvSet([]string{"P", "prod-value"}, "prod", false, ""); err != nil {
		t.Fatalf("seed prod: %v", err)
	}
	if err := runEnvSet([]string{"S", "staging-value"}, "staging", false, ""); err != nil {
		t.Fatalf("seed staging: %v", err)
	}

	// Wipe staging by saving an empty object.
	t.Setenv("EDITOR", fakeEditor(t, `{}`))
	if err := runEnvEdit(nil, "staging"); err != nil {
		t.Fatalf("runEnvEdit: %v", err)
	}

	// prod is untouched.
	if got := readStoredValue(t, "prod", "P"); got != "prod-value" {
		t.Errorf("prod.P = %v, want %q (other envs must not be affected)", got, "prod-value")
	}
}
