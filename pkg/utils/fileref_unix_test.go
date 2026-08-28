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
