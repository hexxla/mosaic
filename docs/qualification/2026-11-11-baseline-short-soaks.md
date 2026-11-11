# Baseline and short-soak qualification — 2026-11-11

**Result: baseline/recovery, 30-minute lexical and 1-hour lexical stages passed.**
Qualification owner: **void, the requesting repository maintainer**. The stack
remains **alpha**. The 24-hour beta workload and 14-day operating pilot are
pending; this report grants no beta/v1 or general production sign-off.

The [qualification plan](../RELEASE_QUALIFICATION.md) owns the gates; the
[soak runbook](../SOAK_RUNBOOK.md) owns workload and acceptance targets. This
report records completed evidence so roadmaps can focus on remaining work.

## Exact candidate and evidence scope

| Component | Published candidate | Source SHA |
| --- | --- | --- |
| HexxlaDB | `v0.8.0-alpha.1` | `96070acab89eebca49aab51fe598d572f42041f0` |
| Mosaic | `v0.4.0-alpha.2` | `e21858d3def444ea6a1cb5e484902b0b973b7612` |
| MCP Ratchet | `v0.4.0-alpha.1` | `43d4ffb110d4ea0180d0d7104943a61a44bbdaa9` |

Go 1.27.4, Linux/amd64, local btrfs, `GOWORK=off`, normal published dependency
resolution, MCP SDK v1.8.0. Baseline checks used GOMAXPROCS=4 per check. The
soaks used a 1 GiB/no-swap/one-core cgroup, GOMAXPROCS=2, one database owner
and one serial MCP caller. The dataset contained 128 overwrite slots and one
near-anchor cell with a fixed idempotency receipt, using a synchronous plaintext
changefeed. Core/advanced inventories alternated at maintenance restarts;
representative core operations ran in both profiles. This does not establish
coverage of every advanced tool or policy variant in the soak.

Both soaks used the same runner binary SHA256:

```text
d3de94d7a40f33da2bbb27b00ccc1e1e9fd2081ed69917fe04ed3f9fc7a4114e
```

Its source and runbook were uncommitted during the runs and subsequently
committed as Mosaic `c33bf1d` (bounded staged qualification runner). The source
hashes were checked unchanged between stages. HexxlaDB `b0a12a1` adds README
examples only. These commits do not alter the runtime services exercised by
the published candidate. Neither these documentation changes nor a new binary's
build provenance should be presented as a separate completed runtime run.
Material runtime/dependency/profile changes require assessment and affected
qualification to be rerun. Existing tags are unchanged.

## Completed baseline and recovery

| Repository | Commands | Result |
| --- | --- | --- |
| HexxlaDB | `task ci`, `task govulncheck`, `task integration`, `task fuzz`, `task build-all` | Pass |
| Mosaic | `task ci`, `task build-all` | Pass |
| MCP Ratchet | `make ci`, `make build-all` | Pass |

All nine baseline commands exited successfully. Crash integration exercised
SIGKILL commit/changefeed phases, authenticated reclaim recovery and MVCC
reopen/churn/pruning. Fuzzing was a short decoder smoke run; cross-builds are
build evidence, not runtime qualification of other platforms. Scanners reported
zero reachable vulnerabilities; HexxlaDB also reported a module-level finding
outside imported/called code.

Eight disposable cross-stack recovery configurations passed, then passed again
with race instrumentation: plaintext/encrypted × sync/lazy changefeed ×
core/advanced MCP. Separate preparation/restoration processes checked real HTTP
MCP and Ratchet gates, matched backup recovery sets, records/vectors/facet/edge/
seam/history oracles, cursor preservation, receipt replay without duplication,
post-restore writes, pruning, supported compaction and encryption rotation.
Vectors were deterministic 2D fixtures; no live Ollama inference was used.

Published HexxlaDB v0.7.0 → v0.8.0-alpha.1 supported-format upgrade and
backup-based rollback passed for plaintext and encrypted data. This is not a
claim that old binaries can read every new file or that retired-format
conversion has been operationally qualified. Initial rehearsal errors involving
slice comparison and unsupported encrypted-compaction procedures were diagnosed
and corrected in the harness; failed attempts remain in the local bundle.

Encrypted compaction creates a new salt and rejects the old encrypted feed.
The encrypted drill verified that refusal and preserved its original restored
set instead of adopting the compacted candidate. Rotation on the retained set
passed. History-preserving encrypted compaction remains an open limitation.

## Completed short soaks

| Observation | 30-minute lexical | 1-hour lexical |
| --- | --- | --- |
| UTC start | 13:19:49 | 13:55:00 |
| UTC finish | 13:49:51 | 14:55:01 |
| Reviewed result / service exit | Pass / 0 | Pass / 0 |
| Successful MCP calls, excluding seeding | 2,144 | 2,084 |
| Required call floor | 450 | 900 |
| Verified backups / compactions / reopens | 3 / 3 / 3 | 3 / 3 / 3 |
| Receipt replays / pre-dispatch cancellation probes | 3 / 3 | 3 / 3 |
| Expected fresh-session Ratchet denials | 4 | 5 |
| Idle interval | 10 seconds | 31 minutes |
| p95 / p99 MCP latency | 17 / 19 ms | 17 / 19 ms |
| Observed OS memory peak, including startup | 23.58 MiB | 24.39 MiB |
| Workload-sampled memory peak | 17.39 MiB | 17.54 MiB |
| Sampled memory growth after warmup | 7.29 MiB | 6.51 MiB |
| Peak sampled run-directory size | 3.04 MiB | 2.93 MiB |
| Swap / recorded OOM or memory-limit events | Zero | Zero |
| Minimum observed host available RAM | 7.00 GiB | 6.37 GiB |
| Minimum observed filesystem availability | 832.29 GiB | 832.11 GiB |
| Timed observations | 180 | 359 |

The limits were p95 ≤5 seconds, p99 ≤30 seconds, 1 GiB runner memory, ≤256 MiB
post-warmup growth, a monitored 512 MiB directory stop budget, hard 128 MiB
per-file limit, and host reserves of 4 GiB available RAM / 10 GiB free disk.
All reviewed targets passed. Final directories including review/source artifacts
were approximately 3.2 / 3.1 MiB. Periodic sample peaks differ from OS lifetime
peaks and may miss short startup/maintenance bursts.

The 1-hour samples show 1,435 operations unchanged from 14:15:08 to 14:45:59
UTC, then continuing work. The successful post-idle assertion and reconnection
verified session expiry and fresh Ratchet prerequisites. It intentionally had
about 29 active minutes, explaining its lower call count than the 30-minute
stage. Final compaction reduced the source primary from 596 to 68 KiB in the
first stage and 552 to 68 KiB in the second; all 129 live cells passed the final
oracle. Feed history was not reset to obtain a pass.

## Remaining gates and priorities

1. **Complete qualification for a declared support scope.** Run/review the
   24-hour workload. For vector-enabled claims, first configure a dedicated
   resource-bounded Ollama service, pin provider/model identity, pass live
   preflight and repeat short stages with real embeddings. Add representative
   concurrent callers/readers and advanced/custom-policy failure workloads to
   cover the broader beta gates; the current serial lexical runner alone does
   not satisfy them. Regenerate applicable capacity evidence at the intended
   operating size. Keep unsupported profiles explicitly excluded.
2. **Define sustainable maintenance.** Design consumer-coordinated changefeed
   archival/reclamation and history-preserving encrypted compaction before
   claiming those long-lived operating capabilities. The sidecar remains
   append-only; primary compaction does not bound it. A watchdog that stops on
   budget exhaustion is containment, not indefinite capacity reclamation.
3. **Simplify supported compatibility.** Decide the minimum HexxlaDB library
   version and remove Mosaic's legacy introspection fallbacks when policy
   permits. Preserve supported database-file recovery fixtures separately.
4. **Bound expensive responses.** Profile and consolidate response shaping,
   pre-estimation and per-call clamps. Native diagnostics/rebuilds need work/
   memory limits before broader exposure. Exact tokenizer accounting stays
   host-owned.
5. **Let the first real use case select features.** Bulk deletion is the clearest
   pending native helper; define chunking, cancellation, outcomes, progress and
   changefeed semantics before implementation. Caching/index/graph/provenance/
   terrain research remains deferred until a workload demonstrates benefit.

After beta gates pass, complete the named-owner 14-day real-workflow pilot,
upgrade/restore/rollback evidence and v1 contract/support commitments. No further
stage or feature work was started by this documentation update.

## Evidence custody

The raw bundles are local, ignored development artifacts under Mosaic:

- `.tmp/qualification/alpha-baseline-20261111T124645Z/`: manifest, command logs,
  harness source/hashes, failed-attempt diagnosis, final recovery/upgrade results.
- `.tmp/qualification/soak-20261111T131949Z-698233/`: 30-minute manifest, summary,
  samples, reviewed checks, source snapshot and successful service-exit evidence.
- `.tmp/qualification/soak-20261111T135500Z-746693/`: equivalent 1-hour evidence,
  including raw idle/resumption observations and 22 passing review checks.

This tracked report preserves the reviewed summary. Raw logs/data are not
included in Git and are not available merely by cloning the repositories.
Attach or archive the reviewed, credential-free bundles in a durable qualification
issue/artifact location before promotion; do not infer their availability from
this report. They contain synthetic data only. The exact publication/install
check was performed in the preceding publication step, separately from the
nine-command baseline.
