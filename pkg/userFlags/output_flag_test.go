package userflags

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xaaha/hulak/pkg/utils"
)

func TestResolveOutputPath(t *testing.T) {
	dir := t.TempDir()
	existingSubdir := filepath.Join(dir, "exists")
	if err := os.Mkdir(existingSubdir, utils.DirPer); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	cases := []struct {
		name      string
		outPath   string
		canonical string
		knownExts []string
		want      string
	}{
		{
			name:      "trailing slash → dir mode",
			outPath:   "requests/",
			canonical: "example.hk.yaml",
			knownExts: []string{".yaml", ".yml"},
			want:      filepath.Join("requests", "example.hk.yaml"),
		},
		{
			name:      "existing dir → dir mode",
			outPath:   existingSubdir,
			canonical: "example.hk.yaml",
			knownExts: []string{".yaml"},
			want:      filepath.Join(existingSubdir, "example.hk.yaml"),
		},
		{
			name:      "matches known ext → file mode (verbatim)",
			outPath:   "sub/health.hk.yaml",
			canonical: "example.hk.yaml",
			knownExts: []string{".yaml", ".yml"},
			want:      filepath.Join("sub", "health.hk.yaml"),
		},
		{
			name:      "no extension, restricted exts → DWIM dir mode",
			outPath:   "dir/path",
			canonical: "example.hk.yaml",
			knownExts: []string{".yaml"},
			want:      filepath.Join("dir", "path", "example.hk.yaml"),
		},
		{
			name:      "any-ext mode when knownExts empty → .txt is file mode",
			outPath:   "secrets/key.txt",
			canonical: "identity.txt",
			knownExts: nil,
			want:      filepath.Join("secrets", "key.txt"),
		},
		{
			name:      "any-ext mode, no extension → DWIM dir mode",
			outPath:   "secrets-dir",
			canonical: "identity.txt",
			knownExts: nil,
			want:      filepath.Join("secrets-dir", "identity.txt"),
		},
		{
			name:      "extension mismatch → DWIM dir mode",
			outPath:   "weird.unknown",
			canonical: "store.age.bak",
			knownExts: []string{".age"},
			want:      filepath.Join("weird.unknown", "store.age.bak"),
		},
		{
			name:      "knownExts accept form with or without leading dot",
			outPath:   "out.age",
			canonical: "store.age.bak",
			knownExts: []string{"age"},
			want:      "out.age",
		},
		{
			name:      "uppercase extension matches case-insensitively",
			outPath:   "Out.YAML",
			canonical: "example.hk.yaml",
			knownExts: []string{".yaml"},
			want:      "Out.YAML",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveOutputPath(tc.outPath, tc.canonical, tc.knownExts...)
			if err != nil {
				t.Fatalf("resolveOutputPath: %v", err)
			}
			if got != tc.want {
				t.Errorf("resolveOutputPath(%q, %q, %v) = %q, want %q",
					tc.outPath, tc.canonical, tc.knownExts, got, tc.want)
			}
		})
	}
}

func TestResolveOutputPath_NormalizesCwd(t *testing.T) {
	// "", ".", "./" all mean "use cwd" and should produce the same absolute
	// path. CreatePath joins against cwd, so we can't predict the prefix —
	// just verify all three agree and the basename matches.
	got1, err := resolveOutputPath("", "example.hk.yaml", ".yaml")
	if err != nil {
		t.Fatalf("empty: %v", err)
	}
	got2, err := resolveOutputPath(".", "example.hk.yaml", ".yaml")
	if err != nil {
		t.Fatalf(".: %v", err)
	}
	got3, err := resolveOutputPath("./", "example.hk.yaml", ".yaml")
	if err != nil {
		t.Fatalf("./: %v", err)
	}
	if got1 != got2 || got2 != got3 {
		t.Errorf("cwd normalization disagrees: %q / %q / %q", got1, got2, got3)
	}
	if filepath.Base(got1) != "example.hk.yaml" {
		t.Errorf("expected basename example.hk.yaml, got %q", got1)
	}
	if !filepath.IsAbs(got1) {
		t.Errorf("CreatePath should return absolute, got %q", got1)
	}
}
