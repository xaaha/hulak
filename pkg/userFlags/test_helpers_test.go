package userflags

import (
	"bytes"
	"os"
	"testing"
)

// captureStdout redirects os.Stdout to a pipe, runs fn, and returns
// everything that was written to stdout as a string. Used by snapshot_test
// to assert on `cmd.PrintHelp()` output.
//
// Each subpackage that needs the same helper carries its own copy. Twelve
// lines twice is cheaper than a shared testutil package that every test
// binary would have to import.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create pipe: %v", err)
	}
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("could not read from pipe: %v", err)
	}
	return buf.String()
}
