package mcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	apicalls "github.com/xaaha/hulak/pkg/apiCalls"
	"github.com/xaaha/hulak/pkg/envparser"
	"github.com/xaaha/hulak/pkg/runner"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

type callRequestInput struct {
	Name    string `json:"name"              jsonschema:"request name, e.g. login (with or without extension)"`
	Env     string `json:"env"               jsonschema:"environment to resolve secrets against, e.g. staging (required)"`
	Project string `json:"project,omitempty" jsonschema:"project to search; omit to search all projects"`
	Save    bool   `json:"save,omitempty"    jsonschema:"also write the response next to the request as a {name}_response file (extension from the response content type, e.g. .json or .txt); off by default (the response is always returned)"`
	Debug   bool   `json:"debug,omitempty"   jsonschema:"return full request, response headers, and TLS details; use to diagnose a failing request"`
	Timeout string `json:"timeout,omitempty" jsonschema:"per-request timeout as a Go duration, e.g. 30s or 2m; a timeout field in the request file wins; default 60s"`
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
			"response is not saved to disk unless save is true, in which case it " +
			"is written next to the request as a {name}_response file whose " +
			"extension follows the response content type (e.g. .json or .txt).",
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

	timeout, err := resolveCallTimeout(in.Timeout, m.Path)
	if err != nil {
		return nil, out, err
	}

	var body, status string
	err = s.withProjectDir(s.projects[m.Project], func() error {
		secrets, err := envparser.LoadSecretsMap(in.Env)
		if err != nil {
			return err
		}
		callCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		respBytes, st, err := apicalls.SendAndSaveAPIRequest(callCtx, apicalls.RequestOptions{
			Secrets: secrets,
			Path:    m.Path,
			// Agents default to no-save so they don't litter the repo with
			// response files; saving is opt-in. The CLI stays save-by-default.
			NoSave: !in.Save,
			Debug:  in.Debug,
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

// resolveCallTimeout mirrors `hulak run`: the request file's own `timeout:`
// field wins, then the tool's timeout arg, then HULAK_TIMEOUT / the 60s
// default. Guarantees every call is bounded so a hung endpoint can't block the
// server forever.
func resolveCallTimeout(arg, path string) (time.Duration, error) {
	if cfg, err := yamlparser.PeekConfig(path); err == nil {
		if d, err := cfg.ParsedTimeout(); err == nil && d > 0 {
			return d, nil
		}
	}
	var flagT time.Duration
	if strings.TrimSpace(arg) != "" {
		d, err := time.ParseDuration(arg)
		if err != nil {
			return 0, fmt.Errorf("invalid timeout %q: %w", arg, err)
		}
		if d <= 0 {
			return 0, fmt.Errorf("timeout must be positive, got %q", arg)
		}
		flagT = d
	}
	return runner.ResolveBaseTimeout(flagT)
}
