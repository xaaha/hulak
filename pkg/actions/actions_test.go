package actions

import (
	"encoding/base64"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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

// D1: a parsed response file is reused only while the file on disk still
// matches it. Both halves matter — memoizing the value is the #253 bug, and
// re-reading unconditionally is what the cache exists to avoid.
func TestD1_ParsedFileReusedUntilRewritten(t *testing.T) {
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
	first, err := readJSONFile(path)
	if err != nil {
		t.Fatalf("first read: %v", err)
	}
	second, err := readJSONFile(path)
	if err != nil {
		t.Fatalf("second read: %v", err)
	}
	if !sameParsedTree(first, second) {
		t.Error("unchanged file was parsed twice — the content cache is not being used")
	}

	write("a-longer-fresh-token")
	if got := GetValueOf("access_token", path); got != "a-longer-fresh-token" {
		t.Errorf("after rewrite: got %v, want a-longer-fresh-token", got)
	}
}

// D2: the freshness check includes file identity, so a rewrite that lands the
// same byte count under the same timestamp is still seen. Response files are
// renamed into place, so identity is the one signal that always moves.
func TestD2_RenamedFileSeenDespiteMatchingSizeAndMtime(t *testing.T) {
	root := setupHulakProject(t)
	t.Cleanup(ResetCache)
	ResetCache()

	path := filepath.Join(root, "auth"+utils.ResponseFileName)
	if err := os.WriteFile(path, []byte(`{"access_token": "aaa"}`), utils.FilePer); err != nil {
		t.Fatalf("failed to write response file: %v", err)
	}
	original, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}

	if got := GetValueOf("access_token", path); got != "aaa" {
		t.Fatalf("first read: got %v, want aaa", got)
	}

	// Same byte count, restored timestamp: size and mtime alone cannot tell
	// this apart from the file already parsed.
	tmp := filepath.Join(root, "replacement.json")
	if err := os.WriteFile(tmp, []byte(`{"access_token": "bbb"}`), utils.FilePer); err != nil {
		t.Fatalf("failed to write replacement: %v", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		t.Fatalf("rename: %v", err)
	}
	if err := os.Chtimes(path, original.ModTime(), original.ModTime()); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	refreshed, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat after rename: %v", err)
	}
	if refreshed.Size() != original.Size() || !refreshed.ModTime().Equal(original.ModTime()) {
		t.Fatalf("setup failed to match size and mtime: %d/%v vs %d/%v",
			refreshed.Size(), refreshed.ModTime(), original.Size(), original.ModTime())
	}

	if got := GetValueOf("access_token", path); got != "bbb" {
		t.Errorf("after rename: got %v, want bbb", got)
	}
}

// D3: a bare filename is walked for once and the resolved path reused, and the
// cache is keyed by the root the walk started from so the same name in another
// project resolves to that project's file.
func TestD3_BareNameWalkedOncePerRoot(t *testing.T) {
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

// D4: an identical getValueOf diagnostic reaches stderr once, not once per
// resolution. Without the value cache absorbing repeat failures, a response
// file that does not exist yet is the common case at the start of a run.
func TestD4_RepeatedDiagnosticPrintedOnce(t *testing.T) {
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

	// ResetCache re-arms it: a fresh tool call should say so again.
	again := captureStderr(t, func() {
		ResetCache()
		GetValueOf("access_token", missing)
	})
	if n := strings.Count(again, "does not exist"); n != 1 {
		t.Errorf("after ResetCache printed %d times, want 1:\n%s", n, again)
	}
}

// D5: when a bare name matches more than one request file the warning still
// fires, and fires once for that name however many keys are read out of it.
// The resolved file is the same every time, which is the point of warning.
func TestD5_AmbiguousNameWarnsOncePerName(t *testing.T) {
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

	var token, kind any
	out := captureStderr(t, func() {
		token = GetValueOf("access_token", "auth")
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

// captureStderr swaps os.Stderr for a pipe, runs fn, restores os.Stderr,
// and returns whatever fn wrote to stderr.
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

// newProjectWithResponse builds a hulak project in its own temp dir holding
// auth.hk_response.json with the given token, and returns the project root.
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

// enterDir chdirs into dir for the rest of the test, restoring the previous
// working directory afterwards.
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

// sameParsedTree reports whether two parsed JSON documents are the same
// in-memory map, which is how a cache hit is distinguished from a re-parse
// that happens to produce an equal value.
func sameParsedTree(a, b any) bool {
	av, bv := reflect.ValueOf(a), reflect.ValueOf(b)
	if av.Kind() != reflect.Map || bv.Kind() != reflect.Map {
		return false
	}
	return av.Pointer() == bv.Pointer()
}

// ResetCache drops parsed response files wholesale. Entries already invalidate
// themselves against the file on disk; this is for the staleness a stat can't
// see, which is why the MCP server calls it between tool calls.
func TestResetCache(t *testing.T) {
	setupHulakProject(t)

	const authDir = "auth"
	if err := os.Mkdir(authDir, utils.DirPer); err != nil {
		t.Fatalf("failed to create auth dir: %v", err)
	}
	relPath := filepath.Join(authDir, "getAuth"+utils.ResponseFileName)

	write := func(token string) {
		t.Helper()
		body := `{"access_token": "` + token + `"}`
		if err := os.WriteFile(relPath, []byte(body), utils.FilePer); err != nil {
			t.Fatalf("failed to write response file: %v", err)
		}
	}

	t.Cleanup(ResetCache)

	ResetCache()
	write("stale-token")
	if got := GetValueOf("access_token", relPath); got != "stale-token" {
		t.Fatalf("first read: got %v, want stale-token", got)
	}

	write("fresh-token")

	ResetCache()
	if got := GetValueOf("access_token", relPath); got != "fresh-token" {
		t.Errorf("after ResetCache: got %v, want fresh-token", got)
	}
}
