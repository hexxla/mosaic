package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"

	"github.com/gorilla/websocket"
)

func main() {
	host := flag.String("host", "localhost:8787", "mosaic-mcp host:port")
	path := flag.String("path", "/observability/stream", "WebSocket endpoint path")
	sessionID := flag.String("session-id", "mosaic-mcp-session", "session ID to subscribe to")
	flag.Parse()

	u := url.URL{Scheme: "ws", Host: *host, Path: *path}
	u.RawQuery = "session_id=" + *sessionID
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer func() { _ = c.Close() }()

	log.Println("connected to WebSocket, listening for events...")

	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			return
		}
		fmt.Printf("EVENT: %s\n", message)
	}
}
