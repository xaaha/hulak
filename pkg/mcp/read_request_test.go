package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestReadRequest_260(t *testing.T) {
	api := evalSymlinks(t, projectDir(t))
	mobile := evalSymlinks(t, projectDir(t))
	request := "kind: GraphQL\nmethod: POST\nurl: \"{{.graphql_url}}\"\nbody:\n  graphql:\n" +
		"    query: '{{getFile \"gql/users.gql\"}}'\n    variables:\n      first: 10\n"
	writeFileAt(t, filepath.Join(api, "svc", "users.hk.yaml"), request)
	writeFileAt(t, filepath.Join(api, "gql", "users.gql"), "query { users { id } }")
	writeFileAt(t, filepath.Join(api, "broken.hk.yaml"),
		"method: POST\nurl: http://x\nbody:\n  raw: '{{getFile \"gql/missing.gql\"}}{{getFile \"/etc/hosts\"}}'\n")
	writeReq(t, api, "login.hk.yaml")
	writeReq(t, mobile, "login.hk.yaml")

	s, err := NewServer(map[string]string{"api": api, "mobile": mobile}, "v")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	t.Run("returns the file as written with its dependency", func(t *testing.T) {
		_, out, err := s.handleReadRequest(ctx, nil, readRequestInput{Name: "users"})
		if err != nil {
			t.Fatal(err)
		}
		if out.Project != "api" || out.Path != filepath.Join("svc", "users.hk.yaml") {
			t.Errorf("project, path = %s, %s, want api, svc/users.hk.yaml", out.Project, out.Path)
		}
		if out.Content != request {
			t.Errorf("content = %q, want the file byte for byte", out.Content)
		}
		want := []DependencyFile{{Path: filepath.Join("gql", "users.gql"), Content: "query { users { id } }"}}
		if len(out.Deps) != 1 || out.Deps[0] != want[0] {
			t.Errorf("deps = %+v, want %+v", out.Deps, want)
		}
	})

	t.Run("a missing or outside dependency is listed without content", func(t *testing.T) {
		_, out, err := s.handleReadRequest(ctx, nil, readRequestInput{Name: "broken", Project: "api"})
		if err != nil {
			t.Fatal(err)
		}
		want := []DependencyFile{
			{Path: filepath.Join("gql", "missing.gql"), Missing: true},
			{Path: "/etc/hosts", Missing: true},
		}
		if len(out.Deps) != len(want) {
			t.Fatalf("deps = %+v, want %+v", out.Deps, want)
		}
		for i := range want {
			if out.Deps[i] != want[i] {
				t.Errorf("deps[%d] = %+v, want %+v", i, out.Deps[i], want[i])
			}
		}
	})

	t.Run("an ambiguous name lists the choices", func(t *testing.T) {
		_, _, err := s.handleReadRequest(ctx, nil, readRequestInput{Name: "login"})
		if err == nil || !strings.Contains(err.Error(), "ambiguous") {
			t.Fatalf("err = %v, want an ambiguity error", err)
		}
		_, out, err := s.handleReadRequest(ctx, nil, readRequestInput{Name: "login", Project: "mobile"})
		if err != nil || out.Project != "mobile" {
			t.Errorf("project mobile: out = %+v, err = %v", out, err)
		}
	})

	t.Run("an unknown name is an error", func(t *testing.T) {
		if _, _, err := s.handleReadRequest(ctx, nil, readRequestInput{Name: "nope"}); err == nil {
			t.Error("want an error for an unknown request")
		}
	})

	t.Run("a name is required", func(t *testing.T) {
		_, _, err := s.handleReadRequest(ctx, nil, readRequestInput{Name: "  "})
		if err == nil || !strings.Contains(err.Error(), "name is required") {
			t.Errorf("err = %v, want name is required", err)
		}
	})
}

func TestReadRequest_260_KeepsSecretsAndOutsideFilesBack(t *testing.T) {
	api := evalSymlinks(t, projectDir(t))
	outside := evalSymlinks(t, t.TempDir())
	writeFileAt(t, filepath.Join(outside, "secret.txt"), "OUTSIDE-SECRET")
	writeFileAt(t, filepath.Join(outside, "outer.hk.yaml"), "method: GET\nurl: http://outside\n")
	writeFileAt(t, filepath.Join(api, "env", "global.env"), "api_token=ENV-SECRET\n")
	writeFileAt(t, filepath.Join(api, ".hulak", "store.age"), "VAULT-BYTES")
	writeFileAt(t, filepath.Join(api, "q", "chain.txt"), `{{getFile "q/never_read.txt"}}`)
	writeFileAt(t, filepath.Join(api, "q", "never_read.txt"), "CHAINED-CONTENT")
	writeFileAt(t, filepath.Join(api, "svc", "users.gql"), "query { users { id } }")
	writeFileAt(t, filepath.Join(api, "svc", "users.hk.yaml"),
		"kind: GraphQL\nmethod: POST\nurl: http://x\nbody:\n  graphql:\n    query: '{{getFile \"*.gql\"}}'\n")
	if err := os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(api, "link.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "outer.hk.yaml"), filepath.Join(api, "outer.hk.yaml")); err != nil {
		t.Fatal(err)
	}
	writeFileAt(t, filepath.Join(api, "leak.hk.yaml"), "method: POST\nurl: http://x\nbody:\n  raw: '"+
		`{{getFile "env/global.env"}}{{getFile ".hulak/store.age"}}{{getFile "link.txt"}}{{getFile "q/chain.txt"}}`+"'\n")

	s, err := NewServer(map[string]string{"api": api}, "v")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	t.Run("env, vault, outside and chained files are held back", func(t *testing.T) {
		_, out, err := s.handleReadRequest(ctx, nil, readRequestInput{Name: "leak"})
		if err != nil {
			t.Fatal(err)
		}
		want := []DependencyFile{
			{Path: filepath.Join("env", "global.env"), Withheld: true},
			{Path: filepath.Join(".hulak", "store.age"), Withheld: true},
			{Path: "link.txt", Missing: true},
			{Path: filepath.Join("q", "chain.txt"), Content: `{{getFile "q/never_read.txt"}}`},
		}
		if !slices.Equal(out.Deps, want) {
			t.Errorf("deps = %+v\nwant %+v", out.Deps, want)
		}
		raw, err := json.Marshal(out)
		if err != nil {
			t.Fatal(err)
		}
		for _, leak := range []string{"ENV-SECRET", "VAULT-BYTES", "OUTSIDE-SECRET", "CHAINED-CONTENT"} {
			if strings.Contains(string(raw), leak) {
				t.Errorf("read_request returned %s: %s", leak, raw)
			}
		}
	})

	t.Run("a sibling query is read", func(t *testing.T) {
		_, out, err := s.handleReadRequest(ctx, nil, readRequestInput{Name: "users"})
		if err != nil {
			t.Fatal(err)
		}
		want := []DependencyFile{{Path: filepath.Join("svc", "users.gql"), Content: "query { users { id } }"}}
		if !slices.Equal(out.Deps, want) {
			t.Errorf("deps = %+v, want %+v", out.Deps, want)
		}
	})

	t.Run("a request file that links outside the project is refused", func(t *testing.T) {
		_, out, err := s.handleReadRequest(ctx, nil, readRequestInput{Name: "outer"})
		if err == nil || strings.Contains(out.Content, "outside") {
			t.Errorf("err = %v, content = %q, want an error and no content", err, out.Content)
		}
	})
}
