package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TODO: FIX THIS TEST
func BenchmarkListFiles(b *testing.B) {
	root := b.TempDir()

	// build a directory tree: depth=5, width=6
	createBenchmarkTree(b, root, 5, 6)

	for b.Loop() {
		_, err := ListFiles(root)
		if err != nil {
			b.Fatalf("ListFiles error: %v", err)
		}
	}
}

func createBenchmarkTree(b *testing.B, root string, depth, width int) {
	b.Helper()

	var build func(path string, d int)
	build = func(path string, d int) {
		if d == 0 {
			return
		}

		for i := range width {
			// unique directory name
			dir := filepath.Join(path, fmt.Sprintf("dir_%d_depth_%d", i, d))
			if err := os.MkdirAll(dir, 0o755); err != nil {
				b.Fatalf("mkdir failed: %v", err)
			}

			// create unique file names to avoid collisions
			files := []string{
				filepath.Join(dir, "file_"+fmt.Sprintf("%d_%d.yaml", i, d)),
				filepath.Join(dir, "file_"+fmt.Sprintf("%d_%d.yml", i, d)),
				filepath.Join(dir, "file_"+fmt.Sprintf("%d_%d.json", i, d)),
			}

			for _, f := range files {
				if err := os.WriteFile(f, []byte("data"), 0o644); err != nil {
					b.Fatalf("file write failed: %v", err)
				}
			}

			build(dir, d-1)
		}
	}

	build(root, depth)
}
