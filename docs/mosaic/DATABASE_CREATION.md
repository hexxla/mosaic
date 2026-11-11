# Creating a Mosaic HexxlaDB file

Use [mosaic-create-db](../../cmd/mosaic-create-db) for an empty database or
[mosaic-seed](../../cmd/mosaic-seed) for the demo corpus with Ollama embeddings.
Both commands build a private database and run HexxlaDB integrity checks before
copying the recovery set to the destination. The destination must be unused.
The server opens an existing primary and reports an error for a missing path.

Current HexxlaDB opens retain a `{primary}.lock` coordination sidecar, even
after close. Leave it in place; it is not the WAL or a recoverable data file.
Do not unlink it while any process may use that database path.

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

## Change the embedding model

Mosaic records the configured Ollama model name and tag inside HexxlaDB.
`mosaic-seed` records it before writing vectors. An empty vector store binds
on its first MCP startup without contacting Ollama. Startup rejects a
different model name even when its vector width matches. A database with
existing vectors and no recorded model requires the same migration.

Stop the server, keep a backup of the recovery set, and explicitly regenerate
embeddings with the target model available in Ollama.

~~~sh
MOSAIC_EMBED_MODEL=target-model:tag mosaic-mcp -db PATH -reembed
~~~

Use the same database credentials and custom changefeed options as normal.
If the policy YAML specifies `ollama.embed_model`, set the target there
because YAML takes precedence over the environment. The command exits after
regenerating vectors from every live cell's current text. Start Mosaic again
with the same model configuration after it succeeds.

Re-embedding preserves the database's existing vector width and distance
metric. The vectors and model identity commit together, subject to
HexxlaDB's transaction and WAL limits. A failed model call leaves both
unchanged. Choose a new database layout when the target model needs a different
vector width.

Model names and tags are configuration identifiers. If the weights behind a
tag change, run `-reembed` explicitly with that same tag. When supplying raw
vectors through the advanced tool, use the database's recorded model.
Older Mosaic binaries ignore the recorded model. Use the pre-migration backup
if you need to return to an older writer.

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

## Continue a HexxlaDB changefeed

Mosaic automatically continues an existing `<database-path>-changelog` sidecar.
Use `mosaic-mcp -db PATH -changefeed` to start a feed at the default path, or
`mosaic-mcp -db PATH -changefeed-path SIDECAR` to select a custom path. For an
existing custom feed, pass its matching sidecar on every startup. Mosaic
cannot discover a custom sidecar from the primary path.

Mosaic uses HexxlaDB's synchronous changefeed mode. HexxlaDB validates retained
history and recovers pending feed entries on open. If the primary has retained
changefeed history, pending entries, or registered consumers and the default
sidecar is missing, Mosaic requires the matching custom sidecar or restoration
of the default one before serving writes.
Starting a new feed records future writes. It does not backfill earlier writes.

## Upgrade an existing database

Mosaic pins HexxlaDB commit `2e5ed270ec49`
(`v0.7.1-0.20261109183456-2e5ed270ec49`) for current storage recovery fixes;
there is no release tag for that commit yet. Plaintext v2 and authenticated v3
remain the supported profiles. Close the old writer cleanly and back up the
primary, matching WAL, and any changelog before upgrading. Recover pending redo
with the matching old binary first.

Current writes persist complete WAL framing and a plaintext-header checksum
that `v0.7.0` writers cannot maintain. Encrypted changelogs upgrade to v3 through
validated replacement; allow temporary space for the full sidecar. Derived
HNSW state may fall back to exact scoring until an offline rebuild with the
matching HexxlaDB build. After new writes, use the pre-upgrade recovery set
to return to an older binary. Do not open upgraded files with an older writer.
For engine v1 or legacy AES-XTS files,
use the pinned v0.6.0 offline converter on a copy before upgrading. Keep the
original recovery set for rollback and verify the converted candidate before
switching Mosaic to it.
[HexxlaDB offline conversion](https://github.com/hexxla/hexxladb/blob/2e5ed270ec49/docs/hexxladb/OPERATIONS.md#offline-conversion-of-retired-files)

Previously stored vectors may already be stale after an earlier cell
overwrite. If affected, regenerate embeddings from the current cell content.
Rebuilding HNSW alone reuses the stored vectors.
