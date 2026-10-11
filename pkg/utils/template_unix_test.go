//go:build unix

package utils

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
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

func TestReferencedFiles_260_DoesNotReadAFIFO(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, EnvironmentFolder), DirPer); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(filepath.Join(root, "pipe.gql"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "pipe.gql"), filepath.Join(root, "alias.gql")); err != nil {
		t.Fatal(err)
	}
	req := filepath.Join(root, "q.hk.yaml")
	writeFile(t, req, buildGetFileContent("pipe.gql")+buildGetFileContent("alias.gql"))
	t.Chdir(root)

	done := make(chan error, 1)
	go func() {
		_, err := ReferencedFiles(req)
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("ReferencedFiles blocked reading a FIFO")
	}
	if _, err := ResolveProjectFile("alias.gql"); err == nil {
		t.Error("ResolveProjectFile accepted a symlink to a FIFO")
	}
}
