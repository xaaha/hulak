package apicalls

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestD9_278_SecretsStayOutOfDecodeErrors(t *testing.T) {
	for name, secret := range map[string]string{
		"newline":    "line-one-SECRET\nline-two-SECRET",
		"tab":        "tab\tSECRET-value",
		"quote":      "quo\"SECRET-value",
		"backslash":  "back\\SECRET-value",
		"html":       "lt<amp&SECRET\nvalue",
		"nbsp":       "pw\u00a0SECRET-value",
		"zero width": "pw\u200bSECRET-value",
		"separator":  "pw\u2028SECRET-value",
		"control":    "ctl\x01SECRET-value",
	} {
		t.Run(name, func(t *testing.T) {
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
			_, err := DryRun(RequestOptions{Secrets: map[string]any{"api_token": secret}, Path: path})
			if err == nil {
				t.Fatal("want a decode error for the list header")
			}
			if strings.Contains(err.Error(), "SECRET") {
				t.Errorf("error shows the secret:\n%v", err)
			}
		})
	}
}
