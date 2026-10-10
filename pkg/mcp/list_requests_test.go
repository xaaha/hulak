package mcp

import (
	"context"
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"filippo.io/age"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
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
	// A request in a SUB-DIRECTORY referencing a root-relative .gql. Guards the
	// project-root resolution wired via withProjectDir: the old file-dir-relative
	// rule doubled the sub-dir here (api/svc/svc/query.gql).
	subGqlPath := filepath.Join(api, "collection", "users.gql")
	writeFileAt(t, subGqlPath, "query { users { id } }")
	subReq := filepath.Join(api, "svc", "getUsers.hk.yaml")
	writeFileAt(t, subReq,
		"kind: GraphQL\nmethod: POST\nurl: http://x\nbody:\n  graphql:\n    query: '{{getFile \"collection/users.gql\"}}'\n")
	// A request in mobile + noise files that must be excluded.
	writeReq(t, mobile, "signup.hk.yaml")
	if err := os.WriteFile(filepath.Join(mobile, "options.yaml"), []byte("kind: API\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mobile, "signup_response.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	s, err := NewServer(map[string]string{"api": api, "mobile": mobile}, "v")
	if err != nil {
		t.Fatal(err)
	}

	byName := func(out listRequestsOutput) map[string]RequestSummary {
		m := map[string]RequestSummary{}
		for _, r := range allRequests(out) {
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
		for _, want := range []string{"getuser", "getusers", "listposts", "signup"} {
			if _, ok := got[want]; !ok {
				t.Errorf("missing request %q in %v", want, keys(got))
			}
		}
		if n := len(allRequests(out)); n != 4 {
			t.Errorf("expected 4 requests (options.yaml + _response.json excluded), got %d: %v",
				n, keys(got))
		}
	})

	t.Run("subdir request resolves root-relative dep without doubling", func(t *testing.T) {
		_, out, err := s.handleListRequests(context.Background(), nil, listRequestsInput{Project: "api"})
		if err != nil {
			t.Fatal(err)
		}
		gu := byName(out)["getusers"]
		want := filepath.Join("collection", "users.gql")
		if len(gu.Deps) != 1 || gu.Deps[0] != want {
			t.Errorf("deps = %v, want [%s] (a doubled path means project-root resolution regressed)",
				gu.Deps, want)
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
		want := filepath.Join("gql", "ListPosts.gql")
		if len(lp.Deps) != 1 || lp.Deps[0] != want {
			t.Errorf("deps = %v, want [%s]", lp.Deps, want)
		}
		if filepath.Join(api, lp.Deps[0]) != gqlPath {
			t.Errorf("root + dep = %s, want %s", filepath.Join(api, lp.Deps[0]), gqlPath)
		}
	})

	t.Run("filters to one project", func(t *testing.T) {
		_, out, err := s.handleListRequests(context.Background(), nil, listRequestsInput{Project: "mobile"})
		if err != nil {
			t.Fatal(err)
		}
		if len(out.Projects) != 1 || out.Projects[0].Name != "mobile" {
			t.Fatalf("expected only the mobile project, got %v", out.Projects)
		}
		if reqs := out.Projects[0].Requests; len(reqs) != 1 || reqs[0].Name != "signup" {
			t.Errorf("expected only mobile/signup, got %v", reqs)
		}
	})

	t.Run("unknown project errors", func(t *testing.T) {
		if _, _, err := s.handleListRequests(context.Background(), nil, listRequestsInput{Project: "nope"}); err == nil {
			t.Error("expected error for unknown project")
		}
	})
}

func allRequests(out listRequestsOutput) []RequestSummary {
	var reqs []RequestSummary
	for _, p := range out.Projects {
		reqs = append(reqs, p.Requests...)
	}
	return reqs
}

func keys(m map[string]RequestSummary) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestListRequests_D3_1_GraphqlBodyKind(t *testing.T) {
	api := evalSymlinks(t, projectDir(t))
	writeFileAt(t, filepath.Join(api, "inferred.hk.yaml"),
		"method: POST\nurl: http://x\nbody:\n  graphql:\n    query: 'query { posts { id } }'\n")

	s, err := NewServer(map[string]string{"api": api}, "v")
	if err != nil {
		t.Fatal(err)
	}
	_, out, err := s.handleListRequests(context.Background(), nil, listRequestsInput{})
	if err != nil {
		t.Fatal(err)
	}
	reqs := allRequests(out)
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	if reqs[0].Kind != "GraphQL" {
		t.Errorf("kind = %q, want GraphQL", reqs[0].Kind)
	}
}

func TestListRequests_D3_1_ScalarBodyKeepsKind(t *testing.T) {
	api := evalSymlinks(t, projectDir(t))
	writeFileAt(t, filepath.Join(api, "scalarbody.hk.yaml"),
		"method: POST\nurl: http://x\nbody: hello\n")

	s, err := NewServer(map[string]string{"api": api}, "v")
	if err != nil {
		t.Fatal(err)
	}
	_, out, err := s.handleListRequests(context.Background(), nil, listRequestsInput{})
	if err != nil {
		t.Fatal(err)
	}
	reqs := allRequests(out)
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	if reqs[0].Kind != "API" {
		t.Errorf("kind = %q, want API", reqs[0].Kind)
	}
}

func TestListRequests_259_GroupsByProjectWithRelativePaths(t *testing.T) {
	api := projectDir(t)
	mobile := projectDir(t)
	writeReq(t, filepath.Join(api, "users"), "getUser.hk.yaml")
	writeReq(t, mobile, "signup.hk.yaml")

	s, err := NewServer(map[string]string{"api": api, "mobile": mobile}, "v")
	if err != nil {
		t.Fatal(err)
	}
	_, out, err := s.handleListRequests(context.Background(), nil, listRequestsInput{})
	if err != nil {
		t.Fatal(err)
	}

	wantRoots := map[string]string{"api": api, "mobile": mobile}
	var names []string
	for _, p := range out.Projects {
		names = append(names, p.Name)
		if p.Root != wantRoots[p.Name] {
			t.Errorf("project %s root = %s, want %s", p.Name, p.Root, wantRoots[p.Name])
		}
		for _, r := range p.Requests {
			if filepath.IsAbs(r.Path) {
				t.Errorf("path %s is absolute, want it relative to %s", r.Path, p.Root)
			}
			if !utils.FileExists(filepath.Join(p.Root, r.Path)) {
				t.Errorf("root %s + path %s is not the request file", p.Root, r.Path)
			}
		}
	}
	if want := []string{"api", "mobile"}; !slices.Equal(names, want) {
		t.Errorf("projects = %v, want %v in that order", names, want)
	}
	if got := out.Projects[0].Requests[0].Path; got != filepath.Join("users", "getUser.hk.yaml") {
		t.Errorf("path = %s, want users/getUser.hk.yaml", got)
	}
}

func TestProjectRelative_259(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "work", "api")
	tests := []struct {
		name string
		path string
		want string
	}{
		{"inside", filepath.Join(root, "users", "get.hk.yaml"), filepath.Join("users", "get.hk.yaml")},
		{"sibling with shared prefix", filepath.Join(string(filepath.Separator), "work", "api-evil", "x.gql"),
			filepath.Join(string(filepath.Separator), "work", "api-evil", "x.gql")},
		{"parent", filepath.Join(string(filepath.Separator), "work", "x.gql"),
			filepath.Join(string(filepath.Separator), "work", "x.gql")},
		{"relative stays as is", filepath.Join("gql", "x.gql"), filepath.Join("gql", "x.gql")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := projectRelative(root, tc.path); got != tc.want {
				t.Errorf("projectRelative(%s, %s) = %s, want %s", root, tc.path, got, tc.want)
			}
		})
	}
}

// legacyRequestSummary is the pre-#259 entry shape the size budget measures against.
type legacyRequestSummary struct {
	Name    string   `json:"name"`
	Project string   `json:"project"`
	Path    string   `json:"path"`
	Kind    string   `json:"kind,omitempty"`
	Deps    []string `json:"deps,omitempty"`
}

type budgetRequest struct {
	content string
	deps    []string
}

func budgetFixture(t *testing.T) (string, map[string]budgetRequest) {
	t.Helper()
	root := evalSymlinks(t, projectDir(t))
	bearer := "headers:\n  Authorization: Bearer {{getValueOf \"access_token\" \"get_m2m_token\"}}\n  Content-Type: application/json\n"
	memberGql := filepath.Join("graphql", "queries", "member_profile.gql")
	employersGql := filepath.Join("graphql", "queries", "list_employers.gql")
	files := map[string]budgetRequest{
		"auth/get_m2m_token.hk.yaml": {content: "method: POST\nurl: \"{{.auth_base_url}}/oauth/token\"\n" +
			"body:\n  urlencodedformdata:\n    client_id: \"{{.client_id}}\"\n    client_secret: \"{{.client_secret}}\"\n    grant_type: client_credentials\n"},
		"opn/migration/get_enrollment.hk.yaml": {content: "method: GET\nurl: \"{{.opn_base_url}}/v2/enrollments/{{.enrollment_id}}\"\n" + bearer},
		"opn/migration/list_programs.hk.yaml":  {content: "method: GET\nurl: \"{{.opn_base_url}}/v2/programs\"\nurlparams:\n  page: 1\n" + bearer},
		"tuition_reimbursement/create_claim.hk.yaml": {content: "method: POST\nurl: \"{{.tr_base_url}}/claims\"\n" + bearer +
			"body:\n  raw: '{\"member_id\": \"{{.member_id}}\", \"amount\": 100}'\n"},
		"tuition_reimbursement/get_claim.hk.yaml": {content: "method: GET\nurl: \"{{.tr_base_url}}/claims/{{.claim_id}}\"\n" + bearer},
		"graphql/member_profile.hk.yaml": {
			content: "kind: GraphQL\nmethod: POST\nurl: \"{{.graphql_url}}\"\n" + bearer +
				"body:\n  graphql:\n    query: '{{getFile \"graphql/queries/member_profile.gql\"}}'\n    variables:\n      memberId: \"{{.member_id}}\"\n      includeHistory: true\n",
			deps: []string{memberGql},
		},
		"graphql/list_employers.hk.yaml": {
			content: "kind: GraphQL\nmethod: POST\nurl: https://graphql-gateway.staging.example.com/graphql\n" +
				"headers:\n  x-api-key: \"{{.appsync_api_key}}\"\n" +
				"body:\n  graphql:\n    query: '{{getFile \"graphql/queries/list_employers.gql\"}}'\n    variables:\n      first: 20\n",
			deps: []string{employersGql},
		},
		"github/oauth.hk.yaml": {content: "kind: Auth\nmethod: POST\nurl: https://github.com/login/oauth/authorize\n" +
			"urlparams:\n  client_id: \"{{.gh_client_id}}\"\nauth:\n  type: OAuth2.0\n  access_token_url: https://github.com/login/oauth/access_token\n"},
		"health.hk.yaml": {content: "method: GET\nurl: https://status.example.com/health\n"},
	}
	for rel, req := range files {
		writeFileAt(t, filepath.Join(root, rel), req.content)
	}
	writeFileAt(t, filepath.Join(root, memberGql), "query { member { id } }")
	writeFileAt(t, filepath.Join(root, employersGql), "query { employers { id } }")
	return root, files
}

func TestListRequests_259_SizeBudget(t *testing.T) {
	root, files := budgetFixture(t)
	s, err := NewServer(map[string]string{"guild": root}, "v")
	if err != nil {
		t.Fatal(err)
	}
	_, out, err := s.handleListRequests(context.Background(), nil, listRequestsInput{})
	if err != nil {
		t.Fatal(err)
	}

	const typicalRoot = "/Users/someone/work/project"
	kinds := map[string]string{}
	for i, p := range out.Projects {
		out.Projects[i].Root = typicalRoot
		for _, r := range p.Requests {
			kinds[r.Name] = r.Kind
		}
	}
	if len(kinds) != len(files) {
		t.Fatalf("listed %d requests, want the fixture's %d", len(kinds), len(files))
	}
	var legacy []legacyRequestSummary
	for rel, req := range files {
		name := utils.RequestStem(filepath.Base(rel))
		var deps []string
		for _, d := range req.deps {
			deps = append(deps, filepath.Join(typicalRoot, d))
		}
		legacy = append(legacy, legacyRequestSummary{
			Name:    name,
			Project: "guild",
			Path:    filepath.Join(typicalRoot, rel),
			Kind:    kinds[name],
			Deps:    deps,
		})
	}

	before, err := json.Marshal(struct {
		Requests []legacyRequestSummary `json:"requests"`
	}{legacy})
	if err != nil {
		t.Fatal(err)
	}
	after, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) > len(before) {
		t.Errorf("default listing is %d bytes, larger than the %d-byte legacy listing", len(after), len(before))
	}
}

func TestListRequests_259_Host(t *testing.T) {
	api := projectDir(t)
	writeFileAt(t, filepath.Join(api, "templated.hk.yaml"), "method: GET\nurl: \"{{.base_url}}/v1/users\"\n")
	writeFileAt(t, filepath.Join(api, "literal.hk.yaml"), "method: GET\nURL: https://api.example.com/v1\n")
	writeFileAt(t, filepath.Join(api, "broken.hk.yaml"), "method: [GET\n")

	s, err := NewServer(map[string]string{"api": api}, "v")
	if err != nil {
		t.Fatal(err)
	}
	_, out, err := s.handleListRequests(context.Background(), nil, listRequestsInput{})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, r := range allRequests(out) {
		got[r.Name] = r.Host
	}
	want := map[string]string{"templated": "{{.base_url}}", "literal": "api.example.com", "broken": ""}
	if !maps.Equal(got, want) {
		t.Errorf("hosts = %v, want %v", got, want)
	}
}

func TestListRequests_259_AuthOnlyWithDetail(t *testing.T) {
	api := projectDir(t)
	writeFileAt(t, filepath.Join(api, "me.hk.yaml"),
		"method: GET\nurl: https://api.example.com/me\nHeaders:\n  Authorization: Bearer {{getValueOf \"access_token\" \"login\"}}\n")
	writeFileAt(t, filepath.Join(api, "gh.hk.yaml"),
		"kind: Auth\nmethod: POST\nurl: https://github.com/login/oauth/authorize\n")
	writeReq(t, api, "open.hk.yaml")

	s, err := NewServer(map[string]string{"api": api}, "v")
	if err != nil {
		t.Fatal(err)
	}
	auths := func(detail bool) map[string]string {
		t.Helper()
		_, out, err := s.handleListRequests(context.Background(), nil, listRequestsInput{Detail: detail})
		if err != nil {
			t.Fatal(err)
		}
		got := map[string]string{}
		for _, r := range allRequests(out) {
			got[r.Name] = r.Auth
		}
		return got
	}

	want := map[string]string{"me": "bearer from login", "gh": "oauth2", "open": ""}
	if got := auths(true); !maps.Equal(got, want) {
		t.Errorf("detail auth = %v, want %v", got, want)
	}
	want = map[string]string{"me": "", "gh": "", "open": ""}
	if got := auths(false); !maps.Equal(got, want) {
		t.Errorf("default auth = %v, want none outside a detail listing", got)
	}
}

func TestListRequests_259_VariablesOnlyWithDetail(t *testing.T) {
	api := projectDir(t)
	writeFileAt(t, filepath.Join(api, "profile.hk.yaml"),
		"kind: GraphQL\nmethod: POST\nurl: \"{{.graphql_url}}\"\nheaders:\n  x-tenant: \"{{.tenant}}\"\n"+
			"body:\n  graphql:\n    query: 'query { me { id } }'\n    variables:\n      memberId: \"{{.member_id}}\"\n      first: 10\n")
	writeReq(t, api, "plain.hk.yaml")
	writeFileAt(t, filepath.Join(api, "broken.hk.yaml"), "url: \"{{.base_url}}\"\nmethod: [GET\n")

	s, err := NewServer(map[string]string{"api": api}, "v")
	if err != nil {
		t.Fatal(err)
	}
	list := func(detail bool) map[string]RequestSummary {
		t.Helper()
		_, out, err := s.handleListRequests(context.Background(), nil, listRequestsInput{Detail: detail})
		if err != nil {
			t.Fatal(err)
		}
		got := map[string]RequestSummary{}
		for _, r := range allRequests(out) {
			got[r.Name] = r
		}
		return got
	}

	got := list(true)
	if len(got) != 3 {
		t.Fatalf("listed %v, want profile, plain, and broken", keys(got))
	}
	profile := got["profile"]
	if want := []string{"graphql_url", "tenant", "member_id"}; !slices.Equal(profile.EnvVars, want) {
		t.Errorf("profile env_vars = %v, want %v", profile.EnvVars, want)
	}
	if want := []string{"memberId", "first"}; !slices.Equal(profile.Variables, want) {
		t.Errorf("profile variables = %v, want %v", profile.Variables, want)
	}
	if p := got["plain"]; p.EnvVars != nil || p.Variables != nil {
		t.Errorf("plain env_vars = %v, variables = %v, want none", p.EnvVars, p.Variables)
	}

	for name, r := range list(false) {
		if r.EnvVars != nil || r.Variables != nil {
			t.Errorf("default %s env_vars = %v, variables = %v, want none outside a detail listing",
				name, r.EnvVars, r.Variables)
		}
	}
}

func TestListRequests_259_Filter(t *testing.T) {
	api := projectDir(t)
	mobile := projectDir(t)
	writeReq(t, filepath.Join(api, "opn", "migration"), "get_enrollment.hk.yaml")
	writeReq(t, filepath.Join(api, "tuition_reimbursement"), "get_claim.hk.yaml")
	writeReq(t, filepath.Join(api, "auth"), "Login.hk.yaml")
	writeReq(t, mobile, "login.hk.yaml")
	writeReq(t, mobile, "signup.hk.yaml")

	s, err := NewServer(map[string]string{"api": api, "mobile": mobile}, "v")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		filter string
		want   []string
	}{
		{"none", "", []string{"api/get_enrollment", "api/get_claim", "api/login", "mobile/login", "mobile/signup"}},
		{"directory", "opn/", []string{"api/get_enrollment"}},
		{"nested directory", "opn/migration/get", []string{"api/get_enrollment"}},
		{"name in mixed case", "LOGIN", []string{"api/login", "mobile/login"}},
		{"no match", "nothing", nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, out, err := s.handleListRequests(context.Background(), nil, listRequestsInput{Filter: tc.filter})
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, p := range out.Projects {
				if len(p.Requests) == 0 {
					t.Errorf("project %s listed with no requests", p.Name)
				}
				for _, r := range p.Requests {
					got = append(got, p.Name+"/"+r.Name)
				}
			}
			slices.Sort(got)
			want := slices.Sorted(slices.Values(tc.want))
			if !slices.Equal(got, want) {
				t.Errorf("filter %q listed %v, want %v", tc.filter, got, want)
			}
		})
	}
}

func TestListRequests_259_EmptyResultOverMCP(t *testing.T) {
	ctx := context.Background()
	serverT, clientT := mcpsdk.NewInMemoryTransports()
	api := projectDir(t)
	writeReq(t, api, "login.hk.yaml")
	s, err := NewServer(map[string]string{"api": api}, "test")
	if err != nil {
		t.Fatal(err)
	}
	ss, err := s.srv.Connect(ctx, serverT, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ss.Close()
	cs, err := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "c", Version: "0"}, nil).Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()

	res, err := cs.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      "list_requests",
		Arguments: map[string]any{"filter": "nothing"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("empty filter result is a tool error: %v", res.Content)
	}
	text := res.Content[0].(*mcpsdk.TextContent).Text
	if text != `{"projects":[]}` {
		t.Errorf("content = %s, want an empty projects list", text)
	}
}

func TestListRequests_259_EnvResolvesHosts(t *testing.T) {
	api := projectDir(t)
	mobile := projectDir(t)
	writeFileAt(t, filepath.Join(api, "env", "staging.env"),
		"base_url=https://svc:hunter2-secret@api.staging.example.com/v1\ntenant=acme\n"+
			"slash_pw=https://svc:12/hunter2-secret@api.example.com/v1\n"+
			"query_pw=https://svc:pa?hunter2-secret@api.example.com/v1\n"+
			"at_pw=https://svc:p@hunter2-secret@api.example.com/v1\n"+
			"bare=api.bare.example.com/v1\n"+
			"empty=\n")
	for _, name := range []string{"slash_pw", "query_pw", "at_pw", "bare", "empty"} {
		writeFileAt(t, filepath.Join(api, name+".hk.yaml"), "method: GET\nurl: \"{{."+name+"}}\"\n")
	}
	writeFileAt(t, filepath.Join(api, "env", "empty.env"), "")
	writeFileAt(t, filepath.Join(api, "users.hk.yaml"), "method: GET\nurl: \"{{.base_url}}/users?page=1\"\n")
	writeFileAt(t, filepath.Join(api, "tenant.hk.yaml"), "method: GET\nurl: \"https://{{.tenant}}.example.com/me\"\n")
	writeFileAt(t, filepath.Join(api, "unresolved.hk.yaml"), "method: GET\nurl: \"{{.nope}}/x\"\n")
	writeFileAt(t, filepath.Join(api, "literal.hk.yaml"), "method: GET\nurl: https://status.example.com/health\n")
	writeFileAt(t, filepath.Join(mobile, "signup.hk.yaml"), "method: POST\nurl: \"{{.base_url}}/signup\"\n")

	s, err := NewServer(map[string]string{"api": api, "mobile": mobile}, "v")
	if err != nil {
		t.Fatal(err)
	}
	list := func(in listRequestsInput) (listRequestsOutput, error) {
		_, out, err := s.handleListRequests(context.Background(), nil, in)
		return out, err
	}

	t.Run("resolves hosts where the env exists", func(t *testing.T) {
		out, err := list(listRequestsInput{Env: "staging"})
		if err != nil {
			t.Fatal(err)
		}
		got := map[string]string{}
		missing := map[string]bool{}
		for _, p := range out.Projects {
			missing[p.Name] = p.EnvMissing
			for _, r := range p.Requests {
				got[p.Name+"/"+r.Name] = r.Host
			}
		}
		want := map[string]string{
			"api/users":      "api.staging.example.com",
			"api/tenant":     "acme.example.com",
			"api/unresolved": "{{.nope}}",
			"api/literal":    "status.example.com",
			"api/slash_pw":   "{{.slash_pw}}",
			"api/query_pw":   "{{.query_pw}}",
			"api/at_pw":      "api.example.com",
			"api/bare":       "api.bare.example.com",
			"api/empty":      "{{.empty}}",
			"mobile/signup":  "{{.base_url}}",
		}
		if !maps.Equal(got, want) {
			t.Errorf("hosts = %v, want %v", got, want)
		}
		if want := map[string]bool{"api": false, "mobile": true}; !maps.Equal(missing, want) {
			t.Errorf("env_missing = %v, want %v", missing, want)
		}
		raw, err := json.Marshal(out)
		if err != nil {
			t.Fatal(err)
		}
		for _, leak := range []string{"hunter2-secret", "svc", "/v1", "page=1"} {
			if strings.Contains(string(raw), leak) {
				t.Errorf("listing leaks %q beyond the host: %s", leak, raw)
			}
		}
	})

	t.Run("an empty env is found", func(t *testing.T) {
		out, err := list(listRequestsInput{Env: "empty", Project: "api"})
		if err != nil {
			t.Fatal(err)
		}
		if out.Projects[0].EnvMissing {
			t.Error("env_missing = true for an env file that exists but is empty")
		}
	})

	t.Run("an env no project has is an error", func(t *testing.T) {
		_, err := list(listRequestsInput{Env: "prod"})
		if err == nil || !strings.Contains(err.Error(), `env "prod" not found in any project`) {
			t.Errorf("err = %v, want env not found in any project", err)
		}
	})

	t.Run("an env missing from the only target project is an error", func(t *testing.T) {
		if _, err := list(listRequestsInput{Env: "staging", Project: "mobile"}); err == nil {
			t.Error("want an error when the only target project lacks the env")
		}
	})
}

func TestListRequests_259_EnvResolvesHostsInVaultProject(t *testing.T) {
	root := evalSymlinks(t, t.TempDir())
	if err := os.Mkdir(filepath.Join(root, utils.HiddenProjectName), utils.DirPer); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	configDir, err := utils.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(configDir, utils.DirPer); err != nil {
		t.Fatal(err)
	}
	id, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	if err := vault.SetIdentity(id.String()); err != nil {
		t.Fatal(err)
	}
	store := &vault.Store{Envs: map[string]vault.Env{
		"global": {},
		"prod":   {"base_url": "https://api.prod.example.com/v2"},
	}}
	if err := vault.WriteStore(store, id.Recipient()); err != nil {
		t.Fatal(err)
	}
	writeFileAt(t, filepath.Join(root, "users.hk.yaml"), "method: GET\nurl: \"{{.base_url}}/users\"\n")

	s, err := NewServer(map[string]string{"vaulted": root}, "v")
	if err != nil {
		t.Fatal(err)
	}
	_, out, err := s.handleListRequests(context.Background(), nil, listRequestsInput{Env: "prod"})
	if err != nil {
		t.Fatal(err)
	}
	if got := allRequests(out)[0].Host; got != "api.prod.example.com" {
		t.Errorf("host = %q, want api.prod.example.com resolved from the vault", got)
	}
}
