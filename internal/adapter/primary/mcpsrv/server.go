// Package mcpsrv is the primary (driving) adapter for the MCP protocol using the official Go SDK.
// Core domains and ports stay free of github.com/modelcontextprotocol/go-sdk; composition happens in cmd.
package mcpsrv

import (
	"log/slog"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewServer builds an MCP [mcp.Server] with no registered tools yet. Tools register here or via helpers
// in this package once application ports are wired.
func NewServer(name, version string) *mcp.Server {
	return mcp.NewServer(&mcp.Implementation{
		Name:    name,
		Version: version,
	}, nil)
}

// StreamableHTTPHandler exposes the MCP server over MCP Streamable HTTP (JSON-RPC over HTTP POST/SSE GET).
//
// Clients send JSON-RPC to the configured listen address and path (see MOSAIC_MCP_ADDR / MOSAIC_MCP_PATH).
func StreamableHTTPHandler(srv *mcp.Server, log *slog.Logger) http.Handler {
	opts := &mcp.StreamableHTTPOptions{}
	if log != nil {
		opts.Logger = log
	}
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return srv }, opts)
}
