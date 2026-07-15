package mcp

import (
	"context"
	"fmt"
	"strings"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	apicalls "github.com/xaaha/hulak/pkg/apiCalls"
	"github.com/xaaha/hulak/pkg/envparser"
)

type callRequestInput struct {
	Name    string `json:"name"              jsonschema:"request name, e.g. login (with or without extension)"`
	Env     string `json:"env"               jsonschema:"environment to resolve secrets against, e.g. staging (required)"`
	Project string `json:"project,omitempty" jsonschema:"project to search; omit to search all projects"`
	NoSave  bool   `json:"no_save,omitempty" jsonschema:"do not write the {name}_response.json file; return the response only"`
}

type callRequestOutput struct {
	Project string `json:"project"`
	Path    string `json:"path"`
	Status  string `json:"status"` // e.g. "200 OK"
	Body    string `json:"body"`
}

// registerCallRequest adds the call_request tool: send a request and return
// the response. Marked destructive — it performs a real network call.
func (s *Server) registerCallRequest() {
	destructive := true
	mcpsdk.AddTool(s.srv, &mcpsdk.Tool{
		Name: "call_request",
		Description: "Send a hulak request against an environment and return the " +
			"response status and body. This performs a real network call. The " +
			"response is saved as {name}_response.json unless no_save is true.",
		Annotations: &mcpsdk.ToolAnnotations{DestructiveHint: &destructive},
	}, s.handleCallRequest)
}

func (s *Server) handleCallRequest(
	ctx context.Context,
	_ *mcpsdk.CallToolRequest,
	in callRequestInput,
) (*mcpsdk.CallToolResult, callRequestOutput, error) {
	var out callRequestOutput
	if strings.TrimSpace(in.Env) == "" {
		return nil, out, fmt.Errorf(`env is required (e.g. "global" or "staging")`)
	}

	m, err := s.ResolveRequest(in.Project, in.Name)
	if err != nil {
		return nil, out, err
	}

	var body, status string
	err = s.withProjectDir(s.projects[m.Project], func() error {
		secrets, err := envparser.LoadSecretsMap(in.Env)
		if err != nil {
			return err
		}
		respBytes, st, err := apicalls.SendAndSaveAPIRequest(ctx, apicalls.RequestOptions{
			Secrets: secrets,
			Path:    m.Path,
			NoSave:  in.NoSave,
		})
		if err != nil {
			return err
		}
		body, status = string(respBytes), st
		return nil
	})
	if err != nil {
		return nil, out, err
	}

	return nil, callRequestOutput{Project: m.Project, Path: m.Path, Status: status, Body: body}, nil
}
