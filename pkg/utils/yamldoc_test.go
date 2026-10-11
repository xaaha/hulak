package utils

import (
	"bytes"
	"io/fs"
	"os"
	"os/exec"
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
		{"two leading separators", "---\n---\nmethod: POST\nurl: https://e.com\n", 0},
		{"trailing separator then blank lines", "method: POST\n---\n\n\n", 0},
		{"second document is only a comment", "method: POST\n---\n# just a comment\n", 0},
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

func TestD22GuardMessageNamesTheFileAndTheLine(t *testing.T) {
	err := ValidateSingleYAMLDoc("req.hk.yaml", []byte("method: POST\n---\nurl: https://e.com\n"))
	want := "req.hk.yaml: a request file must be a single YAML document, " +
		"but a second document starts at line 2"
	if err == nil || err.Error() != want {
		t.Fatalf("error = %v, want %q", err, want)
	}
	if want != testutil.SingleDocError("req.hk.yaml", 2) {
		t.Errorf("testutil.SingleDocError no longer builds %q", want)
	}
}

func TestD22ValidateSingleYAMLDocReportsParseFailure(t *testing.T) {
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

const (
	corpusEnv     = "HULAK_YAML_CORPUS"
	repoYAMLCount = 35
)

var (
	corpusSkipDirs = []string{
		"node_modules",
		"vendor",
		"dist",
		"build",
		"target",
		"tmp",
		"venv",
		"__pycache__",
	}

	corpusFixtureDirs = []string{
		filepath.Join("assets", "demo"),
		filepath.Join("e2etests", "test_collection"),
		filepath.Join("pkg", "userFlags", "example", "examples"),
	}
)

func skipCorpusDir(name string) bool {
	return strings.HasPrefix(name, ".") || slices.Contains(corpusSkipDirs, name)
}

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

func collectYAMLFiles(root string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			if isWalkPermissionError(err) {
				return nil
			}
			return err
		}
		if entry.IsDir() {
			if path != root && skipCorpusDir(entry.Name()) {
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
		return nil, err
	}
	return paths, nil
}

func readCorpusFile(t *testing.T, path string) ([]byte, bool) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		if isWalkPermissionError(err) {
			return nil, false
		}
		t.Fatal(err)
	}
	return content, true
}

func corpusYAMLFiles(t *testing.T) (repo, extra []string) {
	t.Helper()
	dir := repoRoot(t)
	walked, err := collectYAMLFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	repo = walked
	if tracked, ok := gitTrackedFiles(dir); ok {
		repo = slices.DeleteFunc(repo, func(p string) bool { return !tracked[p] })
	}
	root := os.Getenv(corpusEnv)
	if root == "" {
		return repo, nil
	}
	extra, err = collectYAMLFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(extra) == 0 {
		t.Fatalf("%s=%s holds no YAML files", corpusEnv, root)
	}
	return repo, extra
}

// gitTrackedFiles reports false outside a git work tree, e.g. a release tarball.
func gitTrackedFiles(root string) (map[string]bool, bool) {
	out, err := exec.Command("git", "-C", root, "ls-files", "-z").Output()
	if err != nil {
		return nil, false
	}
	tracked := map[string]bool{}
	for _, rel := range bytes.Split(out, []byte{0}) {
		if len(rel) > 0 {
			tracked[filepath.Join(root, string(rel))] = true
		}
	}
	return tracked, true
}

func TestD23RepoYAMLFilesAreSingleDocument(t *testing.T) {
	repo, extra := corpusYAMLFiles(t)

	if len(repo) != repoYAMLCount {
		t.Fatalf("the repo walk found %d YAML files, want %d: update repoYAMLCount when you add or remove one",
			len(repo), repoYAMLCount)
	}
	root := repoRoot(t)
	for _, dir := range corpusFixtureDirs {
		prefix := filepath.Join(root, dir) + string(filepath.Separator)
		if !slices.ContainsFunc(repo, func(p string) bool { return strings.HasPrefix(p, prefix) }) {
			t.Errorf("the walk checked no file under %s", prefix)
		}
	}

	checked := 0
	for _, path := range slices.Concat(repo, extra) {
		content, ok := readCorpusFile(t, path)
		if !ok {
			continue
		}
		checked++
		if err := ValidateSingleYAMLDoc(path, content); err != nil {
			t.Error(err)
		}
	}
	t.Logf("checked %d YAML files", checked)
}

func TestD23CorpusOverrideAddsToTheRepoWalk(t *testing.T) {
	t.Setenv(corpusEnv, "")
	repoOnly, extraOnly := corpusYAMLFiles(t)
	if len(extraOnly) != 0 {
		t.Fatalf("an unset %s gave %d extra files, want 0", corpusEnv, len(extraOnly))
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "extra.yaml"), []byte("a: 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(corpusEnv, dir)

	repo, extra := corpusYAMLFiles(t)
	if len(repo) != len(repoOnly) || len(extra) != 1 {
		t.Errorf("%s gave %d repo and %d extra files, want the repo's %d and the 1 it adds",
			corpusEnv, len(repo), len(extra), len(repoOnly))
	}
}

func TestD23CorpusWalkSkipsUnreadableDirectories(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "ok.yaml"), []byte("a: 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	locked := filepath.Join(root, "locked")
	if err := os.Mkdir(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o700) })
	if _, err := os.ReadDir(locked); err == nil {
		t.Skip("this user can read a 0000 directory")
	}

	got, err := collectYAMLFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Errorf("walked %d files, want the 1 readable one", len(got))
	}
}

func TestD23CorpusWalkReportsNonPermissionErrors(t *testing.T) {
	_, err := collectYAMLFiles(filepath.Join(t.TempDir(), "missing"))
	if err == nil {
		t.Fatal("want an error for a missing root, got nil")
	}
	if isWalkPermissionError(err) {
		t.Errorf("error = %v, want a non-permission error", err)
	}
}

func TestD23CorpusReadSkipsUnreadableFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "locked.yaml")
	if err := os.WriteFile(path, []byte("a: 1\n"), 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o600) })
	if _, err := os.ReadFile(path); err == nil {
		t.Skip("this user can read a 0000 file")
	}

	if _, ok := readCorpusFile(t, path); ok {
		t.Error("read an unreadable file, want it skipped")
	}
}
