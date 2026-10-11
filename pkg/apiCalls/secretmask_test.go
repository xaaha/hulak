package apicalls

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestD9_278_MultiLineSecretStaysMaskedInDecodeErrors(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.Mkdir(filepath.Join(dir, "env"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "r.hk.yaml")
	content := "method: GET\nurl: https://e.com\nheaders:\n  X-K: \"{{.api_token}}\"\n  X-L: [a]\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := DryRun(RequestOptions{Secrets: map[string]any{"api_token": "line-one-secret\nline-two-secret"}, Path: path})
	if err == nil {
		t.Fatal("want a decode error for the list header")
	}
	if strings.Contains(err.Error(), "line-one-secret") || strings.Contains(err.Error(), "line-two-secret") {
		t.Errorf("error shows the secret:\n%v", err)
	}
}
