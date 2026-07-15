// Package mcp adapts hulak's request files to the Model Context Protocol,
// exposing them to AI agents over JSON-RPC on stdio. It is an adapter layer
// only: tools wrap existing hulak packages and reimplement no HTTP, secret,
// or parsing logic.
package mcp

import (
	"context"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/xaaha/hulak/pkg/utils"
)

// Server is a hulak MCP server bound to a single project directory. Tools
// resolve request files relative to dir.
type Server struct {
	dir string
	srv *mcpsdk.Server
}

// NewServer builds an MCP server rooted at dir. It registers no tools yet —
// the handshake works, but the tool list is empty.
func NewServer(dir, version string) *Server {
	srv := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    utils.ProjectName,
		Version: version,
	}, nil)
	return &Server{dir: dir, srv: srv}
}

// Serve runs the server over stdio until the client disconnects or ctx is
// cancelled.
func (s *Server) Serve(ctx context.Context) error {
	return s.srv.Run(ctx, &mcpsdk.StdioTransport{})
}
