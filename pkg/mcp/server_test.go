package mcp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// projectDir returns a temp dir marked as a hulak project (has env/).
func projectDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "env"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func writeReq(t *testing.T, dir, name string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("kind: API\nmethod: GET\nurl: http://x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestNewServer_Validation(t *testing.T) {
	t.Run("empty projects", func(t *testing.T) {
		if _, err := NewServer(map[string]string{}, "", "v"); err == nil {
			t.Error("expected error for empty projects")
		}
	})
	t.Run("rejects non-project dir", func(t *testing.T) {
		_, err := NewServer(map[string]string{"api": t.TempDir()}, "", "v")
		if err == nil {
			t.Fatal("expected error for a dir that is not a hulak project")
		}
		if !strings.Contains(err.Error(), "not a hulak project") {
			t.Errorf("error should say it is not a hulak project, got: %v", err)
		}
	})
	t.Run("default not a project name", func(t *testing.T) {
		_, err := NewServer(map[string]string{"api": projectDir(t)}, "mobile", "v")
		if err == nil {
			t.Fatal("expected error when default-project is not a configured name")
		}
		if !strings.Contains(err.Error(), "default-project") {
			t.Errorf("error should reference --default-project, got: %v", err)
		}
	})
}

func TestResolveRequest(t *testing.T) {
	api := projectDir(t)
	mobile := projectDir(t)
	writeReq(t, api, "getUser.hk.yaml")   // unique to api
	writeReq(t, api, "login.hk.yaml")     // in both
	writeReq(t, mobile, "login.hk.yaml")  // in both
	writeReq(t, mobile, "signup.hk.yaml") // unique to mobile

	s, err := NewServer(map[string]string{"api": api, "mobile": mobile}, "api", "v")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("unique name resolves without project", func(t *testing.T) {
		m, err := s.ResolveRequest("", "getUser")
		if err != nil {
			t.Fatal(err)
		}
		if m.Project != "api" {
			t.Errorf("project = %q, want api", m.Project)
		}
	})

	t.Run("explicit project", func(t *testing.T) {
		m, err := s.ResolveRequest("mobile", "login")
		if err != nil {
			t.Fatal(err)
		}
		if m.Project != "mobile" {
			t.Errorf("project = %q, want mobile", m.Project)
		}
	})

	t.Run("ambiguous name always asks (even with default)", func(t *testing.T) {
		_, err := s.ResolveRequest("", "login")
		if err == nil {
			t.Fatal("expected ambiguity error")
		}
		msg := err.Error()
		if !strings.Contains(msg, "multiple projects") ||
			!strings.Contains(msg, "api") || !strings.Contains(msg, "mobile") {
			t.Errorf("ambiguity error should list both projects, got: %v", err)
		}
	})

	t.Run("unknown project", func(t *testing.T) {
		if _, err := s.ResolveRequest("nope", "login"); err == nil {
			t.Error("expected unknown-project error")
		}
	})

	t.Run("not found", func(t *testing.T) {
		if _, err := s.ResolveRequest("", "missing"); err == nil {
			t.Error("expected not-found error")
		}
	})
}

// TestNewServer_HandshakeNoTools verifies the bare server completes the MCP
// handshake and advertises zero tools (nothing registered yet).
func TestNewServer_HandshakeNoTools(t *testing.T) {
	ctx := context.Background()
	serverT, clientT := mcpsdk.NewInMemoryTransports()

	s, err := NewServer(map[string]string{"api": projectDir(t)}, "", "test")
	if err != nil {
		t.Fatal(err)
	}
	ss, err := s.srv.Connect(ctx, serverT, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	defer ss.Close()

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-client", Version: "0"}, nil)
	cs, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer cs.Close()

	res, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	if len(res.Tools) != 0 {
		t.Errorf("expected 0 tools, got %d", len(res.Tools))
	}
}
