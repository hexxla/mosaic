# Stack release qualification

**Status: alpha.** This is the qualification procedure for HexxlaDB, Mosaic
and MCP Ratchet. It defines future work; it is not a completed qualification
report. CI success on a development revision does not promote the stack.

Each repository's `VERSIONING.md` owns its release/compatibility gates:
[HexxlaDB](https://github.com/hexxla/hexxladb/blob/main/VERSIONING.md),
[Mosaic](../VERSIONING.md),
[Ratchet](https://github.com/hexxla/mcp-ratchet/blob/main/VERSIONING.md).
Repositories keep independent versions and immutable historical tags. An
individual repository may qualify first; the stack stays alpha until the exact
selected combination satisfies the stack beta gates.

## Owner, scope and initial profile

Name a qualification owner before a run. Start with a maintainer-owned local
agent workflow and synthetic or explicitly authorized data that can be restored.
No external customer is required. A controlled pre-v1 pilot is allowed with
explicit ownership and recovery arrangements; it grants no general production
sign-off.

The first proposed operating profile is Linux/amd64, the exact Go toolchain
in the selected `go.mod` files, one open owner per database on a trusted local
filesystem, Mosaic's loopback-only MCP listener, process-local Ratchet state,
and an explicitly identified Ollama model/digest when vectors are enabled.
HexxlaDB serializes writes; exercise concurrent readers and callers within that
single-owner contract. Declare plaintext/encrypted and changefeed variants used.
Cross-builds for other platforms are build evidence; runtime claims require
separate qualification on those platforms. Public network/multi-tenant service,
distributed writers, replication and automatic failover are excluded.

Before execution, write down:

- Exact source SHAs for all three repositories, intended tags, Go module
  versions, MCP SDK version, toolchain, and whether a workspace/replace was used.
- Hardware, OS/kernel, filesystem/storage, CPU/RAM/cache settings, credentials
  mechanism (no secret values), model name/digest and provider version.
- Dataset size, vector dimension/count, record sizes, tag/graph density, history
  depth, enabled tools/options and expected concurrency/read-write mix.
- Numeric latency/throughput, process-memory, disk-growth and maintenance-window
  targets; backup interval, recovery point objective (RPO) and recovery time
  objective (RTO). Choose targets for the intended use before observing results.
- A workload seed, operation/result oracle, run duration, scheduled restart and
  failure cases, monitoring interval, artifact destination and accountable owner.

Use HexxlaDB's [bounded candidate profile](https://github.com/hexxla/hexxladb/blob/main/VERSIONING.md#production-readiness-profile)
as an upper scope requiring verification, not a capacity guarantee. Start small,
then run the intended operating size plus a larger case that confirms limits
fail explicitly. A short smoke test does not establish the longer profile.

## Practical sequence

These are suggested work sessions, not automatic promotion dates. Missing
runners or evidence are qualification work to complete, not passes to infer.
Use existing tests/evidence runners and a small reproducible workload harness
where needed; record its source and seed with the report.

| Step | Work | Completion evidence |
| --- | --- | --- |
| 1. Select and baseline | Choose sources, dependency combination, deployment profile and targets. Check the declared Go module path against the canonical repository before tagging. Review open defects and migrations. Run the repository checks below. | Manifest, command logs, test/build results and blocker list; any skipped check remains unmet. |
| 2. Exercise contracts | Run both Mosaic profiles through real MCP sessions. Exercise enabled tools, policy acceptance/rejection, session isolation, state/cursor expiry, cancellation, failed SDK results, model mismatch and bounded/sparse/oversized retrieval. | Expected/observed outcomes and bounded-resource measurements; check that invalid calls do not produce unintended effects or authorization completions. |
| 3. Rehearse recovery | Use disposable recovery-set copies for crash/reopen, lost-response replay, sidecar continuity, backup/restore, pruning/compaction and install/upgrade/rollback checks. | Oracle comparison, backup integrity/content checks, actual RPO/RTO and complete recovery instructions. |
| 4. Run beta workload | At least 24 continuous hours of periodic mixed work, with at least 10,000 completed operations/calls distributed through the run. Include idle/expiry, session churn, cancellation, retries, reopen and maintenance. | Timed logs, operation counts, expected-denial/failure counts, oracle checks and latency/memory/disk trends meeting the predeclared targets. Idle time alone cannot pass. |
| 5. Review beta | Reconcile every discrepancy and evaluate all repository/stack beta gates. Repeat affected checks after fixes. | Named-owner report and explicit maintainer decision; no automatic beta tag. |
| 6. Operate pilot | After beta, run a real maintainer-owned workflow for at least 14 consecutive days, with regular daily use or representative scheduled work. Include at least one upgrade and verified restore/rollback drill. | Daily workload/health observations, incident log, dependency manifest, backup/restore and upgrade report. Declare expected activity before starting; idle uptime is insufficient. |
| 7. Freeze and release | Settle v1 contracts and support scope, freeze dependencies, select an RC and rerun final gates. | Reviewed report for the final candidate, supported-profile statement, migration/support guidance and maintainer sign-off. Only then select v1. |

A material code, format, model, dependency or operating-profile change requires
an explicit assessment and rerun of affected checks. Correctness/security or
resource blockers discovered during a soak/pilot must be fixed and the affected
observation window restarted. Preserve unaffected evidence with its original
revision and scope. Documentation-only changes may carry evidence forward only
with the source diff and justification recorded.

## Repository baseline commands

Run commands from each repository's own root using the declared toolchain.
These commands are instructions, not evidence that they have run for a new
candidate. Preserve command output and exit status.

| Repository | Commands | Purpose |
| --- | --- | --- |
| HexxlaDB | `task ci`, `task govulncheck`, `task integration`, `task fuzz`, `task build-all` | API baseline, static/race checks, vulnerability review, crash/recovery, decoder fuzz smoke and cross-builds |
| Mosaic | `task ci`, `task build-all` | Module/workflow/static/race/MCP checks, command builds, vulnerability review and cross-builds |
| MCP Ratchet | `make ci`, `make build-all` | Module/workflow/static/race/SDK checks, vulnerability review and cross-builds |

For the intended HexxlaDB profile, regenerate applicable evidence using
`task evidence-controlled`, `task evidence-observe`, `task evidence-vector-scale`
and `task evidence-lattice-placement`. `task soak-pilot` is a five-minute
reference smoke runner; it does not replace the 24-hour/14-day stages. Follow
the [performance evidence guide](https://github.com/hexxla/hexxladb/blob/main/docs/hexxladb/PERFORMANCE_EVIDENCE.md)
for workload parameters and interpretation. A vector-free profile can mark the
vector workload inapplicable with a reason and must exclude vectors from its
readiness claim.

Verify the distributable dependency path: publish upstream candidate tags or
commits before the consuming candidate, resolve them through ordinary Go module
resolution, and test Mosaic with `GOWORK=off`. Record actual `go list -m all`
output and build provenance. An explicit temporary workspace is useful extra
integration evidence, but cannot replace the published dependency test. Do not
handcraft pseudo-versions or silently change dependencies during qualification.
Before assigning a new tag, verify `go list -m` names the canonical module:
`github.com/hexxla/hexxladb`, `github.com/hexxla/mcp-ratchet`, or
`github.com/hexxla/mosaic`. Local compilation cannot prove that an obsolete
repository/module path remains resolvable. After publication, verify module
download and install the exact tagged Mosaic command into an isolated `GOBIN`
from outside the checkout. Record its build information and pinned dependencies.
If a published snapshot has an identity/install defect, preserve it and issue a
corrected prerelease; never move the tag.

This plan itself does not publish or move any tags.

## Required scenarios and observations

- **Durable memory:** keyed writes and atomic batches, metadata/facet guards,
  vector invalidation, model identity/digest changes, lexical-only admission,
  retained history, graph context and logical changefeed continuity. Compare
  writes and replayed results with a deterministic oracle. Regional exports
  must account for deferred/oversized cells and use one snapshot.
- **Ratchet/transport:** bundled and representative custom policies, ordered
  alternatives, successful/denied/failed/canceled calls, consumed prerequisites,
  final SDK serialization failure, capacity/TTL cleanup, independent sessions
  and graceful shutdown with active work. Restart must require fresh process-local
  prerequisites. Ratchet does not make business writes exactly once.
- **Bounds and observability:** measure p50/p95/p99 latency, operation/error
  counts, memory/state growth, primary/WAL/changefeed sizes and maintenance time.
  Expected policy denials must be separated from unexpected failures. Any
  approximate retrieval result or exhausted budget must be visible to callers.
- **Recovery and upgrade:** restore a complete matched primary/WAL/enabled-feed
  set into a fresh location and verify records, vectors, retained history and
  consumer progress. Exercise durable versus uncertain commit reconciliation,
  lost MCP responses, provider failure/cancellation, accepted format conversion
  and fail-closed unsupported-format opens. Validate crash/fault cases through
  available deterministic tests or disposable test volumes, never by exhausting
  the development host or damaging the original recovery set.

Use [HexxlaDB's operations guide](https://github.com/hexxla/hexxladb/blob/main/docs/hexxladb/OPERATIONS.md)
for backup, conversion and maintenance. Rollback can require restoring a
pre-upgrade recovery set; an old binary opening a new file is not a rollback
plan. Cell JSON transfer is not a complete database backup.

## Decision and report

Keep a report/artifact bundle per candidate (for example under an ignored
`.tmp/qualification/<run-id>/` during development), then attach or link a
reviewed, durable copy from the qualification issue/PR before promotion. Do not
store credentials or private memory payloads in public artifacts.

A sufficient report contains:

1. Owner, stage requested, UTC start/end, source/dependency manifest and profile.
2. Targets, workload/harness revision and seed, operation mix/count and coverage.
3. Each gate's pass/fail/not-run/inapplicable result with logs/artifact links;
   every inapplicable result includes a reason and an excluded support claim.
4. Resource/performance trends and every discrepancy, fix and rerun.
5. Backup/restore/upgrade/rollback results, verified content and measured RPO/RTO.
6. Open defects with impact/owner, supported scope/known limits, and the explicit
   reviewer/maintainer promotion decision.

Block promotion for unresolved data loss/corruption, incorrect durable outcomes,
workflow/session-boundary failures, reachable security defects, or uncontrolled
resource use inside the supported profile. Other limitations require explicit
impact and acceptance; passing CI cannot waive a blocker. Beta requires all
applicable beta gates. RC/v1 require beta, the completed pilot, accepted public
contracts and the final candidate checks. Missing evidence leaves the stage
unchanged. A date, feature count, lack of external users or rising v0 minor
number cannot substitute for the decision.
