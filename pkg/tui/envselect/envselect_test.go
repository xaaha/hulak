package envselect

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func TestFormatNoEnvFilesError(t *testing.T) {
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
