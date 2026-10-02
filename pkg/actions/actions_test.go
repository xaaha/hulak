package actions

import (
	"encoding/base64"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/xaaha/hulak/pkg/utils"
)

func Test_processValueOf(t *testing.T) {
	// Create temporary test files
	tmpDir, err := os.MkdirTemp("", "hulak-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test JSON file with an object at root
	objectJSON := `{
		"name": "xaaha",
		"age": 30,
		"nested": {
			"key": "value",
			"array": [1, 2, 3]
		}
	}`
	objectFilePath := filepath.Join(tmpDir, "object.json")
	if err := os.WriteFile(objectFilePath, []byte(objectJSON), 0600); err != nil {
		t.Fatalf("Failed to write object test file: %v", err)
	}

	// Create a test JSON file with an array at root
	arrayJSON := `[
		{
			"access_token": "pratik",
			"refresh_token": "",
			"id_token": "",
			"scope": "",
			"expires_in": 86400,
			"token_type": "Bearer"
		},
		{
			"access_token": "thapa",
			"refresh_token": "",
			"id_token": "",
			"scope": "",
			"expires_in": 81691643,
			"token_type": "Bearer"
		}
	]`
	arrayFilePath := filepath.Join(tmpDir, "array.json")
	if err := os.WriteFile(arrayFilePath, []byte(arrayJSON), 0600); err != nil {
		t.Fatalf("Failed to write array test file: %v", err)
	}

	// Create an invalid JSON file for testing error cases
	invalidJSON := `{ "invalid": json }`
	invalidFilePath := filepath.Join(tmpDir, "invalid.json")
	if err := os.WriteFile(invalidFilePath, []byte(invalidJSON), 0600); err != nil {
		t.Fatalf("Failed to write invalid test file: %v", err)
	}

	tests := []struct {
		name     string // description of this test case
		key      string
		fileName string
		want     any
	}{
		// Object JSON tests
		{
			name:     "Get direct property from object",
			key:      "name",
			fileName: objectFilePath,
			want:     "xaaha",
		},
		{
			name:     "Get nested property from object",
			key:      "nested.key",
			fileName: objectFilePath,
			want:     "value",
		},
		{
			name:     "Get array element from nested property in object",
			key:      "nested.array[1]",
			fileName: objectFilePath,
			want:     2,
		},
		{
			name:     "Get nonexistent property from object",
			key:      "nonexistent",
			fileName: objectFilePath,
			want:     "",
		},

		// Array JSON tests
		{
			name:     "Get property from array element",
			key:      "[0].access_token",
			fileName: arrayFilePath,
			want:     "pratik",
		},
		{
			name:     "Get property from second array element",
			key:      "[1].access_token",
			fileName: arrayFilePath,
			want:     "thapa",
		},
		{
			name:     "Get expires_in from second array element",
			key:      "[1].expires_in",
			fileName: arrayFilePath,
			want:     81691643,
		},
		{
			name:     "Use invalid syntax for array (missing brackets)",
			key:      "0.access_token",
			fileName: arrayFilePath,
			want:     "",
		},
		{
			name:     "Access out of bounds array index",
			key:      "[2].access_token",
			fileName: arrayFilePath,
			want:     "",
		},

		// Error cases
		{
			name:     "Empty key",
			key:      "",
			fileName: objectFilePath,
			want:     "",
		},
		{
			name:     "Empty filename",
			key:      "name",
			fileName: "",
			want:     "",
		},
		{
			name:     "Nonexistent file",
			key:      "name",
			fileName: filepath.Join(tmpDir, "nonexistent.json"),
			want:     "",
		},
		{
			name:     "Invalid JSON file",
			key:      "invalid",
			fileName: invalidFilePath,
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := processValueOf(tt.key, tt.fileName)

			// Compare results
			if got != tt.want {
				t.Errorf("processValueOf() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBasicAuth(t *testing.T) {
	tests := []struct {
		name     string
		username string
		password string
		want     string
	}{
		{
			name:     "standard credentials",
			username: "admin",
			password: "secret123",
			want:     "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:secret123")),
		},
		{
			name:     "empty username",
			username: "",
			password: "secret",
			want:     "Basic " + base64.StdEncoding.EncodeToString([]byte(":secret")),
		},
		{
			name:     "empty password",
			username: "admin",
			password: "",
			want:     "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:")),
		},
		{
			name:     "both empty",
			username: "",
			password: "",
			want:     "Basic " + base64.StdEncoding.EncodeToString([]byte(":")),
		},
		{
			name:     "special characters in password",
			username: "user@domain.com",
			password: "p@ss:w0rd/with=special+chars",
			want:     "Basic " + base64.StdEncoding.EncodeToString([]byte("user@domain.com:p@ss:w0rd/with=special+chars")),
		},
		{
			name:     "unicode characters",
			username: "usuario",
			password: "contraseña",
			want:     "Basic " + base64.StdEncoding.EncodeToString([]byte("usuario:contraseña")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BasicAuth(tt.username, tt.password)
			if got != tt.want {
				t.Errorf("BasicAuth() = %q, want %q", got, tt.want)
			}
			if !strings.HasPrefix(got, "Basic ") {
				t.Errorf("BasicAuth() should start with 'Basic ', got %q", got)
			}
		})
	}
}

// setupHulakProject creates a temp directory with env/ to simulate a hulak project,
// changes into it, and returns a cleanup function that restores the original cwd.
func setupHulakProject(t *testing.T) string {
	t.Helper()

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}

	tmpDir := t.TempDir()
	// Resolve symlinks (macOS /var -> /private/var) so paths are consistent
	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatalf("failed to resolve symlinks: %v", err)
	}

	if err := os.Mkdir(filepath.Join(tmpDir, utils.EnvironmentFolder), utils.DirPer); err != nil {
		t.Fatalf("failed to create env dir: %v", err)
	}

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir to temp dir: %v", err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Fatal(err)
		}
	})

	return tmpDir
}

func TestGetFile(t *testing.T) {
	projectDir := setupHulakProject(t)

	// Create test files within the project
	testContent := "hello world\nline two\n"
	testFile := filepath.Join(projectDir, "testfile.txt")
	if err := os.WriteFile(testFile, []byte(testContent), utils.FilePer); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	subDir := filepath.Join(projectDir, "subdir")
	if err := os.Mkdir(subDir, utils.DirPer); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}
	nestedFile := filepath.Join(subDir, "nested.txt")
	nestedContent := "nested content"
	if err := os.WriteFile(nestedFile, []byte(nestedContent), utils.FilePer); err != nil {
		t.Fatalf("failed to write nested file: %v", err)
	}

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty path returns error",
			input:   "",
			wantErr: true,
			errMsg:  "file path cannot be empty",
		},
		{
			name:  "reads file with absolute path",
			input: testFile,
			want:  testContent,
		},
		{
			name:  "reads file with relative path",
			input: "testfile.txt",
			want:  testContent,
		},
		{
			name:  "reads nested file with relative path",
			input: "subdir/nested.txt",
			want:  nestedContent,
		},
		{
			name:    "rejects path outside project root",
			input:   "/etc/hosts",
			wantErr: true,
			errMsg:  "access denied",
		},
		{
			name:    "rejects directory path",
			input:   "subdir",
			wantErr: true,
			errMsg:  "is a directory",
		},
		{
			name:    "nonexistent file returns error",
			input:   "does_not_exist.txt",
			wantErr: true,
			errMsg:  "file does not exist",
		},
		{
			name:  "reads file with absolute path in nested dir",
			input: nestedFile,
			want:  nestedContent,
		},
		{
			name:    "path traversal outside project is rejected",
			input:   "../../../etc/passwd",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetFile(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("GetFile(%q) expected error, got nil", tt.input)
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("GetFile(%q) error = %q, want it to contain %q", tt.input, err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Errorf("GetFile(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("GetFile(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestGetFile_NotHulakProject(t *testing.T) {
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}

	tmpDir := t.TempDir()
	// No env/ directory — not a hulak project
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Fatal(err)
		}
	}()

	_, err = GetFile("somefile.txt")
	if err == nil {
		t.Error("GetFile() expected error outside hulak project, got nil")
	}
	if !strings.Contains(err.Error(), "not a hulak project") {
		t.Errorf("GetFile() error = %q, want it to contain 'not a hulak project'", err.Error())
	}
}

func TestGetFile_PreservesFormatting(t *testing.T) {
	projectDir := setupHulakProject(t)

	content := "{\n  \"key\": \"value\",\n  \"nested\": {\n    \"arr\": [1, 2, 3]\n  }\n}\n"
	filePath := filepath.Join(projectDir, "formatted.json")
	if err := os.WriteFile(filePath, []byte(content), utils.FilePer); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	got, err := GetFile("formatted.json")
	if err != nil {
		t.Fatalf("GetFile() unexpected error: %v", err)
	}
	if got != content {
		t.Errorf("GetFile() did not preserve formatting.\ngot:\n%s\nwant:\n%s", got, content)
	}
}

// Reusing a result past a rewrite is the bug; parsing on every call is what
// the cache exists to avoid.
func TestResultReusedUntilContentsChange(t *testing.T) {
	root := setupHulakProject(t)
	t.Cleanup(ResetCache)
	ResetCache()

	path := filepath.Join(root, "auth"+utils.ResponseFileName)
	write := func(token string) {
		t.Helper()
		body := `{"access_token": "` + token + `"}`
		if err := os.WriteFile(path, []byte(body), utils.FilePer); err != nil {
			t.Fatalf("failed to write response file: %v", err)
		}
	}

	write("stale-token")
	if got := GetValueOf("access_token", path); got != "stale-token" {
		t.Fatalf("first read: got %v, want stale-token", got)
	}

	// Poisoning the entry is the only evidence available that the next call
	// answered from the cache rather than parsing the file again.
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	valueCacheMu.Lock()
	valueCache[valueCacheKey(path, raw, "access_token")] = "from-the-cache"
	valueCacheMu.Unlock()

	if got := GetValueOf("access_token", path); got != "from-the-cache" {
		t.Errorf("unchanged file: got %v, want from-the-cache — the result was parsed again", got)
	}

	write("a-longer-fresh-token")
	if got := GetValueOf("access_token", path); got != "a-longer-fresh-token" {
		t.Errorf("after rewrite: got %v, want a-longer-fresh-token", got)
	}
}

// Both rewrites are invisible to a stat: same size, same mtime, and in the
// second case the inode moves, which os.SameFile cannot see on Windows.
func TestContentsDecideFreshnessNotTheStat(t *testing.T) {
	root := setupHulakProject(t)
	t.Cleanup(ResetCache)

	cases := map[string]func(t *testing.T, path string, replacement []byte){
		"in place, size and mtime restored": func(t *testing.T, path string, replacement []byte) {
			t.Helper()
			before, err := os.Stat(path)
			if err != nil {
				t.Fatalf("stat: %v", err)
			}
			f, err := os.OpenFile(path, os.O_WRONLY, utils.FilePer)
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			if _, err := f.WriteAt(replacement, 0); err != nil {
				f.Close()
				t.Fatalf("write at: %v", err)
			}
			if err := f.Close(); err != nil {
				t.Fatalf("close: %v", err)
			}
			if err := os.Chtimes(path, before.ModTime(), before.ModTime()); err != nil {
				t.Fatalf("chtimes: %v", err)
			}
			assertStatUnchanged(t, path, before)
		},
		"renamed over, size and mtime restored": func(t *testing.T, path string, replacement []byte) {
			t.Helper()
			before, err := os.Stat(path)
			if err != nil {
				t.Fatalf("stat: %v", err)
			}
			tmp := path + ".replacement"
			if err := os.WriteFile(tmp, replacement, utils.FilePer); err != nil {
				t.Fatalf("write replacement: %v", err)
			}
			if err := os.Rename(tmp, path); err != nil {
				t.Fatalf("rename: %v", err)
			}
			if err := os.Chtimes(path, before.ModTime(), before.ModTime()); err != nil {
				t.Fatalf("chtimes: %v", err)
			}
			assertStatUnchanged(t, path, before)
		},
	}

	for name, rewrite := range cases {
		t.Run(name, func(t *testing.T) {
			ResetCache()

			path := filepath.Join(root, utils.SanitizeFileName(name)+utils.ResponseFileName)
			if err := os.WriteFile(path, []byte(`{"access_token": "aaa"}`), utils.FilePer); err != nil {
				t.Fatalf("failed to write response file: %v", err)
			}
			if got := GetValueOf("access_token", path); got != "aaa" {
				t.Fatalf("first read: got %v, want aaa", got)
			}

			rewrite(t, path, []byte(`{"access_token": "bbb"}`))

			if got := GetValueOf("access_token", path); got != "bbb" {
				t.Errorf("after rewrite: got %v, want bbb", got)
			}
		})
	}
}

// assertStatUnchanged guards the setup: without it a rewrite that moved the
// size or the mtime would pass for the wrong reason.
func assertStatUnchanged(t *testing.T, path string, before os.FileInfo) {
	t.Helper()

	after, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat after rewrite: %v", err)
	}
	if after.Size() != before.Size() {
		t.Fatalf("size moved: %d, want %d", after.Size(), before.Size())
	}
	if !after.ModTime().Equal(before.ModTime()) {
		t.Fatalf("mtime moved: %v, want %v", after.ModTime(), before.ModTime())
	}
}

// Parallel runner workers share one cached result. Only keys returning a
// map or slice share structure, so scalars alone would not catch a regression.
func TestConcurrentLookupsShareNoMutableState(t *testing.T) {
	root := setupHulakProject(t)
	t.Cleanup(ResetCache)
	ResetCache()

	path := filepath.Join(root, "shapes"+utils.ResponseFileName)
	body := `{"scalar": 1, "obj": {"a": 1, "b": 2}, "arr": [1, 2, 3], "deep": {"in": {"x": 9}}}`
	if err := os.WriteFile(path, []byte(body), utils.FilePer); err != nil {
		t.Fatalf("failed to write response file: %v", err)
	}

	keys := []string{"scalar", "obj", "arr", "deep", "deep.in", "arr[1]", "obj.a"}
	var wg sync.WaitGroup
	for range 32 {
		for _, key := range keys {
			wg.Add(1)
			go func(key string) {
				defer wg.Done()
				if got := GetValueOf(key, path); got == "" {
					t.Errorf("key %q came back empty", key)
				}
			}(key)
		}
	}
	wg.Wait()
}

func TestBareNameWalkedOncePerRoot(t *testing.T) {
	t.Cleanup(ResetCache)
	ResetCache()

	walks := 0
	original := listMatchingFiles
	listMatchingFiles = func(name string, root ...string) ([]string, error) {
		walks++
		return original(name, root...)
	}
	t.Cleanup(func() { listMatchingFiles = original })

	first := newProjectWithResponse(t, "first-token")
	enterDir(t, first)

	if got := GetValueOf("access_token", "auth"); got != "first-token" {
		t.Fatalf("first project: got %v, want first-token", got)
	}
	if got := GetValueOf("access_token", "auth"); got != "first-token" {
		t.Fatalf("first project, repeat: got %v, want first-token", got)
	}
	if walks != 1 {
		t.Errorf("walked %d times for one name, want 1", walks)
	}

	second := newProjectWithResponse(t, "second-token")
	enterDir(t, second)

	if got := GetValueOf("access_token", "auth"); got != "second-token" {
		t.Errorf("second project: got %v, want second-token — the cache key ignores the root", got)
	}
	if walks != 2 {
		t.Errorf("walked %d times across two projects, want 2", walks)
	}
}

func TestRepeatedDiagnosticPrintedOnce(t *testing.T) {
	root := setupHulakProject(t)
	t.Cleanup(ResetCache)
	ResetCache()

	missing := filepath.Join(root, "never-written"+utils.ResponseFileName)

	out := captureStderr(t, func() {
		for range 5 {
			if got := GetValueOf("access_token", missing); got != "" {
				t.Errorf("got %v, want empty for a missing file", got)
			}
		}
	})
	if n := strings.Count(out, "does not exist"); n != 1 {
		t.Errorf("printed the missing-file error %d times, want 1:\n%s", n, out)
	}

	// A fresh tool call should report the same condition again.
	again := captureStderr(t, func() {
		ResetCache()
		GetValueOf("access_token", missing)
	})
	if n := strings.Count(again, "does not exist"); n != 1 {
		t.Errorf("after ResetCache printed %d times, want 1:\n%s", n, again)
	}
}

func TestAmbiguousNameWarnsOncePerName(t *testing.T) {
	root := setupHulakProject(t)
	t.Cleanup(ResetCache)
	ResetCache()

	// Two request files share the stem "auth", in sibling directories.
	for dir, token := range map[string]string{"a-first": "first-token", "z-second": "second-token"} {
		if err := os.Mkdir(filepath.Join(root, dir), utils.DirPer); err != nil {
			t.Fatalf("failed to create %s: %v", dir, err)
		}
		request := filepath.Join(root, dir, "auth"+utils.ProjectExt+utils.YAML)
		if err := os.WriteFile(request, []byte("method: GET\nurl: http://example.com\n"), utils.FilePer); err != nil {
			t.Fatalf("failed to write request file: %v", err)
		}
		body := `{"access_token": "` + token + `", "token_type": "Bearer"}`
		response := filepath.Join(root, dir, "auth"+utils.ProjectExt+utils.ResponseFileName)
		if err := os.WriteFile(response, []byte(body), utils.FilePer); err != nil {
			t.Fatalf("failed to write response file: %v", err)
		}
	}

	// Drop the resolved path between lookups so the warning site is reached
	// twice. Otherwise the path cache holds the count at one by itself and the
	// dedupe is never asked to do anything.
	var token, kind any
	out := captureStderr(t, func() {
		token = GetValueOf("access_token", "auth")
		pathCacheMu.Lock()
		clear(pathCache)
		pathCacheMu.Unlock()
		kind = GetValueOf("token_type", "auth")
	})

	if n := strings.Count(out, "multiple 'auth' files"); n != 1 {
		t.Errorf("warned %d times for one ambiguous name, want 1:\n%s", n, out)
	}
	if token != "first-token" {
		t.Errorf("access_token = %v, want first-token", token)
	}
	if kind != "Bearer" {
		t.Errorf("token_type = %v, want Bearer", kind)
	}
}

// Every diagnostic that names a file abbreviates the path, so two
// same-named response files in different collections render identically and
// deduping on the text alone would report one and swallow the other.
func TestSamePrintedPathStillReportsBothFiles(t *testing.T) {
	tests := []struct {
		name  string
		setUp func(t *testing.T, path string)
		want  string
	}{
		{
			name: "missing key",
			setUp: func(t *testing.T, path string) {
				t.Helper()
				writeResponse(t, path, `{"other_key": "x"}`)
			},
			want: "looking up value 'access_token'",
		},
		{
			name: "malformed json",
			setUp: func(t *testing.T, path string) {
				t.Helper()
				writeResponse(t, path, `{"access_token":`)
			},
			want: "has proper json content",
		},
		{
			name: "unreadable file",
			setUp: func(t *testing.T, path string) {
				t.Helper()
				if err := os.Mkdir(path, utils.DirPer); err != nil {
					t.Fatalf("failed to create directory: %v", err)
				}
			},
			want: "error occurred while reading the file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := setupHulakProject(t)
			t.Cleanup(ResetCache)
			ResetCache()

			var paths []string
			for _, collection := range []string{"graphql", "rest"} {
				dir := filepath.Join(root, collection, "api")
				if err := os.MkdirAll(dir, utils.DirPer); err != nil {
					t.Fatalf("failed to create %s: %v", dir, err)
				}
				path := filepath.Join(dir, "auth"+utils.ResponseFileName)
				tt.setUp(t, path)
				paths = append(paths, path)
			}

			out := captureStderr(t, func() {
				for _, path := range paths {
					GetValueOf("access_token", path)
				}
			})

			if n := strings.Count(out, tt.want); n != 2 {
				t.Errorf("reported %d of 2 files, want 2:\n%s", n, out)
			}
		})
	}
}

func writeResponse(t *testing.T, path, body string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(body), utils.FilePer); err != nil {
		t.Fatalf("failed to write response file: %v", err)
	}
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = w
	t.Cleanup(func() { os.Stderr = orig })

	done := make(chan string, 1)
	go func() {
		var buf strings.Builder
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()
	_ = w.Close()
	os.Stderr = orig
	return <-done
}

func newProjectWithResponse(t *testing.T, token string) string {
	t.Helper()

	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("failed to resolve symlinks: %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, utils.EnvironmentFolder), utils.DirPer); err != nil {
		t.Fatalf("failed to create env dir: %v", err)
	}
	request := filepath.Join(root, "auth"+utils.ProjectExt+utils.YAML)
	if err := os.WriteFile(request, []byte("method: GET\nurl: http://example.com\n"), utils.FilePer); err != nil {
		t.Fatalf("failed to write request file: %v", err)
	}
	body := `{"access_token": "` + token + `"}`
	response := filepath.Join(root, "auth"+utils.ProjectExt+utils.ResponseFileName)
	if err := os.WriteFile(response, []byte(body), utils.FilePer); err != nil {
		t.Fatalf("failed to write response file: %v", err)
	}
	return root
}

func enterDir(t *testing.T, dir string) {
	t.Helper()

	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir to %s: %v", dir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(prev); err != nil {
			t.Fatal(err)
		}
	})
}

// The resolved path is the one thing a result cannot check for itself, so it is
// the only thing ResetCache is still needed for.
func TestResetCache_DropsTheResolvedPath(t *testing.T) {
	root := setupHulakProject(t)
	t.Cleanup(ResetCache)
	ResetCache()

	writePair := func(dir, token string) {
		t.Helper()
		if dir != "" {
			if err := os.MkdirAll(filepath.Join(root, dir), utils.DirPer); err != nil {
				t.Fatalf("failed to create %s: %v", dir, err)
			}
		}
		request := filepath.Join(root, dir, "auth"+utils.ProjectExt+utils.YAML)
		if err := os.WriteFile(request, []byte("method: GET\nurl: http://example.com\n"), utils.FilePer); err != nil {
			t.Fatalf("failed to write request file: %v", err)
		}
		response := filepath.Join(root, dir, "auth"+utils.ProjectExt+utils.ResponseFileName)
		if err := os.WriteFile(response, []byte(`{"access_token": "`+token+`"}`), utils.FilePer); err != nil {
			t.Fatalf("failed to write response file: %v", err)
		}
	}
	removePair := func(dir string) {
		t.Helper()
		for _, name := range []string{
			"auth" + utils.ProjectExt + utils.YAML,
			"auth" + utils.ProjectExt + utils.ResponseFileName,
		} {
			if err := os.Remove(filepath.Join(root, dir, name)); err != nil {
				t.Fatalf("failed to remove %s: %v", name, err)
			}
		}
	}

	writePair("", "at-the-root")
	if got := GetValueOf("access_token", "auth"); got != "at-the-root" {
		t.Fatalf("first read: got %v, want at-the-root", got)
	}

	removePair("")
	writePair("collection", "in-the-subdirectory")

	ResetCache()
	if got := GetValueOf("access_token", "auth"); got != "in-the-subdirectory" {
		t.Errorf("after ResetCache: got %v, want in-the-subdirectory", got)
	}
}

// Every failure returns "", so the stderr line is the only thing that tells the
// user which one happened.
func TestProcessValueOf_ReportsEachFailure(t *testing.T) {
	root := setupHulakProject(t)
	t.Cleanup(ResetCache)

	good := filepath.Join(root, "good.json")
	if err := os.WriteFile(good, []byte(`{"present": "yes"}`), utils.FilePer); err != nil {
		t.Fatalf("failed to write json file: %v", err)
	}
	malformed := filepath.Join(root, "malformed.json")
	if err := os.WriteFile(malformed, []byte(`{"broken":`), utils.FilePer); err != nil {
		t.Fatalf("failed to write malformed file: %v", err)
	}
	// A directory fails the read with something other than "does not exist".
	directory := filepath.Join(root, "a-directory.json")
	if err := os.Mkdir(directory, utils.DirPer); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	tests := []struct {
		name     string
		key      string
		fileName string
		want     string
	}{
		{"no key", "", good, "provide key for getValueOf action"},
		{"no file", "present", "", "provide fileName/path to key for getValueOf action"},
		{"missing file", "present", filepath.Join(root, "absent.json"), "does not exist"},
		{"unreadable file", "present", directory, "error occurred while reading the file"},
		{"malformed json", "present", malformed, "has proper json content"},
		{"absent key", "nope", good, "looking up value 'nope'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetCache()

			var got any
			out := captureStderr(t, func() { got = GetValueOf(tt.key, tt.fileName) })

			if got != "" {
				t.Errorf("returned %v, want empty", got)
			}
			if !strings.Contains(out, tt.want) {
				t.Errorf("stderr = %q, want it to contain %q", out, tt.want)
			}
		})
	}
}

func TestResolveJSONFilePath_BareNameEndingInJSON(t *testing.T) {
	root := setupHulakProject(t)
	t.Cleanup(ResetCache)
	ResetCache()

	want := filepath.Join(root, "seeded_response.json")
	if err := os.WriteFile(want, []byte(`{"access_token": "seeded"}`), utils.FilePer); err != nil {
		t.Fatalf("failed to write response file: %v", err)
	}

	got, err := resolveJSONFilePath("seeded_response.json")
	if err != nil {
		t.Fatalf("resolveJSONFilePath: %v", err)
	}
	if got != want {
		t.Errorf("resolved to %q, want %q", got, want)
	}
	if v := GetValueOf("access_token", "seeded_response.json"); v != "seeded" {
		t.Errorf("GetValueOf = %v, want seeded", v)
	}
}
