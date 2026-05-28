package runcmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseRunArgsSetsFilePath(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.hk.yaml")
	if err := os.WriteFile(tmpFile, []byte("kind: API"), 0o600); err != nil {
		t.Fatal(err)
	}

	f, err := parseRunArgs(runCmdArgs{Args: []string{tmpFile}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if f.FilePath != tmpFile {
		t.Errorf("FilePath = %q, want %q", f.FilePath, tmpFile)
	}
	if f.Dir != "" || f.Dirseq != "" {
		t.Error("Dir/Dirseq should be empty for a file path")
	}
}

// TestParseRunArgsSetsDir verifies that passing a directory sets Dir (concurrent).
func TestParseRunArgsSetsDir(t *testing.T) {
	tmpDir := t.TempDir()

	f, err := parseRunArgs(runCmdArgs{Args: []string{tmpDir}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if f.Dir != tmpDir {
		t.Errorf("Dir = %q, want %q", f.Dir, tmpDir)
	}
	if f.Dirseq != "" {
		t.Error("Dirseq should be empty without --sequential")
	}
	if f.FilePath != "" {
		t.Error("FilePath should be empty for a directory")
	}
}

// TestParseRunArgsRoutesFlags verifies that parsed flag values map to the
// right runner.Flags fields. Flag parsing itself is the framework's job —
// this test only checks the path → Flags translation.
func TestParseRunArgsRoutesFlags(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.hk.yaml")
	if err := os.WriteFile(tmpFile, []byte("kind: API"), 0o600); err != nil {
		t.Fatal(err)
	}

	f, err := parseRunArgs(runCmdArgs{Env: "staging", Debug: true, Args: []string{tmpFile}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !f.Debug {
		t.Error("Debug should mirror the debug arg")
	}
	if f.Env != "staging" {
		t.Errorf("Env = %q, want %q", f.Env, "staging")
	}
	if !f.EnvSet {
		t.Error("EnvSet should be true when an env is provided")
	}
}

// TestParseRunArgsQuietPlumbed verifies quiet=true lands on runner.Flags.
func TestParseRunArgsQuietPlumbed(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.hk.yaml")
	if err := os.WriteFile(tmpFile, []byte("kind: API"), 0o600); err != nil {
		t.Fatal(err)
	}

	f, err := parseRunArgs(runCmdArgs{Quiet: true, Args: []string{tmpFile}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !f.Quiet {
		t.Error("Quiet should be true when quiet arg is true")
	}
}

// TestParseRunArgsDryRunPlumbed verifies DryRun=true lands on runner.Flags.
func TestParseRunArgsDryRunPlumbed(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.hk.yaml")
	if err := os.WriteFile(tmpFile, []byte("kind: API"), 0o600); err != nil {
		t.Fatal(err)
	}

	f, err := parseRunArgs(runCmdArgs{DryRun: true, Args: []string{tmpFile}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !f.DryRun {
		t.Error("DryRun should be true when DryRun arg is true")
	}
}

// TestParseRunArgsShowPlumbed verifies Show=true lands on runner.Flags.
func TestParseRunArgsShowPlumbed(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.hk.yaml")
	if err := os.WriteFile(tmpFile, []byte("kind: API"), 0o600); err != nil {
		t.Fatal(err)
	}

	f, err := parseRunArgs(runCmdArgs{Show: true, Args: []string{tmpFile}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !f.Show {
		t.Error("Show should be true when Show arg is true")
	}
}

// TestParseRunArgsDryRunAndShowPlumbed verifies both flags set together
// land on runner.Flags so --dry-run --show reveals headers as intended.
func TestParseRunArgsDryRunAndShowPlumbed(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.hk.yaml")
	if err := os.WriteFile(tmpFile, []byte("kind: API"), 0o600); err != nil {
		t.Fatal(err)
	}

	f, err := parseRunArgs(runCmdArgs{DryRun: true, Show: true, Args: []string{tmpFile}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !f.DryRun {
		t.Error("DryRun should be true")
	}
	if !f.Show {
		t.Error("Show should be true")
	}
}

// TestParseRunArgsSequentialDir verifies sequential=true on a directory
// routes to Dirseq instead of Dir.
func TestParseRunArgsSequentialDir(t *testing.T) {
	tmpDir := t.TempDir()

	f, err := parseRunArgs(runCmdArgs{Sequential: true, Args: []string{tmpDir}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if f.Dirseq != tmpDir {
		t.Errorf("Dirseq = %q, want %q", f.Dirseq, tmpDir)
	}
	if f.Dir != "" {
		t.Error("Dir should be empty when sequential is true")
	}
}

// TestParseRunArgsTimeoutPlumbed verifies the --timeout flag value lands on
// runner.Flags so the runner can resolve it. Precedence/fallback is asserted
// in runner.TestResolveBaseTimeout.
func TestParseRunArgsTimeoutPlumbed(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.hk.yaml")
	if err := os.WriteFile(tmpFile, []byte("kind: API"), 0o600); err != nil {
		t.Fatal(err)
	}

	want := 5 * time.Minute
	f, err := parseRunArgs(runCmdArgs{Timeout: want, Args: []string{tmpFile}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Timeout != want {
		t.Errorf("Timeout = %v, want %v", f.Timeout, want)
	}
}

// TestParseRunArgsBadPath verifies that a nonexistent path returns an error.
func TestParseRunArgsBadPath(t *testing.T) {
	_, err := parseRunArgs(runCmdArgs{Args: []string{"/nonexistent/path.yaml"}})
	if err == nil {
		t.Fatal("expected an error for nonexistent path")
	}
}

// TestParseRunArgsNoEnv verifies that omitting --env leaves Env empty and
// EnvSet false, so the runner knows to invoke the picker instead of silently
// defaulting to "global".
func TestParseRunArgsNoEnv(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.hk.yaml")
	if err := os.WriteFile(tmpFile, []byte("kind: API"), 0o600); err != nil {
		t.Fatal(err)
	}

	f, err := parseRunArgs(runCmdArgs{Args: []string{tmpFile}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if f.Env != "" {
		t.Errorf("Env = %q, want empty so the runner invokes the picker", f.Env)
	}
	if f.EnvSet {
		t.Error("EnvSet should be false when no env is provided")
	}
}
