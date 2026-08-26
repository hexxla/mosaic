# HexxlaDB + Mosaic MCP — troubleshooting

## Iterative **`mosaic_hexxla_delete_cell`** “stops working”

Typical causes:

1. **`allow_delete_cell`** — Default in Mosaic config is **`false`**. Deletes are rejected until **`allow_delete_cell: true`** appears in the policy YAML loaded at server start (**`-policy`** / **`MOSAIC_POLICY_FILE`**). Call **`mosaic_hexxla_get_persistence_policy`** and confirm **`allow_delete_cell`**.
2. **Restart policy** — Edits to the YAML take effect **only after Mosaic restart** (`mosaic-mcp` reload).
3. **Session retrieval cap** — If **`retrieval.session_approx_token_budget`** is set (> 0), cumulative read MCP traffic can hit the cap; subsequent tools (including health or lookups between deletes) may fail with a budget error. Check **`mosaic_hexxla_retrieval_budget_status`**. Deletes themselves are writes and are **not** charged to that metering in Mosaic.
4. **Idempotent deletes** — **`DeleteCell`** on a missing or already deleted coord returns **nil** (no error). Scripts that infer success only from absence of errors may look “stuck”; verify coords with **`query_cells`** / **`search_cells`** or inspect engine errors in logs.

[Mosaic persistence policy](./PERSISTENCE_POLICY.md), [HEXXLA API coverage](./HEXXLA_API_SURFACE_COVERAGE.md) (*prune/delete safety*).

### **`mosaic_hexxla_health`**: disk footprint and MVCC retention

The JSON response includes **`disk`** (**`primary_path`**, **`primary_bytes`**, **`wal_bytes`**, **`total_bytes`**) from the primary Hexxla file and **`{primary}-wal`**, and **`mvcc_retain_commits_behind_head`** (effective Mosaic policy at open, including injected defaults when post-delete pruning is enabled without an explicit YAML value). Use **`disk.total_bytes`** together with **`mvcc`** (**`commit_seq`**, **`versioned_rows`**, **`logical_cells`**) when debugging extend-only file growth vs WAL size.

---

## **`mosaic_hexxla_health`**: tag/source index warnings on MVCC files

Older HexxlaDB releases reported **tag/source index inconsistencies** (“index entries reference deleted or superseded cells”) after **heavy MVCC churn** — multiple **`PutCell`** updates at the same coordinate, **`DeleteCell`**, or many versioned writes — even when storage was coherent.

Reason: MVCC deliberately **retains historical `tag/` and `source/` secondary keys per commit sequence** so time-pinned reads (**`ViewAt`**) stay correct; those keys reference **past** versions, not necessarily the visible head cell. A health probe that only asked “does this coord have a live head?” misclassified kept history as corruption.

**Fix:** Upgrade **`github.com/hexxla/hexxladb`** to a release containing the **HealthCheck** fix (changelog: MVCC secondary validation by commit seq). After upgrading, regenerate or reopen the DB unchanged — no reindex of tags is required for this check.

### **`seam_count` or `mosaic_hexxla_health` seam totals look doubled after resolve**

On **MVCC** files, each **ResolveSeam** appends another `seam/<ULID>/<seq>` row. Older **HealthCheck** scans counted every physical row, so **PutSeam + ResolveSeam** looked like **two** seams. **`FindSeams`** was always correct (it uses the visible version per ULID). Upgrade **HexxlaDB** to a release that deduplicates MVCC seam versions in **`DB.HealthCheck`**.

Residual real issues HealthCheck still catches:

- **Orphan seams** — **[`Tx.DeleteCell`](https://pkg.go.dev/github.com/hexxla/hexxladb)** does **not** remove seams; deleting an endpoint yields seam warnings even after `ResolveSeam`, because health checks validate resolved endpoints too. The public API has no seam delete. Restore the missing cell, or perform a reviewed offline migration/repair that omits the obsolete seam.
- **True orphan secondaries** — e.g. pruned **`cell/`** rows without matching secondary cleanup (operator edge cases).

For retention and MVCC compaction, Mosaic can run bounded **`PruneScheduler.Tick`** after a successful **`mosaic_hexxla_delete_cell`** when **`database.auto_maintain_after_cell_delete`** is enabled (see [configs/config.yaml](../../configs/config.yaml)); there is no separate MCP tool that calls **`PruneCellVersions`** alone. See [HEXXLA API surface](./HEXXLA_API_SURFACE_COVERAGE.md).

---

## Disk size stays ~fixed (e.g. **576 KiB**) after seed, more writes, or “delete everything”

### 1. **File size reflects allocated pages**

New Mosaic databases default to **`PageSize` 4096** ([`internal/config/mosaic_hexxla_db.go`](../../internal/config/mosaic_hexxla_db.go)); nine logical pages occupy about **36 KiB** before any authenticated-encryption overhead. Existing databases retain their creation-time page size, so a legacy **65536-byte** layout with nine pages remains about **576 KiB**. Both are normal small B+tree footprints.

### 2. **Delete does not automatically shrink the primary file**

Plaintext and legacy encrypted v1/v2 files remain **extend-only** and require compaction to reclaim dead pages. Authenticated v3 records reusable pages and consumes them before extending, but reuse alone does not shorten the file; `ReclaimTail` can remove only a contiguous allocator-owned suffix and compaction repacks fragmented/low-fill pages. See HexxlaDB's upstream `docs/hexxladb/OPERATIONS.md` for the format-specific procedure.

### 3. **MVCC deletes still leave a physical row per coordinate**

On MVCC, **`DeleteCell`** adds a **tombstone** (latest version for that coord). **[`PruneCellVersions`](https://pkg.go.dev/github.com/hexxla/hexxladb#DB.PruneCellVersions)** only removes **non-latest** rows with `commit_seq < beforeSeq`. It **always keeps the latest row** per coord — including when that row is a tombstone — so **you cannot “prune away” all evidence of a coord** while that tombstone remains the head version.

**[`Compact`](https://pkg.go.dev/github.com/hexxla/hexxladb#DB.Compact)** copies **all remaining keys** (including tombstones). So after deleting every **visible** cell, **`StatsMVCC.LogicalCells`** can still reflect **one logical coord per former cell** (tombstone rows count), and the file will **not** approach zero bytes.

### 4. **Why adding cells might not change size**

New **`PutCell`** data often fits into **slack** inside pages already allocated for the btree. The file **only grows when new pages are needed** (splits, overflow, large values). So it is normal for size to stay at **576 KiB** until the workload exceeds what fits in the current page allocation.

### 5. **What to verify if you expect shrink after deletes**

| Check | Why it matters |
| ----- | -------------- |
| **`allow_delete_cell: true`** and policy loaded | Otherwise deletes never run. |
| **`database.auto_maintain_after_cell_delete`** with **`prune: true`** and **`compact: true`** | **`Compact`** is what rewrites the primary to a tight layout; pruning drops **old** versions only. |
| Custom **test client** vs **`mosaic-mcp`** | Post-delete **`CompactSwap`** runs in the Mosaic **[`CellWriterAdapter`](../../internal/adapter/secondary/hexxlastore/cell_writer.go)** after a delete that actually removed a cell. A client that opens HexxlaDB directly never runs that path unless **you** call **`Compact`**. |
| **`stat` both `*.hexxla` and `*.hexxla-wal`** | Total footprint includes the WAL until checkpoint/truncate; Mosaic’s compact swap **removes** the stale WAL after publishing a new primary. |

If you need a **smaller minimum footprint** for tiny databases, create the file with a **smaller `page-size`** (e.g. `4096`) via **`mosaic-create-db` / `mosaic-seed`** flags — **page size is fixed at creation** ([`DATABASE_CREATION.md`](./DATABASE_CREATION.md)).
