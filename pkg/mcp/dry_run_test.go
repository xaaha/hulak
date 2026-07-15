package mcp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFileAt(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestHandleDryRun(t *testing.T) {
	api := projectDir(t)
	writeFileAt(t, filepath.Join(api, "env", "staging.env"), "baseUrl=https://api.example.com\n")
	writeFileAt(t, filepath.Join(api, "getUsers.hk.yaml"),
		"kind: API\nmethod: GET\nurl: \"{{.baseUrl}}/users\"\n")

	s, err := NewServer(map[string]string{"api": api}, "api", "v")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	t.Run("resolves url against env", func(t *testing.T) {
		_, out, err := s.handleDryRun(ctx, nil, dryRunInput{Name: "getUsers", Env: "staging"})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.Request, "GET https://api.example.com/users") {
			t.Errorf("request should show resolved URL, got:\n%s", out.Request)
		}
		if out.Project != "api" {
			t.Errorf("project = %q, want api", out.Project)
		}
	})

	t.Run("env is required", func(t *testing.T) {
		if _, _, err := s.handleDryRun(ctx, nil, dryRunInput{Name: "getUsers"}); err == nil {
			t.Error("expected error when env is missing")
		}
	})

	t.Run("unknown request errors", func(t *testing.T) {
		if _, _, err := s.handleDryRun(ctx, nil, dryRunInput{Name: "missing", Env: "staging"}); err == nil {
			t.Error("expected error for unknown request")
		}
	})
}
