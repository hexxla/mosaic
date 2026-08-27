package main

import (
	"net/url"
	"testing"
)

func TestObserverURLBoundary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{name: "ipv4 loopback", raw: "ws://127.0.0.1:8787/observability/stream"},
		{name: "ipv6 loopback", raw: "ws://[::1]:8787/observability/stream"},
		{name: "localhost", raw: "ws://localhost:8787/observability/stream"},
		{name: "tls unsupported", raw: "wss://127.0.0.1:8787/observability/stream", wantErr: true},
		{name: "public host", raw: "ws://example.com/observability/stream", wantErr: true},
		{name: "credential URL", raw: "ws://user:secret@127.0.0.1:8787/observability/stream", wantErr: true},
		{name: "missing path", raw: "ws://127.0.0.1:8787", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := validateObserverURL(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateObserverURL(%q): err=%v wantErr=%v", tt.raw, err, tt.wantErr)
			}
		})
	}
}

func TestObserverURLQueryBoundary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{name: "empty", raw: "ws://127.0.0.1/stream"},
		{name: "session", raw: "ws://127.0.0.1/stream?session_id=abc"},
		{name: "empty session", raw: "ws://127.0.0.1/stream?session_id=", wantErr: true},
		{name: "duplicate session", raw: "ws://127.0.0.1/stream?session_id=a&session_id=b", wantErr: true},
		{name: "token query", raw: "ws://127.0.0.1/stream?token=secret", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			u, err := url.Parse(tt.raw)
			if err != nil {
				t.Fatalf("parse URL: %v", err)
			}
			err = validateObserverQuery(u)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateObserverQuery(%q): err=%v wantErr=%v", tt.raw, err, tt.wantErr)
			}
		})
	}
}
