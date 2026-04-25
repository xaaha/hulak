package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsHulakProject(t *testing.T) {
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Fatal(err)
		}
	}()

	tests := []struct {
		name     string
		setup    func(t *testing.T, dir string)
		expected bool
	}{
		{
			name: "returns true when env directory exists",
			setup: func(t *testing.T, dir string) {
				if err := os.Mkdir(filepath.Join(dir, EnvironmentFolder), DirPer); err != nil {
					t.Fatal(err)
				}
			},
			expected: true,
		},
		{
			name: "returns true when .hulak directory exists",
			setup: func(t *testing.T, dir string) {
				if err := os.Mkdir(filepath.Join(dir, HiddenProjectName), DirPer); err != nil {
					t.Fatal(err)
				}
			},
			expected: true,
		},
		{
			name:     "returns false when neither marker exists",
			setup:    func(_ *testing.T, _ string) {},
			expected: false,
		},
		{
			name: "returns false when env is a file not a directory",
			setup: func(t *testing.T, dir string) {
				f, err := os.Create(filepath.Join(dir, EnvironmentFolder))
				if err != nil {
					t.Fatal(err)
				}
				f.Close()
			},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tc.setup(t, tmpDir)

			if err := os.Chdir(tmpDir); err != nil {
				t.Fatalf("failed to chdir to temp dir: %v", err)
			}

			result := IsHulakProject()
			if result != tc.expected {
				t.Errorf("IsHulakProject() = %v, want %v", result, tc.expected)
			}
		})
	}
}

func TestFindProjectRootPriority(t *testing.T) {
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Fatal(err)
		}
	}()

	t.Run(".hulak found before env when both exist in parent", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpDir, _ = filepath.EvalSymlinks(tmpDir)

		if err := os.Mkdir(filepath.Join(tmpDir, HiddenProjectName), DirPer); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(filepath.Join(tmpDir, EnvironmentFolder), DirPer); err != nil {
			t.Fatal(err)
		}

		subDir := filepath.Join(tmpDir, "subdir")
		if err := os.Mkdir(subDir, DirPer); err != nil {
			t.Fatal(err)
		}
		if err := os.Chdir(subDir); err != nil {
			t.Fatal(err)
		}

		root, found := FindProjectRoot()
		if !found {
			t.Fatal("FindProjectRoot() not found, want found")
		}
		if root != tmpDir {
			t.Errorf("FindProjectRoot() = %q, want %q", root, tmpDir)
		}
	})

	t.Run(".hulak only project is found", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpDir, _ = filepath.EvalSymlinks(tmpDir)

		if err := os.Mkdir(filepath.Join(tmpDir, HiddenProjectName), DirPer); err != nil {
			t.Fatal(err)
		}
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatal(err)
		}

		_, found := FindProjectRoot()
		if !found {
			t.Error("FindProjectRoot() should find .hulak-only project")
		}
	})
}
