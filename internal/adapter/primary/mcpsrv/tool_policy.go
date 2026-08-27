package mcpsrv

import (
	"fmt"
	"slices"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type mcpToolPolicy struct {
	readOnly    bool
	destructive bool
	idempotent  bool
	openWorld   bool
}

// mcpToolPolicies is the explicit safety classification for every Mosaic tool.
// addTool rejects unclassified tools at startup, and MutationToolNames derives
// Ratchet's required mutation coverage from this same inventory.
var mcpToolPolicies = map[string]mcpToolPolicy{
	"mosaic_hexxla_health":                        {readOnly: true},
	"mosaic_hexxla_query_cells":                   {readOnly: true, openWorld: true},
	"mosaic_hexxla_search_cells":                  {readOnly: true, openWorld: true},
	"mosaic_hexxla_search_embedding":              {readOnly: true, openWorld: true},
	"mosaic_hexxla_load_context_pack":             {readOnly: true},
	"mosaic_hexxla_estimate_context_budget_bytes": {readOnly: true},
	"mosaic_hexxla_retrieval_budget_status":       {readOnly: true},
	"mosaic_hexxla_get_persistence_policy":        {readOnly: true},
	"mosaic_hexxla_list_tags":                     {readOnly: true},
	"mosaic_hexxla_tag_counts":                    {readOnly: true},
	"mosaic_hexxla_find_seams":                    {readOnly: true},
	"mosaic_hexxla_get_facet":                     {readOnly: true},
	"mosaic_hexxla_list_facets":                   {readOnly: true},
	"mosaic_hexxla_get_edge":                      {readOnly: true},
	"mosaic_hexxla_list_edges_from":               {readOnly: true},

	"mosaic_hexxla_put_cell":        {destructive: true},
	"mosaic_hexxla_put_embedding":   {destructive: true, openWorld: true},
	"mosaic_hexxla_delete_cell":     {destructive: true, idempotent: true},
	"mosaic_hexxla_mark_conflict":   {},
	"mosaic_hexxla_mark_supersedes": {},
	"mosaic_hexxla_resolve_seam":    {destructive: true},
	"mosaic_hexxla_put_facet":       {destructive: true},
	"mosaic_hexxla_link_cells":      {destructive: true},
}

func addTool[In, Out any](server *mcp.Server, tool *mcp.Tool, handler mcp.ToolHandlerFor[In, Out]) error {
	policy, ok := mcpToolPolicies[tool.Name]
	if !ok {
		return fmt.Errorf("MCP tool %q has no explicit safety policy", tool.Name)
	}
	if tool.Annotations != nil {
		return fmt.Errorf("MCP tool %q sets annotations outside the safety policy inventory", tool.Name)
	}

	openWorld := policy.openWorld
	tool.Annotations = &mcp.ToolAnnotations{
		DestructiveHint: &policy.destructive,
		IdempotentHint:  policy.idempotent,
		OpenWorldHint:   &openWorld,
		ReadOnlyHint:    policy.readOnly,
	}
	mcp.AddTool(server, tool, handler)
	return nil
}

func registerTools(registrations ...func() error) error {
	for _, register := range registrations {
		if err := register(); err != nil {
			return fmt.Errorf("register MCP tool: %w", err)
		}
	}
	return nil
}

// MutationToolNames returns the sorted tool names that can modify state.
func MutationToolNames() []string {
	names := make([]string, 0, len(mcpToolPolicies))
	for name, policy := range mcpToolPolicies {
		if !policy.readOnly {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names
}
