package initcmd

import (
	"os"
	"testing"
)

// chdirTemp changes the working directory to dir and returns a cleanup func
// that restores the original cwd. Test-only.
func chdirTemp(t *testing.T, dir string) func() {
	t.Helper()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir to %s: %v", dir, err)
	}
	return func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	}
}
