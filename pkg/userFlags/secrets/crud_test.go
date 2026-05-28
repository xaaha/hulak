// Contains tests for hulak secrets set, get, and delete handlers.
package secrets

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

	if err := runEnvSet([]string{"API_KEY", "sk-123"}, "global", false, ""); err != nil {
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

	if err := runEnvSet([]string{"TOKEN"}, "global", true, ""); err != nil {
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

	if err := runEnvSet([]string{"DB_URL", "postgres://x"}, "prod", false, ""); err != nil {
		t.Fatalf("runEnvSet: %v", err)
	}

	if got := readStoredValue(t, "prod", "DB_URL"); got != "postgres://x" {
		t.Errorf("stored value = %v, want %q", got, "postgres://x")
	}
}

func TestRunEnvSet_Upsert(t *testing.T) {
	setupVaultProject(t)

	if err := runEnvSet([]string{"K", "v1"}, "global", false, ""); err != nil {
		t.Fatalf("set v1: %v", err)
	}
	if err := runEnvSet([]string{"K", "v2"}, "global", false, ""); err != nil {
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
			err := runEnvSet(tc.args, tc.envName, tc.stdin, "")
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
	if err := runEnvSet([]string{"FOO", "bar"}, "global", false, ""); err != nil {
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

	// Seed non-string types via the store directly to keep this test focused on
	// runEnvGet's printing behavior independent of runEnvSet/--type.
	if err := vault.WithStoreLock(func() error {
		ageKey, _ := vault.EnsureKeypair()
		store, _ := vault.ReadStore()
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
	if err := runEnvSet([]string{"DB_URL", "postgres"}, "prod", false, ""); err != nil {
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
			if err := runEnvSet([]string{tc.seedKey, "v"}, tc.seedEnv, false, ""); err != nil {
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
	if err := runEnvSet([]string{"FOO", "bar"}, "global", false, ""); err != nil {
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
	if err := runEnvSet([]string{"FOO", "1"}, "global", false, ""); err != nil {
		t.Fatalf("seed FOO: %v", err)
	}
	if err := runEnvSet([]string{"BAR", "2"}, "global", false, ""); err != nil {
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
			if err := runEnvSet([]string{tc.seedKey, "v"}, tc.seedEnv, false, ""); err != nil {
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

	if err := runEnvSet([]string{"BIG", big}, "global", false, ""); err != nil {
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

func TestParseTypedValue_String(t *testing.T) {
	got, err := parseTypedValue("hello world", "string")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello world" {
		t.Errorf("got %v (%T), want %q (string)", got, got, "hello world")
	}
}

func TestParseTypedValue_StringDefault(t *testing.T) {
	got, err := parseTypedValue("anything", "")
	if err != nil {
		t.Fatalf("empty type should default to string, got error: %v", err)
	}
	if got != "anything" {
		t.Errorf("got %v, want %q", got, "anything")
	}
}

func TestParseTypedValue_Int(t *testing.T) {
	got, err := parseTypedValue("3939", "int")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	num, ok := got.(json.Number)
	if !ok {
		t.Fatalf("got %v (%T), want json.Number", got, got)
	}
	if num.String() != "3939" {
		t.Errorf("got %q, want %q", num.String(), "3939")
	}
}

func TestParseTypedValue_IntNegative(t *testing.T) {
	got, err := parseTypedValue("-42", "int")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	num, ok := got.(json.Number)
	if !ok || num.String() != "-42" {
		t.Errorf("got %v, want json.Number(\"-42\")", got)
	}
}

func TestParseTypedValue_IntInvalid(t *testing.T) {
	cases := []string{"3.5", "abc", "", "1e3"}
	for _, raw := range cases {
		t.Run(raw, func(t *testing.T) {
			if _, err := parseTypedValue(raw, "int"); err == nil {
				t.Errorf("expected error for int %q, got nil", raw)
			}
		})
	}
}

func TestParseTypedValue_Float(t *testing.T) {
	got, err := parseTypedValue("3.14", "float")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	num, ok := got.(json.Number)
	if !ok || num.String() != "3.14" {
		t.Errorf("got %v, want json.Number(\"3.14\")", got)
	}
}

func TestParseTypedValue_FloatInvalid(t *testing.T) {
	if _, err := parseTypedValue("not-a-number", "float"); err == nil {
		t.Error("expected error for invalid float, got nil")
	}
}

func TestParseTypedValue_Bool(t *testing.T) {
	cases := map[string]bool{
		"true":  true,
		"false": false,
		"1":     true,
		"0":     false,
		"T":     true,
		"F":     false,
	}
	for raw, want := range cases {
		t.Run(raw, func(t *testing.T) {
			got, err := parseTypedValue(raw, "bool")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			b, ok := got.(bool)
			if !ok {
				t.Fatalf("got %v (%T), want bool", got, got)
			}
			if b != want {
				t.Errorf("got %v, want %v", b, want)
			}
		})
	}
}

func TestParseTypedValue_BoolInvalid(t *testing.T) {
	cases := []string{"yes", "no", "TRUE!", "", "2"}
	for _, raw := range cases {
		t.Run(raw, func(t *testing.T) {
			if _, err := parseTypedValue(raw, "bool"); err == nil {
				t.Errorf("expected error for bool %q, got nil", raw)
			}
		})
	}
}

func TestParseTypedValue_JSONObject(t *testing.T) {
	got, err := parseTypedValue(`{"a":1,"b":"x"}`, "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("got %T, want map[string]any", got)
	}
	num, ok := m["a"].(json.Number)
	if !ok || num.String() != "1" {
		t.Errorf("m[a] = %v (%T), want json.Number(\"1\")", m["a"], m["a"])
	}
	if m["b"] != "x" {
		t.Errorf("m[b] = %v, want %q", m["b"], "x")
	}
}

func TestParseTypedValue_JSONArray(t *testing.T) {
	got, err := parseTypedValue(`[1,2,3]`, "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	arr, ok := got.([]any)
	if !ok {
		t.Fatalf("got %T, want []any", got)
	}
	if len(arr) != 3 {
		t.Fatalf("len=%d, want 3", len(arr))
	}
	num, ok := arr[0].(json.Number)
	if !ok || num.String() != "1" {
		t.Errorf("arr[0] = %v, want json.Number(\"1\")", arr[0])
	}
}

func TestParseTypedValue_JSONInvalid(t *testing.T) {
	cases := []string{`{`, `not json`, ``, `{"a":}`}
	for _, raw := range cases {
		t.Run(raw, func(t *testing.T) {
			if _, err := parseTypedValue(raw, "json"); err == nil {
				t.Errorf("expected error for json %q, got nil", raw)
			}
		})
	}
}

func TestParseTypedValue_JSONRejectsTrailingData(t *testing.T) {
	cases := []string{
		`{"a":1}garbage`,
		`[1,2] extra`,
		`42 99`,
		`"hello" world`,
	}
	for _, raw := range cases {
		t.Run(raw, func(t *testing.T) {
			_, err := parseTypedValue(raw, "json")
			if err == nil {
				t.Errorf("expected error for trailing data in %q, got nil", raw)
			}
		})
	}
}

func TestParseTypedValue_JSONAcceptsTrailingWhitespace(t *testing.T) {
	cases := []string{
		`{"a":1}` + "\n",
		`[1,2]  `,
		"  42  \n\t",
	}
	for _, raw := range cases {
		t.Run(raw, func(t *testing.T) {
			if _, err := parseTypedValue(raw, "json"); err != nil {
				t.Errorf("trailing whitespace should be allowed, got error for %q: %v", raw, err)
			}
		})
	}
}

func TestParseTypedValue_UnknownType(t *testing.T) {
	_, err := parseTypedValue("v", "integer")
	if err == nil {
		t.Fatal("expected error for unknown type, got nil")
	}
	msg := err.Error()
	for _, want := range []string{"string", "int", "float", "bool", "json"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error %q should list valid type %q", msg, want)
		}
	}
}

func TestRunEnvSet_TypedInt(t *testing.T) {
	setupVaultProject(t)
	if err := runEnvSet([]string{"userAge", "3939"}, "global", false, "int"); err != nil {
		t.Fatalf("runEnvSet: %v", err)
	}
	got := readStoredValue(t, "global", "userAge")
	num, ok := got.(json.Number)
	if !ok {
		t.Fatalf("stored value = %v (%T), want json.Number", got, got)
	}
	if num.String() != "3939" {
		t.Errorf("stored = %q, want %q", num.String(), "3939")
	}
}

func TestRunEnvSet_TypedFloat(t *testing.T) {
	setupVaultProject(t)
	if err := runEnvSet([]string{"ratio", "1.5"}, "global", false, "float"); err != nil {
		t.Fatalf("runEnvSet: %v", err)
	}
	got := readStoredValue(t, "global", "ratio")
	num, ok := got.(json.Number)
	if !ok || num.String() != "1.5" {
		t.Errorf("stored = %v (%T), want json.Number(\"1.5\")", got, got)
	}
}

func TestRunEnvSet_TypedBool(t *testing.T) {
	setupVaultProject(t)
	if err := runEnvSet([]string{"ENABLED", "true"}, "global", false, "bool"); err != nil {
		t.Fatalf("runEnvSet: %v", err)
	}
	got := readStoredValue(t, "global", "ENABLED")
	b, ok := got.(bool)
	if !ok || !b {
		t.Errorf("stored = %v (%T), want true (bool)", got, got)
	}
}

func TestRunEnvSet_TypedJSON(t *testing.T) {
	setupVaultProject(t)
	if err := runEnvSet([]string{"config", `{"port":8000}`}, "global", false, "json"); err != nil {
		t.Fatalf("runEnvSet: %v", err)
	}
	got := readStoredValue(t, "global", "config")
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("stored = %v (%T), want map[string]any", got, got)
	}
	num, ok := m["port"].(json.Number)
	if !ok || num.String() != "8000" {
		t.Errorf("config.port = %v (%T), want json.Number(\"8000\")", m["port"], m["port"])
	}
}

func TestRunEnvSet_TypedStdin(t *testing.T) {
	setupVaultProject(t)

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	if _, err := w.WriteString("42\n"); err != nil {
		t.Fatalf("write pipe: %v", err)
	}
	_ = w.Close()

	orig := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = orig })

	if err := runEnvSet([]string{"port"}, "global", true, "int"); err != nil {
		t.Fatalf("runEnvSet: %v", err)
	}

	got := readStoredValue(t, "global", "port")
	num, ok := got.(json.Number)
	if !ok || num.String() != "42" {
		t.Errorf("stored = %v, want json.Number(\"42\")", got)
	}
}

func TestRunEnvSet_TypedInvalidValueAbortsBeforeStore(t *testing.T) {
	setupVaultProject(t)
	// Seed an existing value so we can verify it stays untouched.
	if err := runEnvSet([]string{"port", "8000"}, "global", false, "int"); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Reset with invalid int should error and not touch the stored value.
	err := runEnvSet([]string{"port", "not-a-number"}, "global", false, "int")
	if err == nil {
		t.Fatal("expected error for invalid int, got nil")
	}
	if !strings.Contains(err.Error(), "invalid int") {
		t.Errorf("error %q should mention 'invalid int'", err.Error())
	}

	got := readStoredValue(t, "global", "port")
	num, ok := got.(json.Number)
	if !ok || num.String() != "8000" {
		t.Errorf("seed value mutated by failed set: got %v, want json.Number(\"8000\")", got)
	}
}

func TestRunEnvSet_TypedUnknownTypeErrors(t *testing.T) {
	setupVaultProject(t)
	err := runEnvSet([]string{"K", "v"}, "global", false, "integer")
	if err == nil {
		t.Fatal("expected error for unknown type, got nil")
	}
	if !strings.Contains(err.Error(), "unknown type") {
		t.Errorf("error %q should mention 'unknown type'", err.Error())
	}
}

func TestRunEnvSet_DefaultTypeIsString(t *testing.T) {
	setupVaultProject(t)
	// Number-shaped string with no --type stays a string (backward compat).
	if err := runEnvSet([]string{"token", "3939"}, "global", false, ""); err != nil {
		t.Fatalf("runEnvSet: %v", err)
	}
	got := readStoredValue(t, "global", "token")
	if s, ok := got.(string); !ok || s != "3939" {
		t.Errorf("stored = %v (%T), want \"3939\" (string)", got, got)
	}
}

// TestKeysSetCmd_FlagWiring exercises the full keysSetCmd() path: build the
// command, parse flag args through the FlagSet, then invoke Run. This is the
// only test that catches a regression in the `-t`/`--type` alias wiring or
// the StringVar pair pointing to the same variable (a bug here would have
// integration tests still pass because they bypass flag parsing).
func TestKeysSetCmd_FlagWiring(t *testing.T) {
	testCases := []struct {
		name string
		args []string // full args after `secrets keys set`
		key  string
	}{
		{"long flag --type int", []string{"--env", "global", "--type", "int", "longInt", "100"}, "longInt"},
		{"short flag -t int", []string{"--env", "global", "-t", "int", "shortInt", "200"}, "shortInt"},
		{"long flag --type bool", []string{"--env", "global", "--type", "bool", "longBool", "true"}, "longBool"},
		{"short flag -t bool", []string{"--env", "global", "-t", "bool", "shortBool", "false"}, "shortBool"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			setupVaultProject(t)
			cmd := keysSetCmd()
			if err := cmd.Flags.Parse(tc.args); err != nil {
				t.Fatalf("Flags.Parse(%v): %v", tc.args, err)
			}
			if err := cmd.Run(cmd.Flags.Args()); err != nil {
				t.Fatalf("cmd.Run: %v", err)
			}

			got := readStoredValue(t, "global", tc.key)
			// Stored value should NOT be the raw string. Any non-string type
			// proves --type was honored end-to-end via the flag binding.
			if s, ok := got.(string); ok {
				t.Errorf("stored as string %q — flag binding did not honor --type", s)
			}
		})
	}
}
