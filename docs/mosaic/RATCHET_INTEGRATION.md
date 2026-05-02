# MCP-Ratchet Integration with Mosaic

**Status:** Design proposal for integrating mcp-ratchet into mosaic-mcp to enforce tool call order and workflow compliance.

**Purpose:** This document describes how mcp-ratchet can be integrated into the Mosaic MCP server to enforce correct tool usage patterns, guide LLMs through recommended workflows, and improve overall agent reliability when using Mosaic.

**Design Philosophy:** Maximum compliance and guardrails - `one_time_use: true` for state-dependent operations to ensure fresh checks, short expiry times (1-2 minutes) for current state verification, and strong enforcement for all critical workflows.

---

## What is mcp-ratchet?

[mcp-ratchet](https://github.com/hexxla/mcp-ratchet) is a Go package for enforcing tool call order in MCP servers using configurable token-based dependencies. It ensures that LLMs follow specific workflows by:

- Requiring tokens from prerequisite tools before dependent tools can be called
- Issuing tokens after successful tool execution
- Supporting token expiration and one-time use
- Providing custom error messages to guide LLMs to the correct next step
- Detecting circular dependencies in rule configurations

---

## Why Integrate mcp-ratchet with Mosaic?

### Current Challenges

Mosaic's MCP tools provide powerful capabilities for structured agent memory, but LLMs often:

1. **Skip retrieval steps** - Call `load_context_pack` without first calling retrieval tools, resulting in empty or incomplete context
2. **Ignore tag vocabulary** - Create new tags with `put_cell` without consulting existing tags via `list_tags`, leading to inconsistent taxonomy
3. **Miss health checks** - Skip `health` checks before heavy operations, potentially encountering database issues
4. **Bypass budget estimation** - Call `load_context_pack` without estimating context budget, potentially exceeding token limits
5. **Call tools in wrong order** - Attempt operations without necessary prerequisites

### Benefits of Integration

**Workflow Enforcement:**

- Ensures agents follow the retrieve-then-assemble pattern (search → load_context_pack)
- Encourages tag reuse by requiring tag listing before writing
- Guides agents through recommended Mosaic workflows with clear error messages

**Improved Reliability:**

- Reduces failed API calls due to missing prerequisites
- Prevents contextless use of critical tools
- Ensures database health checks before heavy operations

**Better Tag Hygiene:**

- Encourages reuse of existing tag vocabulary
- Reduces tag fragmentation and inconsistent taxonomy
- Improves search effectiveness through consistent tagging

**Context Efficiency:**

- Encourages budget estimation before loading context packs
- Reduces wasted token usage from oversized context loads
- Improves overall token efficiency

---

## Tool Dependency Map

### Recommended Dependency Rules

Based on the [MCP Agent Blueprint](./MCP_AGENT_BLUEPRINT.md), the following dependency rules are recommended:

#### Level 1: Discovery Tools (No Prerequisites)

These tools have no prerequisites and can be called at any time. They are explicitly listed in the ratchet configuration for consistency and future scalability. All use `one_time_use: false` with 5-minute expiry to allow token reuse within the session.

- `mosaic_hexxla_health` - Database health check
- `mosaic_hexxla_list_tags` - List all distinct tags
- `mosaic_hexxla_tag_counts` - Per-tag frequency counts
- `mosaic_hexxla_search_embedding` - Semantic ANN search
- `mosaic_hexxla_query_cells` - Structured cell query
- `mosaic_hexxla_search_cells` - Lexical relevance search
- `mosaic_hexxla_estimate_context_budget_bytes` - Budget estimation

#### Level 2: Context Assembly (Requires Retrieval + Health + Budget)

- `mosaic_hexxla_load_context_pack` - Requires (one of each):
  - Retrieval: `mosaic_hexxla_search_embedding` OR `mosaic_hexxla_query_cells` OR `mosaic_hexxla_search_cells` (one-time use)
  - Health: `mosaic_hexxla_health` (one-time use)
  - Budget: `mosaic_hexxla_estimate_context_budget_bytes` (one-time use)

**Rationale:** `load_context_pack` needs fresh seed coordinates from retrieval tools to assemble meaningful context, health check to verify database integrity, and budget estimation to avoid exceeding context limits. All tokens are one-time use to ensure fresh state (expiry is irrelevant for one-time use tokens).

#### Level 3: Write Operations (Strong Tag + Health Requirements)

- `mosaic_hexxla_put_cell` - Required prerequisites:
  - Tag check: `mosaic_hexxla_list_tags` OR `mosaic_hexxla_tag_counts` (one-time use)
  - Health: `mosaic_hexxla_health` (one-time use)

- `mosaic_hexxla_put_embedding` - Required prerequisite:
  - Health: `mosaic_hexxla_health` (one-time use)

**Rationale:** Write operations must verify current tag vocabulary (to prevent fragmentation) and database integrity (to ensure writes succeed). One-time use ensures fresh checks on every write operation.

#### Level 4: Delete Operations (Strong Health Requirement)

- `mosaic_hexxla_delete_cell` - Required prerequisite:
  - Health: `mosaic_hexxla_health` (one-time use)

**Rationale:** Destructive operations require fresh health check to verify database integrity before modification. One-time use ensures database state is current.

#### Level 5: Seam Operations (Strong Health Requirement)

- `mosaic_hexxla_mark_conflict` - Required prerequisite:
  - Health: `mosaic_hexxla_health` (one-time use)

- `mosaic_hexxla_mark_supersedes` - Required prerequisite:
  - Health: `mosaic_hexxla_health` (one-time use)

- `mosaic_hexxla_resolve_seam` - Required prerequisite:
  - Health: `mosaic_hexxla_health` (one-time use)

**Rationale:** Seam operations modify conflict/supersession state and require fresh health check to ensure database integrity and consistent state.

#### Level 6: Facet/Edge Operations (Requires Retrieval)

- `mosaic_hexxla_put_facet` - Required prerequisite:
  - Retrieval: `mosaic_hexxla_search_embedding` OR `mosaic_hexxla_query_cells` OR `mosaic_hexxla_search_cells` (one-time use)

- `mosaic_hexxla_link_cells` - Required prerequisite:
  - Retrieval: `mosaic_hexxla_search_embedding` OR `mosaic_hexxla_query_cells` OR `mosaic_hexxla_search_cells` (one-time use)

**Rationale:** Facet/edge operations require valid cell coordinates from fresh retrieval. One-time use ensures coordinates are current.

#### Level 7: Read-Only Operations (Optional Health Check)

- `mosaic_hexxla_get_facet` - Recommended prerequisite:
  - Health: `mosaic_hexxla_health` (one-time use: false, 5 min expiry)

- `mosaic_hexxla_get_edge` - Recommended prerequisite:
  - Health: `mosaic_hexxla_health` (one-time use: false, 5 min expiry)

**Rationale:** Read operations are less critical but benefit from health checks. Uses longer expiry and reusable tokens since state changes are less impactful for reads.

---

## YAML Configuration

### Maximum Compliance Ratchet Configuration

```yaml
# DESIGN PHILOSOPHY: Maximum compliance and guardrails
# - one_time_use: true for state-dependent operations to ensure fresh checks
# - one_time_use: false only for operations where state is stable within expiry
# - Expiry is ignored when one_time_use: true (token consumed immediately)
# - Strong enforcement for all critical workflows
# - Tools with no prerequisites are explicitly listed for consistency and future scalability

rules:
  # Level 1: Discovery Tools (No Prerequisites)
  # These tools can be called at any time without prerequisites
  # They issue tokens that dependent tools can validate against
  # Listed explicitly for consistency and future scalability

  - tool: "mosaic_hexxla_health"
    prerequisite: ""
    expiry: "5m"
    error_message: ""
    one_time_use: false

  - tool: "mosaic_hexxla_list_tags"
    prerequisite: ""
    expiry: "5m"
    error_message: ""
    one_time_use: false

  - tool: "mosaic_hexxla_tag_counts"
    prerequisite: ""
    expiry: "5m"
    error_message: ""
    one_time_use: false

  - tool: "mosaic_hexxla_search_embedding"
    prerequisite: ""
    expiry: "5m"
    error_message: ""
    one_time_use: false

  - tool: "mosaic_hexxla_query_cells"
    prerequisite: ""
    expiry: "5m"
    error_message: ""
    one_time_use: false

  - tool: "mosaic_hexxla_search_cells"
    prerequisite: ""
    expiry: "5m"
    error_message: ""
    one_time_use: false

  - tool: "mosaic_hexxla_estimate_context_budget_bytes"
    prerequisite: ""
    expiry: "5m"
    error_message: ""
    one_time_use: false

  # Core Retrieval-Assemble Pattern (one_time_use: true - expiry ignored)
  - tool: "mosaic_hexxla_load_context_pack"
    prerequisite: "mosaic_hexxla_search_embedding"
    error_message: "You must call a retrieval tool immediately before calling mosaic_hexxla_load_context_pack to provide fresh seed coordinates. Each context pack requires fresh retrieval."
    one_time_use: true

  - tool: "mosaic_hexxla_load_context_pack"
    prerequisite: "mosaic_hexxla_query_cells"
    error_message: "You must call a retrieval tool immediately before calling mosaic_hexxla_load_context_pack to provide fresh seed coordinates. Each context pack requires fresh retrieval."
    one_time_use: true

  - tool: "mosaic_hexxla_load_context_pack"
    prerequisite: "mosaic_hexxla_search_cells"
    error_message: "You must call a retrieval tool immediately before calling mosaic_hexxla_load_context_pack to provide fresh seed coordinates. Each context pack requires fresh retrieval."
    one_time_use: true

  # Tag Hygiene (one_time_use: true - expiry ignored)
  - tool: "mosaic_hexxla_put_cell"
    prerequisite: "mosaic_hexxla_list_tags"
    error_message: "You must call mosaic_hexxla_list_tags immediately before calling mosaic_hexxla_put_cell to review current tag vocabulary. This is required on every put_cell call. Tags may have changed."
    one_time_use: true

  - tool: "mosaic_hexxla_put_cell"
    prerequisite: "mosaic_hexxla_tag_counts"
    error_message: "You must call mosaic_hexxla_tag_counts immediately before calling mosaic_hexxla_put_cell to understand current tag frequency. This is required on every put_cell call. Tags may have changed."
    one_time_use: true

  # Budget Estimation (one_time_use: true - expiry ignored)
  - tool: "mosaic_hexxla_load_context_pack"
    prerequisite: "mosaic_hexxla_estimate_context_budget_bytes"
    error_message: "You must call mosaic_hexxla_estimate_context_budget_bytes immediately before calling mosaic_hexxla_load_context_pack to preview token usage for this specific operation."
    one_time_use: true

  # Health Check - All Operations (one_time_use: true - expiry ignored)
  - tool: "mosaic_hexxla_load_context_pack"
    prerequisite: "mosaic_hexxla_health"
    error_message: "You must call mosaic_hexxla_health immediately before calling mosaic_hexxla_load_context_pack to verify database integrity."
    one_time_use: true

  - tool: "mosaic_hexxla_put_cell"
    prerequisite: "mosaic_hexxla_health"
    error_message: "You must call mosaic_hexxla_health immediately before calling mosaic_hexxla_put_cell to verify database integrity."
    one_time_use: true

  - tool: "mosaic_hexxla_put_embedding"
    prerequisite: "mosaic_hexxla_health"
    error_message: "You must call mosaic_hexxla_health immediately before calling mosaic_hexxla_put_embedding to verify database integrity."
    one_time_use: true

  # Delete Operations (one_time_use: true - expiry ignored)
  - tool: "mosaic_hexxla_delete_cell"
    prerequisite: "mosaic_hexxla_health"
    error_message: "You must call mosaic_hexxla_health immediately before calling mosaic_hexxla_delete_cell to verify database integrity before destructive operations."
    one_time_use: true

  # Seam Operations (one_time_use: true - expiry ignored)
  - tool: "mosaic_hexxla_mark_conflict"
    prerequisite: "mosaic_hexxla_health"
    error_message: "You must call mosaic_hexxla_health immediately before calling mosaic_hexxla_mark_conflict to verify database integrity and ensure consistent state."
    one_time_use: true

  - tool: "mosaic_hexxla_mark_supersedes"
    prerequisite: "mosaic_hexxla_health"
    error_message: "You must call mosaic_hexxla_health immediately before calling mosaic_hexxla_mark_supersedes to verify database integrity and ensure consistent state."
    one_time_use: true

  - tool: "mosaic_hexxla_resolve_seam"
    prerequisite: "mosaic_hexxla_health"
    error_message: "You must call mosaic_hexxla_health immediately before calling mosaic_hexxla_resolve_seam to verify database integrity and ensure consistent state."
    one_time_use: true

  # Facet/Edge Operations (one_time_use: true - expiry ignored)
  - tool: "mosaic_hexxla_put_facet"
    prerequisite: "mosaic_hexxla_search_embedding"
    error_message: "You must call a retrieval tool immediately before calling mosaic_hexxla_put_facet to ensure you have valid current cell coordinates."
    one_time_use: true

  - tool: "mosaic_hexxla_link_cells"
    prerequisite: "mosaic_hexxla_search_embedding"
    error_message: "You must call a retrieval tool immediately before calling mosaic_hexxla_link_cells to ensure you have valid current cell coordinates."
    one_time_use: true

  # Read-Only Operations (one_time_use: false, expiry applies)
  - tool: "mosaic_hexxla_get_facet"
    prerequisite: "mosaic_hexxla_health"
    expiry: "5m"
    error_message: "Consider calling mosaic_hexxla_health before calling mosaic_hexxla_get_facet to verify database integrity. Recommended for read operations."
    one_time_use: false

  - tool: "mosaic_hexxla_get_edge"
    prerequisite: "mosaic_hexxla_health"
    expiry: "5m"
    error_message: "Consider calling mosaic_hexxla_health before calling mosaic_hexxla_get_edge to verify database integrity. Recommended for read operations."
    one_time_use: false
```

### Configuration Notes

1. **Multiple Prerequisites:** The configuration shows that tools can have multiple valid prerequisites (e.g., `load_context_pack` can follow any of the three retrieval tools). The implementation should check if ANY of the valid prerequisites has been called.

2. **Explicit Tool Listing:** All tools (including those with no prerequisites) are explicitly listed in the configuration for consistency and future scalability. Tools with no prerequisites use `prerequisite: ""` and can be called at any time. This makes the configuration self-documenting and easier to extend.

3. **One-Time Use Philosophy:** Critical operations use `one_time_use: true` to ensure fresh state checks on every operation. This prevents stale state from being reused and ensures agents always verify current conditions (tags, health, coordinates) before performing operations. **Important:** When `one_time_use: true`, the token is consumed immediately upon validation, so expiry time is irrelevant. Expiry is only applicable when `one_time_use: false`.

4. **Expiry Only for Reusable Tokens:** Expiry times are only set for rules with `one_time_use: false` (discovery tools and read-only operations). This allows token reuse within the expiry window for operations where state is stable. For `one_time_use: true` rules, expiry is omitted since tokens are consumed immediately.

5. **Health Check Ubiquity:** Health checks are required before all write, delete, and seam operations to verify database integrity. This prevents operations on corrupted or unhealthy databases.

6. **Tag Hygiene Enforcement:** `put_cell` requires fresh tag listing on every call (one-time use). This ensures agents always check current tag vocabulary before creating new tags, preventing fragmentation. No expiry is needed since the token is consumed immediately.

7. **Budget Estimation Required:** `load_context_pack` requires budget estimation to avoid exceeding context limits. This ensures efficient token usage and prevents oversized context loads.

---

## Implementation Approach

### Integration Points

1. **mosaic-mcp Composition Root** (`cmd/mosaic-mcp/main.go`):
   - Initialize ratchet service alongside existing services
   - Load ratchet configuration from YAML file
   - Pass ratchet service to tool registration functions

2. **Tool Registration** (`internal/adapter/primary/mcpsrv/`):
   - Wrap each tool handler with ratchet validation
   - Call `ValidateToolCall` before executing tool logic
   - Call `IssueToken` after successful tool execution
   - Pass session ID derived from MCP session

3. **Session Management**:
   - Use MCP session ID as ratchet session ID
   - Create session on first tool call
   - Track tokens per session

### Example Tool Wrapper

```go
// Example wrapper for load_context_pack
func RegisterLoadContextPackTool(server *mcp.Server, ratchetSvc ratchetPorts.RatchetService, sessionStore ratchetSecondary.SessionStore) {
    originalHandler := loadContextPackHandler

    wrappedHandler := func(ctx context.Context, req *mcp.CallToolRequest, input LoadContextPackInput) (*mcp.CallToolResult, LoadContextPackOutput, error) {
        // Derive session ID from MCP context
        sessionID := deriveSessionID(req)

        // Get or create session
        session, err := sessionStore.Get(ctx, sessionID)
        if err != nil {
            session = ratchetDomain.NewSession(sessionID)
            sessionStore.Create(ctx, session)
        }

        // Get existing token for this tool
        var token ratchetDomain.TokenValue
        if tokens, ok := session.Tokens["mosaic_hexxla_load_context_pack"]; ok && len(tokens) > 0 {
            token = tokens[len(tokens)-1]
        }

        // Validate tool call
        err = ratchetSvc.ValidateToolCall(ctx, sessionID, "mosaic_hexxla_load_context_pack", token)
        if err != nil {
            return nil, LoadContextPackOutput{}, fmt.Errorf("ratchet validation failed: %w", err)
        }

        // Execute original handler
        result, output, err := originalHandler(ctx, req, input)
        if err != nil {
            return result, output, err
        }

        // Issue token after successful execution
        _, err = ratchetSvc.IssueToken(ctx, sessionID, "mosaic_hexxla_load_context_pack")
        if err != nil {
            return result, output, fmt.Errorf("failed to issue token: %w", err)
        }

        // Update session
        session.RecordToolCall("mosaic_hexxla_load_context_pack")
        sessionStore.Update(ctx, session)

        return result, output, nil
    }

    // Register wrapped tool with MCP SDK
    mcp.AddTool(server, &mcp.Tool{
        Name:        "mosaic_hexxla_load_context_pack",
        Description: "Load budgeted context pack from seed coordinates",
    }, wrappedHandler)
}
```

---

## Conditional Logic Considerations

### Multiple Valid Prerequisites

The initial configuration shows `load_context_pack` can follow any of three retrieval tools. The implementation needs logic to check if ANY of the valid prerequisites has been called:

```go
// Check if any retrieval tool has been called
func hasRetrievalToken(session *ratchetDomain.Session) bool {
    return session.HasToolBeenCalled("mosaic_hexxla_search_embedding") ||
           session.HasToolBeenCalled("mosaic_hexxla_query_cells") ||
           session.HasToolBeenCalled("mosaic_hexxla_search_cells")
}
```

### Budget-Based Enforcement

For conditional enforcement based on operation size (e.g., requiring health check only for large context loads), the wrapper would need to inspect input parameters:

```go
// Check if operation is "heavy" and requires health check
func requiresHealthCheck(input LoadContextPackInput) bool {
    return input.MaxRing > 5 || input.MaxBudgetBytes > 10000
}
```

---

## Gradual Rollout Strategy

### Phase 1: Core Retrieval-Assemble Pattern

- Enforce `load_context_pack` requires a retrieval tool
- This is the most critical workflow pattern
- Highest impact on agent reliability

### Phase 2: Tag Hygiene

- Add soft enforcement for `put_cell` requiring `list_tags`
- Monitor tag vocabulary consistency improvements
- Adjust error messaging based on agent behavior

### Phase 3: Advanced Patterns

- Add conditional health check requirements for heavy operations
- Add budget estimation recommendations before large context loads
- Experiment with more sophisticated dependency rules

### Phase 4: Evaluation and Refinement

- Monitor agent success rates with ratchet enabled
- Collect metrics on tool call patterns
- Refine rules based on observed behavior
- Consider making certain rules optional or configurable

---

## Configuration Options

### Enable/Disable Ratchet

Add a flag to mosaic-mcp to enable or disable ratchet enforcement:

```bash
./mosaic-mcp -policy configs/config.yaml -db ./data/mosaic.db -ratchet-config configs/ratchet.yaml -enable-ratchet
```

If `-enable-ratchet` is not provided, tools operate without ratchet validation (current behavior).

### Strict vs. Lenient Mode

- **Strict mode:** All rules enforced with mandatory error messages
- **Lenient mode:** Some rules use "recommendation" error messages that don't block execution

This could be configured via a top-level setting in the ratchet YAML:

```yaml
mode: strict # or "lenient"
rules:
  # ...
```

---

## Testing Strategy

### Unit Tests

- Test ratchet service logic with mock stores
- Test rule validation and circular dependency detection
- Test token issuance and expiry

### Integration Tests

- Test tool wrappers with real MCP SDK
- Test session management across multiple tool calls
- Test token consumption and renewal

### Agent Behavior Tests

- Test that agents follow enforced workflows
- Test error message clarity and effectiveness
- Test that agents can recover from validation failures

---

## Monitoring and Observability

### Metrics to Track

- Ratchet validation success/failure rates
- Most common validation failures
- Token issuance and expiry patterns
- Session lifecycle statistics

### Logging

- Log ratchet validation attempts and results
- Log token issuance and consumption
- Log rule violations with context

---

## Future Enhancements

### Dynamic Rule Updates

- Allow runtime rule updates without server restart
- Support hot-reloading of ratchet configuration

### Per-Session Rule Sets

- Different rule sets for different agent types or sessions
- Configurable rule profiles (conservative, balanced, permissive)

### Dependency Graph Visualization

- Visual representation of tool dependency rules
- Debugging tool for understanding rule chains

### Integration with Mosaic Policy

- Include ratchet rules in the main Mosaic policy YAML
- Single source of truth for all Mosaic configuration

---

## References

- [mcp-ratchet README](https://github.com/hexxla/mcp-ratchet)
- [Mosaic MCP Agent Blueprint](./MCP_AGENT_BLUEPRINT.md)
- [Mosaic Configuration](./MOSAIC_CONFIG.md)
- [Hexagonal Architecture](../architecture/hexagonal-architecture-simplified.md)
