# Ratchet integration

Mosaic has an optional integration with [mcp-ratchet](https://github.com/hexxla/mcp-ratchet) for enforcing tool-call prerequisites. It is implemented as one MCP receiving middleware, so tool handlers remain independent of workflow policy.

## Activation

Ratchet is disabled by default. Enable it with either configuration path:

```bash
mosaic-mcp -db ./data/mosaic.hexxla -ratchet-config configs/ratchet.yaml
```

```bash
MOSAIC_RATCHET_CONFIG_FILE=configs/ratchet.yaml \
  mosaic-mcp -db ./data/mosaic.hexxla
```

The flag takes precedence over the environment variable. A missing, invalid,
non-regular, circular, or mutation-incomplete configuration fails server
startup. When Ratchet is enabled, every tool classified by Mosaic as a mutation
must have at least one prerequisite and must not have a free-pass rule.
Omitting both settings preserves the documented opt-in behavior: Ratchet is not
an authorization layer and Mosaic's normal runtime write policies still apply.

## Enforcement contract

The middleware applies only to tools named in the loaded rules. A configured tool call:

1. obtains the actual MCP server-session ID;
2. creates or loads process-local Ratchet state for that session;
3. validates the configured prerequisite;
4. reserves a one-time prerequisite, when configured, before running the handler;
5. records the call and issues a token only after the handler succeeds.

Calls without an MCP session fail closed for configured tools. State is isolated between clients, held in memory, and lost when the process restarts. Same-session calls are serialized through bounded lock striping so concurrent requests cannot reuse one one-time token.

If a protected handler fails after its one-time prerequisite was reserved, the client must repeat the prerequisite before retrying. This prevents two concurrent mutations from both passing on one token.

## Supplied policy

[`configs/ratchet.yaml`](../../configs/ratchet.yaml) defines this example operator policy:

| Tool | Requirement | Token behavior |
| --- | --- | --- |
| `mosaic_hexxla_list_tags` | none | issues a 10-minute token |
| `mosaic_hexxla_query_cells`, `mosaic_hexxla_search_cells`, `mosaic_hexxla_search_embedding` | none | issue 10-minute tokens |
| `mosaic_hexxla_find_seams` | none | issues a 10-minute token |
| `mosaic_hexxla_put_cell` | `mosaic_hexxla_list_tags` | consumed once |
| `mosaic_hexxla_put_embedding`, `mosaic_hexxla_put_facet` | any listed retrieval tool | reusable until expiry |
| `mosaic_hexxla_load_context_pack` | any listed retrieval tool | reusable until expiry |
| `mosaic_hexxla_link_cells` | `mosaic_hexxla_load_context_pack` | consumed once |
| `mosaic_hexxla_mark_conflict`, `mosaic_hexxla_mark_supersedes`, `mosaic_hexxla_resolve_seam` | `mosaic_hexxla_find_seams` | consumed once |
| `mosaic_hexxla_delete_cell` | any listed retrieval tool | reusable until expiry |

Read-only tools omitted from the file are unrestricted. A mutation omitted from
the file—or given any empty-prerequisite rule—causes startup to fail while
Ratchet is enabled. Multiple rules with the same `tool` are alternatives (OR),
not cumulative requirements. An empty `prerequisite` makes that rule
immediately callable and allows it to issue a token after success.

`one_time_use` belongs to the protected tool rule. The supplied
multi-prerequisite OR rules deliberately use reusable tokens: the pinned
mcp-ratchet version consumes the first matching rule's prerequisite for
one-time dependencies, which cannot safely represent one-time OR consumption.
A single prerequisite such as `list_tags`, `load_context_pack`, or `find_seams`
does not have that ambiguity.

## Tool safety inventory

Mosaic keeps one explicit classification for all 23 tools. Registration fails
at startup if a tool is absent from that inventory. The same inventory:

- publishes MCP `readOnlyHint`, `destructiveHint`, `idempotentHint`, and
  `openWorldHint` annotations;
- identifies the eight mutation tools Ratchet must protect when enabled; and
- classifies Ollama-backed operations as open-world because the configured
  Ollama URL may be a separate service.

These annotations help clients present and approve calls, but they are hints,
not security controls. Ratchet middleware and Mosaic runtime policy remain the
enforcement mechanisms. A regression test rejects production registrations
that bypass the classification wrapper.

## Security boundary

Ratchet is workflow ordering, not authentication, authorization, or a substitute for Mosaic's write/delete policy. Continue to bind Mosaic to a trusted interface and apply the normal transport/access controls described in the operator documentation.

Mosaic does not expose Ratchet's tokens, session store, historical event store, or HTTP statistics. It also does not log session IDs or token values. An operator may explicitly enable a live-only WebSocket stream; it requires a private bearer-token file, shares Mosaic's loopback-only listener, rejects requests carrying a browser `Origin`, and omits capability tokens and event metadata from every event.

## Live WebSocket observability

Observability is disabled unless both Ratchet and a bearer-token file are configured. Create a random private token, start the server, and run the bundled observer in a second terminal:

```bash
umask 077
openssl rand -hex 32 > ./observer.token

mosaic-mcp \
  -db ./data/mosaic.hexxla \
  -ratchet-config configs/ratchet.yaml \
  -ratchet-observability-token-file ./observer.token
```

```bash
mosaic-observe -token-file ./observer.token
```

The default stream URL is `ws://127.0.0.1:8787/observability/stream`.
Use `-ratchet-observability-path` on the server and `-url` on the observer to
change it. The observer receives all live sessions by default; use
`-session-id <mcp-session-id>` to filter one session. Events are NDJSON with
only `id`, `type`, `session_id`, optional `tool_name`, and `timestamp`.

The stream is operational telemetry, not a durable audit log: it retains no
history, accepts at most 16 simultaneous observers, and drops events for a
subscriber whose bounded buffer is full rather than delaying MCP enforcement.
The bearer token is accepted only in the `Authorization` header; Mosaic never
accepts it in a URL or command-line value. Keep the token file private (`0600`
on Unix-like systems) and rotate it by replacing the file and restarting the
server.

## Current limitations

- State is local to one Mosaic process; it is not shared across replicas and does not survive restart.
- Rules are loaded only at startup; there is no hot reload.
- Ratchet observability is live-only and process-local; there is no replay, durable event store, aggregate statistics endpoint, or browser dashboard.
- Repeated rules implement OR semantics. AND workflows require a separate policy design or upstream Ratchet support.
- Multi-prerequisite one-time OR enforcement is not enabled for the reason described above.
- Tokens attest only that a prerequisite tool succeeded in the same session;
  they do not bind returned coordinates or records to a later mutation's
  arguments. Clients must inspect the relevant cell/context/seam, and Mosaic's
  domain validation remains authoritative.

These constraints keep the integration small and explicit while preserving the normal Mosaic API when Ratchet is not configured.

## Verification

The integration tests use the real MCP Streamable HTTP transport and independent clients to verify:

- configured prerequisites deny calls before discovery;
- all eight Mosaic mutation tools are protected by the supplied policy;
- one client cannot authorize another client's call;
- a successful prerequisite authorizes the protected call;
- a one-time token cannot be reused;
- failed prerequisite handlers do not issue authorization tokens.
- every registered production tool goes through the safety inventory.

Run the focused tests with:

```bash
go test ./internal/adapter/primary/mcpsrv ./internal/config ./cmd/mosaic-mcp
```
