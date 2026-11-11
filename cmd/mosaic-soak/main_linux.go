//go:build linux

// Command mosaic-soak exercises a disposable, bounded Linux qualification profile.
// Use scripts/soak.sh to enforce the required cgroup limits before running it.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/json/v2"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/hexxla/hexxladb"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hexxla/mosaic/internal/adapter/secondary/hexxlastore"
	"github.com/hexxla/mosaic/internal/core/domain"
	"github.com/hexxla/mosaic/internal/core/ports/secondary"
)

const (
	mib         = int64(1 << 20)
	gib         = 1024 * mib
	slots       = 128
	maxDisk     = 512 * mib
	reserveDisk = 10 * gib
	reserveRAM  = 4 * gib
)

type options struct {
	Duration    time.Duration
	Interval    time.Duration
	Maintenance time.Duration
	Dir         string
	Owner       string
	Mode        string
	URL         string
	Model       string
	Digest      string
	Dim         uint
	Verify      bool
	Preflight   bool
}
type summary struct {
	CallsByTool       map[string]int
	RunnerSHA256      string
	Targets           map[string]any
	Replays           int
	CancelChecks      int
	Owner             string
	Options           options
	Started           time.Time
	Ended             time.Time
	Status            string
	Failure           string
	Operations        int
	Denials           int
	Restarts          int
	Compactions       int
	Backups           int
	Embeddings        int
	IdleSeconds       int
	P95MS             int64
	P99MS             int64
	PeakMemoryBytes   int64
	PeakDiskBytes     int64
	MemoryGrowthBytes int64
	ProviderVersion   string
	Build             *debug.BuildInfo
}
type sample struct {
	At                time.Time
	Operations        int
	MemoryBytes       int64
	AvailableRAMBytes int64
	DiskBytes         int64
	FreeDiskBytes     uint64
}
type workload struct {
	cfg        options
	out        summary
	db         *hexxladb.DB
	live       *hexxlastore.LiveDB
	wire       *wire
	opts       *hexxladb.Options
	embedder   secondary.TextEmbedder
	identity   string
	path       string
	expected   map[int]string
	vectors    map[int][]float32
	latencies  []time.Duration
	log        *os.File
	generation int
	warmMemory int64
	receipt    domain.PutCellMutationResult
}

func main() {
	var c options
	flag.DurationVar(&c.Duration, "duration", 30*time.Minute, "explicit stage: 30m, 1h, or 24h")
	flag.DurationVar(&c.Interval, "interval", time.Second, "minimum pause between workload cycles")
	flag.DurationVar(&c.Maintenance, "maintenance-every", 10*time.Minute, "prune, backup, compact, reopen interval")
	flag.StringVar(&c.Dir, "dir", "", "new disposable output directory (must not exist)")
	flag.StringVar(&c.Owner, "owner", "void", "qualification owner: requesting repository maintainer")
	flag.StringVar(&c.Mode, "mode", "lexical", "lexical or ollama")
	flag.StringVar(&c.URL, "ollama-url", "http://127.0.0.1:11435", "dedicated resource-limited Ollama")
	flag.StringVar(&c.Model, "model", "all-minilm:latest", "Ollama embedding model")
	flag.StringVar(&c.Digest, "digest", "", "required provider SHA256 digest for live mode")
	flag.UintVar(&c.Dim, "dimension", 384, "expected live embedding dimension")
	flag.BoolVar(&c.Verify, "verify", false, "short runner verification, never qualifies a soak")
	flag.BoolVar(&c.Preflight, "preflight", false, "check resources/provider only, then exit")
	flag.Parse()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := execute(ctx, c)
	stop()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func validate(c options) error {
	if c.Dir == "" || strings.TrimSpace(c.Owner) == "" {
		return fmt.Errorf("dir and qualification owner are required")
	}
	if c.Mode != "lexical" && c.Mode != "ollama" {
		return fmt.Errorf("mode must be lexical or ollama")
	}
	if c.Verify {
		if c.Duration < 5*time.Second || c.Duration > 2*time.Minute {
			return fmt.Errorf("verification duration must be 5s–2m")
		}
	} else if c.Duration != 30*time.Minute && c.Duration != time.Hour && c.Duration != 24*time.Hour {
		return fmt.Errorf("select one stage: 30m, 1h, or 24h")
	}
	if c.Interval < 100*time.Millisecond || c.Interval > time.Second || c.Maintenance < time.Second || c.Maintenance > 10*time.Minute {
		return fmt.Errorf("interval must be 100ms–1s; maintenance 1s–10m")
	}
	if !c.Verify && (c.Interval != time.Second || c.Maintenance != 10*time.Minute) {
		return fmt.Errorf("qualification stages use frozen 1s cycles and 10m maintenance")
	}
	if c.Mode == "ollama" {
		if c.Dim == 0 || c.Dim > 1024 || len(c.Digest) != 64 {
			return fmt.Errorf("live mode requires dimension 1–1024 and a 64-character SHA256 digest")
		}
		for _, b := range c.Digest {
			if !strings.ContainsRune("0123456789abcdef", b) {
				return fmt.Errorf("digest must be lowercase SHA256 hex")
			}
		}
		u, err := url.Parse(c.URL)
		if err != nil || u.Scheme != "http" || u.Host != "127.0.0.1:11435" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
			return fmt.Errorf("live provider must be the dedicated http://127.0.0.1:11435")
		}
	}
	return nil
}
func execute(ctx context.Context, c options) (err error) {
	if err := validate(c); err != nil {
		return err
	}
	if err := checkFileLimit(); err != nil {
		return err
	}
	group, err := ownGroup()
	if err != nil {
		return err
	}
	if err := checkGroup(group, gib); err != nil {
		return err
	}
	var providerGroup string
	if c.Mode == "ollama" {
		providerGroup, err = ollamaGroup(ctx)
		if err != nil {
			return err
		}
		if err := checkGroup(providerGroup, 2*gib); err != nil {
			return err
		}
	}
	parent := filepath.Dir(c.Dir)
	if _, err = resourceSamplePath(parent, "", group, providerGroup); err != nil {
		return err
	}
	if c.Preflight {
		_, version, e := provider(ctx, c)
		if e == nil {
			fmt.Printf("preflight pass; mode=%s provider=%s owner=%s\n", c.Mode, version, c.Owner)
		}
		return e
	}
	if err = os.Mkdir(c.Dir, 0o700); err != nil {
		return fmt.Errorf("require new disposable directory: %w", err)
	}
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	w := &workload{cfg: c, expected: map[int]string{}, vectors: map[int][]float32{}}
	w.out = summary{Owner: c.Owner, Options: c, Started: time.Now().UTC(), Status: "running"}
	w.out.Build, _ = debug.ReadBuildInfo()
	executable, e := os.Open("/proc/self/exe")
	if e != nil {
		return e
	}
	hash := sha256.New()
	_, e = io.Copy(hash, executable)
	e = errors.Join(e, executable.Close())
	if e != nil {
		return e
	}
	w.out.RunnerSHA256 = fmt.Sprintf("%x", hash.Sum(nil))
	w.out.Targets = map[string]any{"cells": slots + 1, "disk_stop_bytes": maxDisk, "disk_reserve_bytes": reserveDisk, "ram_reserve_bytes": reserveRAM, "runner_memory_max_bytes": gib, "provider_memory_max_bytes": 2 * gib, "warm_memory_growth_max_bytes": 256 * mib, "p95_ms": 5000, "p99_ms": 30000, "maintenance_max_seconds": 30}
	defer func() {
		if v := recover(); v != nil {
			err = fmt.Errorf("qualification failed: %v", v)
			if cause := context.Cause(ctx); cause != nil {
				err = errors.Join(err, cause)
			}
		}
		// Close admitted MCP calls before closing storage, including on failure.
		if w.wire != nil {
			err = errors.Join(err, w.wire.session.Close())
			for s := range w.wire.server.Sessions() {
				err = errors.Join(err, s.Close())
			}
			w.wire.http.Close()
		}
		if w.live != nil {
			err = errors.Join(err, w.live.Close())
		}
		if w.log != nil {
			err = errors.Join(err, w.log.Close())
		}
		w.out.Ended = time.Now().UTC()
		w.out.Status = "pass"
		if c.Verify {
			w.out.Status = "verification-pass"
		}
		if err != nil {
			w.out.Status = "fail"
			w.out.Failure = err.Error()
		}
		err = errors.Join(err, writeJSON(filepath.Join(c.Dir, "summary.json"), w.out))
	}()
	watchCtx, stopWatch := context.WithCancel(ctx)
	watchDone := make(chan struct{})
	go func() {
		defer close(watchDone)
		t := time.NewTicker(time.Second)
		defer t.Stop()
		for {
			select {
			case <-watchCtx.Done():
				return
			case <-t.C:
				if _, e := resourceSample(c.Dir, group, providerGroup); e != nil {
					cancel(e)
					return
				}
			}
		}
	}()
	defer func() { stopWatch(); <-watchDone }()
	must(writeJSON(filepath.Join(c.Dir, "manifest.json"), w.out))
	w.log, err = os.OpenFile(filepath.Join(c.Dir, "samples.jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	must(err)
	w.embedder, w.out.ProviderVersion, err = provider(ctx, c)
	must(err)
	dim := uint16(0)
	if c.Mode == "ollama" {
		dim = uint16(c.Dim) //nolint:gosec // validate bounds the live dimension to 1–1024 before conversion.
		w.identity = c.Model + "@sha256:" + c.Digest
	}
	w.opts = &hexxladb.Options{ChangelogEnabled: true, EmbeddingDimension: dim, MaxValueBytes: 16384}
	must(os.Mkdir(filepath.Join(c.Dir, "a"), 0o700))
	w.path = filepath.Join(c.Dir, "a", "live.db")
	w.db, err = hexxladb.Open(w.path, w.opts)
	must(err)
	if dim > 0 {
		must(hexxlastore.EnsureEmbeddingModel(w.db, w.identity))
	}
	// Seed over MCP so live mode computes actual embeddings through the application.
	w.connect(ctx)
	for q := range slots {
		w.put(q, 0)
	}
	w.retry()
	w.oracle(ctx, w.db)
	start := time.Now()
	end := start.Add(c.Duration)
	nextMaintenance := start.Add(c.Maintenance)
	nextSample := start
	w.out.Operations = 0
	w.out.CallsByTool = map[string]int{}
	w.latencies = nil
	observe := func() {
		if time.Now().After(nextSample) {
			s, e := resourceSample(c.Dir, group, providerGroup)
			must(e)
			s.Operations = w.out.Operations
			w.out.PeakDiskBytes = max(w.out.PeakDiskBytes, s.DiskBytes)
			w.out.PeakMemoryBytes = max(w.out.PeakMemoryBytes, s.MemoryBytes)
			if time.Since(start) >= 5*time.Minute && w.warmMemory == 0 {
				w.warmMemory = s.MemoryBytes
			}
			if w.warmMemory != 0 {
				w.out.MemoryGrowthBytes = max(w.out.MemoryGrowthBytes, s.MemoryBytes-w.warmMemory)
				require(w.out.MemoryGrowthBytes <= 256*mib, "cgroup memory growth exceeded 256 MiB after warmup")
			}
			b, e := json.Marshal(s)
			must(e)
			_, e = w.log.Write(append(b, '\n'))
			must(e)
			nextSample = time.Now().Add(10 * time.Second)
		}
	}
	idleAt := start.Add(c.Duration / 3)
	idle := 10 * time.Second
	if c.Duration >= time.Hour {
		idle = 31 * time.Minute
	}
	if c.Verify {
		idle = 0
	}
	idleDone := false
	step := 0
	for time.Now().Before(end) {
		must(context.Cause(ctx))
		if _, err = resourceSample(c.Dir, group, providerGroup); err != nil {
			return err
		}
		if !idleDone && time.Now().After(idleAt) {
			idleDone = true
			w.out.IdleSeconds = int(idle.Seconds())
			idleEnd := time.Now().Add(idle)
			for time.Now().Before(idleEnd) {
				observe()
				must(pause(ctx, min(time.Second, time.Until(idleEnd))))
			}
			if idle >= 31*time.Minute {
				count := 0
				for range w.wire.server.Sessions() {
					count++
				}
				require(count == 0, "idle HTTP session failed to expire")
				_ = w.wire.session.Close()
				w.wire.http.Close()
				w.wire = nil
				w.connect(ctx)
			}
		}
		if time.Now().After(nextMaintenance) {
			w.maintain(ctx)
			nextMaintenance = time.Now().Add(c.Maintenance)
		}
		q := step % slots
		switch {
		case step%5 == 0:
			w.put(q, step+1)
		case step%2 == 0:
			var out domain.CellLookupResponse
			w.decode("get_cell", map[string]any{"q": q, "r": 0}, &out)
			require(out.Found && out.Cell != nil && out.Cell.RawContent == w.expected[q], "MCP lookup oracle mismatch")
		default:
			mode := "lexical"
			if c.Mode == "ollama" && step%7 == 0 {
				mode = "semantic"
			}
			var out domain.CellHitsResponse
			w.decode("search_cells", map[string]any{"mode": mode, "query": "qualification", "max_results": 8, "max_scan_rows": 4096}, &out)
			require(len(out.Hits) > 0, "search returned no fixture cells")
		}
		step++
		observe()
		must(pause(ctx, c.Interval))
	}
	w.maintain(ctx)
	w.oracle(ctx, w.db)
	sorted := slices.Sorted(slices.Values(w.latencies))
	w.out.P95MS = percentile(sorted, 95).Milliseconds()
	w.out.P99MS = percentile(sorted, 99).Milliseconds()
	require(w.out.P95MS <= 5000 && w.out.P99MS <= 30000, "MCP latency target exceeded (p95 5s, p99 30s)")
	minimum := int(c.Duration.Seconds() / 4)
	if c.Duration == 24*time.Hour {
		minimum = max(minimum, 10000)
	}
	if c.Verify {
		minimum = 1
	}
	require(w.out.Operations >= minimum, fmt.Sprintf("completed operations %d below %d", w.out.Operations, minimum))
	must(context.Cause(ctx))
	return nil
}
func pause(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
func percentile(values []time.Duration, p int) time.Duration {
	if len(values) == 0 {
		return 0
	}
	return values[(len(values)-1)*p/100]
}
func (w *workload) connect(ctx context.Context) {
	w.live = hexxlastore.NewLiveDB(w.db)
	w.wire = openWire(ctx, w.live, w.path, w.generation%2 == 1, w.opts.EmbeddingDimension, w.identity, w.embedder)
	w.wire.denied("put_cell", map[string]any{"q": 0, "r": 0, "raw_content": "denied", "source_id": "qualification", "confidence": 1})
	w.out.Denials++
}
func (w *workload) decode(name string, args, out any) {
	start := time.Now()
	r := w.wire.call(name, args)
	elapsed := time.Since(start)
	limit := 5 * time.Second
	if w.cfg.Mode == "ollama" {
		limit = 30 * time.Second
	}
	require(elapsed <= limit, "individual MCP call exceeded latency ceiling")
	if len(w.latencies) < 1000000 {
		w.latencies = append(w.latencies, elapsed)
	} else {
		panic("latency sample capacity exhausted")
	}
	w.out.Operations++
	if w.out.CallsByTool == nil {
		w.out.CallsByTool = map[string]int{}
	}
	w.out.CallsByTool[name]++
	if out != nil {
		b, err := json.Marshal(r.StructuredContent)
		must(err)
		must(json.Unmarshal(b, out))
	}
}
func (w *workload) put(q, version int) {
	w.decode("list_tags", nil, nil)
	body := fmt.Sprintf("qualification slot=%03d version=%08d %s", q, version, strings.Repeat("bounded-memory ", 32))
	w.decode("put_cell", map[string]any{"q": q, "r": 0, "raw_content": body, "source_id": "qualification", "confidence": 1, "allow_overwrite": version > 0, "embed": w.cfg.Mode == "ollama"}, nil)
	w.expected[q] = body
	if w.cfg.Mode == "ollama" {
		w.out.Embeddings++
		must(w.db.View(func(tx *hexxladb.Tx) error {
			k, err := hexxladb.Pack(hexxladb.Coord{Q: q})
			if err != nil {
				return err
			}
			v, ok, err := tx.GetEmbedding(k)
			if err != nil {
				return err
			}
			require(ok && len(v) == int(w.cfg.Dim), "embedding missing")
			w.vectors[q] = slices.Clone(v)
			return nil
		}))
	}
}
func (w *workload) oracle(ctx context.Context, db *hexxladb.DB) {
	must(db.View(func(tx *hexxladb.Tx) error {
		for q, body := range w.expected {
			k, err := hexxladb.Pack(hexxladb.Coord{Q: q})
			if err != nil {
				return err
			}
			rec, ok, err := tx.GetCell(k)
			if err != nil {
				return err
			}
			require(ok && rec.RawContent == body, "database cell oracle mismatch")
			if w.cfg.Mode == "ollama" {
				v, ok, err := tx.GetEmbedding(k)
				if err != nil {
					return err
				}
				require(ok && slices.Equal(v, w.vectors[q]), "database vector oracle mismatch")
			}
		}
		return nil
	}))
	report, err := db.HealthCheck(ctx, hexxladb.DefaultHealthCheckConfig())
	must(err)
	require(report.CellCount == len(w.expected) && report.TagIndexErrors == 0 && report.SourceIndexErrors == 0 && len(report.OrphanedSeams) == 0, "database health mismatch")
}
func (w *workload) maintain(ctx context.Context) {
	start := time.Now()
	usage, e := directoryBytes(w.cfg.Dir)
	must(e)
	require(usage < maxDisk/3, "insufficient run-budget headroom for backup and compaction")
	w.oracle(ctx, w.db)
	feed, err := w.db.ReadChangelogSince(0, 100000)
	must(err)
	require(len(feed) > 0, "changefeed unexpectedly empty")
	cursor, found, err := w.db.GetChangelogConsumerCursor("soak")
	must(err)
	if !found {
		cursor = 0
	}
	latest := feed[len(feed)-1].Seq
	// Drain bounded pages rather than assuming the first page is the tail.
	for len(feed) == 100000 {
		feed, err = w.db.ReadChangelogSince(latest, 100000)
		must(err)
		if len(feed) > 0 {
			latest = feed[len(feed)-1].Seq
		}
	}
	must(w.db.AdvanceChangelogConsumer("soak", cursor, latest))
	var seq uint64
	must(w.db.View(func(tx *hexxladb.Tx) error { seq = tx.ReadSequence(); return nil }))
	for {
		n, e := w.db.PruneCellVersions(seq, 512)
		must(e)
		if n == 0 {
			break
		}
		must(context.Cause(ctx))
	}
	next := "b"
	if w.generation%2 == 1 {
		next = "a"
	}
	target := filepath.Join(w.cfg.Dir, next)
	must(os.RemoveAll(target))
	must(os.Mkdir(target, 0o700)) // Only the inactive directory in this newly created run is removed.
	backup := filepath.Join(target, "backup.db")
	must(w.db.BackupTo(ctx, backup))
	b, err := hexxladb.OpenExisting(backup, w.opts)
	must(err)
	w.oracle(ctx, b)
	backupCursor, exists, e := b.GetChangelogConsumerCursor("soak")
	must(e)
	require(exists && backupCursor == latest, "backup consumer mismatch")
	must(b.Close())
	w.out.Backups++
	w.wire.close()
	w.wire = nil
	must(w.live.Close())
	w.live = nil
	candidate := filepath.Join(target, "live.db")
	must(hexxladb.CompactToWithOptions(ctx, w.path, candidate, w.opts, &hexxladb.CompactOptions{BatchSize: 128, VerifyDestination: true}))
	w.copyFile(w.path+"-changelog", candidate+"-changelog")
	w.db, err = hexxladb.OpenExisting(candidate, w.opts)
	must(err)
	w.oracle(ctx, w.db)
	got, ok, err := w.db.GetChangelogConsumerCursor("soak")
	must(err)
	require(ok && got == latest, "consumer cursor changed through compaction")
	w.path = candidate
	w.generation++
	w.out.Compactions++
	w.out.Restarts++
	w.connect(ctx)
	w.retry()
	w.cancelProbe(ctx)
	require(time.Since(start) <= 30*time.Second, "maintenance exceeded 30s window")
}
func (w *workload) copyFile(from, to string) {
	root, err := os.OpenRoot(w.cfg.Dir)
	must(err)
	defer func() { must(root.Close()) }()
	src, err := filepath.Rel(w.cfg.Dir, from)
	must(err)
	dst, err := filepath.Rel(w.cfg.Dir, to)
	must(err)
	f, err := root.Open(src)
	must(err)
	defer func() { must(f.Close()) }()
	g, err := root.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	must(err)
	defer func() { must(g.Close()) }()
	n, err := io.Copy(g, io.LimitReader(f, 128*mib+1))
	must(err)
	require(n <= 128*mib, "sidecar exceeds per-file copy budget")
	must(g.Sync())
}
func readSequence(db *hexxladb.DB) uint64 {
	var seq uint64
	must(db.View(func(tx *hexxladb.Tx) error { seq = tx.ReadSequence(); return nil }))
	return seq
}
func (w *workload) retry() {
	w.decode("list_tags", nil, nil)
	before := readSequence(w.db)
	args := map[string]any{"q": 1000, "r": 0, "raw_content": "qualification durable replay", "source_id": "qualification", "confidence": 1, "placement": "near_anchor", "max_radius": 1, "idempotency_key": "soak-fixed-receipt", "embed": w.cfg.Mode == "ollama"}
	var out domain.PutCellMutationResult
	w.decode("put_cell", args, &out)
	if w.receipt.OK {
		require(out == w.receipt && readSequence(w.db) == before, "receipt retry changed placement or committed again")
		w.out.Replays++
		return
	}
	w.receipt = out
	require(out.OK && out.Coord.R == 0, "unexpected near-anchor placement")
	q := out.Coord.Q
	w.expected[q] = "qualification durable replay"
	if w.cfg.Mode == "ollama" {
		must(w.db.View(func(tx *hexxladb.Tx) error {
			k, err := hexxladb.Pack(hexxladb.Coord{Q: q})
			if err != nil {
				return err
			}
			v, ok, err := tx.GetEmbedding(k)
			if err != nil {
				return err
			}
			require(ok, "receipt embedding missing")
			w.vectors[q] = slices.Clone(v)
			return nil
		}))
	}
}
func (w *workload) cancelProbe(ctx context.Context) {
	before := readSequence(w.db)
	c, cancel := context.WithCancel(ctx)
	cancel()
	_, err := w.wire.session.CallTool(c, &mcp.CallToolParams{Name: "mosaic_hexxla_put_cell", Arguments: map[string]any{"q": 0, "r": 0, "raw_content": "canceled", "source_id": "qualification", "confidence": 1, "allow_overwrite": true}})
	require(err != nil && readSequence(w.db) == before, "pre-dispatch cancellation mutated storage")
	w.out.CancelChecks++
}

func writeJSON(path string, v any) error {
	b, err := json.Marshal(v, json.WithMarshalers(json.MarshalFunc(func(d time.Duration) ([]byte, error) { return json.Marshal(d.String()) })))
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o600)
}
