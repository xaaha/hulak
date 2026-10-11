//go:build unix

package mcp

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestReadRequest_260_UnreadableSecretsFolderFailsClosed(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads a mode 000 folder")
	}
	root := evalSymlinks(t, projectDir(t))
	locked := filepath.Join(root, "env", "locked")
	writeFileAt(t, filepath.Join(locked, "x.env"), "LOCKED-SECRET")
	if err := os.Link(filepath.Join(locked, "x.env"), filepath.Join(root, "h.json")); err != nil {
		t.Fatal(err)
	}
	writeFileAt(t, filepath.Join(root, "probe.hk.yaml"), "method: POST\nurl: http://x\nbody:\n  raw: '{{getFile \"h.json\"}}'\n")
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	s, err := NewServer(map[string]string{"api": root}, "v")
	if err != nil {
		t.Fatal(err)
	}
	_, out, err := s.handleReadRequest(context.Background(), nil, readRequestInput{Name: "probe"})
	if err == nil {
		t.Errorf("read_request returned %+v, want an error while env/ cannot be fully checked", out.Deps)
	}
}

func TestReadRequest_260_UncheckableSecretsFailClosed(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads a mode 000 folder")
	}
	probe := func(t *testing.T, root string) error {
		t.Helper()
		writeFileAt(t, filepath.Join(root, "probe.hk.yaml"), "method: GET\nurl: http://x\n")
		s, err := NewServer(map[string]string{"api": root}, "v")
		if err != nil {
			t.Fatal(err)
		}
		_, _, err = s.handleReadRequest(context.Background(), nil, readRequestInput{Name: "probe"})
		return err
	}

	t.Run("an env link into an unreadable folder", func(t *testing.T) {
		root := evalSymlinks(t, projectDir(t))
		locked := filepath.Join(t.TempDir(), "locked")
		writeFileAt(t, filepath.Join(locked, "real.env"), "SECRET")
		if err := os.Symlink(filepath.Join(locked, "real.env"), filepath.Join(root, "env", "link.env")); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(locked, 0); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
		if err := probe(t, root); err == nil {
			t.Error("read_request succeeded although env/link.env could not be checked")
		}
	})

	t.Run("an env folder that is a symlink loop", func(t *testing.T) {
		root := evalSymlinks(t, t.TempDir())
		if err := os.Mkdir(filepath.Join(root, ".hulak"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Join(root, "env"), filepath.Join(root, "env")); err != nil {
			t.Fatal(err)
		}
		if err := probe(t, root); err == nil {
			t.Error("read_request succeeded although env/ could not be resolved")
		}
	})
}
