// Command mosaic-observe streams Mosaic's authenticated live Ratchet events as NDJSON.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gorilla/websocket"

	"github.com/sploitzberg/mosaic/internal/config"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if err := run(log); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	streamURLFlag := flag.String("url", "ws://127.0.0.1:8787"+config.DefaultRatchetObservabilityPath, "loopback Ratchet observability WebSocket URL")
	tokenFileFlag := flag.String("token-file", "", "private bearer-token file (required)")
	sessionIDFlag := flag.String("session-id", "", "optional MCP session ID filter")
	flag.Parse()

	if strings.TrimSpace(*tokenFileFlag) == "" {
		return errors.New("-token-file is required")
	}
	streamURL, err := validateObserverURL(*streamURLFlag)
	if err != nil {
		return fmt.Errorf("observer URL: %w", err)
	}
	if sessionID := strings.TrimSpace(*sessionIDFlag); sessionID != "" {
		query := streamURL.Query()
		if query.Has("session_id") {
			return errors.New("session_id must be supplied either in -url or -session-id, not both")
		}
		query.Set("session_id", sessionID)
		streamURL.RawQuery = query.Encode()
	}
	if err := validateObserverQuery(streamURL); err != nil {
		return fmt.Errorf("observer URL query: %w", err)
	}

	observerConfig, err := config.LoadRatchetObservability(*tokenFileFlag, streamURL.Path)
	if err != nil {
		return fmt.Errorf("observer credentials: %w", err)
	}
	header := http.Header{"Authorization": {"Bearer " + string(observerConfig.BearerToken)}}
	conn, _, err := websocket.DefaultDialer.Dial(streamURL.String(), header)
	if err != nil {
		return fmt.Errorf("connect observer: %w", err)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			log.Warn("close observer", "err", closeErr)
		}
	}()

	loggedURL := *streamURL
	loggedURL.RawQuery = ""
	log.Info("connected to Ratchet live observability", "url", loggedURL.Redacted())
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("read observer stream: %w", err)
		}
		if _, err := fmt.Fprintln(os.Stdout, string(message)); err != nil {
			return fmt.Errorf("write event: %w", err)
		}
	}
}

func validateObserverURL(raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, fmt.Errorf("parse observer URL: %w", err)
	}
	if u.Scheme != "ws" || u.User != nil || u.Host == "" || u.Path == "" || u.Fragment != "" {
		return nil, fmt.Errorf("observer URL must be an absolute ws:// loopback URL with a path: %q", raw)
	}
	hostname := u.Hostname()
	if hostname != "localhost" {
		ip := net.ParseIP(hostname)
		if ip == nil || !ip.IsLoopback() {
			return nil, fmt.Errorf("observer URL must use a loopback host: %q", hostname)
		}
	}
	return u, nil
}

func validateObserverQuery(u *url.URL) error {
	query := u.Query()
	for key, values := range query {
		if key != "session_id" || len(values) != 1 || strings.TrimSpace(values[0]) == "" || len(values[0]) > 256 {
			return errors.New("observer URL query may contain one non-empty session_id only")
		}
	}
	return nil
}
