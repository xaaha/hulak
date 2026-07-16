package mcp

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestHandleListRequests(t *testing.T) {
	// Resolve symlinks (macOS /var -> /private/var) so the absolute dep paths
	// returned by getFile resolution — run from inside the project dir — compare
	// equal to the paths built here.
	api := evalSymlinks(t, projectDir(t))
	mobile := evalSymlinks(t, projectDir(t))

	// API request in api.
	writeReq(t, api, "getUser.hk.yaml")
	// GraphQL request in api that references a sibling .gql via getFile.
	gqlDir := filepath.Join(api, "gql")
	if err := os.MkdirAll(gqlDir, 0o755); err != nil {
		t.Fatal(err)
	}
	gqlPath := filepath.Join(gqlDir, "ListPosts.gql")
	if err := os.WriteFile(gqlPath, []byte("query { posts { id } }"), 0o600); err != nil {
		t.Fatal(err)
	}
	gqlReq := filepath.Join(api, "listPosts.hk.yaml")
	body := "kind: GraphQL\nmethod: POST\nurl: http://x\nbody:\n  graphql:\n    query: '{{getFile \"gql/ListPosts.gql\"}}'\n"
	if err := os.WriteFile(gqlReq, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	// A request in mobile + noise files that must be excluded.
	writeReq(t, mobile, "signup.hk.yaml")
	if err := os.WriteFile(filepath.Join(mobile, "options.yaml"), []byte("kind: API\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mobile, "signup_response.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	s, err := NewServer(map[string]string{"api": api, "mobile": mobile}, "api", "v")
	if err != nil {
		t.Fatal(err)
	}

	byName := func(out listRequestsOutput) map[string]RequestSummary {
		m := map[string]RequestSummary{}
		for _, r := range out.Requests {
			m[r.Name] = r
		}
		return m
	}

	t.Run("lists all projects, excludes noise", func(t *testing.T) {
		_, out, err := s.handleListRequests(context.Background(), nil, listRequestsInput{})
		if err != nil {
			t.Fatal(err)
		}
		got := byName(out)
		for _, want := range []string{"getuser", "listposts", "signup"} {
			if _, ok := got[want]; !ok {
				t.Errorf("missing request %q in %v", want, keys(got))
			}
		}
		if len(out.Requests) != 3 {
			t.Errorf("expected 3 requests (options.yaml + _response.json excluded), got %d: %v",
				len(out.Requests), keys(got))
		}
	})

	t.Run("graphql request surfaces its .gql dep and kind", func(t *testing.T) {
		_, out, err := s.handleListRequests(context.Background(), nil, listRequestsInput{})
		if err != nil {
			t.Fatal(err)
		}
		lp := byName(out)["listposts"]
		if lp.Kind != "GraphQL" {
			t.Errorf("kind = %q, want GraphQL", lp.Kind)
		}
		if len(lp.Deps) != 1 || lp.Deps[0] != gqlPath {
			t.Errorf("deps = %v, want [%s]", lp.Deps, gqlPath)
		}
		if lp.Project != "api" {
			t.Errorf("project = %q, want api", lp.Project)
		}
	})

	t.Run("filters to one project", func(t *testing.T) {
		_, out, err := s.handleListRequests(context.Background(), nil, listRequestsInput{Project: "mobile"})
		if err != nil {
			t.Fatal(err)
		}
		if len(out.Requests) != 1 || out.Requests[0].Name != "signup" {
			t.Errorf("expected only mobile/signup, got %v", out.Requests)
		}
	})

	t.Run("unknown project errors", func(t *testing.T) {
		if _, _, err := s.handleListRequests(context.Background(), nil, listRequestsInput{Project: "nope"}); err == nil {
			t.Error("expected error for unknown project")
		}
	})
}

func keys(m map[string]RequestSummary) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
