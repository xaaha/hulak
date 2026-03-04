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
			name:     "returns false when env directory does not exist",
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
