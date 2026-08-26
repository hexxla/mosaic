package main

import "github.com/hexxla/hexxladb"

// seedMessage is a single conversation turn with a role, content, and tags.
type seedMessage struct {
	role    string
	content string
	tags    []string
}

// seedConversation is the full corpus seeded into the demo database.
// It spans four thematic sessions deliberately chosen to exercise every index
// path (tag/, source/, time/, full-scan), multiple contradiction / supersession
// patterns, a rich tag vocabulary, and varied confidence levels.
//
// This slice matches github.com/hexxla/hexxladb/examples/conversational_memory/seed_data.go
// (82 turns · 5 thematic sessions — the canonical conversational_memory demo corpus).
var seedConversation = []seedMessage{
	// ──────────────────────────────────────────────────────────────
	// SESSION A: Communication preferences & workflow (turns 0-11)
	// Establishes a contradiction / supersession pair used in Phase 3/4.
	// ──────────────────────────────────────────────────────────────
	{
		"user",
		"I prefer detailed technical explanations with code examples — I learn best by reading full implementations.",
		[]string{"preference", "communication-style", "learning", "user-123"},
	},
	{
		"assistant",
		"Understood. I will provide thorough technical responses, complete runnable snippets, and inline comments.",
		[]string{"acknowledgment", "commitment", "communication-style", "user-123"},
	},
	{
		"user",
		"Actually, please keep responses concise. I changed my mind — bullet points over prose.",
		[]string{"preference", "communication-style", "contradiction", "update", "user-123"},
	},
	{
		"assistant",
		"Got it. Switching to concise bullet-point style. I will only expand into prose when you ask for depth.",
		[]string{"acknowledgment", "commitment", "communication-style", "user-123"},
	},
	{
		"user",
		"One more update: for complex architectural topics I still want depth, otherwise keep it short.",
		[]string{"preference", "communication-style", "conditional", "nuance", "user-123"},
	},
	{
		"assistant",
		"Perfect. Rule applied: architecture and design → detailed; implementation Q&A → concise bullets.",
		[]string{"acknowledgment", "rule", "communication-style", "user-123"},
	},
	{
		"user",
		"I work across three time zones. Please always include UTC offsets when you mention schedules.",
		[]string{"preference", "workflow", "scheduling", "timezones", "user-123"},
	},
	{
		"assistant",
		"Noted. All time references will include UTC offsets, e.g. 14:00 UTC+1 / 09:00 UTC-4.",
		[]string{"acknowledgment", "commitment", "scheduling", "user-123"},
	},
	{
		"user",
		"I use GitHub Flow: feature branches, PRs, squash merges. No gitflow or trunk-based for my personal projects.",
		[]string{"preference", "workflow", "git", "branching-strategy", "user-123"},
	},
	{
		"assistant",
		"Understood. GitHub Flow: short-lived feature branches from main, PR review, squash merge, delete branch.",
		[]string{"acknowledgment", "fact", "git", "workflow", "user-123"},
	},
	{
		"user",
		"My editor is Neovim with LSP. I do not use VS Code. Please don't suggest VS Code extensions.",
		[]string{"preference", "tooling", "editor", "neovim", "user-123"},
	},
	{
		"assistant",
		"Confirmed. Will recommend Neovim plugins and LSP configs only. No VS Code suggestions.",
		[]string{"acknowledgment", "commitment", "tooling", "user-123"},
	},

	// ──────────────────────────────────────────────────────────────
	// SESSION B: HexxlaDB technical deep-dive (turns 12-29)
	// Exercises the 'fact', 'technical', 'api', 'architecture' tags.
	// ──────────────────────────────────────────────────────────────
	{
		"user",
		"What makes HexxlaDB different from a traditional key-value store like RocksDB or BadgerDB?",
		[]string{"question", "architecture", "comparison", "database", "user-123"},
	},
	{
		"assistant",
		"HexxlaDB stores data in a hexagonal lattice. Pack encodes 2-D axial coordinates (q,r) into stable Morton-ordered keys with useful spatial locality. Exact rings are enumerated with Ring, WalkRings, or transaction ring walks rather than assumed to be one contiguous key range.",
		[]string{"fact", "architecture", "hexagonal-grid", "morton-encoding", "user-123"},
	},
	{
		"user",
		"How does MVCC work in HexxlaDB — what exactly is a CommitSeq?",
		[]string{"question", "mvcc", "internals", "user-123"},
	},
	{
		"assistant",
		"Each Update transaction increments a monotonic uint64 CommitSeq. Cell writes are stored with a version suffix (8-byte big-endian CommitSeq). ViewAt(seq) pins a read snapshot to that generation; cells written after seq are invisible to the snapshot.",
		[]string{"fact", "mvcc", "snapshot", "internals", "user-123"},
	},
	{
		"user",
		"Can I store embeddings alongside cell content for hybrid retrieval?",
		[]string{"question", "embeddings", "retrieval", "roadmap", "user-123"},
	},
	{
		"assistant",
		"Yes. PutEmbedding stores one vector per cell and persists the dimension detected from the first write unless it was configured at Open. SearchByEmbedding uses the maintained HNSW index with a flat fallback, and QueryCells can combine embedding similarity with structured filters.",
		[]string{"fact", "embeddings", "retrieval", "api", "user-123"},
	},
	{
		"user",
		"What is the max key size and value size? I have some large JSON payloads.",
		[]string{"question", "limits", "configuration", "user-123"},
	},
	{
		"assistant",
		"Max key: 256 bytes. Max value: configurable per-database via Options.MaxValueBytes (512 to 1048576 bytes; default 8192). Values exceeding the inline leaf threshold spill to overflow pages automatically. Always-on DEFLATE compression reduces stored size transparently for values >= 64 bytes.",
		[]string{"fact", "limits", "configuration", "api", "compression", "user-123"},
	},
	{
		"user",
		"Walk me through the B+ tree page layout — I want to understand on-disk format.",
		[]string{"question", "internals", "btree", "on-disk-format", "user-123"},
	},
	{
		"assistant",
		"Page size is persisted per database: 4, 8, 16, or 64 KiB, with 4 KiB the default for new files. Page 0 reserves a 512-byte database header. B+ tree nodes use a 64-byte common header; leaf entries store key and value lengths, while internal nodes store separator keys and 8-byte child page IDs. Multi-byte fields are big-endian.",
		[]string{"fact", "internals", "btree", "page-layout", "user-123"},
	},
	{
		"user",
		"What does the WAL give me that fsync on the data file alone doesn't?",
		[]string{"question", "durability", "wal", "internals", "user-123"},
	},
	{
		"assistant",
		"The WAL allows atomic multi-page commits. A crash after WAL sync but before data-file write leaves the WAL intact; on next open, pages are replayed deterministically. Without a WAL, a partial multi-page write could leave the B+ tree in an inconsistent split state.",
		[]string{"fact", "durability", "wal", "crash-safety", "user-123"},
	},
	{
		"user",
		"How does the changefeed work and what op codes does it emit?",
		[]string{"question", "changefeed", "api", "user-123"},
	},
	{
		"assistant",
		"Enable with Options.ChangelogEnabled. PutCell, PutSeam, ResolveSeam, PutFacet, PutEdge, and DeleteCell project recoverable at-least-once records to <dbpath>-changelog. Stable operation codes run from ChangelogOpPutCell (0x01) through ChangelogOpDeleteCell (0x06). ReadChangelogFiltered scopes reads by operation code and key prefix.",
		[]string{"fact", "changefeed", "api", "op-codes", "user-123"},
	},
	{
		"user",
		"Is at-rest encryption available? What cipher is used?",
		[]string{"question", "security", "encryption", "user-123"},
	},
	{
		"assistant",
		"Yes. New encrypted databases use authenticated XChaCha20-Poly1305 pages; existing AES-256-XTS files remain readable through the legacy path. A raw Options.EncryptionKey is expanded with HKDF-SHA256, while Options.Passphrase uses Argon2id with the per-database salt. Official encryption also authenticates the header, WAL, verifier, and optional changelog.",
		[]string{"fact", "security", "encryption", "aes-xts", "argon2id", "user-123"},
	},
	{
		"user",
		"How do seams work for tracking contradictions?",
		[]string{"question", "seams", "mvcc", "contradiction", "user-123"},
	},
	{
		"assistant",
		"A seam is a typed relation between two cell coordinates identified by a ULID. SeamTypeConflict marks open contradiction; SeamTypeSupersedes records directional replacement (superseder replaces superseded). Seams have their own secondary indexes (seam-source/, seam-time/) and can be resolved with ResolveSeam. FilterSuperseded in context assembly excludes stale cells and promotes successors.",
		[]string{"fact", "seams", "contradiction", "supersession", "api", "user-123"},
	},
	{
		"user",
		"Can I run SnapshotDiff to feed a CDC pipeline?",
		[]string{"question", "snapshot-diff", "cdc", "mvcc", "user-123"},
	},
	{
		"assistant",
		"DB.SnapshotDiff(ctx, fromSeq, toSeq, config) returns retained cell and seam versions in (fromSeq, toSeq] on MVCC formats v2 and v3. Pruned versions and non-cell/seam mutations are absent, so it is a bounded diagnostic rather than complete replication, audit, or CDC input. Use HexxlaDB's durable changelog consumers for complete logical change processing.",
		[]string{"fact", "snapshot-diff", "cdc", "api", "user-123"},
	},

	// ──────────────────────────────────────────────────────────────
	// SESSION C: Go programming & architecture (turns 30-51)
	// Exercises 'go', 'architecture', 'best-practice', 'opinion' tags.
	// ──────────────────────────────────────────────────────────────
	{
		"user",
		"What's the idiomatic way to handle errors in Go — sentinel errors vs types vs wrapping?",
		[]string{"question", "go", "errors", "best-practice", "user-123"},
	},
	{
		"assistant",
		"Prefer errors.Is / errors.As for comparison; use fmt.Errorf with %w to wrap. Sentinel errors (var ErrFoo = errors.New(...)) for stable public API; error types for structured metadata. Avoid string comparison. In public packages export sentinels, not implementation details.",
		[]string{"fact", "go", "errors", "best-practice", "user-123"},
	},
	{
		"user",
		"I think generics in Go are overused. Most code I see using them would be cleaner without.",
		[]string{"opinion", "go", "generics", "criticism", "user-123"},
	},
	{
		"assistant",
		"A valid perspective shared by many experienced Go engineers. Generics shine for container types, constraints on numeric types, and eliminating genuine duplication. They add cognitive overhead — prefer interface{} + type assertion for small cases; reserve generics for measurable abstraction wins.",
		[]string{"fact", "go", "generics", "nuance", "user-123"},
	},
	{
		"user",
		"What is the hexagonal architecture pattern and why would I use it?",
		[]string{"question", "architecture", "hexagonal", "design-patterns", "user-123"},
	},
	{
		"assistant",
		"Hexagonal (Ports & Adapters) architecture separates domain logic from infrastructure. The domain defines ports (interfaces); adapters implement them (DB, HTTP, queues). Benefits: domain is testable in isolation, adapters are swappable, import cycles are impossible by construction (domain never imports adapters).",
		[]string{"fact", "architecture", "hexagonal", "design-patterns", "user-123"},
	},
	{
		"user",
		"How do I profile memory allocations in a Go program efficiently?",
		[]string{"question", "go", "profiling", "performance", "user-123"},
	},
	{
		"assistant",
		"Use go test -memprofile=mem.out then go tool pprof mem.out. For runtime profiling: import net/http/pprof, hit /debug/pprof/heap. Key metrics: inuse_objects (live), alloc_objects (total). For allocation hotspots: pprof -alloc_objects. Benchmark with -benchmem to catch per-call allocations.",
		[]string{"fact", "go", "profiling", "performance", "tooling", "user-123"},
	},
	{
		"user",
		"What's the best way to structure a large Go module — flat vs nested packages?",
		[]string{"question", "go", "project-structure", "best-practice", "user-123"},
	},
	{
		"assistant",
		"Prefer packages that encode a single concept over flat megapackages. Use internal/ for module-private code. Avoid circular imports by ensuring lower-level packages (types, errors) never import higher-level (business logic). Keep cmd/ thin — composition root only, no business logic.",
		[]string{"fact", "go", "project-structure", "best-practice", "user-123"},
	},
	{
		"user",
		"I've been burned by goroutine leaks. What patterns prevent them?",
		[]string{"question", "go", "concurrency", "goroutine-leaks", "user-123"},
	},
	{
		"assistant",
		"Key patterns: always pair goroutine launch with a cancellation mechanism (context.Context or done channel); use sync.WaitGroup for fan-out; never start a goroutine without knowing when it will stop; use goleak in tests to catch leaks. Avoid fire-and-forget goroutines in library code.",
		[]string{"fact", "go", "concurrency", "goroutine-leaks", "best-practice", "user-123"},
	},
	{
		"user",
		"How should I approach database migration in a zero-downtime deployment?",
		[]string{"question", "database", "migrations", "operations", "user-123"},
	},
	{
		"assistant",
		"The expand-contract pattern: (1) add column/index (backwards-compatible), deploy app that writes both old + new shape, (2) migrate data, (3) deploy app that reads new shape only, (4) drop old column. Never rename or drop in the same deployment that first uses the new shape.",
		[]string{"fact", "database", "migrations", "operations", "best-practice", "user-123"},
	},
	{
		"user",
		"What benchmarking gotchas do Go developers commonly miss?",
		[]string{"question", "go", "benchmarking", "performance", "user-123"},
	},
	{
		"assistant",
		"Common gotchas: (1) not calling b.ResetTimer() after expensive setup; (2) ignoring compiler elimination of results — assign to a package-level sink; (3) GOMAXPROCS=1 hiding lock contention; (4) benchmark duration too short — use -benchtime=5s; (5) not using testing.B.Loop (Go 1.24+) for correct iteration semantics.",
		[]string{"fact", "go", "benchmarking", "performance", "user-123"},
	},
	{
		"user",
		"Do you think dependency injection frameworks are worth the complexity in Go?",
		[]string{"opinion", "go", "dependency-injection", "architecture", "user-123"},
	},
	{
		"assistant",
		"Generally no for idiomatic Go. Manual wiring in main() (or a composition root) is explicit, refactorable, and requires no reflection magic. Wire (Google) is acceptable for very large codebases. Avoid runtime DI containers — they break go vet, make refactoring harder, and hide coupling.",
		[]string{"fact", "go", "dependency-injection", "architecture", "opinion", "user-123"},
	},
	{
		"user",
		"I want to use slog for structured logging. What fields should every log entry carry?",
		[]string{"question", "go", "logging", "observability", "slog", "user-123"},
	},
	{
		"assistant",
		"Minimum required: timestamp (automatic with slog.Default), level, message, and a caller source. Add: request_id / trace_id for correlation, component or subsystem label, and error (as slog.Any(\"err\", err)) on error paths. Avoid logging secrets. Use slog.Group for structured sub-objects.",
		[]string{"fact", "go", "logging", "observability", "best-practice", "user-123"},
	},

	// ──────────────────────────────────────────────────────────────
	// SESSION D: LLM systems & product design (turns 52-71)
	// Exercises 'llm', 'product', 'retrieval', 'rag' tags.
	// ──────────────────────────────────────────────────────────────
	{
		"user",
		"What's the difference between RAG and fine-tuning for adding domain knowledge to an LLM?",
		[]string{"question", "llm", "rag", "fine-tuning", "user-123"},
	},
	{
		"assistant",
		"RAG retrieves relevant context at inference time and injects it into the prompt — no weight updates, always fresh. Fine-tuning bakes knowledge into weights — faster at inference, but stale and expensive to refresh. Use RAG for dynamic, proprietary, or frequently-updated knowledge; fine-tuning for style/format adaptation.",
		[]string{"fact", "llm", "rag", "fine-tuning", "comparison", "user-123"},
	},
	{
		"user",
		"How should I chunk documents for effective RAG retrieval?",
		[]string{"question", "rag", "chunking", "retrieval", "user-123"},
	},
	{
		"assistant",
		"Prefer semantic chunks over fixed-size sliding windows. Options: sentence boundaries, paragraph boundaries, or recursive character splitters respecting markdown headings. Target 256-512 tokens per chunk with 10-15% overlap for boundary context. Smaller chunks → higher precision; larger chunks → more context per hit.",
		[]string{"fact", "rag", "chunking", "retrieval", "best-practice", "user-123"},
	},
	{
		"user",
		"What is token budget management and why does it matter for conversational memory?",
		[]string{"question", "llm", "token-budget", "memory", "user-123"},
	},
	{
		"assistant",
		"LLMs have fixed context windows, but exact token accounting depends on the provider, model, rendered messages, tool definitions, and reserved output. HexxlaDB returns a deterministic MaxCells-bounded candidate set; Mosaic applies a provider-neutral byte ceiling, while the client that renders the final request remains responsible for exact tokenizer accounting.",
		[]string{"fact", "llm", "token-budget", "memory", "hexxladb", "user-123"},
	},
	{
		"user",
		"I'm building a multi-agent system. How should agents share a memory store?",
		[]string{"question", "llm", "multi-agent", "architecture", "user-123"},
	},
	{
		"assistant",
		"Each agent writes cells with its own source_id. Shared facts can be tagged with a common topic. Use ViewAt for read isolation — agents reading concurrently pin their own CommitSeq snapshot. Seams (MarkConflict, MarkSupersedes) handle disagreements between agents. SnapshotDiff lets a coordinator observe what changed between rounds.",
		[]string{"fact", "llm", "multi-agent", "architecture", "hexxladb", "user-123"},
	},
	{
		"user",
		"What's the best strategy for pruning old conversation history without losing important facts?",
		[]string{"question", "llm", "memory", "pruning", "strategy", "user-123"},
	},
	{
		"assistant",
		"Keep logical-retention policy in the application: select obsolete cells by tags, provenance, confidence, and age, then delete them explicitly. PruneCellVersions removes eligible old MVCC versions according to commit retention; it does not choose which current facts to forget. Run compaction afterward when physical disk reclamation is required.",
		[]string{"fact", "llm", "memory", "pruning", "strategy", "hexxladb", "user-123"},
	},
	{
		"user",
		"How do I detect when the user's preference has changed and update the memory accordingly?",
		[]string{"question", "llm", "memory", "preference-tracking", "user-123"},
	},
	{
		"assistant",
		"Pattern: on each new user turn, QueryCells with RequireTags=['preference'] and the relevant topic tag. If a highly-confident existing cell contradicts the new statement, call MarkSupersedes(newCoord, oldCoord, reason) so FilterSuperseded removes the stale cell from future context packs.",
		[]string{"fact", "llm", "memory", "preference-tracking", "hexxladb", "user-123"},
	},
	{
		"user",
		"I think most LLM chatbots have terrible memory — they forget context within the same session.",
		[]string{"opinion", "llm", "memory", "criticism", "user-123"},
	},
	{
		"assistant",
		"A widely-held view among practitioners. The root cause is stateless inference: each API call starts with a fresh window. Persistent memory stores like HexxlaDB address this by externalising context, making it queryable, and assembling only the most relevant subset per turn.",
		[]string{"fact", "llm", "memory", "criticism", "hexxladb", "user-123"},
	},
	{
		"user",
		"How should I evaluate the quality of a memory-augmented conversational system?",
		[]string{"question", "llm", "evaluation", "metrics", "user-123"},
	},
	{
		"assistant",
		"Key dimensions: (1) Recall — does the system retrieve relevant past facts when needed? Measure with recall@k against a golden set. (2) Precision — does context assembly avoid irrelevant noise? (3) Freshness — are stale/superseded facts excluded? (4) Latency — context assembly time at p99. (5) Token efficiency — relevant tokens / total context tokens.",
		[]string{"fact", "llm", "evaluation", "metrics", "retrieval", "user-123"},
	},
	{
		"user",
		"What role do confidence scores play in memory systems?",
		[]string{"question", "llm", "memory", "confidence", "user-123"},
	},
	{
		"assistant",
		"Confidence encodes epistemic certainty. HexxlaDB stores and returns confidence without silently changing retrieval meaning. Mosaic can rank user-confirmed or verified facts ahead of inferred material; any time-based confidence or retention policy should use explicit, auditable writes.",
		[]string{"fact", "llm", "memory", "confidence", "hexxladb", "user-123"},
	},

	// ──────────────────────────────────────────────────────────────
	// SESSION E: Security, compliance & operations (turns 72-83)
	// Exercises 'security', 'operations', 'compliance' tags.
	// ──────────────────────────────────────────────────────────────
	{
		"user",
		"What are the OWASP Top 10 vulnerabilities I should always check for in a web API?",
		[]string{"question", "security", "owasp", "api", "user-123"},
	},
	{
		"assistant",
		"OWASP Top 10 (2021): A01 Broken Access Control, A02 Cryptographic Failures, A03 Injection, A04 Insecure Design, A05 Security Misconfiguration, A06 Vulnerable Components, A07 Auth Failures, A08 Software Integrity Failures, A09 Security Logging Failures, A10 SSRF. Prioritise A01 and A03 for API surfaces.",
		[]string{"fact", "security", "owasp", "api", "compliance", "user-123"},
	},
	{
		"user",
		"How should I handle secrets in a Kubernetes deployment?",
		[]string{"question", "security", "kubernetes", "secrets", "operations", "user-123"},
	},
	{
		"assistant",
		"Never hardcode secrets. Use Kubernetes Secrets (base64 encoded, not encrypted by default) combined with envelope encryption via a KMS (AWS KMS, GCP KMS, HashiCorp Vault). Use external-secrets-operator to sync from Vault/AWS Secrets Manager. Mount as env vars or volume files; avoid CLI args (visible in ps output).",
		[]string{"fact", "security", "kubernetes", "secrets", "operations", "user-123"},
	},
	{
		"user",
		"What observability stack do you recommend for a Go microservice?",
		[]string{"question", "observability", "operations", "go", "user-123"},
	},
	{
		"assistant",
		"OpenTelemetry SDK for Go (traces + metrics) exporting to a collector. Prometheus + Grafana for metrics dashboards. Loki or Elasticsearch for logs. Jaeger or Tempo for traces. Use slog with otelslog handler to correlate trace IDs in logs. Instrument HTTP handlers with otelhttp middleware.",
		[]string{"fact", "observability", "operations", "go", "opentelemetry", "user-123"},
	},
	{
		"user",
		"How do I make a database backup strategy for a small team?",
		[]string{"question", "operations", "backup", "database", "user-123"},
	},
	{
		"assistant",
		"3-2-1 rule: 3 copies, 2 different media, 1 offsite. For HexxlaDB: snapshot the data file + WAL while the DB is closed (or use RotateEncryption with a backup key as a copy mechanism). Test restores monthly. Use object storage (S3, GCS) with versioning for offsite. Automate and alert on backup age.",
		[]string{"fact", "operations", "backup", "database", "hexxladb", "user-123"},
	},
	{
		"user",
		"What GDPR considerations apply to an LLM memory system that stores user conversations?",
		[]string{"question", "compliance", "gdpr", "llm", "privacy", "user-123"},
	},
	{
		"assistant",
		"Key considerations: (1) Lawful basis for storing conversation data (consent or legitimate interest). (2) Right to erasure — implement a delete path that removes all cells/seams attributed to a user (filter by source_id). (3) Data minimisation — prune old turns. (4) Transparency — users should know their conversations are persisted. (5) Jurisdiction — check where data at rest is stored.",
		[]string{"fact", "compliance", "gdpr", "llm", "privacy", "user-123"},
	},
	{
		"user",
		"What incident response steps should I have for a database compromise?",
		[]string{"question", "security", "incident-response", "operations", "user-123"},
	},
	{
		"assistant",
		"Runbook: (1) Isolate — revoke credentials, network-isolate the instance. (2) Preserve evidence — snapshot the data file before any recovery attempt. (3) Assess scope — audit logs, changefeed, MVCC diff to see what was read/written. (4) Rotate keys — RotateEncryption to re-key with new credentials. (5) Notify — per GDPR/breach-notification laws. (6) Post-mortem — document root cause.",
		[]string{"fact", "security", "incident-response", "operations", "hexxladb", "user-123"},
	},
}

// spiralCoord assigns deterministic axial coordinates across a rectangular window
// (same layout as github.com/hexxla/hexxladb/examples/conversational_memory).
func spiralCoord(index int) hexxladb.Coord {
	const cols = 11
	q := (index % cols) - (cols / 2)
	r := (index / cols) - 4
	return hexxladb.Coord{Q: q, R: r}
}
