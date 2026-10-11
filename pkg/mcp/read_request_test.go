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
	writeFileAt(t, filepath.Join(api, "child", "env", "global.env"), "CHILD-SECRET")
	evil := api + "-evil"
	writeFileAt(t, filepath.Join(evil, "secret.txt"), "EVIL-SECRET")
	t.Cleanup(func() { _ = os.RemoveAll(evil) })
	for link, target := range map[string]string{
		"inner.txt":    filepath.Join(api, "env", "global.env"),
		"evil.txt":     filepath.Join(evil, "secret.txt"),
		"peek.hk.yaml": filepath.Join(api, "env", "global.env"),
	} {
		if err := os.Symlink(target, filepath.Join(api, link)); err != nil {
			t.Fatal(err)
		}
	}
	writeFileAt(t, filepath.Join(api, "env", "inenv.hk.yaml"), "method: GET\nurl: http://inenv\n")
	for _, name := range []string{"hard.json", "hardreq.hk.yaml"} {
		if err := os.Link(filepath.Join(api, "env", "global.env"), filepath.Join(api, name)); err != nil {
			t.Fatal(err)
		}
	}
	writeFileAt(t, filepath.Join(api, "leak.hk.yaml"), "method: POST\nurl: http://x\nbody:\n  raw: '"+
		`{{getFile "env/global.env"}}{{getFile ".hulak/store.age"}}{{getFile "link.txt"}}{{getFile "q/chain.txt"}}`+
		`{{getFile "inner.txt"}}{{getFile "child/env/global.env"}}{{getFile "evil.txt"}}{{getFile "hard.json"}}`+"'\n")
	writeFileAt(t, filepath.Join(api, "upper", "ENV", "global.env"), "UPPER-ENV-SECRET")
	writeFileAt(t, filepath.Join(api, "upper", ".HULAK", "store.age"), "UPPER-VAULT-BYTES")
	writeFileAt(t, filepath.Join(api, "env", "nested", "x.env"), "NESTED-SECRET")
	writeFileAt(t, filepath.Join(api, "cased.hk.yaml"), "method: POST\nurl: http://x\nbody:\n  raw: '"+
		`{{getFile "upper/ENV/global.env"}}{{getFile "upper/.HULAK/store.age"}}{{getFile "env/nested/x.env"}}`+"'\n")
	writeFileAt(t, filepath.Join(api, "environments", "staging.json"), "STAGING")
	writeFileAt(t, filepath.Join(api, "svc", "env"), "FILE-NAMED-ENV")
	writeFileAt(t, filepath.Join(api, "dev.env.json"), "DEV")
	writeFileAt(t, filepath.Join(api, "plain.hk.yaml"), "method: POST\nurl: http://x\nbody:\n  raw: '"+
		`{{getFile "environments/staging.json"}}{{getFile "svc/env"}}{{getFile "dev.env.json"}}`+"'\n")

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
			{Path: "inner.txt", Withheld: true},
			{Path: filepath.Join("child", "env", "global.env"), Withheld: true},
			{Path: "evil.txt", Missing: true},
			{Path: "hard.json", Withheld: true},
		}
		if !slices.Equal(out.Deps, want) {
			t.Errorf("deps = %+v\nwant %+v", out.Deps, want)
		}
		raw, err := json.Marshal(out)
		if err != nil {
			t.Fatal(err)
		}
		for _, leak := range []string{"ENV-SECRET", "VAULT-BYTES", "OUTSIDE-SECRET", "CHAINED-CONTENT", "CHILD-SECRET", "EVIL-SECRET"} {
			if strings.Contains(string(raw), leak) {
				t.Errorf("read_request returned %s: %s", leak, raw)
			}
		}
	})

	t.Run("cased and nested secrets folders are held back", func(t *testing.T) {
		_, out, err := s.handleReadRequest(ctx, nil, readRequestInput{Name: "cased"})
		if err != nil {
			t.Fatal(err)
		}
		want := []DependencyFile{
			{Path: filepath.Join("upper", "ENV", "global.env"), Withheld: true},
			{Path: filepath.Join("upper", ".HULAK", "store.age"), Withheld: true},
			{Path: filepath.Join("env", "nested", "x.env"), Withheld: true},
		}
		if !slices.Equal(out.Deps, want) {
			t.Errorf("deps = %+v\nwant %+v", out.Deps, want)
		}
	})

	t.Run("names that only resemble a secrets folder are read", func(t *testing.T) {
		_, out, err := s.handleReadRequest(ctx, nil, readRequestInput{Name: "plain"})
		if err != nil {
			t.Fatal(err)
		}
		want := []DependencyFile{
			{Path: filepath.Join("environments", "staging.json"), Content: "STAGING"},
			{Path: filepath.Join("svc", "env"), Content: "FILE-NAMED-ENV"},
			{Path: "dev.env.json", Content: "DEV"},
		}
		if !slices.Equal(out.Deps, want) {
			t.Errorf("deps = %+v\nwant %+v", out.Deps, want)
		}
	})

	for _, name := range []string{"peek", "inenv", "hardreq"} {
		t.Run("request "+name+" in a secrets folder is refused", func(t *testing.T) {
			_, out, err := s.handleReadRequest(ctx, nil, readRequestInput{Name: name})
			if err == nil || out.Content != "" {
				t.Errorf("err = %v, content = %q, want an error and no content", err, out.Content)
			}
		})
	}

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

func TestReadRequest_260_ProjectUnderSymlinkedParent(t *testing.T) {
	parent := t.TempDir()
	writeFileAt(t, filepath.Join(parent, "proj", "env", "global.env"), "")
	writeFileAt(t, filepath.Join(parent, "proj", "gql", "q.gql"), "query { a }")
	writeFileAt(t, filepath.Join(parent, "proj", "q.hk.yaml"),
		"kind: GraphQL\nmethod: POST\nurl: http://x\nbody:\n  graphql:\n    query: '{{getFile \"gql/q.gql\"}}'\n")
	link := filepath.Join(t.TempDir(), "via")
	if err := os.Symlink(parent, link); err != nil {
		t.Fatal(err)
	}
	s, err := NewServer(map[string]string{"api": filepath.Join(link, "proj")}, "v")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PWD", filepath.Join(link, "proj"))
	_, out, err := s.handleReadRequest(context.Background(), nil, readRequestInput{Name: "q"})
	if err != nil {
		t.Fatal(err)
	}
	want := []DependencyFile{{Path: filepath.Join("gql", "q.gql"), Content: "query { a }"}}
	if !slices.Equal(out.Deps, want) {
		t.Errorf("deps = %+v, want %+v", out.Deps, want)
	}
}
