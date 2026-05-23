package envselect

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"filippo.io/age"
	"golang.org/x/term"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

func setupTestEnvDir(t *testing.T, envFiles []string) func() {
	t.Helper()

	tmpDir := t.TempDir()
	envDir := filepath.Join(tmpDir, "env")
	if err := os.MkdirAll(envDir, 0o755); err != nil {
		t.Fatal(err)
	}

	for _, name := range envFiles {
		f, err := os.Create(filepath.Join(envDir, name))
		if err != nil {
			t.Fatal(err)
		}
		_ = f.Close()
	}

	oldWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	return func() {
		err := os.Chdir(oldWd)
		if err != nil {
			t.Errorf("error on setupTestEnvDir: %v", err)
		}
	}
}

func TestEnvItemsWithEnvFiles(t *testing.T) {
	cleanup := setupTestEnvDir(t, []string{"dev.env", "prod.env", "staging.env"})
	defer cleanup()

	items, err := envItems()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) != 3 {
		t.Errorf("expected 3 items, got %d", len(items))
	}

	expected := map[string]bool{"dev": true, "prod": true, "staging": true}
	for _, item := range items {
		if !expected[item] {
			t.Errorf("unexpected item: %s", item)
		}
	}
}

func TestEnvItemsWithNoEnvFiles(t *testing.T) {
	cleanup := setupTestEnvDir(t, []string{})
	defer cleanup()

	items, err := envItems()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

func TestEnvItemsIgnoresNonEnvFiles(t *testing.T) {
	cleanup := setupTestEnvDir(t, []string{"dev.env", "readme.txt", "config.yaml"})
	defer cleanup()

	items, err := envItems()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}
	if items[0] != "dev" {
		t.Errorf("expected 'dev', got '%s'", items[0])
	}
}

func TestEnvItemsFromVault(t *testing.T) {
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Fatal(err)
		}
	}()

	tmpDir := t.TempDir()
	tmpDir, _ = filepath.EvalSymlinks(tmpDir)

	hulakDir := filepath.Join(tmpDir, utils.HiddenProjectName)
	if err := os.Mkdir(hulakDir, utils.DirPer); err != nil {
		t.Fatal(err)
	}

	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	configTmp := t.TempDir()
	configTmp, _ = filepath.EvalSymlinks(configTmp)
	t.Setenv("XDG_CONFIG_HOME", configTmp)
	t.Cleanup(func() {
		if err := os.Setenv("XDG_CONFIG_HOME", oldXDG); err != nil {
			t.Fatal(err)
		}
	})

	configDir, err := utils.UserConfigDir()
	if err != nil {
		t.Fatalf("UserConfigDir() error: %v", err)
	}
	if err := os.MkdirAll(configDir, utils.DirPer); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	id, _ := age.GenerateX25519Identity()
	if err := vault.SetIdentity(id.String()); err != nil {
		t.Fatalf("SetIdentity() error: %v", err)
	}

	store := &vault.Store{Envs: map[string]vault.Env{
		"global":  {"URL": "https://example.com"},
		"prod":    {"API_KEY": "sk-xxx"},
		"staging": {"API_KEY": "sk-staging"},
	}}
	if err := vault.WriteStore(store, id.Recipient()); err != nil {
		t.Fatalf("WriteStore() error: %v", err)
	}

	items, err := envItems()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) != 3 {
		t.Fatalf("EnvItems() len = %d, want 3", len(items))
	}
	expected := []string{"global", "prod", "staging"}
	for i, want := range expected {
		if items[i] != want {
			t.Errorf("EnvItems()[%d] = %q, want %q", i, items[i], want)
		}
	}
}

func TestNoEnvFilesError(t *testing.T) {
	// Isolate cwd so vault.DetectStore doesn't find an ancestor .hulak/
	// (which happens when the test runs inside a hulak project clone).
	// Mirrors the setup in TestNoEnvFilesErrorVaultMode but without
	// creating a store, so the classic branch is exercised.
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Fatal(err)
		}
	}()

	tmpDir := t.TempDir()
	tmpDir, _ = filepath.EvalSymlinks(tmpDir)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	err = noEnvFilesError()

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "no '.env' files found") {
		t.Error("error should mention no env files found")
	}
	if !strings.Contains(errStr, "Possible solutions") {
		t.Error("error should include possible solutions")
	}
}

func TestNoEnvFilesErrorVaultMode(t *testing.T) {
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Fatal(err)
		}
	}()

	tmpDir := t.TempDir()
	tmpDir, _ = filepath.EvalSymlinks(tmpDir)
	hulakDir := filepath.Join(tmpDir, utils.HiddenProjectName)
	if err := os.Mkdir(hulakDir, utils.DirPer); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hulakDir, utils.StoreFile), []byte("encrypted"), utils.SecretPer); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	gotErr := noEnvFilesError()
	if gotErr == nil {
		t.Fatal("expected error, got nil")
	}

	errStr := gotErr.Error()
	if !strings.Contains(errStr, "encrypted store") {
		t.Errorf("vault error should mention encrypted store, got: %s", errStr)
	}
	if strings.Contains(errStr, "hulak init") {
		t.Errorf("vault error should not mention 'hulak init', got: %s", errStr)
	}
}

// setupVaultProject prepares a working dir with a .hulak/ marker and an
// isolated XDG_CONFIG_HOME pointing at a temp dir. Returns the project dir.
// Used by the broken-vault regressions for #209.
func setupVaultProject(t *testing.T) string {
	t.Helper()

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Errorf("chdir back: %v", err)
		}
	})

	tmpDir := t.TempDir()
	tmpDir, _ = filepath.EvalSymlinks(tmpDir)
	hulakDir := filepath.Join(tmpDir, utils.HiddenProjectName)
	if err := os.Mkdir(hulakDir, utils.DirPer); err != nil {
		t.Fatal(err)
	}

	configTmp := t.TempDir()
	configTmp, _ = filepath.EvalSymlinks(configTmp)
	t.Setenv("XDG_CONFIG_HOME", configTmp)
	t.Setenv(utils.MasterKey, "") // ensure no env-var identity bleeds in

	configDir, err := utils.UserConfigDir()
	if err != nil {
		t.Fatalf("UserConfigDir: %v", err)
	}
	if err := os.MkdirAll(configDir, utils.DirPer); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	return tmpDir
}

// TestRunEnvSelector_NonTTYFailsFast asserts the selector refuses to launch
// when stdin is not a terminal. Without this guard, bubbletea would fall back
// to /dev/tty and hang in CI (PTY-allocated jobs) or emit a cryptic open
// error (detached contexts). The error must point the user at --env so the
// recovery is obvious.
func TestRunEnvSelector_NonTTYFailsFast(t *testing.T) {
	if term.IsTerminal(int(os.Stdin.Fd())) { //nolint:gosec // G115 fd is small non-neg
		t.Skip("test runner has a TTY on stdin; cannot exercise the non-TTY guard")
	}

	tmpDir := setupVaultProject(t)

	id, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("GenerateX25519Identity: %v", err)
	}
	if err := vault.SetIdentity(id.String()); err != nil {
		t.Fatalf("SetIdentity: %v", err)
	}
	store := &vault.Store{Envs: map[string]vault.Env{
		"global": {"K": "v"},
	}}
	if err := vault.WriteStore(store, id.Recipient()); err != nil {
		t.Fatalf("WriteStore: %v", err)
	}
	_ = tmpDir

	_, err = RunEnvSelector()
	if err == nil {
		t.Fatal("expected non-TTY guard to refuse, got nil error")
	}
	if !strings.Contains(err.Error(), "not a terminal") {
		t.Errorf("error should mention non-terminal context, got: %v", err)
	}
	if !strings.Contains(err.Error(), "--env") {
		t.Errorf("error should suggest passing --env, got: %v", err)
	}
}

// TestEnvItems_VaultErrors regression for #209 — vault read failures used to
// be swallowed and shown to the user as "no envs found", masking the real
// problem. Each subtest sets up a broken vault and asserts envItems surfaces
// the error instead of returning a silent empty list.
func TestEnvItems_VaultErrors(t *testing.T) {
	t.Run("missing identity", func(t *testing.T) {
		tmpDir := setupVaultProject(t)
		// Write a store file but no identity. ResolveIdentity must fail.
		storePath := filepath.Join(tmpDir, utils.HiddenProjectName, utils.StoreFile)
		if err := os.WriteFile(storePath, []byte("encrypted"), utils.SecretPer); err != nil {
			t.Fatal(err)
		}

		items, err := envItems()
		if err == nil {
			t.Fatalf("expected error, got items=%v", items)
		}
		if items != nil {
			t.Errorf("expected nil items on error, got %v", items)
		}
		if !strings.Contains(err.Error(), "vault") {
			t.Errorf("expected error to wrap with 'vault', got %v", err)
		}
	})

	t.Run("corrupted store", func(t *testing.T) {
		tmpDir := setupVaultProject(t)

		id, err := age.GenerateX25519Identity()
		if err != nil {
			t.Fatalf("GenerateX25519Identity: %v", err)
		}
		if err := vault.SetIdentity(id.String()); err != nil {
			t.Fatalf("SetIdentity: %v", err)
		}
		// Write garbage where the encrypted store should be — ReadStore will
		// fail to decrypt.
		storePath := filepath.Join(tmpDir, utils.HiddenProjectName, utils.StoreFile)
		if err := os.WriteFile(storePath, []byte("not a valid age ciphertext"), utils.SecretPer); err != nil {
			t.Fatal(err)
		}

		items, err := envItems()
		if err == nil {
			t.Fatalf("expected error, got items=%v", items)
		}
		if !strings.Contains(err.Error(), "vault") {
			t.Errorf("expected error to wrap with 'vault', got %v", err)
		}
	})
}
