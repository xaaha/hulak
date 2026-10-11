//go:build unix

package utils

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestResolveProjectFile_260_RefusesNonRegularFiles(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, EnvironmentFolder), DirPer); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(filepath.Join(root, "pipe.json"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	if _, err := ResolveProjectFile("pipe.json"); err == nil {
		t.Error("ResolveProjectFile accepted a FIFO")
	}
}
