package mcp

import (
	"context"
	"fmt"
	"strings"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/xaaha/hulak/pkg/envparser"
)

type listEnvsInput struct {
	Project string `json:"project,omitempty" jsonschema:"limit to this project; omit to list every project"`
}

// projectEnvs holds the environment names available in one project. Envs
// differ per project, so results are grouped rather than flattened.
type projectEnvs struct {
	Project string   `json:"project"`
	Envs    []string `json:"envs"`
}

type listEnvsOutput struct {
	Environments []projectEnvs `json:"environments"`
}

// registerListEnvs adds the list_envs tool: list the environment names a
// request can be run against. Names only — no secret values are read.
func (s *Server) registerListEnvs() {
	mcpsdk.AddTool(s.srv, &mcpsdk.Tool{
		Name: "list_envs",
		Description: "List the environment names available in each project (e.g. " +
			"global, staging, prod). Use one of these as the `env` argument to " +
			"dry_run or call_request. Returns names only — never secret values. " +
			"Omit `project` to list every configured project.",
		Annotations: &mcpsdk.ToolAnnotations{ReadOnlyHint: true},
	}, s.handleListEnvs)
}

func (s *Server) handleListEnvs(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	in listEnvsInput,
) (*mcpsdk.CallToolResult, listEnvsOutput, error) {
	targets := s.projects
	if in.Project != "" {
		path, ok := s.projects[in.Project]
		if !ok {
			return nil, listEnvsOutput{}, fmt.Errorf(
				"unknown project %q; configured projects: %s",
				in.Project, strings.Join(projectNames(s.projects), ", "),
			)
		}
		targets = map[string]string{in.Project: path}
	}

	var out listEnvsOutput
	for _, name := range projectNames(targets) {
		var envs []string
		// Enumeration is cwd-dependent (vault detection and env/ lookup both key
		// off the working directory), so it must run inside the project dir.
		err := s.withProjectDir(targets[name], func() error {
			var err error
			envs, err = envparser.ListEnvironments()
			return err
		})
		if err != nil {
			return nil, listEnvsOutput{}, fmt.Errorf("project %q: %w", name, err)
		}
		out.Environments = append(out.Environments, projectEnvs{Project: name, Envs: envs})
	}
	return nil, out, nil
}
