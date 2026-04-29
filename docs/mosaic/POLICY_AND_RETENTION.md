# Mosaic operator policy / retention (`mosaic.policy.yaml`)

**Audience:** Humans running `mosaic-mcp` who want explicit acknowledgement before exposing mutating tools to agents (optional), and declarative retention *intent* Hexxla can hold while enforcement stays out-of-band.

## Is a policy file next to the binary a good convention?

**Partially.**

- Putting config **strictly beside `argv[0]`** breaks many installs (`/usr/bin`, AppImages, snaps) and symlink layouts. Mosaic therefore resolves **`MOSAIC_POLICY_FILE`**, otherwise **`./mosaic.policy.yaml`** in the **process working directory**, not implicitly next to the binary.
- If you deploy a systemd unit or container, set **`WorkingDirectory`** and/or **`MOSAIC_POLICY_FILE=/etc/mosaic/mosaic.policy.yaml`**.

Duplicating rules in Cursor `.mdc` **drifts** from the file Git reviews. Prefer: **YAML is canonical**, rules say *“enforce `mosaic.policy.yaml` (see MOSAIC_POLICY_REQUIRE_ACK)”*.

## Write acknowledgement gate (fail-fast)

When **`MOSAIC_POLICY_REQUIRE_ACK`** is **`true|1|yes|on`**, `mosaic-mcp` **refuses to start** unless:

1. The policy file exists (path from **`MOSAIC_POLICY_FILE`**, or default **`mosaic.policy.yaml`** in cwd), and
2. YAML parses as **version `1`**, and
3. **`writes.enabled: true`** — the operator line that mutating MCP tools are allowed on this database.

This is a **logic / social** lock, not encryption: it stops “forgot to copy policy into prod” and LLM mass-deletes before anyone reviewed retention. It does **not** stop a malicious local user from unsetting the env var.

**Default:** acknowledgement is **off** so local dev and `make` workflows keep working without a file.

## Retention (declarative only in `mosaic-mcp`)

Hexxla stores cells until something removes them. **Retention is not auto-applied** by `mosaic-mcp` (no scheduler, no TTL worker). The **`retention:`** section in `mosaic.policy.yaml` is for:

- Operator comments and numeric hints (`max_cell_age_days`, `tags_eligible_for_prune`) for **offline jobs**, `mosaic-seed` labelling, or future `mosaic-retention` tooling.
- Agent instructions: delete only cells matching agreed tags/windows.

**Recommended pattern**

1. Encode intent in YAML (and optionally mirror with cell tags such as `scratch`, `session:<id>`).
2. Run a **separate** batch (script, cron, CI) that queries by tag/time and calls **`DeleteCell`** (via dedicated tool or native Go), or use operator-only Hexxla maintenance when applicable.
3. Keep **one writer process per `MOSAIC_DB_PATH`** to avoid corruption from concurrent embedded access.

See [MCP_AGENT_BLUEPRINT.md](./MCP_AGENT_BLUEPRINT.md) (persistence model) and [MCP_BLUEPRINT.md](./MCP_BLUEPRINT.md) (server layout).

## Example

[`configs/config.yaml`](../../configs/config.yaml) at repo root (`retention`, `allow_delete_cell`, and room for more top-level sections).

## Related environment variables

| Variable | Meaning |
| --- | --- |
| `MOSAIC_POLICY_REQUIRE_ACK` | When truthy, require valid policy file + `writes.enabled: true` before `mosaic-mcp` binds. |
| `MOSAIC_POLICY_FILE` | Optional path to the YAML file; default `mosaic.policy.yaml` (relative to cwd). |
