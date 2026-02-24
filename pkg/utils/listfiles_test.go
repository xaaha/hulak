package utils

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
)

func touch(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("test"), 0o600); err != nil {
		t.Fatalf("failed to create file %s: %v", path, err)
	}
}

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("failed to create directory %s: %v", path, err)
	}
}

func TestListFiles_Basic(t *testing.T) {
	root := t.TempDir()

	file1 := filepath.Join(root, "a.yaml")
	file2 := filepath.Join(root, "b.json")
	touch(t, file1)
	touch(t, file2)

	got, err := ListFiles(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{file1, file2}
	// Order is not guaranteed — compare as sets
	if !reflect.DeepEqual(asSet(got), asSet(expected)) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
}

func TestListFiles_Recursive(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	mkdir(t, sub)

	file := filepath.Join(sub, "nested.yml")
	touch(t, file)

	got, err := ListFiles(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !contains(got, file) {
		t.Fatalf("expected recursive file %s, got %v", file, got)
	}
}

func TestListFiles_SkipDefaultDirs(t *testing.T) {
	root := t.TempDir()
	nd := filepath.Join(root, "node_modules")
	mkdir(t, nd)

	skipped := filepath.Join(nd, "ignored.json")
	touch(t, skipped)

	got, err := ListFiles(root)
	if err == nil {
		t.Fatalf("expected error (no files), got: %v", got)
	}
}

func TestListFiles_CustomSkipDirs(t *testing.T) {
	root := t.TempDir()

	mkdir(t, filepath.Join(root, "skipme"))
	f1 := filepath.Join(root, "skipme", "file.yaml")
	touch(t, f1)

	// File is skipped due to custom setting
	got, err := ListFiles(root, WithSkipDirs([]string{"skipme"}))
	if err == nil || len(got) != 0 {
		t.Fatalf("expected no files due to skip dir, got: %v", got)
	}
}

func TestListFiles_RespectDotDirsFalse(t *testing.T) {
	root := t.TempDir()
	dot := filepath.Join(root, ".config")
	mkdir(t, dot)

	f := filepath.Join(dot, "dot.yaml")
	touch(t, f)

	got, err := ListFiles(root, WithRespectDotDirs(false))
	if err == nil {
		t.Fatalf("expected error since no files should be seen")
	}
	if len(got) != 0 {
		t.Fatalf("dot directory should have been skipped; got %v", got)
	}
}

func TestListFiles_RespectDotDirsTrue(t *testing.T) {
	root := t.TempDir()
	dot := filepath.Join(root, ".config")
	mkdir(t, dot)

	f := filepath.Join(dot, "dot.yaml")
	touch(t, f)

	got, err := ListFiles(root, WithRespectDotDirs(true))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !contains(got, f) {
		t.Fatalf("expected to find file in dot dir: %s", f)
	}
}

func TestListFiles_NonExistentDir(t *testing.T) {
	_, err := ListFiles("does-not-exist-12345")
	if err == nil {
		t.Fatal("expected an error for non-existent directory")
	}
}

func TestListFiles_PathIsFile(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "file.json")
	touch(t, file)

	_, err := ListFiles(file) // passing file path instead of dir
	if err == nil {
		t.Fatal("expected error when path is a file")
	}
}

func TestListFiles_NoMatchingFiles(t *testing.T) {
	root := t.TempDir()

	touch(t, filepath.Join(root, "readme.txt"))

	_, err := ListFiles(root)
	if err == nil {
		t.Fatal("expected error when no YAML/YML/JSON files exist")
	}
}

// --- Helpers ---

func asSet(list []string) map[string]bool {
	set := make(map[string]bool)
	for _, v := range list {
		set[v] = true
	}
	return set
}

func contains(list []string, item string) bool {
	return slices.Contains(list, item)
}
