package userflags

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

// setupVaultProject prepares an isolated hulak project + config dir for tests.
// It chdirs into a fresh temp dir and points XDG_CONFIG_HOME at another temp dir
// so vault.EnsureKeypair stores the identity outside the user's real config.
func setupVaultProject(t *testing.T) string {
	t.Helper()

	configDir := t.TempDir()
	configDir, err := filepath.EvalSymlinks(configDir)
	if err != nil {
		t.Fatalf("resolve symlinks: %v", err)
	}
	t.Setenv("XDG_CONFIG_HOME", configDir)

	projectDir := t.TempDir()
	projectDir, err = filepath.EvalSymlinks(projectDir)
	if err != nil {
		t.Fatalf("resolve symlinks: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectDir, utils.HiddenProjectName), utils.DirPer); err != nil {
		t.Fatalf("mkdir .hulak: %v", err)
	}

	t.Cleanup(chdirTemp(t, projectDir))
	return projectDir
}

// readStoredValue decrypts the store and returns the value at envName/key.
func readStoredValue(t *testing.T, envName, key string) any {
	t.Helper()
	id, err := vault.LoadIdentity()
	if err != nil {
		t.Fatalf("LoadIdentity: %v", err)
	}
	store, err := vault.ReadStore(id)
	if err != nil {
		t.Fatalf("ReadStore: %v", err)
	}
	env := store.GetEnv(envName)
	if env == nil {
		t.Fatalf("env %q not found in store", envName)
	}
	return env[key]
}

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
		t.Errorf("stored value = %v, want %q (trailing newline should be trimmed)", got, "stdin-secret")
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
		{"too many positionals", []string{"K", "hello", "world"}, "global", false, "too many arguments"},
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
		store, _ := vault.ReadStore(ageKey.Identity)
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
		{"missing key in env", "K", "global", []string{"NOPE"}, "global", `key "NOPE" not found in environment "global"`},
		{"missing env", "K", "global", []string{"K"}, "doesnotexist", `environment "doesnotexist" not found`},
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
		{"missing key in env", "K", "global", []string{"NOPE"}, "global", `key "NOPE" not found in environment "global"`},
		{"missing env", "K", "global", []string{"K"}, "doesnotexist", `environment "doesnotexist" not found`},
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

func TestRunEnvList_PrintsSortedNames(t *testing.T) {
	setupVaultProject(t)

	// Seed: prod, global, staging — listing should sort alphabetically.
	for _, env := range []string{"prod", "global", "staging"} {
		if err := runEnvSet([]string{"K", "v"}, env, false); err != nil {
			t.Fatalf("seed %s: %v", env, err)
		}
	}

	var runErr error
	out := captureStdout(t, func() {
		runErr = runEnvList(nil)
	})
	if runErr != nil {
		t.Fatalf("runEnvList: %v", runErr)
	}

	want := "global\nprod\nstaging\n"
	if out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}
}

func TestRunEnvList_EmptyStore(t *testing.T) {
	setupVaultProject(t)

	// Generate a keypair + empty store so LoadIdentity / ReadStore succeed.
	if err := vault.WithStoreLock(func() error {
		ageKey, _ := vault.EnsureKeypair()
		store, _ := vault.ReadStore(ageKey.Identity)
		return vault.WriteStore(store, ageKey.Recipient)
	}); err != nil {
		t.Fatalf("seed empty: %v", err)
	}

	var runErr error
	out := captureStdout(t, func() {
		runErr = runEnvList(nil)
	})
	if runErr != nil {
		t.Fatalf("runEnvList: %v", runErr)
	}
	if out != "" {
		t.Errorf("stdout = %q, want empty", out)
	}
}

func TestRunEnvList_TooManyArgs(t *testing.T) {
	setupVaultProject(t)
	err := runEnvList([]string{"unexpected"})
	if err == nil {
		t.Fatal("expected error for extra arg")
	}
	if !strings.Contains(err.Error(), "too many arguments") {
		t.Errorf("error %q should mention too many arguments", err.Error())
	}
}

func TestRunEnvKeys_MasksByDefault(t *testing.T) {
	setupVaultProject(t)
	if err := runEnvSet([]string{"API_KEY", "sk-123"}, "global", false); err != nil {
		t.Fatalf("seed: %v", err)
	}

	var runErr error
	out := captureStdout(t, func() {
		runErr = runEnvKeys(nil, "global", "", false)
	})
	if runErr != nil {
		t.Fatalf("runEnvKeys: %v", runErr)
	}
	if !strings.Contains(out, "API_KEY") {
		t.Errorf("output should contain key name; got %q", out)
	}
	if !strings.Contains(out, maskedValue) {
		t.Errorf("output should contain mask %q; got %q", maskedValue, out)
	}
	if strings.Contains(out, "sk-123") {
		t.Errorf("output should NOT contain real value; got %q", out)
	}
}

func TestRunEnvKeys_ShowReveals(t *testing.T) {
	setupVaultProject(t)
	if err := runEnvSet([]string{"API_KEY", "sk-123"}, "global", false); err != nil {
		t.Fatalf("seed: %v", err)
	}

	var runErr error
	out := captureStdout(t, func() {
		runErr = runEnvKeys(nil, "global", "", true)
	})
	if runErr != nil {
		t.Fatalf("runEnvKeys: %v", runErr)
	}
	if !strings.Contains(out, "sk-123") {
		t.Errorf("--show output should reveal value; got %q", out)
	}
	if strings.Contains(out, maskedValue) {
		t.Errorf("--show output should NOT contain mask; got %q", out)
	}
}

func TestRunEnvKeys_SortsKeys(t *testing.T) {
	setupVaultProject(t)
	for _, k := range []string{"ZETA", "alpha", "MIDDLE"} {
		if err := runEnvSet([]string{k, "v"}, "global", false); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	var runErr error
	out := captureStdout(t, func() {
		runErr = runEnvKeys(nil, "global", "", false)
	})
	if runErr != nil {
		t.Fatalf("runEnvKeys: %v", runErr)
	}

	// sort.Strings is byte-order: uppercase before lowercase, ZETA's Z(0x5A) < a(0x61).
	want := []string{"MIDDLE", "ZETA", "alpha"}
	prevIdx := -1
	prevKey := ""
	for _, k := range want {
		idx := strings.Index(out, k)
		if idx == -1 {
			t.Fatalf("output missing %q", k)
		}
		if prevIdx > idx {
			t.Errorf("expected %q to appear after %q in sorted output; got %q", k, prevKey, out)
		}
		prevIdx = idx
		prevKey = k
	}
}

func TestRunEnvKeys_SearchSubstring(t *testing.T) {
	setupVaultProject(t)
	for _, k := range []string{"API_KEY", "DB_URL", "api_token", "TIMEOUT"} {
		if err := runEnvSet([]string{k, "v"}, "global", false); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	var runErr error
	out := captureStdout(t, func() {
		// "api" — substring, case-insensitive: should match API_KEY and api_token.
		runErr = runEnvKeys(nil, "global", "api", false)
	})
	if runErr != nil {
		t.Fatalf("runEnvKeys: %v", runErr)
	}
	if !strings.Contains(out, "API_KEY") || !strings.Contains(out, "api_token") {
		t.Errorf("substring match should include both API_KEY and api_token; got %q", out)
	}
	if strings.Contains(out, "DB_URL") || strings.Contains(out, "TIMEOUT") {
		t.Errorf("substring match should exclude DB_URL/TIMEOUT; got %q", out)
	}
}

func TestRunEnvKeys_SearchGlob(t *testing.T) {
	setupVaultProject(t)
	for _, k := range []string{"API_KEY", "API_TOKEN", "DB_URL"} {
		if err := runEnvSet([]string{k, "v"}, "global", false); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	var runErr error
	out := captureStdout(t, func() {
		// Glob, case-sensitive: matches "API_*".
		runErr = runEnvKeys(nil, "global", "API_*", false)
	})
	if runErr != nil {
		t.Fatalf("runEnvKeys: %v", runErr)
	}
	if !strings.Contains(out, "API_KEY") || !strings.Contains(out, "API_TOKEN") {
		t.Errorf("glob match should include both API_KEY and API_TOKEN; got %q", out)
	}
	if strings.Contains(out, "DB_URL") {
		t.Errorf("glob match should exclude DB_URL; got %q", out)
	}
}

func TestRunEnvKeys_SearchNoMatches(t *testing.T) {
	setupVaultProject(t)
	if err := runEnvSet([]string{"API_KEY", "v"}, "global", false); err != nil {
		t.Fatalf("seed: %v", err)
	}

	var runErr error
	out := captureStdout(t, func() {
		runErr = runEnvKeys(nil, "global", "ZZZ", false)
	})
	if runErr != nil {
		t.Fatalf("runEnvKeys: %v", runErr)
	}
	if out != "" {
		t.Errorf("expected empty output for no matches; got %q", out)
	}
}

func TestRunEnvKeys_Errors(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		envName     string
		search      string
		errContains string
	}{
		{"too many args", []string{"unexpected"}, "global", "", "too many arguments"},
		{"invalid env", []string{}, "bad name", "", "invalid"},
		{"missing env", []string{}, "doesnotexist", "", `environment "doesnotexist" not found`},
		{"bad glob", []string{}, "global", "[unclosed", "invalid glob pattern"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setupVaultProject(t)
			if err := runEnvSet([]string{"K", "v"}, "global", false); err != nil {
				t.Fatalf("seed: %v", err)
			}

			err := runEnvKeys(tc.args, tc.envName, tc.search, false)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.errContains) {
				t.Errorf("error %q should contain %q", err.Error(), tc.errContains)
			}
		})
	}
}

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
	if err := runEnvSet([]string{"FOO", "bar"}, "global", false); err != nil {
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
	if err := runEnvSet([]string{"FOO", "bar"}, "global", false); err != nil {
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
	if err := runEnvSet([]string{"OLD", "v"}, "staging", false); err != nil {
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
	if err := runEnvSet([]string{"FOO", "bar"}, "global", false); err != nil {
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
	if err := runEnvSet([]string{"FOO", "bar"}, "global", false); err != nil {
		t.Fatalf("seed: %v", err)
	}

	prevPicker := envPicker
	t.Cleanup(func() { envPicker = prevPicker })
	pickerCalled := false
	envPicker = func() (string, error) {
		pickerCalled = true
		return "", errors.New("picker disabled in tests")
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
		t.Errorf("global was edited despite empty --env (got FOO=%v); edit must prompt, not default", got)
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

// TestEscapeControlChars verifies that control characters which would break
// table alignment (\n shifts rows; \r blanks the line; \t expands to tab stops)
// are rendered as visible escapes, while plain text passes through unchanged.
func TestEscapeControlChars(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain text unchanged", "hello", "hello"},
		{"empty unchanged", "", ""},
		{"newline escaped", "a\nb", `a\nb`},
		{"carriage return escaped", "a\rb", `a\rb`},
		{"crlf escaped", "a\r\nb", `a\r\nb`},
		{"tab escaped", "a\tb", `a\tb`},
		{"bell escaped as hex", "a\x07b", `a\x07b`},
		{"esc escaped as hex", "a\x1bb", `a\x1bb`},
		{"del escaped as hex", "a\x7fb", `a\x7fb`},
		{"mixed", "k\rev\ny", `k\rev\ny`},
		{"high-bit unchanged", "café", "café"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := escapeControlChars(tc.in); got != tc.want {
				t.Errorf("escapeControlChars(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestFormatTableValue_EscapesControlChars asserts that the table formatter
// uses escapeControlChars for string values. A regression here would let a
// stored value containing \r or \n shift downstream rows or blank a line.
func TestFormatTableValue_EscapesControlChars(t *testing.T) {
	got := formatTableValue("k\rinjected\nrow")
	want := `k\rinjected\nrow`
	if got != want {
		t.Errorf("formatTableValue = %q, want %q", got, want)
	}
}

// TestRunEnvKeys_AlignsValuesContainingControlChars guards the end-to-end
// behavior: a value with a CR must not blank or misalign the table line.
// We render two rows; both must appear on their own line, in order.
func TestRunEnvKeys_AlignsValuesContainingControlChars(t *testing.T) {
	setupVaultProject(t)
	if err := runEnvSet([]string{"FIRST", "value\rwith-cr"}, "global", false); err != nil {
		t.Fatalf("seed FIRST: %v", err)
	}
	if err := runEnvSet([]string{"SECOND", "plain"}, "global", false); err != nil {
		t.Fatalf("seed SECOND: %v", err)
	}

	var runErr error
	out := captureStdout(t, func() {
		runErr = runEnvKeys(nil, "global", "", true)
	})
	if runErr != nil {
		t.Fatalf("runEnvKeys: %v", runErr)
	}
	// Raw CR must not survive into the rendered output — it would erase
	// the start of the line in many terminals.
	if strings.ContainsRune(out, '\r') {
		t.Errorf("output should not contain raw CR; got %q", out)
	}
	// Both rows present, in order.
	first := strings.Index(out, "FIRST")
	second := strings.Index(out, "SECOND")
	if first == -1 || second == -1 || first >= second {
		t.Errorf("rows missing or out of order: %q", out)
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
		if err := runEnvSet([]string{kv[0], kv[1]}, "global", false); err != nil {
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
	if err := runEnvSet([]string{"P", "prod-value"}, "prod", false); err != nil {
		t.Fatalf("seed prod: %v", err)
	}
	if err := runEnvSet([]string{"S", "staging-value"}, "staging", false); err != nil {
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
