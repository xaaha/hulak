package runner

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGenerateFilePathList_FpOnly(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.hk.yaml")
	if err := os.WriteFile(tmpFile, []byte("method: GET\nurl: http://example.com"), 0o600); err != nil {
		t.Fatal(err)
	}

	list, err := generateFilePathList("", tmpFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 || list[0] != tmpFile {
		t.Errorf("expected [%s], got %v", tmpFile, list)
	}
}

func TestGenerateFilePathList_BothEmpty(t *testing.T) {
	_, err := generateFilePathList("", "")
	if err == nil {
		t.Fatal("expected error when both fileName and fp are empty")
	}
}

func TestDiscoverFilePaths_FpReturnsFileList(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "req.hk.yaml")
	if err := os.WriteFile(tmpFile, []byte("method: GET\nurl: http://example.com"), 0o600); err != nil {
		t.Fatal(err)
	}

	fileList, concurrent, sequential, err := discoverFilePaths(
		"",      // fileName
		tmpFile, // fp
		"",      // dir
		"",      // dirseq
		false,   // hasDirFlags
	)

	if err != nil {
		t.Fatalf("discoverFilePaths: %v", err)
	}
	if len(fileList) != 1 || fileList[0] != tmpFile {
		t.Errorf("fileList = %v, want [%s]", fileList, tmpFile)
	}
	if len(concurrent) != 0 {
		t.Errorf("concurrent should be empty, got %v", concurrent)
	}
	if len(sequential) != 0 {
		t.Errorf("sequential should be empty, got %v", sequential)
	}
}

func TestDiscoverFilePaths_DirReturnsConcurrent(t *testing.T) {
	tmpDir := t.TempDir()
	for _, name := range []string{"a.yaml", "b.yaml"} {
		content := "method: GET\nurl: http://example.com"
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	fileList, concurrent, sequential, err := discoverFilePaths(
		"",     // fileName
		"",     // fp
		tmpDir, // dir
		"",     // dirseq
		true,   // hasDirFlags
	)

	if err != nil {
		t.Fatalf("discoverFilePaths: %v", err)
	}
	if len(fileList) != 0 {
		t.Errorf("fileList should be empty, got %v", fileList)
	}
	if len(concurrent) != 2 {
		t.Errorf("expected 2 concurrent files, got %d: %v", len(concurrent), concurrent)
	}
	if len(sequential) != 0 {
		t.Errorf("sequential should be empty, got %v", sequential)
	}
}

func TestDiscoverFilePaths_DirseqReturnsSequential(t *testing.T) {
	tmpDir := t.TempDir()
	for _, name := range []string{"a.yaml", "b.yaml"} {
		content := "method: GET\nurl: http://example.com"
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	fileList, concurrent, sequential, err := discoverFilePaths(
		"",     // fileName
		"",     // fp
		"",     // dir
		tmpDir, // dirseq
		true,   // hasDirFlags
	)
	if err != nil {
		t.Fatalf("discoverFilePaths: %v", err)
	}

	if len(fileList) != 0 {
		t.Errorf("fileList should be empty, got %v", fileList)
	}
	if len(concurrent) != 0 {
		t.Errorf("concurrent should be empty, got %v", concurrent)
	}
	if len(sequential) != 2 {
		t.Errorf("expected 2 sequential files, got %d: %v", len(sequential), sequential)
	}
}

func TestContainsTemplateVars_NoTemplates(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "plain.yaml")
	if err := os.WriteFile(tmpFile, []byte("method: GET\nurl: http://example.com"), 0o600); err != nil {
		t.Fatal(err)
	}

	if containsTemplateVars([]string{tmpFile}) {
		t.Error("expected false for file without template vars")
	}
}

func TestContainsTemplateVars_WithTemplates(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "templated.yaml")
	if err := os.WriteFile(tmpFile, []byte("method: GET\nurl: '{{.apiUrl}}'"), 0o600); err != nil {
		t.Fatal(err)
	}

	if !containsTemplateVars([]string{tmpFile}) {
		t.Error("expected true for file with template vars")
	}
}

func TestContainsTemplateVars_EmptyList(t *testing.T) {
	if containsTemplateVars(nil) {
		t.Error("expected false for empty list")
	}
}

func TestDiscoverFilePaths_EmptyInputs(t *testing.T) {
	fileList, concurrent, sequential, err := discoverFilePaths(
		"", "", "", "", false,
	)
	if err != nil {
		t.Fatalf("discoverFilePaths: %v", err)
	}

	if len(fileList) != 0 {
		t.Errorf("fileList should be empty, got %v", fileList)
	}
	if len(concurrent) != 0 {
		t.Errorf("concurrent should be empty, got %v", concurrent)
	}
	if len(sequential) != 0 {
		t.Errorf("sequential should be empty, got %v", sequential)
	}
}

// --- envSelector tests ---

func TestExecute_EnvNotSet_CallsSelector(t *testing.T) {
	orig := envSelector
	defer func() { envSelector = orig }()

	called := false
	envSelector = func() (string, error) {
		called = true
		return "picked-env", nil
	}

	// File with template vars
	tmpFile := filepath.Join(t.TempDir(), "tmpl.hk.yaml")
	if err := os.WriteFile(tmpFile, []byte("url: '{{.apiUrl}}'"), 0o600); err != nil {
		t.Fatal(err)
	}

	f := &Flags{
		Env:      "global",
		EnvSet:   false,
		FilePath: tmpFile,
	}

	// Execute will return an error in test context (no hulak project), but we
	// only care that envSelector ran first. Discard the error.
	_ = Execute(f)

	if !called {
		t.Error("envSelector should be called when EnvSet is false and files have template vars")
	}
	if f.Env != "picked-env" {
		t.Errorf("Env = %q, want %q", f.Env, "picked-env")
	}
}

func TestExecute_EnvSet_SkipsSelector(t *testing.T) {
	orig := envSelector
	defer func() { envSelector = orig }()

	called := false
	envSelector = func() (string, error) {
		called = true
		return "should-not-be-used", nil
	}

	// File with template vars
	tmpFile := filepath.Join(t.TempDir(), "tmpl.hk.yaml")
	if err := os.WriteFile(tmpFile, []byte("url: '{{.apiUrl}}'"), 0o600); err != nil {
		t.Fatal(err)
	}

	f := &Flags{
		Env:      "staging",
		EnvSet:   true,
		FilePath: tmpFile,
	}

	_ = Execute(f)

	if called {
		t.Error("envSelector should NOT be called when EnvSet is true")
	}
	if f.Env != "staging" {
		t.Errorf("Env = %q, want %q (should remain unchanged)", f.Env, "staging")
	}
}

func TestExecute_NoTemplateVars_SkipsSelector(t *testing.T) {
	orig := envSelector
	defer func() { envSelector = orig }()

	called := false
	envSelector = func() (string, error) {
		called = true
		return "should-not-be-used", nil
	}

	// File WITHOUT template vars
	tmpFile := filepath.Join(t.TempDir(), "plain.hk.yaml")
	if err := os.WriteFile(tmpFile, []byte("method: GET\nurl: http://example.com"), 0o600); err != nil {
		t.Fatal(err)
	}

	f := &Flags{
		Env:      "global",
		EnvSet:   false,
		FilePath: tmpFile,
	}

	_ = Execute(f)

	if called {
		t.Error("envSelector should NOT be called when files have no template vars")
	}
}

func TestDiscoverFilePaths_FpAndDirTogether(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "single.yaml")
	content := "method: GET\nurl: http://example.com"
	if err := os.WriteFile(tmpFile, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	fileList, concurrent, sequential, err := discoverFilePaths(
		"",      // fileName
		tmpFile, // fp
		tmpDir,  // dir
		"",      // dirseq
		true,    // hasDirFlags
	)
	if err != nil {
		t.Fatalf("discoverFilePaths: %v", err)
	}

	if len(fileList) != 1 {
		t.Errorf("fileList should have 1 entry, got %v", fileList)
	}
	if len(concurrent) != 1 {
		t.Errorf("concurrent should have 1 entry from dir, got %v", concurrent)
	}
	if len(sequential) != 0 {
		t.Errorf("sequential should be empty, got %v", sequential)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{"zero", 0, "0ms"},
		{"sub-millisecond rounds down", 500 * time.Microsecond, "0ms"},
		{"under one second", 142 * time.Millisecond, "142ms"},
		{"just under one second", 999 * time.Millisecond, "999ms"},
		{"one second exactly switches to seconds", time.Second, "1.0s"},
		{"under one minute", 1234 * time.Millisecond, "1.2s"},
		{"one minute exactly", time.Minute, "1m0s"},
		{"minutes and seconds", 83 * time.Second, "1m23s"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatDuration(tc.d); got != tc.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tc.d, got, tc.want)
			}
		})
	}
}

func TestSplitErrorForOutcome(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		wantHeadline string
		wantHint     string
	}{
		{
			name:         "nil error returns empty",
			err:          nil,
			wantHeadline: "",
			wantHint:     "",
		},
		{
			name:         "plain error has no hint",
			err:          errors.New("something broke"),
			wantHeadline: "something broke",
			wantHint:     "",
		},
		{
			name: "env-key-missing hint is split off",
			err: errors.New(
				`substituting "client_id": key "client_id" not found in environment "global". Add "client_id=<value>" to env/global.env`,
			),
			wantHeadline: `substituting "client_id": key "client_id" not found in environment "global".`,
			wantHint:     `Add "client_id=<value>" to env/global.env`,
		},
		{
			name:         "marker requires opening quote — false-positive prose stays unsplit",
			err:          errors.New("you must Add this header before retrying"),
			wantHeadline: "you must Add this header before retrying",
			wantHint:     "",
		},
		{
			name:         "embedded newlines collapse to spaces",
			err:          errors.New("line one\nline two\nline three"),
			wantHeadline: "line one line two line three",
			wantHint:     "",
		},
		{
			name:         "ANSI escape codes are stripped",
			err:          errors.New("\x1b[31mred error\x1b[0m happened"),
			wantHeadline: "red error happened",
			wantHint:     "",
		},
		{
			name: "ANSI + newline + hint together",
			err: errors.New(
				"\x1b[31mwrapped\x1b[0m:\nkey \"X\" not found in environment \"prod\". Add \"X=<value>\" to env/prod.env",
			),
			wantHeadline: `wrapped: key "X" not found in environment "prod".`,
			wantHint:     `Add "X=<value>" to env/prod.env`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotHL, gotHint := splitErrorForOutcome(tc.err)
			if gotHL != tc.wantHeadline {
				t.Errorf("headline = %q, want %q", gotHL, tc.wantHeadline)
			}
			if gotHint != tc.wantHint {
				t.Errorf("hint = %q, want %q", gotHint, tc.wantHint)
			}
		})
	}
}

func TestRunFailureError_String(t *testing.T) {
	tests := []struct {
		name string
		err  *runFailureError
		want string
	}{
		{"single file", &runFailureError{failed: 1, total: 1}, "request failed"},
		{"multiple files", &runFailureError{failed: 2, total: 5}, "2 of 5 files failed"},
		{"all failed", &runFailureError{failed: 4, total: 4}, "4 of 4 files failed"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.err.Error(); got != tc.want {
				t.Errorf("Error() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestIsRunFailure(t *testing.T) {
	rf := &runFailureError{failed: 1, total: 3}
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil is not a run failure", nil, false},
		{"plain error is not", errors.New("boom"), false},
		{"runFailureError is", rf, true},
		{"wrapped runFailureError is detectable via errors.As", fmt.Errorf("wrap: %w", rf), true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsRunFailure(tc.err); got != tc.want {
				t.Errorf("IsRunFailure(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

// TestResolveBaseTimeout covers the flag → env → default precedence and
// rejects malformed env values up front (#206).
func TestResolveBaseTimeout(t *testing.T) {
	tests := []struct {
		name     string
		flag     time.Duration
		env      string // unset if empty
		want     time.Duration
		wantErr  bool
		errMatch string
	}{
		{"all unset → default", 0, "", DefaultTimeout, false, ""},
		{"env only", 0, "5m", 5 * time.Minute, false, ""},
		{"flag only", 2 * time.Minute, "", 2 * time.Minute, false, ""},
		{"flag wins over env", 2 * time.Minute, "5m", 2 * time.Minute, false, ""},
		{"invalid env duration", 0, "not-a-duration", 0, true, "invalid HULAK_TIMEOUT"},
		{"non-positive env duration", 0, "0s", 0, true, "must be positive"},
		{"negative env duration", 0, "-1s", 0, true, "must be positive"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.env == "" {
				t.Setenv(HulakTimeoutEnv, "")
				_ = os.Unsetenv(HulakTimeoutEnv)
			} else {
				t.Setenv(HulakTimeoutEnv, tc.env)
			}
			got, err := resolveBaseTimeout(tc.flag)
			if (err != nil) != tc.wantErr {
				t.Fatalf("error = %v, wantErr = %v", err, tc.wantErr)
			}
			if tc.wantErr {
				if !strings.Contains(err.Error(), tc.errMatch) {
					t.Errorf("error %q does not contain %q", err.Error(), tc.errMatch)
				}
				return
			}
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// TestProcessFilesSequentially_TimeoutEnforced verifies the sequential path
// honors baseTimeout: when the request takes longer than the timeout, the
// context deadline cancels the HTTP request and the outcome surfaces the
// error. Regression guard for the gap that existed before #206 was bundled
// into this branch (concurrent path enforced timeouts; sequential path was
// unbounded).
func TestProcessFilesSequentially_TimeoutEnforced(t *testing.T) {
	// Hold the handler open until the request's context cancels or the test
	// releases it. Using a channel instead of time.Sleep avoids slow-CI
	// flakiness and lets httptest.Server.Close() return immediately.
	block := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		select {
		case <-block:
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(server.Close)
	t.Cleanup(func() { close(block) })

	yaml := fmt.Sprintf("method: GET\nurl: %q\n", server.URL)
	path := filepath.Join(t.TempDir(), "slow.hk.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}

	timeout := 100 * time.Millisecond
	start := time.Now()
	outcomes := processFilesSequentially([]string{path}, nil, false, false, timeout)
	elapsed := time.Since(start)

	if len(outcomes) != 1 {
		t.Fatalf("got %d outcomes, want 1", len(outcomes))
	}
	o := outcomes[0]
	if o.ok {
		t.Errorf("outcome.ok = true, want false (request should have timed out)")
	}
	if o.err == nil || !strings.Contains(o.err.Error(), "context deadline exceeded") {
		t.Errorf("err = %v, want one containing 'context deadline exceeded'", o.err)
	}
	// Cap the slack generously so slow CI doesn't false-positive but a
	// regression that re-introduces an unbounded sequential mode would
	// clearly exceed the bound.
	if elapsed > 500*time.Millisecond {
		t.Errorf("elapsed %v — timeout did not cancel the in-flight request", elapsed)
	}
}

// TestProcessFilesSequentially_YAMLTimeoutWins is the end-to-end check that
// a YAML `timeout:` value actually reaches context.WithTimeout inside
// processTask. With a generous base (10s), a wiring slip that dropped the
// per-file override would cause the request to block until the test cleanup
// fires, blowing past the elapsed bound.
func TestProcessFilesSequentially_YAMLTimeoutWins(t *testing.T) {
	block := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		select {
		case <-block:
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(server.Close)
	t.Cleanup(func() { close(block) })

	yaml := fmt.Sprintf("method: GET\nurl: %q\ntimeout: 100ms\n", server.URL)
	path := filepath.Join(t.TempDir(), "yaml-timeout.hk.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}

	// Generous base — if the YAML field were ignored or lost in the wiring,
	// the request would block on the server until cleanup, and elapsed would
	// approach this base instead of the YAML's 100ms.
	base := 10 * time.Second
	start := time.Now()
	outcomes := processFilesSequentially([]string{path}, nil, false, false, base)
	elapsed := time.Since(start)

	if len(outcomes) != 1 {
		t.Fatalf("got %d outcomes, want 1", len(outcomes))
	}
	o := outcomes[0]
	if o.ok {
		t.Errorf("outcome.ok = true, want false")
	}
	if o.err == nil || !strings.Contains(o.err.Error(), "context deadline exceeded") {
		t.Errorf("err = %v, want one containing 'context deadline exceeded'", o.err)
	}
	// The elapsed time is the real proof that YAML override worked: ~100ms
	// not ~10s. If YAML timeout were ignored, elapsed would approach base.
	if elapsed > 500*time.Millisecond {
		t.Errorf("elapsed %v — YAML timeout did not override base %v", elapsed, base)
	}
}

func TestIsRetryable(t *testing.T) {
	cfgErr := &configError{errors.New("missing key")}
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil never retries", nil, false},
		{"configError fails fast", cfgErr, false},
		{"wrapped configError fails fast (errors.As)", fmt.Errorf("wrap: %w", cfgErr), false},
		{"plain error retries (assumed transport)", errors.New("dial tcp: timeout"), true},
		{"context.DeadlineExceeded-style retries", errors.New("timeout after 60s"), true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isRetryable(tc.err); got != tc.want {
				t.Errorf("isRetryable(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
