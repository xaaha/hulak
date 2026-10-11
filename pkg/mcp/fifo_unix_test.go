//go:build unix

package mcp

import (
	"context"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/xaaha/hulak/pkg/utils/testutil"
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

	fifos := []string{filepath.Join(api, "pipe.txt"), filepath.Join(api, "fifo.hk.yaml")}
	within := func(t *testing.T, call func() error) error {
		t.Helper()
		return testutil.WithinFIFOTimeout(t, call, fifos...)
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
