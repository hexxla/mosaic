# MCP Tool Description Audit

**Status:** Historical audit snapshot from May 2, 2026. Findings and quoted descriptions below describe that date, not the current tool surface.

**Purpose:** This document records the May 2, 2026 audit of Mosaic MCP tool descriptions. For the current write and placement contract, see [`MCP_AGENT_BLUEPRINT.md`](MCP_AGENT_BLUEPRINT.md#cell-placement-on-writes).

---

## Audit Date

May 2, 2026

---

## Tools Analyzed

| Tool                                          | File                                                              | Current Description Quality                        |
| --------------------------------------------- | ----------------------------------------------------------------- | -------------------------------------------------- |
| `mosaic_hexxla_put_cell`                      | `internal/adapter/primary/mcpsrv/cell_mutation_tool.go`           | ⚠️ Needs improvement - lacks tag category guidance |
| `mosaic_hexxla_put_embedding`                 | `internal/adapter/primary/mcpsrv/cell_mutation_tool.go`           | ✅ Good                                            |
| `mosaic_hexxla_delete_cell`                   | `internal/adapter/primary/mcpsrv/cell_mutation_tool.go`           | ✅ Good                                            |
| `mosaic_hexxla_query_cells`                   | `internal/adapter/primary/mcpsrv/cell_query_tool.go`              | ⚠️ Could improve - has one tag example             |
| `mosaic_hexxla_search_cells`                  | `internal/adapter/primary/mcpsrv/cell_search_tool.go`             | ⚠️ Could improve - no tag examples                 |
| `mosaic_hexxla_search_embedding`              | `internal/adapter/primary/mcpsrv/embedding_search_tool.go`        | ⚠️ Could improve - no tag examples                 |
| `mosaic_hexxla_load_context_pack`             | `internal/adapter/primary/mcpsrv/context_pack_tool.go`            | ✅ Good                                            |
| `mosaic_hexxla_list_tags`                     | `internal/adapter/primary/mcpsrv/tag_tools.go`                    | ✅ Good                                            |
| `mosaic_hexxla_tag_counts`                    | `internal/adapter/primary/mcpsrv/tag_tools.go`                    | ✅ Good                                            |
| `mosaic_hexxla_health`                        | `internal/adapter/primary/mcpsrv/health_tool.go`                  | ✅ Good                                            |
| `mosaic_hexxla_put_facet`                     | `internal/adapter/primary/mcpsrv/facet_edge_tools.go`             | ✅ Good                                            |
| `mosaic_hexxla_link_cells`                    | `internal/adapter/primary/mcpsrv/facet_edge_tools.go`             | ✅ Good                                            |
| `mosaic_hexxla_get_facet`                     | `internal/adapter/primary/mcpsrv/facet_edge_read_tools.go`        | ✅ Good                                            |
| `mosaic_hexxla_list_facets`                   | `internal/adapter/primary/mcpsrv/facet_edge_read_tools.go`        | ✅ Good                                            |
| `mosaic_hexxla_get_edge`                      | `internal/adapter/primary/mcpsrv/facet_edge_read_tools.go`        | ✅ Good                                            |
| `mosaic_hexxla_list_edges_from`               | `internal/adapter/primary/mcpsrv/facet_edge_read_tools.go`        | ✅ Good                                            |
| `mosaic_hexxla_find_seams`                    | `internal/adapter/primary/mcpsrv/seam_tools.go`                   | ✅ Good                                            |
| `mosaic_hexxla_mark_conflict`                 | `internal/adapter/primary/mcpsrv/seam_tools.go`                   | ✅ Good                                            |
| `mosaic_hexxla_mark_supersedes`               | `internal/adapter/primary/mcpsrv/seam_tools.go`                   | ✅ Good                                            |
| `mosaic_hexxla_resolve_seam`                  | `internal/adapter/primary/mcpsrv/seam_tools.go`                   | ✅ Good                                            |
| `mosaic_hexxla_estimate_context_budget_bytes` | `internal/adapter/primary/mcpsrv/context_budget_estimate_tool.go` | ✅ Good                                            |
| `mosaic_hexxla_get_persistence_policy`        | `internal/adapter/primary/mcpsrv/persistence_policy_tool.go`      | ✅ Good                                            |
| `mosaic_hexxla_retrieval_budget_status`       | `internal/adapter/primary/mcpsrv/retrieval_budget.go`             | ✅ Good                                            |

---

## Issues Identified

### Summary

Out of 23 tools analyzed, 4 have description issues related to tag category guidance. The remaining 19 tools have clear, adequate descriptions.

### 1. put_cell - Missing Tag Category Guidance

**Current Description:**

```text
Write one cell at axial (q,r) via HexxlaDB PutCell (DB.Update). Before choosing tags, call mosaic_hexxla_list_tags (and mosaic_hexxla_tag_counts) when taxonomy is unknown — reuse existing tags when relevant instead of inventing near-duplicates. kind=fact uses a fact template; user_message / assistant_response match conversational_memory-style tags. source_id is required (provenance source for fact, session id for user/assistant templates).
```

**Issue:** The description mentions "conversational_memory-style tags" but doesn't explain what tag categories should be used. LLMs will likely invent inconsistent tags across sessions, leading to fragmentation and poor search effectiveness.

**Impact:** High - This is the primary write tool and tag consistency is critical for memory retrieval.

---

### 2. query_cells - Limited Tag Examples

**Current Description:**

```text
Indexed cell query (HexxlaDB QueryCells): tags, source, time window, spatial radius, sort, explain. Use require_tags (e.g. preference) for structured slices instead of vague query-only search when fetching tagged memories.
```

**Issue:** Mentions "preference" as a single example tag, but doesn't provide a comprehensive list of common categories.

**Impact:** Medium - LLMs may not know what other tag categories are available.

---

### 3. search_cells - No Tag Examples

**Current Description:**

```text
Lexical relevance search (HexxlaDB SearchCells): scored substring/tag/source matches; optional scan radius; optional embed_query_text for ANN-accelerated hybrid retrieval.
```

**Issue:** No tag category examples provided.

**Impact:** Medium - Same as query_cells.

---

### 4. search_embedding - No Tag Examples

**Current Description:**

```text
Semantic retrieval: embed the query via Ollama (MOSAIC_EMBED_MODEL) and run HexxlaDB SearchByEmbedding ANN — returns top-K similar cells (coords, scores, text, tags).
```

**Issue:** No tag category examples provided, even though results include tags.

**Impact:** Medium - LLMs may not know how to filter results by tag categories.

---

## Recommended Tag Categories

Based on common conversation patterns and memory organization needs, the following tag categories are recommended:

### Content Nature Categories

- **preference** - User likes/dislikes, settings, configurations
- **fact** - Objective, verifiable information
- **opinion** - Subjective views, beliefs, perspectives
- **idea** - Concepts, proposals, brainstorming
- **code** - Snippets, technical content, algorithms
- **signal** - Important, high-value information
- **noise** - Low-value, irrelevant content

### Action/Task Categories

- **task** - Action items, to-do items
- **decision** - Made choices, resolved issues
- **question** - Queries, requests for information
- **answer** - Responses, solutions

### Work/Project Categories

- **project** - Work-related activities
- **bug** - Issues, errors, problems
- **feature** - Capabilities, functionality
- **requirement** - Needs, specifications
- **goal** - Objectives, targets
- **milestone** - Progress markers
- **deadline** - Time constraints
- **priority** - Importance level (high/medium/low)
- **status** - Current state (active/pending/completed)

### Communication Categories

- **meeting** - Discussions, syncs
- **email** - Communications
- **document** - Files, references
- **note** - General notes, observations

### Personal Categories

- **personal** - Private information
- **contact** - People information
- **location** - Geographical data

---

## Suggested Improvements

### 1. put_cell - Add Explicit Tag Category Guidance

**Suggested Description:**

```text
Write one cell at axial (q,r) via HexxlaDB PutCell (DB.Update). Before choosing tags, call mosaic_hexxla_list_tags (and mosaic_hexxla_tag_counts) when taxonomy is unknown — reuse existing tags when relevant instead of inventing near-duplicates.

**Tag Categories:** Use semantic tags to categorize content nature:
- Content nature: preference (user likes/dislikes), fact (objective information), opinion (subjective views), idea (concepts/proposals), code (snippets/technical), signal (important/high-value), noise (low-value/irrelevant)
- Actions: task (action items), decision (made choices), question (queries), answer (responses)
- Work: project (work-related), bug (issues/errors), feature (capabilities), requirement (needs), goal (objectives), milestone (progress), deadline (time constraints), priority (importance), status (current state)
- Communication: meeting (discussions), email (communications), document (files), note (general notes)
- Personal: personal (private info), contact (people), location (geographical)

kind=fact uses a fact template; user_message / assistant_response match conversational_memory-style tags. source_id is required (provenance source for fact, session id for user/assistant templates).
```

### 2. query_cells - Expand Tag Examples

**Suggested Description:**

```text
Indexed cell query (HexxlaDB QueryCells): tags, source, time window, spatial radius, sort, explain. Use require_tags (e.g. preference, fact, opinion, idea, code, signal, task, project, bug, feature, goal, milestone, meeting, deadline, priority, status) for structured slices instead of vague query-only search when fetching tagged memories.
```

### 3. search_cells - Add Tag Examples

**Suggested Description:**

```text
Lexical relevance search (HexxlaDB SearchCells): scored substring/tag/source matches; optional scan radius; optional embed_query_text for ANN-accelerated hybrid retrieval. Use tag filters (e.g. preference, fact, opinion, idea, code, signal, task, project) to narrow results by category.
```

### 4. search_embedding - Add Tag Filtering Note

**Suggested Description:**

```text
Semantic retrieval: embed the query via Ollama (MOSAIC_EMBED_MODEL) and run HexxlaDB SearchByEmbedding ANN — returns top-K similar cells (coords, scores, text, tags). Results include tags that can be used for category filtering (e.g. preference, fact, opinion, idea, code, signal, task, project). Use as a first step to get seed coordinates; then call mosaic_hexxla_load_context_pack with 1-3 of those {q,r} to expand hex-neighbourhood context.
```

## Implementation Priority

| Priority   | Tool                             | Reason                                       |
| ---------- | -------------------------------- | -------------------------------------------- |
| **High**   | `mosaic_hexxla_put_cell`         | Primary write tool, tag consistency critical |
| **Medium** | `mosaic_hexxla_query_cells`      | Frequently used for retrieval                |
| **Medium** | `mosaic_hexxla_search_cells`     | Frequently used for retrieval                |
| **Medium** | `mosaic_hexxla_search_embedding` | Primary semantic search tool                 |

**Note:** The remaining 19 tools have adequate descriptions and do not require changes. These include facet/edge operations, seam operations, budget estimation, persistence policy, and retrieval budget status tools, which are either write operations with clear technical descriptions or read-only operations that don't require tag category guidance.

---

## Additional Considerations

### Tag Hygiene

- The ratchet configuration already enforces `list_tags`/`tag_counts` before `put_cell`
- However, without explicit category guidance, LLMs may still choose suboptimal tags
- Consider adding a "recommended tags" section to the documentation

### Tag Versioning

- Consider whether tag categories should be versioned (e.g., `preference-v1`)
- This could allow evolution of the taxonomy without breaking existing content

### Tag Hierarchies

- Consider supporting hierarchical tags (e.g., `work/project`, `work/meeting`)
- This would provide more granular organization capabilities

---

## Next Steps

1. Update `mosaic_hexxla_put_cell` description with explicit tag category guidance
2. Update retrieval tool descriptions (`query_cells`, `search_cells`, `search_embedding`) with tag examples
3. Consider creating a separate tag taxonomy reference document
4. Test updated descriptions with LLMs to verify improved tag consistency
5. Monitor tag usage patterns after deployment to identify missing categories

---

## References

- MCP Agent Blueprint: `docs/mosaic/MCP_AGENT_BLUEPRINT.md`
- Ratchet Integration: `docs/mosaic/RATCHET_INTEGRATION.md`
- Tool definitions: `internal/adapter/primary/mcpsrv/`

---

## Go MCP SDK Documentation Review

Analyzed the Go MCP SDK documentation at <https://go.sdk.modelcontextprotocol.io/server/> for best practices related to tool descriptions.

### Findings

The Go MCP SDK documentation focuses on **implementation patterns** rather than **description content quality**:

**SDK Best Practices (Implementation):**

- Use the generic `AddTool` function to handle schema inference, validation, marshaling, and error handling automatically
- Use `jsonschema` struct tags to provide field-level descriptions for input/output structs (e.g., `jsonschema:"user location"`)
- Can customize inferred schemas using `TypeSchemas` map for complex types
- Can tweak inferred schemas (e.g., setting min/max constraints)

**Missing Guidance:**

- No explicit best practices for tool description content quality
- No guidance on what information should be included in the `Tool.Description` field
- No recommendations for tag category conventions or taxonomy design
- No examples of effective tool descriptions for LLM guidance

**Conclusion:** The Go SDK provides excellent guidance for tool implementation but does not address the content quality of tool descriptions. Our audit's focus on tag category guidance and description clarity is not covered by the SDK documentation, making our recommendations complementary to the SDK's implementation guidance.

---

## Mosaic Codebase SDK Compliance

Analyzed Mosaic codebase for compliance with Go MCP SDK best practices.

### Compliance Status: ✅ Fully Compliant

The Mosaic codebase follows all Go MCP SDK best practices:

**1. Uses Generic AddTool Function:**

- All 23 tools use `mcp.AddTool(server, &mcp.Tool{...}, handler)` pattern
- Handler functions use the recommended signature: `func(ctx context.Context, req *mcp.CallToolRequest, input In) (*mcp.CallToolResult, output Out, error)`
- This allows the SDK to automatically handle schema inference, validation, marshaling, and error handling

**2. Uses jsonschema Struct Tags:**

- All input structs use `jsonschema` tags for field-level descriptions
- Example from `cell_mutation_tool.go`:

```go
type putCellInput struct {
    Q          int      `json:"q" jsonschema:"axial q"`
    R          int      `json:"r" jsonschema:"axial r"`
    RawContent string   `json:"raw_content" jsonschema:"cell body text"`
    Tags       []string `json:"tags,omitempty" jsonschema:"optional tag strings (merged with template tags)"`
    SourceID   string   `json:"source_id" jsonschema:"provenance source or session id"`
    Confidence float64  `json:"confidence" jsonschema:"0..1"`
    Kind       string   `json:"kind,omitempty" jsonschema:"fact (default) | user_message | assistant_response"`
}
```

- Similar patterns used across all 23 tools in facet_edge_tools.go, seam_tools.go, etc.

**3. Custom Schema Support:**

- While not currently using `TypeSchemas` map for complex type customization, the codebase structure supports this if needed
- Current approach with inline `jsonschema` tags provides sufficient clarity for the existing types

**Conclusion:** Mosaic codebase fully complies with Go MCP SDK implementation best practices. The tools are correctly implemented using the generic `AddTool` function with proper `jsonschema` struct tags for schema inference and validation.
