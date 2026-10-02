package utils

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xaaha/hulak/pkg/utils/testutil"
)

func TestD22OnlyNonEmptyDocumentsCount(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantLine int
	}{
		{"plain mapping", "method: POST\nurl: https://e.com\n", 0},
		{"leading separator", "---\nmethod: POST\nurl: https://e.com\n", 0},
		{"trailing separator", "method: POST\nurl: https://e.com\n---\n", 0},
		{"leading and trailing separators", "---\nmethod: POST\n---\n", 0},
		{"two trailing separators", "method: POST\n---\n---\n", 0},
		{"trailing separator then blank lines", "method: POST\n---\n\n\n", 0},
		{"yaml directive", "%YAML 1.2\n---\nmethod: POST\nurl: https://e.com\n", 0},
		{
			"tag directive",
			"%TAG !e! tag:example.com,2000:app/\n---\nmethod: POST\nurl: https://e.com\n",
			0,
		},
		{"stray line splits the mapping", "method: POST\n// stray\nurl: https://e.com\n", 2},
		{"deliberate second document", "method: POST\n---\nurl: https://e.com\n", 2},
		{"second document is an explicit null", "method: POST\n---\nnull\n", 2},
		{"split after a leading separator", "---\nmethod: POST\n// stray\n", 3},
		{"split after a yaml directive", "%YAML 1.2\n---\nmethod: POST\n---\nurl: x\n", 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSingleYAMLDoc("req.hk.yaml", []byte(tt.content))
			if tt.wantLine == 0 {
				if err != nil {
					t.Fatalf("want no error, got %v", err)
				}
				return
			}
			want := testutil.SingleDocError("req.hk.yaml", tt.wantLine)
			if err == nil || err.Error() != want {
				t.Errorf("error = %v, want %q", err, want)
			}
		})
	}
}

func TestValidateSingleYAMLDocReportsParseFailure(t *testing.T) {
	err := ValidateSingleYAMLDoc("req.hk.yaml", []byte("{method: POST\n"))
	if err == nil {
		t.Fatal("want an error, got nil")
	}
	if !strings.HasPrefix(err.Error(), "parsing req.hk.yaml: ") {
		t.Errorf("error = %q, want it to start with the parsing prefix and the name", err)
	}
	if !strings.Contains(err.Error(), "unterminated flow mapping") {
		t.Errorf("error = %q, want it to carry the parser's own message", err)
	}
}

// repoYAMLFloor catches the walk quietly shrinking. The repo holds 35 YAML
// files outside the directories ListFiles skips.
const (
	corpusEnv     = "HULAK_YAML_CORPUS"
	repoYAMLFloor = 30
)

func repoRoot(t *testing.T) string {
	t.Helper()
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

func collectYAMLFiles(t *testing.T, root string) []string {
	t.Helper()
	opts := defaultOptions()
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != root && shouldSkipDir(entry.Name(), opts) {
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
	return paths
}

func corpusYAMLFiles(t *testing.T) []string {
	t.Helper()
	paths := collectYAMLFiles(t, repoRoot(t))
	if extra := os.Getenv(corpusEnv); extra != "" {
		paths = append(paths, collectYAMLFiles(t, extra)...)
	}
	return paths
}

func TestD23RepoYAMLFilesAreSingleDocument(t *testing.T) {
	paths := corpusYAMLFiles(t)

	if len(paths) < repoYAMLFloor {
		t.Fatalf("walked the corpus and found %d YAML files, want at least %d",
			len(paths), repoYAMLFloor)
	}
	e2e := filepath.Join(repoRoot(t), "e2etests") + string(filepath.Separator)
	if !slices.ContainsFunc(paths, func(p string) bool { return strings.HasPrefix(p, e2e) }) {
		t.Errorf("the walk checked no file under %s", e2e)
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

func TestD23CorpusOverrideAddsToTheRepoWalk(t *testing.T) {
	t.Setenv(corpusEnv, "")
	repoOnly := len(corpusYAMLFiles(t))

	extra := t.TempDir()
	if err := os.WriteFile(filepath.Join(extra, "extra.yaml"), []byte("a: 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(corpusEnv, extra)

	if got := len(corpusYAMLFiles(t)); got != repoOnly+1 {
		t.Errorf("%s gave %d files, want the repo's %d plus the 1 it adds",
			corpusEnv, got, repoOnly+1)
	}
}
