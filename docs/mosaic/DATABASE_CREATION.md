# Creating a Mosaic HexxlaDB file

HexxlaDB creates the on-disk file when you call **`hexxladb.Open`** and the path does not exist yet. Mosaic wraps **`Open`** in two commands that apply the **same layout defaults** (MVCC, page size, embeddings — see [`internal/config/mosaic_hexxla_db.go`](../../internal/config/mosaic_hexxla_db.go)) and optionally **at-rest encryption**.

---

## Encrypted database and MCP config (end-to-end)

1. **Create** the file with a passphrase (pick **one** secret channel; same precedence when opening):

   ```bash
   go run ./cmd/mosaic-create-db -db "$HOME/mosaic/secret.hexxla" -db-passphrase "$MOSAIC_DB_PASSPHRASE"
   ```

   Prefer **`MOSAIC_DB_PASSPHRASE`** in the environment instead of **`-db-passphrase`** on the command line (avoids **`ps`** leakage); see [Encrypted database](#encrypted-database-passphrase-or-raw-key) below.

2. **Open with `mosaic-mcp`** using the **same** credentials (**`-db-passphrase`** → **`MOSAIC_DB_PASSPHRASE`** → YAML **`database.passphrase`**):

   ```bash
   export MOSAIC_DB_PATH="$HOME/mosaic/secret.hexxla"
   export MOSAIC_DB_PASSPHRASE='…'
   go run ./cmd/mosaic-mcp -policy configs/config.yaml
   ```

   Optional: add **`database.passphrase`** under **`database:`** in policy YAML ([`MOSAIC_CONFIG.md`](./MOSAIC_CONFIG.md)) only if storing a hint on disk is acceptable — env is usually safer.

3. **Align paths:** **`MOSAIC_DB_PATH`**, **`-db`**, or **`-name`** / **`MOSAIC_DB_DIR`** must resolve to the **same file** you created ([`ResolveMosaicDBPath`](../../internal/config/mosaic_db_path.go)).

Further detail: **[`MOSAIC_CONFIG.md`](./MOSAIC_CONFIG.md)** (`database`), **[`PERSISTENCE_POLICY.md`](./PERSISTENCE_POLICY.md)** (encryption).

---

### Naming the database file (`-name`, `-db`, env)

Path resolution is centralized in **`internal/config/mosaic_db_path.go`** ([**`ResolveMosaicDBPath`](../../internal/config/mosaic_db_path.go)).

| Precedence | Mechanism |
| ---------- | ------- |
| 1 | **`-db`** — full path to the `.hexxla` file |
| 2 | **`-name`** — writes **`<db-dir>/<name>.hexxla`** · **`db-dir`** from **`-db-dir`**, else **`MOSAIC_DB_DIR`**, else **`.tmp`** |
| 3 | **`MOSAIC_DB_PATH`** |
| 4 | Default **`mosaic.hexxla`** in the **process working directory** (shell cwd) · *(create-db / seed only; MCP requires 1–3)* |

**`-db`** and **`-name`** are mutually exclusive. **`mosaic-mcp`** accepts **`-db`** / **`-name`** / **`-db-dir`** the same way and no longer requires **`MOSAIC_DB_PATH`** when you pass **`-name`** or **`-db`**.

| Tool | When to use |
| ---- | ----------- |
| **`mosaic-create-db`** | Empty database only (no Ollama). Fast path for “create file then point **`mosaic-mcp`** at **`MOSAIC_DB_PATH`**. |
| **`mosaic-seed`** | Same creation step, then fills demo cells/embeddings if the built-in corpus is non-empty (needs Ollama for embeddings). |

**`mosaic-mcp`** does not create files; it opens an existing path from **`MOSAIC_DB_PATH`** (and optional encryption — **`BuildHexxlaOpenOptions`** / **`ApplyHexxlaEncryption`**).

---

## Encrypted database (passphrase or raw key)

Use **one** of: **`-db-passphrase`**, **`MOSAIC_DB_PASSPHRASE`**, optional **`database.passphrase`** in policy YAML (via **`-policy`** / **`MOSAIC_POLICY_FILE`**), or **`MOSAIC_DB_ENCRYPTION_KEY_HEX`** (raw key; never combine with passphrase sources). Precedence for passphrase: flag → env → YAML.

**About `-force` / `-replace`:** Both flags mean the same thing: **delete the existing file at the resolved path and create (or seed) again.** They are **not** required for encryption itself — only when you want to overwrite a database that is already on disk. If a file already exists and you did not pass **`-replace`/`-force`**, the command **errors** (create-db) or **skips seed** (seed) and tells you to pick another **`-db`** / **`-name`** or overwrite explicitly.

For a **new path**, omit them:

```bash
export MOSAIC_DB_PATH="$PWD/.tmp/brand-new.hexxla"
go run ./cmd/mosaic-create-db -db-passphrase 'your-passphrase-here'
```

These examples create an **encrypted** file at **`MOSAIC_DB_PATH`**. Replace placeholder secrets; avoid committing real passphrases.

### Empty encrypted DB with the password flag (new file)

```bash
export MOSAIC_DB_PATH="$PWD/.tmp/secret.hexxla"
go run ./cmd/mosaic-create-db -db-passphrase 'your-passphrase-here'
```

### Same using environment (often preferable — no passphrase in shell history from `-db-passphrase`)

```bash
export MOSAIC_DB_PATH="$PWD/.tmp/secret.hexxla"
export MOSAIC_DB_PASSPHRASE='your-passphrase-here'
go run ./cmd/mosaic-create-db
```

### Overwriting an existing file

Use **`-force`** or **`-replace`** when **`MOSAIC_DB_PATH`** already exists:

```bash
go run ./cmd/mosaic-create-db -replace -db-passphrase 'your-passphrase-here'
```

### Encrypted seed database (Ollama required for embeddings)

```bash
export MOSAIC_DB_PATH="$PWD/.tmp/seed-secret.hexxla"
go run ./cmd/mosaic-seed -db "$MOSAIC_DB_PATH" -db-passphrase 'your-passphrase-here'
```

(Re-seed from scratch: add **`-replace`** or **`-force`** if the file already exists.)

### Open encrypted DB with MCP (existing file + passphrase)

```bash
export MOSAIC_DB_PATH="$PWD/.tmp/secret.hexxla"
go run ./cmd/mosaic-mcp -policy configs/config.yaml -db-passphrase 'your-passphrase-here'
```

Or only env (no flag):

```bash
export MOSAIC_DB_PATH="$PWD/.tmp/secret.hexxla"
export MOSAIC_DB_PASSPHRASE='your-passphrase-here'
go run ./cmd/mosaic-mcp -policy configs/config.yaml
```

### Make (append flags; quote multiple arguments)

```bash
# New encrypted DB at Makefile MOSAIC_DB_PATH (no -replace unless file exists)
make create-db MOSAIC_CREATE_DB_FLAGS='-db-passphrase your-passphrase-here'

# Overwrite existing file
make create-db MOSAIC_CREATE_DB_FLAGS='-replace -db-passphrase your-passphrase-here'

make run-mosaic-mcp MOSAIC_MCP_FLAGS='-policy configs/config.yaml -db-passphrase your-passphrase-here'
```

**Security:** **`-db-passphrase`** may appear in process listings (`ps`). Prefer **`MOSAIC_DB_PASSPHRASE`** or a secrets manager that injects env for production-like setups.

---

## Examples: shell (`go run`)

From the **repository root**, paths are stable relative to `configs/config.yaml`.

### Empty DB at the default path (`mosaic.hexxla` in cwd, or **`MOSAIC_DB_PATH`**)

```bash
go run ./cmd/mosaic-create-db
```

### Empty DB at an explicit path, replace if present, load passphrase hints from policy YAML

```bash
go run ./cmd/mosaic-create-db -db ./.tmp/my.hexxla -force -policy configs/config.yaml
```

### Encrypted empty DB

See **[Encrypted database](#encrypted-database-passphrase-or-raw-key)** above (`-db-passphrase` or **`MOSAIC_DB_PASSPHRASE`**).

### Seed with defaults (Ollama must be up; **`MOSAIC_OLLAMA_URL`**, **`MOSAIC_EMBED_MODEL`**)

```bash
export MOSAIC_DB_PATH="$PWD/mosaic.hexxla"
go run ./cmd/mosaic-seed -db "$MOSAIC_DB_PATH"
```

### Reseed from scratch with non-default page size

```bash
go run ./cmd/mosaic-seed -db ./mosaic.hexxla -force -page-size 4096
```

### Run MCP against an existing DB + policy file

```bash
export MOSAIC_DB_PATH="$PWD/mosaic.hexxla"
go run ./cmd/mosaic-mcp -policy configs/config.yaml
```

Layout and encryption flags are documented on **`go run ./cmd/mosaic-create-db -help`**, **`go run ./cmd/mosaic-seed -help`**, **`go run ./cmd/mosaic-mcp -help`**.

---

## Examples: Make

The Makefile sets **`MOSAIC_DB_PATH`** (default **`./mosaic.hexxla`** and passes it as **`-db`** to create-db/seed), Ollama URL/model for seed, and MCP listen options. Extra **`go run`** arguments are appended via:

| Variable | Used by |
| -------- | ------- |
| **`MOSAIC_CREATE_DB_FLAGS`** | **`make create-db`** / **`make run-mosaic-create-db`** |
| **`MOSAIC_SEED_FLAGS`** | **`make seed`**, **`make reseed`**, **`make run-mosaic-seed`** |
| **`MOSAIC_MCP_FLAGS`** | **`make run-mosaic-mcp`**, **`make mosaic-dev`** (MCP half only) |

Quote the value when passing multiple flags.

```bash
# Empty DB at Makefile default path
make create-db

# Empty DB with force + policy file (YAML database.passphrase optional)
make create-db MOSAIC_CREATE_DB_FLAGS='-force -policy configs/config.yaml'

# Custom DB path + smaller pages for new file
make create-db MOSAIC_DB_PATH=./projects/custom.hexxla MOSAIC_CREATE_DB_FLAGS='-force -page-size 4096'

# Seed (skips if DB exists unless you reseed)
make seed

make reseed MOSAIC_SEED_FLAGS='-page-size 8192'

# MCP with persistence policy YAML
make run-mosaic-mcp MOSAIC_MCP_FLAGS='-policy configs/config.yaml'

# Encrypted DB / MCP (see "Encrypted database" section — use -replace only if path exists)
make create-db MOSAIC_CREATE_DB_FLAGS='-db-passphrase your-passphrase-here'
make run-mosaic-mcp MOSAIC_MCP_FLAGS='-policy configs/config.yaml -db-passphrase your-passphrase-here'

# Dev loop: seed (if missing) then MCP — pass seed layout flags only on the seed step:
make mosaic-dev MOSAIC_SEED_FLAGS='-embedding-dim 384' MOSAIC_MCP_FLAGS='-policy configs/config.yaml'
```

**Note:** **`make mosaic-dev`** runs **`seed`** first. Only **`MOSAIC_SEED_FLAGS`** affects that step; **`MOSAIC_MCP_FLAGS`** applies when **`mosaic-mcp`** starts.

---

## Quick reference (defaults)

Unless overridden by flags, **`DefaultMosaicDatabaseLayout`** uses: MVCC on, page size **65536**, max value bytes **16384**, embedding dimension **384**, distance metric **cosine**. See [`mosaic_hexxla_db.go`](../../internal/config/mosaic_hexxla_db.go).
