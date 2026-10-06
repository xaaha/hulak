package mcp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xaaha/hulak/pkg/utils/testutil"
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

	s, err := NewServer(map[string]string{"api": api}, "v")
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

// The show argument has to reach the formatter, not just the tool schema.
func TestD1_3_DryRunShowReachesTheFormatter(t *testing.T) {
	const secret = "super-secret-client-value"
	api := projectDir(t)
	writeFileAt(t, filepath.Join(api, "env", "staging.env"),
		"baseUrl=https://api.example.com\nclient_secret="+secret+"\n")
	writeFileAt(t, filepath.Join(api, "token.hk.yaml"),
		"kind: API\nmethod: GET\nurl: \"{{.baseUrl}}/token\"\n"+
			"urlparams:\n  client_secret: \"{{.client_secret}}\"\n"+
			"headers:\n  Authorization: \"Bearer {{.client_secret}}\"\n")

	s, err := NewServer(map[string]string{"api": api}, "v")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	_, masked, err := s.handleDryRun(ctx, nil, dryRunInput{Name: "token", Env: "staging"})
	if err != nil {
		t.Fatal(err)
	}
	testutil.AssertNoSecretForm(t, "dry_run without show", masked.Request, secret)

	_, shown, err := s.handleDryRun(ctx, nil, dryRunInput{Name: "token", Env: "staging", Show: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(shown.Request, secret) {
		t.Errorf("show must reveal the resolved secret:\n%s", shown.Request)
	}
	if !strings.Contains(shown.Request, "authorization: Bearer "+secret) {
		t.Errorf("show must reveal the sensitive header:\n%s", shown.Request)
	}
}
