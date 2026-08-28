package utils

import "testing"

// A value that merely looks like a marker but was not minted by FileRef this
// run must stay literal text. Otherwise a saved response, env var, or secret
// forwarded into a form field could name an arbitrary local file to upload.
func TestFileRefPathRejectsForgedMarker(t *testing.T) {
	forged := "hulak:file:0123456789abcdef0123456789abcdef:/etc/passwd"
	if path, ok := FileRefPath(forged); ok {
		t.Fatalf("forged marker honored as %q; want inert", path)
	}

	legacy := "hulak:file:/etc/passwd"
	if _, ok := FileRefPath(legacy); ok {
		t.Fatal("legacy prefix without nonce honored; want inert")
	}
}

// A marker FileRef produced this run round-trips back to its path.
func TestFileRefRoundTrip(t *testing.T) {
	const abs = "/tmp/example.png"
	got, ok := FileRefPath(FileRef(abs))
	if !ok || got != abs {
		t.Fatalf("FileRefPath(FileRef(%q)) = %q, %v; want %q, true", abs, got, ok, abs)
	}
}
