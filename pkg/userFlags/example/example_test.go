package example

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xaaha/hulak/pkg/utils"
)

// TestScaffoldExample_AllTypes verifies that every advertised type writes
// the expected filename and that the on-disk content matches the embedded
// asset byte-for-byte. Aliases share their canonical's expected filename.
func TestScaffoldExample_AllTypes(t *testing.T) {
	cases := []struct {
		arg      string
		wantFile string
	}{
		{"api", "example-api.hk.yaml"},
		{"formdata", "example-formdata.hk.yaml"},
		{"urlencoded", "example-urlencoded.hk.yaml"},
		{"urlencodedformdata", "example-urlencoded.hk.yaml"}, // alias → urlencoded
		{"graphql", "example-graphql.hk.yaml"},
		{"gql", "example-graphql.hk.yaml"}, // alias → graphql
		{"auth", "example-auth.hk.yaml"},
		{"options", utils.OptionsReference},
	}

	for _, tc := range cases {
		t.Run(tc.arg, func(t *testing.T) {
			dir := t.TempDir()
			t.Cleanup(chdirTemp(t, dir))

			if err := scaffoldExample(tc.arg, ""); err != nil {
				t.Fatalf("scaffoldExample(%q): %v", tc.arg, err)
			}

			dest := filepath.Join(dir, tc.wantFile)
			got, err := os.ReadFile(dest)
			if err != nil {
				t.Fatalf("read %s: %v", tc.wantFile, err)
			}

			want, err := embeddedExamples.ReadFile("examples/" + tc.wantFile)
			if err != nil {
				t.Fatalf("read embedded examples/%s: %v", tc.wantFile, err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("scaffolded %s differs from embedded asset", tc.wantFile)
			}
		})
	}
}

// TestScaffoldExample_CaseInsensitive verifies that the type arg is matched
// case-insensitively — `API` and `Api` should resolve to `api`.
func TestScaffoldExample_CaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	t.Cleanup(chdirTemp(t, dir))

	if err := scaffoldExample("API", ""); err != nil {
		t.Fatalf("scaffoldExample(API): %v", err)
	}
	if !utils.FileExists(filepath.Join(dir, "example-api.hk.yaml")) {
		t.Error("expected example-api.hk.yaml after `API` arg")
	}
}

// TestScaffoldExample_Idempotent verifies that re-running for the same type
// in a directory where the file already exists does NOT clobber edits.
func TestScaffoldExample_Idempotent(t *testing.T) {
	dir := t.TempDir()
	t.Cleanup(chdirTemp(t, dir))

	if err := scaffoldExample("api", ""); err != nil {
		t.Fatalf("first scaffold: %v", err)
	}

	dest := filepath.Join(dir, "example-api.hk.yaml")
	custom := []byte("# user-edited\nmethod: GET\nurl: http://localhost/\n")
	if err := os.WriteFile(dest, custom, utils.FilePer); err != nil {
		t.Fatalf("simulate user edit: %v", err)
	}

	if err := scaffoldExample("api", ""); err != nil {
		t.Fatalf("second scaffold: %v", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("re-read: %v", err)
	}
	if !bytes.Equal(got, custom) {
		t.Error("scaffoldExample clobbered user-edited file — must be idempotent")
	}
}

// TestScaffoldExample_UnknownType verifies that an unknown type returns a
// helpful error listing the available types, and does not write anything.
func TestScaffoldExample_UnknownType(t *testing.T) {
	dir := t.TempDir()
	t.Cleanup(chdirTemp(t, dir))

	err := scaffoldExample("nope", "")
	if err == nil {
		t.Fatal("expected error for unknown type, got nil")
	}
	if msg := err.Error(); !strings.Contains(msg, "nope") || !strings.Contains(msg, "api") {
		t.Errorf("error should mention the bad arg and list types, got: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("unknown type should not write anything, found: %v", entries)
	}
}

// TestScaffoldExample_OutDirectory verifies that an --out value that names a
// directory (trailing slash or existing dir) writes the canonical filename
// inside it, creating the directory if needed.
func TestScaffoldExample_OutDirectory(t *testing.T) {
	t.Run("trailing slash auto-creates parent", func(t *testing.T) {
		dir := t.TempDir()
		t.Cleanup(chdirTemp(t, dir))

		if err := scaffoldExample("api", "requests/"); err != nil {
			t.Fatalf("scaffoldExample: %v", err)
		}
		want := filepath.Join(dir, "requests", "example-api.hk.yaml")
		if !utils.FileExists(want) {
			t.Errorf("expected %s to exist", want)
		}
	})

	t.Run("existing directory keeps canonical name", func(t *testing.T) {
		dir := t.TempDir()
		t.Cleanup(chdirTemp(t, dir))
		if err := os.Mkdir(filepath.Join(dir, "reqs"), utils.DirPer); err != nil {
			t.Fatalf("mkdir: %v", err)
		}

		if err := scaffoldExample("api", "reqs"); err != nil {
			t.Fatalf("scaffoldExample: %v", err)
		}
		want := filepath.Join(dir, "reqs", "example-api.hk.yaml")
		if !utils.FileExists(want) {
			t.Errorf("expected %s to exist", want)
		}
	})

	t.Run("non-existing path without yaml ext treated as dir", func(t *testing.T) {
		dir := t.TempDir()
		t.Cleanup(chdirTemp(t, dir))

		// No trailing slash, no extension, doesn't exist — DWIM: treat as dir.
		if err := scaffoldExample("api", "dir/path"); err != nil {
			t.Fatalf("scaffoldExample: %v", err)
		}
		want := filepath.Join(dir, "dir", "path", "example-api.hk.yaml")
		if !utils.FileExists(want) {
			t.Errorf("expected %s to exist", want)
		}
		// Must NOT have created a file literally named "dir/path"
		if utils.FileExists(filepath.Join(dir, "dir", "path")) {
			info, _ := os.Stat(filepath.Join(dir, "dir", "path"))
			if info != nil && !info.IsDir() {
				t.Error("created a file named 'path' instead of a directory")
			}
		}
	})
}

// TestScaffoldExample_OutFilePath verifies that an --out value pointing at a
// file path is honored verbatim — caller can both relocate and rename.
func TestScaffoldExample_OutFilePath(t *testing.T) {
	dir := t.TempDir()
	t.Cleanup(chdirTemp(t, dir))

	if err := scaffoldExample("api", "requests/health.hk.yaml"); err != nil {
		t.Fatalf("scaffoldExample: %v", err)
	}
	want := filepath.Join(dir, "requests", "health.hk.yaml")
	if !utils.FileExists(want) {
		t.Errorf("expected %s to exist", want)
	}
	// Canonical name must NOT exist when caller specified a full path
	if utils.FileExists(filepath.Join(dir, "example-api.hk.yaml")) {
		t.Error("canonical filename should not have been written when --out is a file path")
	}
}

// TestExampleType_ReferencesOptionsReference makes sure the `options` arg
// scaffolds the same filename the directory-mode runner skips. If these
// drift, `hulak example options` would create a file that bulk runs try to
// execute as a request.
func TestExampleType_ReferencesOptionsReference(t *testing.T) {
	if exampleType["options"] != utils.OptionsReference {
		t.Errorf(
			"exampleType[options]=%q should equal utils.OptionsReference=%q",
			exampleType["options"], utils.OptionsReference,
		)
	}
}
