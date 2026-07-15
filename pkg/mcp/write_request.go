package mcp

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/xaaha/hulak/pkg/utils"
)

type writeRequestInput struct {
	Name        string `json:"name"                jsonschema:"file name, e.g. login or auth/login (a .hk.yaml extension is added when omitted)"`
	YamlContent string `json:"yaml_content"        jsonschema:"the request YAML to write"`
	Project     string `json:"project,omitempty"   jsonschema:"project to write into; required when more than one is configured"`
	Overwrite   bool   `json:"overwrite,omitempty" jsonschema:"overwrite an existing file (default false)"`
}

type writeRequestOutput struct {
	Project     string `json:"project"`
	Path        string `json:"path"`
	Overwritten bool   `json:"overwritten"`
}

// registerWriteRequest adds the write_request tool: create or overwrite a
// request file. Destructive — it writes to disk.
func (s *Server) registerWriteRequest() {
	destructive := true
	mcpsdk.AddTool(s.srv, &mcpsdk.Tool{
		Name: "write_request",
		Description: "Create or overwrite a hulak request file in a project. The " +
			"content must be valid YAML. Refuses to overwrite an existing file " +
			"unless overwrite is true. A .hk.yaml extension is added when the name " +
			"has none.",
		Annotations: &mcpsdk.ToolAnnotations{DestructiveHint: &destructive},
	}, s.handleWriteRequest)
}

func (s *Server) handleWriteRequest(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	in writeRequestInput,
) (*mcpsdk.CallToolResult, writeRequestOutput, error) {
	var out writeRequestOutput
	if strings.TrimSpace(in.Name) == "" {
		return nil, out, fmt.Errorf("name is required")
	}
	if strings.TrimSpace(in.YamlContent) == "" {
		return nil, out, fmt.Errorf("yaml_content is required")
	}
	if err := s.validateRequestContent(in.YamlContent); err != nil {
		return nil, out, err
	}

	projName, root, err := s.resolveProjectRoot(in.Project)
	if err != nil {
		return nil, out, err
	}
	dest, err := destPath(root, in.Name)
	if err != nil {
		return nil, out, err
	}

	existed := utils.FileExists(dest)
	if existed && !in.Overwrite {
		return nil, out, fmt.Errorf(
			"%s already exists; pass overwrite=true to replace it", dest,
		)
	}

	if err := utils.AtomicWriteFile(dest, []byte(in.YamlContent), utils.FilePer, utils.DirPer); err != nil {
		return nil, out, err
	}
	return nil, writeRequestOutput{Project: projName, Path: dest, Overwritten: existed}, nil
}

// resolveProjectRoot picks the project to write into. An explicit project is
// used as given; otherwise the sole project is used, and an ambiguous omission
// errors — a write must never guess its target.
func (s *Server) resolveProjectRoot(project string) (string, string, error) {
	if project != "" {
		root, ok := s.projects[project]
		if !ok {
			return "", "", fmt.Errorf(
				"unknown project %q; configured projects: %s",
				project, strings.Join(projectNames(s.projects), ", "),
			)
		}
		return project, root, nil
	}
	names := projectNames(s.projects)
	if len(names) == 1 {
		return names[0], s.projects[names[0]], nil
	}
	return "", "", fmt.Errorf(
		"multiple projects configured; pass `project` to choose where to write: %s",
		strings.Join(names, ", "),
	)
}

// destPath resolves name to a path inside root, appending .hk.yaml when the
// name has no request extension. Rejects absolute paths and traversal outside
// the project.
func destPath(root, name string) (string, error) {
	rel := name
	if !utils.IsRequestFile(rel) {
		rel += utils.ProjectExt + utils.YAML
	}
	clean := filepath.Clean(rel)
	if filepath.IsAbs(clean) || clean == ".." ||
		strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("name %q must stay within the project", name)
	}
	return filepath.Join(root, clean), nil
}

// validateRequestContent rejects content that isn't a valid hulak request
// before it lands on disk. Always checks it's a non-empty YAML mapping; when a
// request schema is configured, also validates against it (required method/url,
// known kind/method, graphql shape, etc.) so the agent can't write garbage or
// a hallucinated shape.
func (s *Server) validateRequestContent(content string) error {
	var doc any
	if err := yaml.Unmarshal([]byte(content), &doc); err != nil {
		return fmt.Errorf("yaml_content is not valid YAML: %w", err)
	}
	m, ok := doc.(map[string]any)
	if !ok || len(m) == 0 {
		return fmt.Errorf("yaml_content must be a non-empty YAML mapping")
	}
	if s.reqSchema != nil {
		if err := s.reqSchema.Validate(doc); err != nil {
			return fmt.Errorf("request does not match the hulak schema: %w", err)
		}
	}
	return nil
}
