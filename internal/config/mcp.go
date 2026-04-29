package config

import (
	"fmt"
	"net"
	"os"
	"strings"
)

const (
	// DefaultMCPAddr is the implicit listen address when MOSAIC_MCP_ADDR is unset (loopback-only).
	DefaultMCPAddr = "127.0.0.1:8787"
	// DefaultMCPPath is the implicit MCP JSON-RPC endpoint path when MOSAIC_MCP_PATH is unset.
	DefaultMCPPath = "/mcp"
	envMCPAddr     = "MOSAIC_MCP_ADDR"
	envMCPPath     = "MOSAIC_MCP_PATH"
)

// MCP holds configuration for the local MCP Streamable HTTP server.
type MCP struct {
	// Addr is a TCP listen address (host:port). Only loopback interfaces are allowed.
	Addr string

	// Path is the HTTP path at which MCP JSON-RPC is served (single endpoint).
	Path string
}

// LoadMCPFromEnv reads MCP settings from MOSAIC_MCP_ADDR and MOSAIC_MCP_PATH.
//
// MOSAIC_MCP_ADDR defaults to 127.0.0.1:8787. MOSAIC_MCP_PATH defaults to /mcp.
func LoadMCPFromEnv() (MCP, error) {
	addr := strings.TrimSpace(os.Getenv(envMCPAddr))
	if addr == "" {
		addr = DefaultMCPAddr
	}
	if err := validateLoopbackListenAddr(addr); err != nil {
		return MCP{}, err
	}
	path := strings.TrimSpace(os.Getenv(envMCPPath))
	if path == "" {
		path = DefaultMCPPath
	}
	if !strings.HasPrefix(path, "/") {
		return MCP{}, fmt.Errorf("%s must start with '/': got %q", envMCPPath, path)
	}
	return MCP{Addr: addr, Path: path}, nil
}

// validateLoopbackListenAddr rejects addresses that bind to all interfaces or public interfaces.
func validateLoopbackListenAddr(addr string) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid MCP listen address %q: %w", addr, err)
	}
	if port == "" {
		return fmt.Errorf("invalid MCP listen address %q: missing port", addr)
	}
	if host == "" {
		return fmt.Errorf("MCP listen address must include loopback host (e.g. 127.0.0.1:8787): got %q", addr)
	}

	if host != "localhost" {
		ip := net.ParseIP(host)
		if ip == nil {
			return fmt.Errorf("MCP listen host must be 127.0.0.1, ::1, or localhost: got %q", host)
		}
		if !ip.IsLoopback() {
			return fmt.Errorf("MCP must listen on loopback only: got host %q", host)
		}
	}

	// Extra guard against binding to unspecified / public addresses.
	tcpAddr := net.JoinHostPort(host, port)
	if host == "localhost" {
		tcpAddr = net.JoinHostPort("127.0.0.1", port)
	}
	if la, err := net.ResolveTCPAddr("tcp", tcpAddr); err == nil {
		if la.IP.IsUnspecified() {
			return fmt.Errorf("MCP must not bind to unspecified address: %q", addr)
		}
		if !la.IP.IsLoopback() {
			return fmt.Errorf("MCP must listen on loopback only: got %q", addr)
		}
	}

	return nil
}
