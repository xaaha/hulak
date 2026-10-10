package mcp

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

// RequestSummary describes one request file for the list_requests tool.
// Path and Deps are relative to the project root when they live inside it.
type RequestSummary struct {
	Name string   `json:"name"`
	Path string   `json:"path"`
	Kind string   `json:"kind,omitempty"`
	Deps []string `json:"deps,omitempty"` // referenced files, e.g. a GraphQL .gql
}

// ProjectRequests is one project's request files and the root their paths are
// relative to.
type ProjectRequests struct {
	Name     string           `json:"name"`
	Root     string           `json:"root"`
	Requests []RequestSummary `json:"requests"`
}

type listRequestsInput struct {
	Project string `json:"project,omitempty" jsonschema:"limit to this project; omit to list every project"`
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
			"that root, kind (API/GraphQL), and any dependency files it references " +
			"(e.g. a GraphQL query .gql that lives next to the request). Omit " +
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

	var out listRequestsOutput
	for _, name := range projectNames(targets) {
		var reqs []RequestSummary
		// Run inside the project dir: dependency resolution (utils.ReferencedFiles
		// -> getFile) is project-root-relative and keys off the working
		// directory, exactly as a real run does.
		err := s.withProjectDir(targets[name], func() error {
			var err error
			reqs, err = listProjectRequests(targets[name])
			return err
		})
		if err != nil {
			return nil, listRequestsOutput{}, err
		}
		out.Projects = append(out.Projects, ProjectRequests{
			Name:     name,
			Root:     targets[name],
			Requests: reqs,
		})
	}
	return nil, out, nil
}

// listProjectRequests returns a summary of every request file under root.
func listProjectRequests(root string) ([]RequestSummary, error) {
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
		if !utils.IsRequestFile(filepath.Base(f)) {
			continue
		}
		// Deps are best-effort: a missing/unreadable referenced file should
		// not drop the request from the listing.
		deps, _ := utils.ReferencedFiles(f)
		for i, d := range deps {
			deps[i] = projectRelative(depRoot, d)
		}
		out = append(out, RequestSummary{
			Name: utils.RequestStem(filepath.Base(f)),
			Path: projectRelative(root, f),
			Kind: requestKind(f),
			Deps: deps,
		})
	}
	return out, nil
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
