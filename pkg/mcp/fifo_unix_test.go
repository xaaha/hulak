//go:build unix

package mcp

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestFIFOs_260_DoNotBlockTheServer(t *testing.T) {
	api := evalSymlinks(t, projectDir(t))
	writeReq(t, api, "ok.hk.yaml")
	writeFileAt(t, filepath.Join(api, "dep.hk.yaml"),
		"method: POST\nurl: http://x\nbody:\n  raw: '{{getFile \"pipe.txt\"}}'\n")
	for _, name := range []string{"pipe.txt", "fifo.hk.yaml"} {
		if err := syscall.Mkfifo(filepath.Join(api, name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	s, err := NewServer(map[string]string{"api": api}, "v")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	within := func(t *testing.T, call func() error) error {
		t.Helper()
		done := make(chan error, 1)
		go func() { done <- call() }()
		select {
		case err := <-done:
			return err
		case <-time.After(5 * time.Second):
			unblockFIFOs(t, filepath.Join(api, "pipe.txt"), filepath.Join(api, "fifo.hk.yaml"))
			<-done
			t.Fatal("the call blocked on a FIFO")
			return nil
		}
	}

	t.Run("list_requests skips a FIFO request and does not read a FIFO dependency", func(t *testing.T) {
		var out listRequestsOutput
		err := within(t, func() error {
			var err error
			_, out, err = s.handleListRequests(ctx, nil, listRequestsInput{Detail: true})
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range allRequests(out) {
			if r.Name == "fifo" {
				t.Error("list_requests listed a FIFO as a request")
			}
		}
	})

	t.Run("read_request refuses a FIFO request file", func(t *testing.T) {
		err := within(t, func() error {
			_, _, err := s.handleReadRequest(ctx, nil, readRequestInput{Name: "fifo"})
			return err
		})
		if err == nil {
			t.Error("read_request read a FIFO request file")
		}
	})

	t.Run("read_request lists a FIFO dependency as missing", func(t *testing.T) {
		var out readRequestOutput
		err := within(t, func() error {
			var err error
			_, out, err = s.handleReadRequest(ctx, nil, readRequestInput{Name: "dep"})
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(out.Deps) != 1 || !out.Deps[0].Missing {
			t.Errorf("deps = %+v, want pipe.txt missing", out.Deps)
		}
	})
}

// unblockFIFOs opens each FIFO for writing so a blocked reader gets EOF and releases the server lock.
func unblockFIFOs(t *testing.T, paths ...string) {
	t.Helper()
	for _, path := range paths {
		f, err := os.OpenFile(path, os.O_RDWR, 0)
		if err != nil {
			t.Log(err)
			continue
		}
		_ = f.Close()
	}
}

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
