// Package mcpsrv is the primary (driving) adapter for the MCP protocol using the official Go SDK.
// Core domains and ports stay free of github.com/modelcontextprotocol/go-sdk; composition happens in cmd.
package mcpsrv

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewServer builds an MCP [mcp.Server] with no registered tools yet. Tools register here or via helpers
// in this package once application ports are wired.
//
// opts may set [mcp.ServerOptions.Instructions] (e.g. persistence policy text from config.MCPPolicyInstructions).
func NewServer(name, version string, opts *mcp.ServerOptions) *mcp.Server {
	return mcp.NewServer(&mcp.Implementation{
		Name:    name,
		Version: version,
	}, opts)
}

// ServerInstructions combines operator instructions with mutation recovery
// guidance sent to clients at initialize.
func ServerInstructions(instructions string) *mcp.ServerOptions {
	const mutationRecovery = "If a mutation reports a durable-commit or uncertain finalization error, reconcile the database before retrying. A failed response does not prove that nothing was saved. In particular, do not blindly retry near-anchor placement: search by source_id and content to locate a possible committed cell first."
	if instructions != "" {
		instructions += "\n\n"
	}
	return &mcp.ServerOptions{Instructions: instructions + mutationRecovery}
}

// StreamableHTTPHandler exposes the MCP server over MCP Streamable HTTP.
// Responses to POST may use SSE. Standalone GET streams are unnecessary for
// this fixed tool inventory and would keep HTTP shutdown waiting indefinitely.
//
// Clients send JSON-RPC to the configured listen address and path (see MOSAIC_MCP_ADDR / MOSAIC_MCP_PATH).
func StreamableHTTPHandler(srv *mcp.Server, log *slog.Logger) http.Handler {
	opts := &mcp.StreamableHTTPOptions{SessionTimeout: 30 * time.Minute}
	if log != nil {
		opts.Logger = log
	}
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return srv }, opts)
	protected := http.NewCrossOriginProtection().Handler(handler)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodDelete {
			w.Header().Set("Allow", "POST, DELETE")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		protected.ServeHTTP(w, r)
	})
}
