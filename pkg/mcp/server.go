// Package mcp adapts hulak's request files to the Model Context Protocol,
// exposing them to AI agents over JSON-RPC on stdio. It is an adapter layer
// only: tools wrap existing hulak packages and reimplement no HTTP, secret,
// or parsing logic.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/google/jsonschema-go/jsonschema"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/xaaha/hulak/pkg/utils"
)

// Server is a hulak MCP server over one or more named projects from user's config.
// Every tool resolves a request within a project; when the caller omits the project,
// the resolver searches all of them and reports ambiguity rather than guessing.
type Server struct {
	projects       map[string]string // name -> absolute project directory
	defaultProject string            // name in projects, or "" if unset
	srv            *mcpsdk.Server
	reqSchema      *jsonschema.Resolved // request-file schema; nil = skip schema validation
	mu             sync.Mutex           // serializes withProjectDir
}

// SetRequestSchema compiles schemaJSON (the hulak request JSON Schema) and uses
// it to validate write_request content. Best-effort: an empty or unparseable
// schema is skipped with a warning rather than failing startup — write_request
// then falls back to a basic YAML-mapping check.
func (s *Server) SetRequestSchema(schemaJSON []byte) {
	if len(schemaJSON) == 0 {
		return
	}
	var sc jsonschema.Schema
	if err := json.Unmarshal(schemaJSON, &sc); err != nil {
		utils.PrintWarningStderr("mcp: request schema ignored (unparseable): " + err.Error())
		return
	}
	resolved, err := sc.Resolve(nil)
	if err != nil {
		utils.PrintWarningStderr("mcp: request schema ignored (unresolvable): " + err.Error())
		return
	}
	s.reqSchema = resolved
}

// Match is a resolved request: the file and the project it was found in.
type Match struct {
	Project string
	Path    string
}

// NewServer builds an MCP server over the named projects. Paths are ~- and abs-expanded.
// Errors when projects is empty, a path can't be resolved, or
// defaultProject (when set) is not one of the projects.
func NewServer(projects map[string]string, defaultProject, version string) (*Server, error) {
	if len(projects) == 0 {
		return nil, fmt.Errorf("at least one project is required")
	}
	resolved := make(map[string]string, len(projects))
	for name, path := range projects {
		abs, err := utils.ExpandPath(path)
		if err != nil {
			return nil, fmt.Errorf("project %q: %w", name, err)
		}
		resolved[name] = abs
	}
	if err := validateProjectDirs(resolved); err != nil {
		return nil, err
	}
	if err := validateDefaultProject(defaultProject, resolved); err != nil {
		return nil, err
	}

	srv := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    utils.ProjectName,
		Version: version,
	}, nil)
	s := &Server{projects: resolved, defaultProject: defaultProject, srv: srv}
	s.registerListRequests()
	s.registerListEnvs()
	s.registerDryRun()
	s.registerCallRequest()
	s.registerWriteRequest()
	return s, nil
}

// withProjectDir runs fn with the process working directory set to root,
// restoring it afterward. Serialized by mu: hulak's secret loading and
// getFile/getValueOf resolution key off the working directory, so only one
// request may hold it at a time.
func (s *Server) withProjectDir(root string, fn func() error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	prev, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := os.Chdir(root); err != nil {
		return fmt.Errorf("entering project %s: %w", root, err)
	}
	defer func() { _ = os.Chdir(prev) }()
	return fn()
}

// Projects returns the resolved name->path map. Used for the startup identity
// precheck.
func (s *Server) Projects() map[string]string { return s.projects }

// validateProjectDirs rejects any project path that is not a hulak project
// (no .hulak/ or env/ marker) — catches a mistyped or wrong --project path at
// startup instead of surfacing empty request lists later.
func validateProjectDirs(resolved map[string]string) error {
	for _, name := range projectNames(resolved) {
		if !utils.IsProjectDir(resolved[name]) {
			return fmt.Errorf(
				"--project %s=%s is not a hulak project: no %s/ or %s/ found there.\n"+
					"Point it at a hulak project directory, or run 'hulak init' there first",
				name, resolved[name], utils.HiddenProjectName, utils.EnvironmentFolder,
			)
		}
	}
	return nil
}

// validateDefaultProject checks that --default-project names one of the
// configured --project entries.
func validateDefaultProject(defaultProject string, resolved map[string]string) error {
	if defaultProject == "" {
		return nil
	}
	if _, ok := resolved[defaultProject]; ok {
		return nil
	}
	return fmt.Errorf(
		"--default-project %q does not match any --project name.\n"+
			"Configured project names: %s",
		defaultProject, strings.Join(projectNames(resolved), ", "),
	)
}

// ResolveRequest locates the request file named name. When project is given,
// it searches only that project; otherwise it searches every project and
// treats a name present in more than one as ambiguous — returning an error
// listing the choices so the client can ask the user and retry with a
// project. The AI must not guess beyond this resolver.
func (s *Server) ResolveRequest(project, name string) (Match, error) {
	searchProjects := projectNames(s.projects)
	if project != "" {
		if _, ok := s.projects[project]; !ok {
			return Match{}, fmt.Errorf(
				"unknown project %q; configured projects: %s",
				project, strings.Join(projectNames(s.projects), ", "),
			)
		}
		searchProjects = []string{project}
	}

	var matches []Match
	for _, proj := range searchProjects {
		files, err := utils.FindRequestFiles(s.projects[proj], name)
		if err != nil {
			return Match{}, err
		}
		for _, file := range files {
			matches = append(matches, Match{Project: proj, Path: file})
		}
	}

	switch len(matches) {
	case 0:
		if project != "" {
			return Match{}, fmt.Errorf("request %q not found in project %q", name, project)
		}
		return Match{}, fmt.Errorf(
			"request %q not found in any project (%s)",
			name, strings.Join(projectNames(s.projects), ", "),
		)
	case 1:
		return matches[0], nil
	default:
		return Match{}, ambiguousError(name, project, matches)
	}
}

// ambiguousError builds the error for a name that matches more than one
// request file. The name may collide across projects, or within a single
// project (same stem in different sub-directories); the hint differs so the
// caller knows whether `project` can disambiguate.
func ambiguousError(name, project string, matches []Match) error {
	var b strings.Builder
	fmt.Fprintf(&b, "request %q is ambiguous; it matches %d files:\n", name, len(matches))
	for _, m := range matches {
		fmt.Fprintf(&b, "  - %s (%s)\n", m.Project, m.Path)
	}
	if project == "" {
		b.WriteString("pass `project` to narrow, or rename so the stem is unique")
	} else {
		b.WriteString("rename so the stem is unique within the project")
	}
	return fmt.Errorf("%s", strings.TrimRight(b.String(), "\n"))
}

// Serve runs the server over stdio until the client disconnects or ctx is
// cancelled.
func (s *Server) Serve(ctx context.Context) error {
	return s.srv.Run(ctx, &mcpsdk.StdioTransport{})
}

// projectNames returns the map keys sorted, for deterministic output.
func projectNames(projects map[string]string) []string {
	names := make([]string, 0, len(projects))
	for n := range projects {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
