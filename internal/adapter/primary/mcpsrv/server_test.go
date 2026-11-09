package mcpsrv

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestStreamableHTTPOriginAndHostProtection(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(StreamableHTTPHandler(NewServer("security-test", "test", nil), nil))
	defer server.Close()
	const initialize = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"test"}}}`
	for _, tc := range []struct {
		name, method, origin, contentType, host string
		want                                    int
	}{
		{"native", "POST", "", "application/json", "", http.StatusOK},
		{"same origin", "POST", server.URL, "application/json", "", http.StatusOK},
		{"foreign origin post", "POST", "https://untrusted.invalid", "application/json", "", http.StatusForbidden},
		{"foreign origin delete", "DELETE", "https://untrusted.invalid", "", "", http.StatusForbidden},
		{"null origin", "POST", "null", "application/json", "", http.StatusForbidden},
		{"browser simple type", "POST", "", "text/plain", "", http.StatusUnsupportedMediaType},
		{"foreign host", "POST", "", "application/json", "untrusted.invalid", http.StatusForbidden},
		{"optional SSE", "GET", "", "", "", http.StatusMethodNotAllowed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequestWithContext(t.Context(), tc.method, server.URL, strings.NewReader(initialize))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", tc.contentType)
			req.Header.Set("Accept", "application/json, text/event-stream")
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			if tc.host != "" {
				req.Host = tc.host
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = resp.Body.Close() }()
			_, _ = io.Copy(io.Discard, resp.Body)
			if resp.StatusCode != tc.want {
				t.Fatalf("status=%d want=%d", resp.StatusCode, tc.want)
			}
			if tc.want == http.StatusOK && !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
				t.Fatalf("POST SSE response lost: %v", resp.Header)
			}
			if tc.want == http.StatusMethodNotAllowed && resp.Header.Get("Allow") != "POST, DELETE" {
				t.Fatalf("Allow=%q", resp.Header.Get("Allow"))
			}
			if id := resp.Header.Get("Mcp-Session-Id"); id != "" {
				closeReq, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, server.URL, nil)
				if err != nil {
					t.Fatal(err)
				}
				closeReq.Header.Set("Mcp-Session-Id", id)
				closed, err := http.DefaultClient.Do(closeReq)
				if err != nil {
					t.Fatal(err)
				}
				_ = closed.Body.Close()
			}
		})
	}
}

func TestDefaultClientCancellationReachesTool(t *testing.T) {
	t.Parallel()
	srv := NewServer("cancellation-test", "test", nil)
	entered, canceled := make(chan struct{}), make(chan struct{})
	mcp.AddTool(srv, &mcp.Tool{Name: "wait"}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, ratchetTestOutput, error) {
		close(entered)
		<-ctx.Done()
		close(canceled)
		return nil, ratchetTestOutput{}, ctx.Err()
	})
	server := httptest.NewServer(StreamableHTTPHandler(srv, nil))
	defer server.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "test"}, nil)
	session, err := client.Connect(t.Context(), &mcp.StreamableClientTransport{Endpoint: server.URL}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = session.Close() }()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "wait"}); done <- err }()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("tool not called")
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("call error=%v", err)
	}
	select {
	case <-canceled:
	case <-time.After(5 * time.Second):
		t.Fatal("tool context not canceled")
	}
}
