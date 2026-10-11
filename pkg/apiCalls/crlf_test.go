package apicalls

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestD14_278_CRLFCommentsKeepTheLineInRunErrors(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.Mkdir(filepath.Join(dir, "env"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "r.hk.yaml")
	if err := os.WriteFile(path, []byte("kind: API # c\r\nmethod: GET # c\r\nurl: http://x\r\nurl: http://y\r\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := DryRun(RequestOptions{Secrets: map[string]any{}, Path: path})
	if err == nil || !strings.Contains(err.Error(), ": [4:1] ") {
		t.Errorf("err = %v, want the repeated key on line 4", err)
	}
}
