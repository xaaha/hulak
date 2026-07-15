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
type RequestSummary struct {
	Name    string   `json:"name"`
	Project string   `json:"project"`
	Path    string   `json:"path"`
	Kind    string   `json:"kind,omitempty"`
	Deps    []string `json:"deps,omitempty"` // referenced files, e.g. a GraphQL .gql
}

type listRequestsInput struct {
	Project string `json:"project,omitempty" jsonschema:"limit to this project; omit to list every project"`
}

type listRequestsOutput struct {
	Requests []RequestSummary `json:"requests"`
}

// registerListRequests adds the list_requests tool to the server.
func (s *Server) registerListRequests() {
	mcpsdk.AddTool(s.srv, &mcpsdk.Tool{
		Name: "list_requests",
		Description: "List hulak request files. Each entry has its name, project, " +
			"file path, kind (API/GraphQL), and any dependency files it references " +
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
		reqs, err := listProjectRequests(name, targets[name])
		if err != nil {
			return nil, listRequestsOutput{}, err
		}
		out.Requests = append(out.Requests, reqs...)
	}
	return nil, out, nil
}

// listProjectRequests returns a summary of every request file under root.
func listProjectRequests(project, root string) ([]RequestSummary, error) {
	files, err := utils.ListFiles(root)
	if err != nil {
		return nil, err
	}
	var out []RequestSummary
	for _, f := range files {
		if !utils.IsRequestFile(filepath.Base(f)) {
			continue
		}
		// Deps are best-effort: a missing/unreadable referenced file should
		// not drop the request from the listing.
		deps, _ := utils.ReferencedFiles(f)
		out = append(out, RequestSummary{
			Name:    utils.RequestStem(filepath.Base(f)),
			Project: project,
			Path:    f,
			Kind:    requestKind(f),
			Deps:    deps,
		})
	}
	return out, nil
}

// requestKind returns the file's kind (API/GraphQL/Auth), best-effort: "" when
// it can't be read. PeekKind reads only the kind field, so template vars and
// getFile references do not block the listing.
func requestKind(path string) string {
	k, err := yamlparser.PeekKind(path)
	if err != nil {
		return ""
	}
	return string(k)
}
