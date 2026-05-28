// Package testutil holds tiny test-only helpers shared across packages.
// Imported only from *_test.go files; nothing in production code should
// depend on this package.
package testutil

import (
	"bytes"
	"io"
	"os"
	"testing"
)

// CaptureStdout swaps os.Stdout for a pipe, runs fn, and returns everything
// fn wrote. Stdout is restored even if fn panics. Used by snapshot and help-
// rendering tests that assert on printed output.
func CaptureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	fn()
	_ = w.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("io.Copy: %v", err)
	}
	return buf.String()
}
