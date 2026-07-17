package gqlexplorer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xaaha/hulak/pkg/utils"
)

// TestSaveResponse_OwnerOnlyPerms guards the fix for issue #166: an explorer
// response body can echo tokens/PII, so it must be written owner-only (0o600),
// not with the world-readable FilePer used for shareable source files.
func TestSaveResponse_OwnerOnlyPerms(t *testing.T) {
	dir := t.TempDir()
	m := &Model{
		responseBody:    `{"access_token":"super-secret"}`,
		filtered:        []UnifiedOperation{{Name: "getUser", Endpoint: "e"}},
		cursor:          0,
		schemaFilePaths: map[string]string{"e": filepath.Join(dir, "schema.json")},
	}

	if cmd := m.saveResponse(); cmd == nil {
		t.Fatal("saveResponse returned nil cmd; expected a save notification")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var saved string
	for _, e := range entries {
		if !e.IsDir() {
			saved = filepath.Join(dir, e.Name())
		}
	}
	if saved == "" {
		t.Fatal("no response file written")
	}

	info, err := os.Stat(saved)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != utils.SecretPer.Perm() {
		t.Errorf("response file perms = %o, want %o (secret-bearing body must be owner-only)",
			perm, utils.SecretPer.Perm())
	}
}
