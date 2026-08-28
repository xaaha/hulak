//go:build unix

package utils

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// attachFile must accept only regular files. A directory, FIFO, or device node
// would either error late, upload forever (/dev/zero), or block the process
// (a FIFO with no writer), which is worse on the long-lived MCP server.
func TestResolveAttachPathRejectsNonRegular(t *testing.T) {
	dir := t.TempDir()

	if _, err := ResolveAttachPath(dir); err == nil {
		t.Error("directory accepted; want rejected")
	}

	fifo := filepath.Join(dir, "pipe")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Skipf("cannot create fifo: %v", err)
	}
	if _, err := ResolveAttachPath(fifo); err == nil {
		t.Error("fifo accepted; want rejected")
	}

	regular := filepath.Join(dir, "real.txt")
	if err := os.WriteFile(regular, []byte("hi"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveAttachPath(regular); err != nil {
		t.Errorf("regular file rejected: %v", err)
	}
}

// A relative attachFile path that is an in-repo symlink pointing outside the
// project root must be rejected: containment has to survive symlinks, not just
// lexical traversal.
func TestResolveAttachPathRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".hulak"), 0o755); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	secret := filepath.Join(outside, "id_rsa")
	if err := os.WriteFile(secret, []byte("SECRET"), 0o600); err != nil {
		t.Fatal(err)
	}
	escaping := filepath.Join(root, "escape_link")
	if err := os.Symlink(secret, escaping); err != nil {
		t.Fatal(err)
	}
	// A symlink that stays inside the root is fine.
	inRoot := filepath.Join(root, "real.txt")
	if err := os.WriteFile(inRoot, []byte("hi"), 0o600); err != nil {
		t.Fatal(err)
	}
	staying := filepath.Join(root, "stay_link")
	if err := os.Symlink(inRoot, staying); err != nil {
		t.Fatal(err)
	}

	cwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(cwd) }()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	if _, err := ResolveAttachPath("escape_link"); err == nil {
		t.Error("symlink escaping the project root was accepted; want rejected")
	}
	if _, err := ResolveAttachPath("stay_link"); err != nil {
		t.Errorf("in-root symlink rejected: %v", err)
	}
}
