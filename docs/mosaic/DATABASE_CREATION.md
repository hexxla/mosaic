# Creating a Mosaic HexxlaDB file

Use [mosaic-create-db](../../cmd/mosaic-create-db) for an empty database or
[mosaic-seed](../../cmd/mosaic-seed) for the demo corpus with Ollama embeddings.
Both commands build a private database and run HexxlaDB integrity checks before
copying the recovery set to the destination. The destination must be unused.
The server opens an existing primary and reports an error for a missing path.

## Naming the database file

| Precedence | Path |
| --- | --- |
| 1 | The file passed to **-db** |
| 2 | **-name** selects a file named **name.hexxla** under **-db-dir**, **MOSAIC_DB_DIR**, or **.tmp**, in that order |
| 3 | **MOSAIC_DB_PATH** |
| 4 | **mosaic.hexxla** in the current directory for the creator and seed commands |

The **-db** and **-name** flags are mutually exclusive.
[Path resolution](../../internal/config/mosaic_db_path.go)

Defaults are 4 KiB pages, a 16 KiB maximum value, and 384-dimensional cosine
embeddings. Both current storage formats use MVCC. Use the layout flags on
the creator or seed command to select a compatible embedding dimension and
distance metric. [Layout defaults](../../internal/config/mosaic_hexxla_db.go)

~~~sh
export MOSAIC_DB_PATH="$PWD/.tmp/mosaic.hexxla"
task create-db
task run-mosaic-mcp
~~~

For the demo, use **task seed** on an unused path. It requires Ollama with the
configured embedding model. A repeat seed on an existing primary skips it.
**task mosaic-dev** runs the seed task before starting MCP.
[Ollama configuration](./MOSAIC_CONFIG.md#ollama)

## Encrypted database (passphrase or raw key)

| Command | Passphrase precedence |
| --- | --- |
| **mosaic-mcp**, **mosaic-create-db**, and **mosaic-seed** | **-db-passphrase**, **MOSAIC_DB_PASSPHRASE**, YAML **database.passphrase** |

All three accept a hexadecimal key in **MOSAIC_DB_ENCRYPTION_KEY_HEX**.
Use one credential source. Prefer environment injection for secrets so the
passphrase stays out of command arguments. Use matching credentials on create
and open. [Configuration reference](./MOSAIC_CONFIG.md)

## Replace a database offline

1. Create or seed a candidate at a **different, unused path**. Wait for the
   command to succeed before opening the destination.
2. Stop Mosaic and any other process using the original database. Preserve the
   original primary and its matching WAL and any changefeed as one recovery set.
3. Point **MOSAIC_DB_PATH** or **-db** at the candidate and start Mosaic with
   its matching credentials. Verify the path, layout, and integrity with
   **mosaic_hexxla_health** with `{"full_check":true}`.
4. To roll back, stop Mosaic and restore the original path and credential.
   Preserve any writes made on the candidate separately.

The creator and seed commands refuse **-replace** and **-force**.
If copying the candidate is interrupted, keep the original configuration and
build at another unused path. Keep each candidate's sidecars with its own primary.

~~~sh
task seed MOSAIC_DB_PATH="$PWD/.tmp/demo-next.hexxla"
# Stop Mosaic before changing its database path.
task run-mosaic-mcp MOSAIC_DB_PATH="$PWD/.tmp/demo-next.hexxla"
~~~

## Upgrade an existing database

Current plaintext v2 and authenticated v3 files use the same supported
storage profiles. Close the old writer cleanly and take a verified backup
before opening with the upgraded server. If the old writer left pending
plaintext v2 redo, recover it with the old binary first.
For engine v1 or legacy AES-XTS files,
use the pinned v0.6.0 offline converter on a copy before upgrading. Keep the
original recovery set for rollback and verify the converted candidate before
switching Mosaic to it.
[HexxlaDB offline conversion](https://github.com/hexxla/hexxladb/blob/v0.7.0/docs/hexxladb/OPERATIONS.md#offline-conversion-of-retired-files)

Previously stored vectors may already be stale after an earlier cell
overwrite. If affected, regenerate embeddings from the current cell content.
Rebuilding HNSW alone reuses the stored vectors.
