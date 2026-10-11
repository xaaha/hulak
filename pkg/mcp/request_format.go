package mcp

import (
	"context"
	_ "embed"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

//go:embed authoring.md
var authoringGuide string

type requestFormatInput struct {
	Schema bool `json:"schema,omitempty" jsonschema:"also return the full request JSON Schema (about 11 KB)"`
}

type requestFormatOutput struct {
	Guide  string `json:"guide"`
	Schema string `json:"schema,omitempty"`
}

func (s *Server) registerRequestFormat() {
	mcpsdk.AddTool(s.srv, &mcpsdk.Tool{
		Name: "request_format",
		Description: "How to write a hulak request file: the YAML shape, body types, " +
			"template actions ({{.var}}, getValueOf, getFile, attachFile, basicAuth, os) " +
			"with their argument order, and worked examples, including passing a " +
			"token from one request to the next. Read it before write_request. " +
			"Pass `schema` to also get the JSON Schema that write_request validates against.",
		Annotations: &mcpsdk.ToolAnnotations{ReadOnlyHint: true},
	}, s.handleRequestFormat)
}

func (s *Server) handleRequestFormat(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	in requestFormatInput,
) (*mcpsdk.CallToolResult, requestFormatOutput, error) {
	out := requestFormatOutput{Guide: authoringGuide}
	if in.Schema {
		if s.reqSchemaJSON == nil {
			return nil, requestFormatOutput{}, fmt.Errorf("no request schema is configured")
		}
		out.Schema = string(s.reqSchemaJSON)
	}
	return nil, out, nil
}
