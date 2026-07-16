package mcp

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestHandleWriteRequest(t *testing.T) {
	ctx := context.Background()
	const body = "kind: API\nmethod: GET\nurl: http://x\n"

	t.Run("creates new file, appends extension", func(t *testing.T) {
		api := projectDir(t)
		s, _ := NewServer(map[string]string{"api": api}, "v")
		_, out, err := s.handleWriteRequest(ctx, nil, writeRequestInput{Name: "login", YamlContent: body})
		if err != nil {
			t.Fatal(err)
		}
		want := filepath.Join(api, "login.hk.yaml")
		if out.Path != want {
			t.Errorf("path = %q, want %q", out.Path, want)
		}
		if out.Overwritten {
			t.Error("Overwritten should be false for a new file")
		}
		got, _ := os.ReadFile(want)
		if string(got) != body {
			t.Errorf("content = %q, want %q", got, body)
		}
	})

	t.Run("refuses to overwrite without flag", func(t *testing.T) {
		api := projectDir(t)
		s, _ := NewServer(map[string]string{"api": api}, "v")
		if _, _, err := s.handleWriteRequest(ctx, nil, writeRequestInput{Name: "x", YamlContent: body}); err != nil {
			t.Fatal(err)
		}
		_, _, err := s.handleWriteRequest(ctx, nil, writeRequestInput{Name: "x", YamlContent: body})
		if err == nil {
			t.Error("expected error overwriting existing file without overwrite=true")
		}
	})

	t.Run("overwrite=true replaces and reports it", func(t *testing.T) {
		api := projectDir(t)
		s, _ := NewServer(map[string]string{"api": api}, "v")
		_, _, _ = s.handleWriteRequest(ctx, nil, writeRequestInput{Name: "x", YamlContent: body})
		_, out, err := s.handleWriteRequest(ctx, nil, writeRequestInput{
			Name: "x", YamlContent: "kind: GraphQL\nurl: http://y\n", Overwrite: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if !out.Overwritten {
			t.Error("Overwritten should be true")
		}
		got, _ := os.ReadFile(out.Path)
		if string(got) != "kind: GraphQL\nurl: http://y\n" {
			t.Errorf("content not replaced, got %q", got)
		}
	})

	t.Run("rejects invalid yaml", func(t *testing.T) {
		api := projectDir(t)
		s, _ := NewServer(map[string]string{"api": api}, "v")
		if _, _, err := s.handleWriteRequest(ctx, nil, writeRequestInput{Name: "bad", YamlContent: "just a scalar"}); err == nil {
			t.Error("expected error for non-mapping YAML")
		}
	})

	t.Run("multiple projects require explicit project", func(t *testing.T) {
		s, _ := NewServer(map[string]string{"api": projectDir(t), "mob": projectDir(t)}, "v")
		if _, _, err := s.handleWriteRequest(ctx, nil, writeRequestInput{Name: "x", YamlContent: body}); err == nil {
			t.Error("expected error when project omitted with multiple projects")
		}
	})

	t.Run("rejects path traversal", func(t *testing.T) {
		api := projectDir(t)
		s, _ := NewServer(map[string]string{"api": api}, "v")
		if _, _, err := s.handleWriteRequest(ctx, nil, writeRequestInput{Name: "../escape", YamlContent: body}); err == nil {
			t.Error("expected error for traversal outside project")
		}
	})
}

func TestHandleWriteRequest_SchemaValidation(t *testing.T) {
	ctx := context.Background()
	schema, err := os.ReadFile("../../assets/schema.json")
	if err != nil {
		t.Fatal(err)
	}

	newServer := func(t *testing.T) *Server {
		t.Helper()
		s, err := NewServer(map[string]string{"api": projectDir(t)}, "v")
		if err != nil {
			t.Fatal(err)
		}
		s.SetRequestSchema(schema)
		return s
	}

	t.Run("accepts valid request with template url", func(t *testing.T) {
		s := newServer(t)
		content := "kind: API\nmethod: GET\nurl: \"{{.baseUrl}}/users\"\n"
		if _, _, err := s.handleWriteRequest(ctx, nil, writeRequestInput{Name: "ok", YamlContent: content}); err != nil {
			t.Errorf("valid request rejected: %v", err)
		}
	})

	t.Run("rejects missing url", func(t *testing.T) {
		s := newServer(t)
		if _, _, err := s.handleWriteRequest(ctx, nil, writeRequestInput{Name: "bad", YamlContent: "kind: API\nmethod: GET\n"}); err == nil {
			t.Error("missing url should be rejected by schema")
		}
	})

	t.Run("rejects unknown kind", func(t *testing.T) {
		s := newServer(t)
		content := "kind: Nonsense\nmethod: GET\nurl: http://x\n"
		if _, _, err := s.handleWriteRequest(ctx, nil, writeRequestInput{Name: "bad", YamlContent: content}); err == nil {
			t.Error("unknown kind should be rejected by schema")
		}
	})

	t.Run("rejects invalid method", func(t *testing.T) {
		s := newServer(t)
		content := "method: FETCH\nurl: http://x\n"
		if _, _, err := s.handleWriteRequest(ctx, nil, writeRequestInput{Name: "bad", YamlContent: content}); err == nil {
			t.Error("invalid method should be rejected by schema")
		}
	})

	t.Run("without schema, structurally-thin yaml still allowed", func(t *testing.T) {
		s, _ := NewServer(map[string]string{"api": projectDir(t)}, "v") // no SetRequestSchema
		content := "kind: API\nmethod: GET\n"                           // no url
		if _, _, err := s.handleWriteRequest(ctx, nil, writeRequestInput{Name: "ok", YamlContent: content}); err != nil {
			t.Errorf("without schema this should pass the mapping-only check, got: %v", err)
		}
	})
}
