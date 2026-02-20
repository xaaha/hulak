package fileselect

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestDir(t *testing.T, files []string) func() {
	t.Helper()

	tmpDir := t.TempDir()

	for _, name := range files {
		dir := filepath.Join(tmpDir, filepath.Dir(name))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}

		f, err := os.Create(filepath.Join(tmpDir, name))
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
			t.Errorf("error on setupTestDir: %v", err)
		}
	}
}

func TestFileItemsWithFiles(t *testing.T) {
	cleanup := setupTestDir(t, []string{"collection/get_users.yaml", "collection/post_data.yml"})
	defer cleanup()

	items, err := FileItems()
	if err != nil {
		t.Fatalf("fileItems returned error: %v", err)
	}

	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}

	expected := map[string]bool{
		filepath.Join("collection", "get_users.yaml"): true,
		filepath.Join("collection", "post_data.yml"):  true,
	}
	for _, item := range items {
		if !expected[item] {
			t.Errorf("unexpected item: %s", item)
		}
	}
}

func TestFileItemsWithNoFiles(t *testing.T) {
	cleanup := setupTestDir(t, []string{})
	defer cleanup()

	items, err := FileItems()
	if err != nil {
		t.Fatalf("fileItems returned error: %v", err)
	}

	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

func TestFileItemsFiltersResponseFiles(t *testing.T) {
	cleanup := setupTestDir(t, []string{"api.yaml", "api_response.json"})
	defer cleanup()

	items, err := FileItems()
	if err != nil {
		t.Fatalf("fileItems returned error: %v", err)
	}

	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}
	if items[0] != "api.yaml" {
		t.Errorf("expected 'api.yaml', got '%s'", items[0])
	}
}

func TestFileItemsFiltersJsonFiles(t *testing.T) {
	cleanup := setupTestDir(t, []string{"api.yaml", "data.json"})
	defer cleanup()

	items, err := FileItems()
	if err != nil {
		t.Fatalf("fileItems returned error: %v", err)
	}

	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}
	if items[0] != "api.yaml" {
		t.Errorf("expected 'api.yaml', got '%s'", items[0])
	}
}

func TestFileItemsFiltersEnvDir(t *testing.T) {
	cleanup := setupTestDir(t, []string{"api.yaml", "env/global.env"})
	defer cleanup()

	items, err := FileItems()
	if err != nil {
		t.Fatalf("fileItems returned error: %v", err)
	}

	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}
	if items[0] != "api.yaml" {
		t.Errorf("expected 'api.yaml', got '%s'", items[0])
	}
}

func TestFileItemsShowsRelativePaths(t *testing.T) {
	cleanup := setupTestDir(t, []string{"collection/get_users.yaml"})
	defer cleanup()

	items, err := FileItems()
	if err != nil {
		t.Fatalf("fileItems returned error: %v", err)
	}

	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}

	if strings.HasPrefix(items[0], "/") {
		t.Errorf("expected relative path, got absolute: %s", items[0])
	}
	if items[0] != filepath.Join("collection", "get_users.yaml") {
		t.Errorf("expected 'collection/get_users.yaml', got '%s'", items[0])
	}
}

func TestFileItemsKeepsYamlWithResponseInName(t *testing.T) {
	cleanup := setupTestDir(t, []string{"create_response.yaml", "plain.yaml"})
	defer cleanup()

	items, err := FileItems()
	if err != nil {
		t.Fatalf("fileItems returned error: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	expected := map[string]bool{"create_response.yaml": true, "plain.yaml": true}
	for _, item := range items {
		if !expected[item] {
			t.Errorf("unexpected item: %s", item)
		}
	}
}

func TestFormatNoFilesError(t *testing.T) {
	err := NoFilesError()

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "no '.yaml' or '.yml' files found") {
		t.Error("error should mention no yaml/yml files found")
	}
	if !strings.Contains(errStr, "Possible solutions") {
		t.Error("error should include possible solutions")
	}
}
