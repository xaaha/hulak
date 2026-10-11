package mcp

import (
	"context"
	"fmt"
	"os"
	"strings"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/xaaha/hulak/pkg/utils"
)

type readRequestInput struct {
	Name    string `json:"name"              jsonschema:"request name, e.g. login (with or without extension)"`
	Project string `json:"project,omitempty" jsonschema:"project to search; omit to search all projects"`
}

// DependencyFile is a file a request pulls in with getFile; Missing means it was not read.
type DependencyFile struct {
	Path    string `json:"path"`
	Content string `json:"content,omitempty"`
	Missing bool   `json:"missing,omitempty"`
}

type readRequestOutput struct {
	Project string           `json:"project"`
	Path    string           `json:"path"`
	Content string           `json:"content"`
	Deps    []DependencyFile `json:"deps,omitempty"`
}

func (s *Server) registerReadRequest() {
	mcpsdk.AddTool(s.srv, &mcpsdk.Tool{
		Name: "read_request",
		Description: "Read a request file as written, plus every file it pulls in " +
			"with getFile (e.g. the .gql query of a GraphQL request). Use it to see " +
			"what a request sends and which {{.variables}} it needs before dry_run " +
			"or call_request, or to copy an existing request as the starting point " +
			"for write_request. Env values are not resolved. A dependency that is " +
			"missing or outside the project is listed with missing: true.",
		Annotations: &mcpsdk.ToolAnnotations{ReadOnlyHint: true},
	}, s.handleReadRequest)
}

func (s *Server) handleReadRequest(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	in readRequestInput,
) (*mcpsdk.CallToolResult, readRequestOutput, error) {
	var out readRequestOutput
	if strings.TrimSpace(in.Name) == "" {
		return nil, out, fmt.Errorf("name is required")
	}
	m, err := s.ResolveRequest(in.Project, in.Name)
	if err != nil {
		return nil, out, err
	}
	root := s.projects[m.Project]
	err = s.withProjectDir(root, func() error {
		content, err := os.ReadFile(m.Path)
		if err != nil {
			return err
		}
		out = readRequestOutput{
			Project: m.Project,
			Path:    projectRelative(root, m.Path),
			Content: string(content),
		}
		out.Deps, err = readDependencies(m.Path)
		return err
	})
	if err != nil {
		return nil, readRequestOutput{}, err
	}
	return nil, out, nil
}

// readDependencies must run inside the project dir: getFile paths resolve against its root.
func readDependencies(path string) ([]DependencyFile, error) {
	refs, err := utils.ReferencedFiles(path)
	if err != nil {
		return nil, err
	}
	depRoot, ok := utils.FindProjectRoot()
	if !ok {
		return nil, fmt.Errorf("no project root above %s", path)
	}
	var deps []DependencyFile
	for _, ref := range refs {
		dep := DependencyFile{Path: projectRelative(depRoot, ref)}
		resolved, err := utils.ResolveProjectFile(ref)
		if err != nil {
			dep.Missing = true
			deps = append(deps, dep)
			continue
		}
		content, err := os.ReadFile(resolved)
		if err != nil {
			return nil, err
		}
		dep.Content = string(content)
		deps = append(deps, dep)
	}
	return deps, nil
}
