package mcp

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"slices"
	"strings"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/xaaha/hulak/pkg/envparser"
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

// RequestSummary describes one request file for the list_requests tool.
// Path and Deps are relative to the project root when they live inside it.
// Auth, EnvVars, and Variables are filled only for a detail listing.
type RequestSummary struct {
	Name      string   `json:"name"`
	Path      string   `json:"path"`
	Kind      string   `json:"kind,omitempty"`
	Host      string   `json:"host,omitempty"`
	Deps      []string `json:"deps,omitempty"` // referenced files, e.g. a GraphQL .gql
	Auth      string   `json:"auth,omitempty"`
	EnvVars   []string `json:"env_vars,omitempty"`
	Variables []string `json:"variables,omitempty"`
}

// ProjectRequests is one project's request files and the root their paths are
// relative to.
type ProjectRequests struct {
	Name       string           `json:"name"`
	Root       string           `json:"root"`
	EnvMissing bool             `json:"env_missing,omitempty"` // env was passed but this project lacks it
	Requests   []RequestSummary `json:"requests"`
}

type listRequestsInput struct {
	Project string `json:"project,omitempty" jsonschema:"limit to this project; omit to list every project"`
	Filter  string `json:"filter,omitempty"  jsonschema:"case-insensitive substring of the project-relative path, e.g. opn/ or login"`
	Detail  bool   `json:"detail,omitempty"  jsonschema:"also report each request's auth mode, env variables, and GraphQL variables"`
	Env     string `json:"env,omitempty"     jsonschema:"resolve each host against this environment, e.g. prod; omit to show hosts as written"`
}

type listRequestsOutput struct {
	Projects []ProjectRequests `json:"projects"`
}

// registerListRequests adds the list_requests tool to the server.
func (s *Server) registerListRequests() {
	mcpsdk.AddTool(s.srv, &mcpsdk.Tool{
		Name: "list_requests",
		Description: "List hulak request files, grouped by project. Each project " +
			"has its absolute root; each entry has its name, file path relative to " +
			"that root, kind (API/GraphQL), target host as written (template " +
			"variables unresolved), and any dependency files it references " +
			"(e.g. a GraphQL query .gql that lives next to the request). Pass " +
			"`detail` to also get each request's auth mode (e.g. \"bearer from " +
			"login\": a bearer token read from the login request's response via " +
			"getValueOf), the env variables it resolves ({{.name}}), and its " +
			"GraphQL variables. Narrow with `filter`, a case-insensitive substring " +
			"of the relative path (a directory like opn/ or part of a name). Pass " +
			"`env` to resolve hosts against that environment; only the host is " +
			"resolved, a host that cannot resolve stays as written, and a project " +
			"without that env is marked env_missing. Omit " +
			"`project` to list every configured project.",
		Annotations: &mcpsdk.ToolAnnotations{ReadOnlyHint: true},
	}, s.handleListRequests)
}

// handleListRequests is the list request handler for the server
func (s *Server) handleListRequests(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	in listRequestsInput,
) (*mcpsdk.CallToolResult, listRequestsOutput, error) {
	targets := s.projects
	if in.Project != "" {
		path, ok := s.projects[in.Project]
		if !ok {
			return nil, listRequestsOutput{}, fmt.Errorf(
				"unknown project %q; configured projects: %s",
				in.Project, strings.Join(projectNames(s.projects), ", "),
			)
		}
		targets = map[string]string{in.Project: path}
	}

	out := listRequestsOutput{Projects: []ProjectRequests{}}
	missing := 0
	for _, name := range projectNames(targets) {
		var reqs []RequestSummary
		envMissing := false
		// Run inside the project dir: dependency resolution (utils.ReferencedFiles
		// -> getFile) and secret loading are project-root-relative and key off
		// the working directory, exactly as a real run does.
		err := s.withProjectDir(targets[name], func() error {
			opts := listOptions{filter: in.Filter, detail: in.Detail}
			if in.Env != "" {
				secrets, found, err := loadProjectEnv(in.Env)
				if err != nil {
					return err
				}
				opts.secrets, envMissing = secrets, !found
			}
			var err error
			reqs, err = listProjectRequests(targets[name], opts)
			return err
		})
		if err != nil {
			return nil, listRequestsOutput{}, err
		}
		if envMissing {
			missing++
		}
		if len(reqs) == 0 {
			continue
		}
		out.Projects = append(out.Projects, ProjectRequests{
			Name:       name,
			Root:       targets[name],
			EnvMissing: envMissing,
			Requests:   reqs,
		})
	}
	if in.Env != "" && missing == len(targets) {
		return nil, listRequestsOutput{}, fmt.Errorf(
			"env %q not found in any project (%s); list_envs shows the available names",
			in.Env, strings.Join(projectNames(targets), ", "),
		)
	}
	return nil, out, nil
}

// loadProjectEnv loads env's secrets for the project in the working directory.
// found is false when the project has no such environment.
func loadProjectEnv(env string) (secrets map[string]any, found bool, err error) {
	envs, err := envparser.ListEnvironments()
	if err != nil {
		return nil, false, err
	}
	if !slices.Contains(envs, env) {
		return nil, false, nil
	}
	secrets, err = envparser.ReadSecretsMap(env)
	return secrets, err == nil, err
}

type listOptions struct {
	filter  string
	detail  bool
	secrets map[string]any
}

// listProjectRequests returns a summary of every request file under root
// whose project-relative path contains opts.filter, ignoring case.
func listProjectRequests(root string, opts listOptions) ([]RequestSummary, error) {
	files, err := utils.ListFiles(root)
	if err != nil {
		return nil, err
	}
	// Deps resolve under the cwd's project root, which can differ from root by symlinks.
	depRoot, ok := utils.FindProjectRoot()
	if !ok {
		depRoot = root
	}
	var out []RequestSummary
	for _, f := range files {
		rel := projectRelative(root, f)
		if !utils.IsRequestFile(filepath.Base(f)) || !pathMatches(rel, opts.filter) {
			continue
		}
		// Deps are best-effort: a missing/unreadable referenced file should
		// not drop the request from the listing.
		deps, _ := utils.ReferencedFiles(f)
		for i, d := range deps {
			deps[i] = projectRelative(depRoot, d)
		}
		// Metadata is best-effort too: an unparseable file is still listed.
		doc, _ := readRequestDoc(f)
		url, _ := doc["url"].(string)
		summary := RequestSummary{
			Name: utils.RequestStem(filepath.Base(f)),
			Path: rel,
			Kind: requestKind(f),
			Host: requestHost(url, f, opts.secrets),
			Deps: deps,
		}
		if opts.detail {
			summary.Auth = requestAuth(summary.Kind, doc)
			summary.EnvVars, summary.Variables, _ = utils.RequestVariables(f)
		}
		out = append(out, summary)
	}
	return out, nil
}

// requestHost returns rawURL's host, with its template resolved against
// secrets when given. A host that fails to resolve, or whose resolved form is
// not certain to exclude userinfo, stays as written.
func requestHost(rawURL, file string, secrets map[string]any) string {
	host := rawHost(rawURL)
	if secrets == nil || !strings.Contains(host, "{{") {
		return host
	}
	resolved, err := envparser.SubstituteVariables(host, secrets, file)
	if text, ok := resolved.(string); err == nil && ok {
		if h, ok := resolvedHost(text); ok {
			return h
		}
	}
	return host
}

// resolvedHost returns the host of a resolved url. ok is false when the url
// does not parse, has no host, or the host does not directly follow its last
// '@', since an unencoded '/', '?' or '#' in a password ends the authority
// early and would put part of it in the host.
func resolvedHost(text string) (string, bool) {
	if !strings.Contains(text, "://") {
		text = "//" + text
	}
	u, err := url.Parse(text)
	if err != nil || u.Host == "" {
		return "", false
	}
	if at := strings.LastIndex(text, "@"); at >= 0 && !strings.HasPrefix(text[at+1:], u.Host) {
		return "", false
	}
	return u.Host, true
}

func pathMatches(rel, filter string) bool {
	return strings.Contains(
		strings.ToLower(filepath.ToSlash(rel)),
		strings.ToLower(filepath.ToSlash(filter)),
	)
}

// projectRelative returns path relative to root, or path unchanged when it
// lies outside root.
func projectRelative(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return path
	}
	return rel
}

// requestKind returns the file's kind (API/GraphQL/Auth), best-effort: "" when
// it can't be read. PeekRequestKind reads only the kind field and the body
// shape, so template vars and getFile references do not block the listing.
func requestKind(path string) string {
	k, err := yamlparser.PeekRequestKind(path)
	if err != nil {
		return ""
	}
	return string(k)
}
