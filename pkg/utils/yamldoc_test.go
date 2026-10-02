package utils

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestD22OnlyNonEmptyDocumentsCount(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantLine string
	}{
		{"plain mapping", "method: POST\nurl: https://e.com\n", ""},
		{"leading separator", "---\nmethod: POST\nurl: https://e.com\n", ""},
		{"trailing separator", "method: POST\nurl: https://e.com\n---\n", ""},
		{"leading and trailing separators", "---\nmethod: POST\n---\n", ""},
		{"two trailing separators", "method: POST\n---\n---\n", ""},
		{"trailing separator then blank lines", "method: POST\n---\n\n\n", ""},
		{"yaml directive", "%YAML 1.2\n---\nmethod: POST\nurl: https://e.com\n", ""},
		{
			"tag directive",
			"%TAG !e! tag:example.com,2000:app/\n---\nmethod: POST\nurl: https://e.com\n",
			"",
		},
		{"stray line splits the mapping", "method: POST\n// stray\nurl: https://e.com\n", "line 2"},
		{"deliberate second document", "method: POST\n---\nurl: https://e.com\n", "line 2"},
		{"second document is an explicit null", "method: POST\n---\nnull\n", "line 2"},
		{"split after a leading separator", "---\nmethod: POST\n// stray\n", "line 3"},
		{"split after a yaml directive", "%YAML 1.2\n---\nmethod: POST\n---\nurl: x\n", "line 4"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSingleYAMLDoc("req.hk.yaml", []byte(tt.content))
			if tt.wantLine == "" {
				if err != nil {
					t.Fatalf("want no error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("want an error, got nil")
			}
			if !strings.Contains(err.Error(), "req.hk.yaml") {
				t.Errorf("error should name the file, got %q", err)
			}
			if !strings.Contains(err.Error(), tt.wantLine) {
				t.Errorf("error should name %s, got %q", tt.wantLine, err)
			}
		})
	}
}

// yamlCorpusRoot is the repo itself, or the directory named by
// HULAK_YAML_CORPUS so the same check can be pointed at another project.
func yamlCorpusRoot(t *testing.T) string {
	t.Helper()
	if root := os.Getenv("HULAK_YAML_CORPUS"); root != "" {
		return root
	}
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("no go.mod above the test's working directory")
		}
		dir = parent
	}
}

func TestD23RepoYAMLFilesAreSingleDocument(t *testing.T) {
	root := yamlCorpusRoot(t)

	var paths []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".yaml", ".yml":
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("walked the repo and found no YAML files")
	}

	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := ValidateSingleYAMLDoc(path, content); err != nil {
			t.Error(err)
		}
	}
	t.Logf("checked %d YAML files", len(paths))
}
