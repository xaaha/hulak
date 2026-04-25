package envselect

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"filippo.io/age"

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

	items := EnvItems()

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

	items := EnvItems()

	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

func TestEnvItemsIgnoresNonEnvFiles(t *testing.T) {
	cleanup := setupTestEnvDir(t, []string{"dev.env", "readme.txt", "config.yaml"})
	defer cleanup()

	items := EnvItems()

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

	items := EnvItems()

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
	err := NoEnvFilesError()

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

	gotErr := NoEnvFilesError()
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
