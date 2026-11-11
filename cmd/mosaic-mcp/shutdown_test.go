package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hexxla/mosaic/internal/adapter/primary/mcpsrv"
)

func TestShutdownDrainsDefaultClientMutationAndClosesSessions(t *testing.T) {
	t.Parallel()
	srv := mcpsrv.NewServer("shutdown-test", "test", nil)
	calls := &activeToolCalls{drained: make(chan struct{})}
	srv.AddReceivingMiddleware(calls.middleware)
	entered, release := make(chan struct{}), make(chan struct{})
	releaseHandler := sync.OnceFunc(func() { close(release) })
	defer releaseHandler()
	mcp.AddTool(srv, &mcp.Tool{Name: "mutate"}, func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, struct {
		OK bool `json:"ok"`
	}, error) {
		close(entered)
		<-release
		return nil, struct {
			OK bool `json:"ok"`
		}{OK: true}, nil
	})
	server := httptest.NewServer(mcpsrv.StreamableHTTPHandler(srv, nil))
	defer server.Close()
	httpTransport := http.DefaultTransport.(*http.Transport).Clone()
	defer httpTransport.CloseIdleConnections()
	httpClient := &http.Client{Transport: httpTransport}
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "test"}, nil)
	session, err := client.Connect(t.Context(), &mcp.StreamableClientTransport{Endpoint: server.URL, HTTPClient: httpClient}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { releaseHandler(); _ = session.Close() }()
	callDone := make(chan error, 1)
	go func() {
		res, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "mutate"})
		if err == nil && res.IsError {
			t.Error("mutation failed")
		}
		callDone <- err
	}()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("mutation not entered")
	}
	// Close speculative idle connections without disturbing the active call.
	httpTransport.CloseIdleConnections()
	stopping := make(chan struct{})
	server.Config.RegisterOnShutdown(func() { close(stopping) })
	shutdownDone := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()
		shutdownDone <- shutdownMCP(ctx, server.Config, srv, calls)
	}()
	<-stopping
	select {
	case err := <-shutdownDone:
		t.Fatalf("shutdown returned before mutation completed: %v", err)
	default:
	}
	releaseHandler()
	if err := <-callDone; err != nil {
		t.Fatalf("mutation did not drain: %v", err)
	}
	if err := <-shutdownDone; err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	for range srv.Sessions() {
		t.Fatal("idle MCP session retained after shutdown")
	}
}

func TestShutdownDrainsDisconnectedSDKCall(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	srv := mcpsrv.NewServer("detached-shutdown", "test", nil)
	calls := &activeToolCalls{drained: make(chan struct{})}
	srv.AddReceivingMiddleware(calls.middleware)
	entered, release, canceled := make(chan struct{}), make(chan struct{}), make(chan struct{})
	releaseHandler := sync.OnceFunc(func() { close(release) })
	defer releaseHandler()
	mcp.AddTool(srv, &mcp.Tool{Name: "mutate"}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, struct{}, error) {
		close(entered)
		select {
		case <-release:
			return nil, struct{}{}, nil
		case <-ctx.Done():
			close(canceled)
			return nil, struct{}{}, ctx.Err()
		}
	})
	server := httptest.NewServer(mcpsrv.StreamableHTTPHandler(srv, nil))
	defer server.Close()
	httpTransport := http.DefaultTransport.(*http.Transport).Clone()
	defer httpTransport.CloseIdleConnections()
	httpClient := &http.Client{Transport: httpTransport}
	client, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "test"}, nil).Connect(ctx, &mcp.StreamableClientTransport{Endpoint: server.URL, HTTPClient: httpClient}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { releaseHandler(); _ = client.Close() }()
	var sessionID string
	for session := range srv.Sessions() {
		sessionID = session.ID()
	}
	callCtx, disconnect := context.WithCancel(ctx)
	defer disconnect()
	req, err := http.NewRequestWithContext(callCtx, http.MethodPost, server.URL, strings.NewReader(`{"jsonrpc":"2.0","id":100,"method":"tools/call","params":{"name":"mutate","arguments":{}}}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Mcp-Session-Id", sessionID)
	callDone := make(chan error, 1)
	go func() {
		response, err := httpClient.Do(req)
		if err == nil {
			_, err = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
		}
		callDone <- err
	}()
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal("handler did not begin")
	}
	disconnect()
	select {
	case err := <-callDone:
		if err == nil {
			t.Fatal("disconnected POST unexpectedly succeeded")
		}
	case <-ctx.Done():
		t.Fatal("POST did not disconnect")
	}
	// Close speculative idle connections without disturbing the active call.
	httpTransport.CloseIdleConnections()
	stopping := make(chan struct{})
	server.Config.RegisterOnShutdown(func() { close(stopping) })
	shutdownDone := make(chan error, 1)
	go func() { shutdownDone <- shutdownMCP(ctx, server.Config, srv, calls) }()
	select {
	case <-stopping:
	case <-ctx.Done():
		t.Fatal("shutdown did not begin")
	}
	select {
	case err := <-shutdownDone:
		t.Fatalf("shutdown returned while detached SDK handler was active: %v", err)
	case <-canceled:
		t.Fatal("shutdown canceled the detached SDK handler")
	case <-time.After(20 * time.Millisecond):
	case <-ctx.Done():
		t.Fatal("shutdown stalled")
	}
	releaseHandler()
	select {
	case err := <-shutdownDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("shutdown did not complete after draining")
	}
	for range srv.Sessions() {
		t.Fatal("MCP session retained after draining")
	}
}
