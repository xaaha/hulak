package mcp

import (
	"context"
	"fmt"
	"strings"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	apicalls "github.com/xaaha/hulak/pkg/apiCalls"
	"github.com/xaaha/hulak/pkg/envparser"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

type dryRunInput struct {
	Name    string `json:"name"              jsonschema:"request name, e.g. login (with or without extension)"`
	Env     string `json:"env"               jsonschema:"environment to resolve secrets against, e.g. staging (required)"`
	Project string `json:"project,omitempty" jsonschema:"project to search; omit to search all projects"`
	Show    bool   `json:"show,omitempty"    jsonschema:"reveal sensitive headers instead of masking them"`
}

type dryRunOutput struct {
	Project string `json:"project"`
	Path    string `json:"path"`
	Request string `json:"request"` // formatted request that would be sent
}

// registerDryRun adds the dry_run tool: resolve a request against an env and
// show what would be sent, without sending it.
func (s *Server) registerDryRun() {
	mcpsdk.AddTool(s.srv, &mcpsdk.Tool{
		Name: "dry_run",
		Description: "Resolve a request against an environment and return the exact " +
			"request that would be sent (method, URL, headers, body) without sending " +
			"it. Use this to check a request's variables resolve in a given env. " +
			"Sensitive headers are masked unless `show` is true.",
		Annotations: &mcpsdk.ToolAnnotations{ReadOnlyHint: true},
	}, s.handleDryRun)
}

func (s *Server) handleDryRun(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	in dryRunInput,
) (*mcpsdk.CallToolResult, dryRunOutput, error) {
	var out dryRunOutput
	if strings.TrimSpace(in.Env) == "" {
		return nil, out, fmt.Errorf(`env is required (e.g. "global" or "staging")`)
	}

	m, err := s.ResolveRequest(in.Project, in.Name)
	if err != nil {
		return nil, out, err
	}
	if k := readKind(m.Path); strings.EqualFold(k, string(yamlparser.KindAuth)) {
		return nil, out, fmt.Errorf(
			"dry_run supports API and GraphQL requests only; %q is kind %s (OAuth2 is out of scope)",
			in.Name,
			k,
		)
	}

	var text string
	err = s.withProjectDir(s.projects[m.Project], func() error {
		secrets, err := envparser.LoadSecretsMap(in.Env)
		if err != nil {
			return err
		}
		text, err = apicalls.DryRun(apicalls.RequestOptions{
			Secrets: secrets,
			Path:    m.Path,
			Show:    in.Show,
		})
		return err
	})
	if err != nil {
		return nil, out, err
	}

	return nil, dryRunOutput{Project: m.Project, Path: m.Path, Request: text}, nil
}
