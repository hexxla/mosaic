//go:build linux

package main

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/hexxla/mosaic/internal/adapter/secondary/ollamaembed"
	"github.com/hexxla/mosaic/internal/core/ports/secondary"
	"github.com/hexxla/mosaic/internal/ollama"
)

const cgroupRoot = "/sys/fs/cgroup"

func numberFile(path string) (int64, error) {
	b, err := os.ReadFile(path) //nolint:gosec // Paths are kernel-discovered cgroup files or test fixtures.
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64)
}
func ownGroup() (string, error) {
	b, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		return "", err
	}
	for line := range strings.SplitSeq(string(b), "\n") {
		if path, ok := strings.CutPrefix(line, "0::/"); ok {
			return filepath.Join(cgroupRoot, path), nil
		}
	}
	return "", fmt.Errorf("cgroup v2 required; use scripts/soak.sh")
}
func checkGroup(path string, limit int64) error {
	memory, err := numberFile(filepath.Join(path, "memory.max"))
	if err != nil || memory <= 0 || memory > limit {
		return fmt.Errorf("cgroup memory.max must be bounded at %d bytes or less", limit)
	}
	swap, err := numberFile(filepath.Join(path, "memory.swap.max"))
	if err != nil || swap != 0 {
		return fmt.Errorf("cgroup memory.swap.max must be zero")
	}
	b, err := os.ReadFile(filepath.Join(path, "cpu.max")) //nolint:gosec // The group path comes from the kernel or the dedicated systemd unit.
	if err != nil {
		return err
	}
	parts := strings.Fields(string(b))
	if len(parts) != 2 {
		return fmt.Errorf("invalid cpu.max")
	}
	quota, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return fmt.Errorf("cgroup CPU quota required: %w", err)
	}
	period, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || period <= 0 || quota <= 0 || quota > period {
		return fmt.Errorf("cgroup CPU quota must be at most one core")
	}
	return nil
}
func ollamaGroup(ctx context.Context) (string, error) {
	b, err := exec.CommandContext(ctx, "systemctl", "--user", "show", "mosaic-soak-ollama.service", "--property=ControlGroup", "--value").Output()
	if err != nil {
		return "", fmt.Errorf("dedicated bounded Ollama service required: %w", err)
	}
	s := strings.TrimSpace(string(b))
	if !strings.HasPrefix(s, "/") || s == "/" {
		return "", fmt.Errorf("dedicated Ollama service is not running")
	}
	return filepath.Join(cgroupRoot, s), nil
}
func availableRAM() (int64, error) {
	b, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	for line := range strings.SplitSeq(string(b), "\n") {
		if strings.HasPrefix(line, "MemAvailable:") {
			p := strings.Fields(line)
			if len(p) != 3 {
				return 0, fmt.Errorf("invalid MemAvailable")
			}
			v, e := strconv.ParseInt(p[1], 10, 64)
			return v * 1024, e
		}
	}
	return 0, fmt.Errorf("MemAvailable unavailable")
}
func resourceSample(dir, group, providerGroup string) (sample, error) {
	return resourceSamplePath(dir, dir, group, providerGroup)
}
func resourceSamplePath(dir, walk, group, providerGroup string) (sample, error) {
	s := sample{At: time.Now().UTC()}
	var err error
	s.MemoryBytes, err = numberFile(filepath.Join(group, "memory.current"))
	if err != nil {
		return s, err
	}
	if providerGroup != "" {
		n, e := numberFile(filepath.Join(providerGroup, "memory.current"))
		if e != nil {
			return s, e
		}
		s.MemoryBytes += n
	}
	s.AvailableRAMBytes, err = availableRAM()
	if err != nil {
		return s, err
	}
	if s.AvailableRAMBytes < reserveRAM {
		return s, fmt.Errorf("host available RAM below 4 GiB reserve")
	}
	var fs syscall.Statfs_t
	if err := syscall.Statfs(dir, &fs); err != nil {
		return s, err
	}
	if fs.Bsize <= 0 || fs.Bavail > math.MaxUint64/uint64(fs.Bsize) {
		return s, fmt.Errorf("invalid filesystem capacity")
	}
	s.FreeDiskBytes = fs.Bavail * uint64(fs.Bsize)
	if s.FreeDiskBytes < uint64(reserveDisk) {
		return s, fmt.Errorf("filesystem available space below 10 GiB reserve")
	}
	if walk == "" {
		return s, nil
	}
	s.DiskBytes, err = directoryBytes(walk)
	if err == nil && s.DiskBytes >= maxDisk {
		err = fmt.Errorf("run directory reached 512 MiB stop budget")
	}

	return s, err
}
func provider(ctx context.Context, c options) (secondary.TextEmbedder, string, error) {
	if c.Mode != "ollama" {
		return nil, "not-used", nil
	}
	u, err := url.Parse(c.URL)
	if err != nil {
		return nil, "", err
	}
	client := ollama.NewClient(u, c.Model)
	client.ExpectedDigest = c.Digest
	client.HTTP = &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return fmt.Errorf("provider redirects refused") }}
	get := func(endpoint string, out any) error {
		req, e := http.NewRequestWithContext(ctx, http.MethodGet, u.JoinPath("api", endpoint).String(), http.NoBody)
		if e != nil {
			return e
		}
		r, e := client.HTTP.Do(req)
		if e != nil {
			return e
		}
		defer func() { _ = r.Body.Close() }()
		if r.StatusCode != http.StatusOK {
			return fmt.Errorf("ollama %s: %s", endpoint, r.Status)
		}
		b, e := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if e != nil {
			return e
		}
		return json.Unmarshal(b, out)
	}
	var version struct {
		Version string `json:"version"`
	}
	if err := get("version", &version); err != nil {
		return nil, "", err
	}
	if version.Version == "" {
		return nil, "", fmt.Errorf("provider version missing")
	}
	if _, err = client.Embed(ctx, "qualification live embedding readiness probe", int(c.Dim)); err != nil {
		return nil, "", err
	}
	var ps struct {
		Models []struct {
			Name     string `json:"name"`
			SizeVRAM int64  `json:"size_vram"`
		} `json:"models"`
	}
	if err := get("ps", &ps); err != nil {
		return nil, "", err
	}
	if len(ps.Models) != 1 || ps.Models[0].Name != c.Model || ps.Models[0].SizeVRAM != 0 {
		return nil, "", fmt.Errorf("CPU-only profile requires exactly the selected model resident with zero GPU bytes")
	}
	return ollamaembed.NewTextEmbedder(client), version.Version, nil
}

func directoryBytes(dir string) (int64, error) {
	var total int64
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, e error) error {
		if e != nil {
			if os.IsNotExist(e) && path != dir {
				return nil
			}
			return e
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("unexpected symlink in disposable run directory")
		}
		if !d.IsDir() {
			info, e := d.Info()
			if os.IsNotExist(e) {
				return nil
			}
			if e != nil {
				return e
			}
			total += info.Size()
		}
		return nil
	})
	return total, err
}

func checkFileLimit() error {
	var r syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_FSIZE, &r); err != nil {
		return err
	}
	if r.Cur > uint64(128*mib) {
		return fmt.Errorf("128 MiB per-file OS limit required; use scripts/soak.sh")
	}
	return nil
}
