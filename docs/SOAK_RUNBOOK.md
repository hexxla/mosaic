# Bounded stack soak

Qualification owner: **void, the requesting repository maintainer**. The stack
remains alpha. This runbook prepares explicit 30-minute, 1-hour and 24-hour
stages; no command automatically chains stages or promotes a release. Review
each result before starting the next stage. The governing gates are in
[release qualification](RELEASE_QUALIFICATION.md).

## Scope and safety limits

The initial profile is Linux/amd64, local btrfs or another trusted local
filesystem, one database owner, one serial MCP caller, plaintext database with
synchronous changefeed, and either lexical-only retrieval or CPU-only live
Ollama embeddings. A real loopback HTTP MCP server composes Mosaic's current
adapters with its pinned HexxlaDB and Ratchet dependencies. Core and advanced
inventories alternate on maintenance restarts. This is an adapter composition
workload; it does not independently qualify the mosaic-mcp CLI's signal handling,
concurrent clients, every advanced tool, encrypted compaction or host power loss.
The baseline/recovery tests supply separate evidence for their stated scope.

The launcher requires systemd user services and cgroup v2. It runs the build and
runner inside the same bounded service, with no background continuation:

| Limit | Enforcement / target |
| --- | --- |
| Runner memory | OS MemoryMax=1 GiB, MemoryHigh=768 MiB, no swap; Go soft target 512 MiB |
| Runner CPU / tasks | OS CPUQuota=100% (one core), TasksMax=128; GOMAXPROCS=2 |
| Individual file size | OS LimitFSIZE=128 MiB, including primary, WAL, feed and logs |
| Whole run directory | Stop at 512 MiB logical file bytes; sampled every second |
| Maintenance headroom | Refuse a maintenance cycle once the directory reaches one third of its budget |
| Host reserve | Stop below 4 GiB MemAvailable or 10 GiB filesystem available space |
| Live Ollama | Separate, dedicated service: maximum 2 GiB, no swap, one CPU core; exactly one CPU-resident model |
| Duration | systemd runtime ceiling adds 10 minutes for setup/cleanup to the selected stage |

The command refuses unbounded memory/CPU/swap/file-size settings. A one-second
watchdog cancels work on resource failure, including during seeding, idle and
maintenance; per-call contexts are bounded. Disk monitoring is a stop budget,
not a filesystem quota, and can overshoot between samples. Per-file OS caps,
fixed file generations and bounded records additionally constrain growth. Other
applications can still consume host resources: these safeguards do not reserve
RAM/disk globally or guarantee the health of unrelated processes. A dedicated
quota/volume is needed if a strict aggregate disk allocation is required.

Each invocation creates a new private directory under `.tmp/qualification/`.
It never opens an existing user database. It retains two alternating database /
verified-backup generations, not an unbounded sequence. Prior invocation evidence
is retained for review and counts against filesystem free space; remove only
reviewed disposable runs when appropriate. There is no automatic deletion of
previous runs or original user data.

## Frozen workload and acceptance targets

- 128 fixed coordinate slots with approximately 500-byte synthetic records,
  plus one near-anchor cell with a fixed idempotency receipt. Overwrites bound
  live-cell count at 129; raw record limit is 16 KiB. Live mode embeds actual
  writes through Mosaic and uses provider-backed semantic searches. Lexical mode
  does not substitute synthetic vectors for live-model qualification.
- One cycle per second, plus request execution time. Every fifth cycle performs
  a gated overwrite (`list_tags`, then `put_cell`). Other cycles alternate exact
  lookup and bounded search (eight results, at most 4096 scanned rows).
- Each ten minutes, and once at the end: compare the record/vector oracle, drain
  and advance the named feed consumer, prune obsolete cell versions, capture and
  verify a full backup, close MCP/storage, compact to the inactive generation,
  copy the matched plaintext sidecar, reopen and verify content/health/cursor.
  Then require fresh Ratchet prerequisites, retry the same durable receipt with
  no extra commit, and check pre-dispatch cancellation has no durable effect.
- The 30-minute stage includes a 10-second idle interval. The 1-hour and 24-hour
  stages include one 31-minute interval to exercise actual HTTP session expiry,
  with continued resource observations, then a fresh session. Work resumes after
  expiry. Longer stages intentionally pause maintenance during this idle window.
- Zero unexpected MCP/provider/storage errors, zero oracle mismatches, zero
  unintended commits, and all maintenance/backup/reopen checks successful.
  Expected policy denials and pre-canceled requests are counted separately.
- p95 MCP latency ≤5 seconds, p99 ≤30 seconds. Individual lexical calls must
  complete within 5 seconds; live-mode calls within 30 seconds. Setup's live
  provider probe warms the model. Seeding is outside stage operation/latency
  counts, but remains subject to watchdogs and per-call ceilings.
- Maintenance ≤30 seconds per cycle; hard memory/disk limits and host reserves
  respected. Combined runner/provider cgroup memory growth after five-minute
  warmup must stay within 256 MiB of its first post-warmup sample. This includes
  charged page cache, so it is a conservative containment target, not proof that
  heap usage is flat. Timed samples allow trend review beyond the pass flag.
- At least 450 successful MCP calls in 30 minutes, 900 in 1 hour, and 21,600 in
  24 hours. These deliberately modest floors allow idle/maintenance/inference;
  the last exceeds the shared plan's 10,000-operation minimum. Calls include
  prerequisites and reads; the fixed mix guarantees ongoing writes as well.
  Calls are spread across the timed stage; seeding cannot satisfy the floor.

**Yes, compaction is used.** Pruning limits retained versions; compaction rewrites
and repacks the primary. It does not compact/reset the append-only changefeed.
The feed and retained backups remain inside the disk stop budget. If history
exhausts the budget, the run fails and stops; it never silently resets consumers
or deletes history to make the stage pass. Encrypted compaction retaining old
feed history remains excluded because the candidate uses a new encryption salt.

These small-workload targets were chosen before long-stage measurement. They
are not production capacity guarantees. Freeze the runner revision, sources,
provider/model and targets before the first stage; changes require recording
and rerunning affected stages. Review the full timeline and per-kind coverage,
not only a final pass. This runner does not alone satisfy every beta gate.

## Commands

Run from the Mosaic root with Go 1.27.4 and `systemd-run` on PATH. First verify
prerequisites without starting a stage:

```bash
scripts/soak.sh --preflight
```

Start and review each stage separately:

```bash
scripts/soak.sh --duration 30m
scripts/soak.sh --duration 1h
scripts/soak.sh --duration 24h
```

The commands above use lexical mode. Do not run all three as an unattended
sequence. `--verify --duration 20s --maintenance-every 8s --interval 200ms` is
available for runner development and reports `verification-pass`, never a soak
pass. It does not replace any stage. Do not suspend the host during a stage.

The launcher prints its `mosaic-soak-<UTC>-<pid>.service` unit name. Stop that
specific unit with `systemctl --user stop <printed-unit-name>`; cancellation
must be reviewed as a failed/incomplete stage, not a pass. A hard cgroup kill
or missing final summary cannot count as success. The command stays attached;
keep the terminal/session alive and inspect the unit result.

Artifacts in the matching `.tmp/qualification/soak-<UTC>-<pid>/` directory:
`manifest.json`, ten-second `samples.jsonl`, and `summary.json`. The summary
contains owner, runner binary hash, build provenance/pinned modules, frozen
options/targets, successful calls, expected denials, retries, cancellations,
maintenance counts, latency percentiles, peaks and failure reason. Attach the
reviewed report plus exact source revision/diff and raw artifacts to the
qualification issue before promotion. Keep historical failures with their cause.

## Live Ollama prerequisites

Host inspection on 2026-11-11 found no Ollama executable, active service or
listener at the normal port 11434. No software/model was downloaded and no
service was installed by runner preparation.

1. Install and record an explicit Ollama version using the
   [official Linux guidance](https://docs.ollama.com/linux). Avoid changing that
   version during qualification. Mosaic currently uses `/api/embeddings`; its
   live preflight verifies that endpoint on the chosen provider version.
2. Use a small embedding-only model. The proposed `all-minilm:latest` download
   is listed as 46 MB in the [official library](https://ollama.com/library/all-minilm).
   Download size is not inference memory. Dimension 384 is a proposed value,
   verified by the actual probe; a mismatch fails before the workload starts.
3. Start a **dedicated** user service on port 11435, with its own model store,
   explicit CPU-only inference and resource limits. Configure one loaded model,
   one parallel request, a small queue and finite keepalive; see
   [Ollama configuration](https://docs.ollama.com/faq). Do not reuse an unrelated
   unrestricted Ollama process. For an installed CPU-capable Ollama binary:

   ```bash
   mkdir -p "$HOME/.local/share/mosaic-soak-models"
   systemd-run --user --collect --unit=mosaic-soak-ollama \
     -p MemoryHigh=1536M -p MemoryMax=2G -p MemorySwapMax=0 \
     -p CPUQuota=100% -p TasksMax=128 -p RuntimeMaxSec=90000 \
     /usr/bin/env OLLAMA_HOST=127.0.0.1:11435 \
     OLLAMA_MODELS="$HOME/.local/share/mosaic-soak-models" \
     OLLAMA_NUM_PARALLEL=1 OLLAMA_MAX_LOADED_MODELS=1 \
     OLLAMA_MAX_QUEUE=4 OLLAMA_KEEP_ALIVE=5m \
     CUDA_VISIBLE_DEVICES=-1 ROCR_VISIBLE_DEVICES=-1 \
     "$(command -v ollama)" serve
   OLLAMA_HOST=127.0.0.1:11435 ollama pull all-minilm:latest
   ```

   Verify CPU use on the installed version: environment variables alone are not
   proof. The preflight rejects nonzero `size_vram` or additional loaded models
   from `/api/ps`. GPU qualification requires separate GPU memory controls and
   is outside this profile. The 2 GiB service limit constrains host RAM, not VRAM.
   Model download/setup is separate from the soak; check disk space before it.
Restart the dedicated service before each stage so its runtime ceiling covers
   that complete stage.
4. Read `/api/tags` on port 11435 (`curl --fail http://127.0.0.1:11435/api/tags`) and record the exact selected model name and
   full lowercase SHA256 digest (strip a leading `sha256:`). Pin that value:

   ```bash
   scripts/soak.sh --preflight --mode ollama \
     --model all-minilm:latest --dimension 384 --digest <full-sha256>
   scripts/soak.sh --duration 30m --mode ollama \
     --model all-minilm:latest --dimension 384 --digest <full-sha256>
   ```

   Use the same live options for subsequent 1-hour and 24-hour stages. Preflight
   checks service cgroup limits, provider version, pinned digest, actual embedding
   dimension/finite values and CPU residency. Digest is rechecked before/after
   every application embedding; model identity is persisted with the database.
   Re-embedding is required if it changes. No automatic model pull occurs.
5. Stop the dedicated provider after the run/review with
   `systemctl --user stop mosaic-soak-ollama.service`; it is separately bounded
   and is not implicitly stopped when the runner exits. Its combined memory
   is included in the watchdog's observations. Keep provider version/service
   logs with the report. Review external journal/model-store growth separately;
   neither is part of the runner's 512 MiB directory budget.

Run the same staged sequence with live embeddings before making vector-enabled
beta claims. A lexical-only pass cannot establish provider correctness or
resource use. A failure needs diagnosis; do not raise a resource limit just to
obtain a passing result.
