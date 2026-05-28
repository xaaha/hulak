// Contains tests for hulak secrets list and keys handlers.
package secrets

import (
	"strings"
	"testing"

	"github.com/xaaha/hulak/pkg/utils/testutil"
	"github.com/xaaha/hulak/pkg/vault"
)

func TestRunEnvList_PrintsSortedNames(t *testing.T) {
	setupVaultProject(t)

	// Seed: prod, global, staging — listing should sort alphabetically.
	for _, env := range []string{"prod", "global", "staging"} {
		if err := runEnvSet([]string{"K", "v"}, env, false, ""); err != nil {
			t.Fatalf("seed %s: %v", env, err)
		}
	}

	var runErr error
	out := testutil.CaptureStdout(t, func() {
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

	// Generate a keypair + empty store so ReadStore succeeds.
	if err := vault.WithStoreLock(func() error {
		ageKey, _ := vault.EnsureKeypair()
		store, _ := vault.ReadStore()
		return vault.WriteStore(store, ageKey.Recipient)
	}); err != nil {
		t.Fatalf("seed empty: %v", err)
	}

	var runErr error
	out := testutil.CaptureStdout(t, func() {
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
	if err := runEnvSet([]string{"API_KEY", "sk-123"}, "global", false, ""); err != nil {
		t.Fatalf("seed: %v", err)
	}

	var runErr error
	out := testutil.CaptureStdout(t, func() {
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
	if err := runEnvSet([]string{"API_KEY", "sk-123"}, "global", false, ""); err != nil {
		t.Fatalf("seed: %v", err)
	}

	var runErr error
	out := testutil.CaptureStdout(t, func() {
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
		if err := runEnvSet([]string{k, "v"}, "global", false, ""); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	var runErr error
	out := testutil.CaptureStdout(t, func() {
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
		if err := runEnvSet([]string{k, "v"}, "global", false, ""); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	var runErr error
	out := testutil.CaptureStdout(t, func() {
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
		if err := runEnvSet([]string{k, "v"}, "global", false, ""); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	var runErr error
	out := testutil.CaptureStdout(t, func() {
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
	if err := runEnvSet([]string{"API_KEY", "v"}, "global", false, ""); err != nil {
		t.Fatalf("seed: %v", err)
	}

	var runErr error
	out := testutil.CaptureStdout(t, func() {
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
			if err := runEnvSet([]string{"K", "v"}, "global", false, ""); err != nil {
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
	if err := runEnvSet([]string{"FIRST", "value\rwith-cr"}, "global", false, ""); err != nil {
		t.Fatalf("seed FIRST: %v", err)
	}
	if err := runEnvSet([]string{"SECOND", "plain"}, "global", false, ""); err != nil {
		t.Fatalf("seed SECOND: %v", err)
	}

	var runErr error
	out := testutil.CaptureStdout(t, func() {
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
