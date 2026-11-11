package mcpsrv

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/sploitzberg/mosaic/internal/core/domain"
)

func TestBrowseCursorRejectsTamperingAndForeignSessions(t *testing.T) {
	codec := newBrowseCursorCodec()
	state := browseCursor{Owner: "session-a", Mode: "browse", Expires: time.Now().Add(time.Minute).Unix(), Query: domain.BrowseQuery{SnapshotSeq: new(uint64(7))}, Ring: 2, Offset: 3}
	token, err := codec.encode(state)
	if err != nil {
		t.Fatal(err)
	}
	body, _, _ := strings.Cut(token, ".")
	decoded, err := base64.RawURLEncoding.DecodeString(body)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(decoded), "session-a") {
		t.Fatal("cursor exposed transport session ID")
	}
	got, err := codec.decode(token, "session-a", "browse")
	if err != nil || got.Ring != 2 || *got.Query.SnapshotSeq != 7 {
		t.Fatalf("valid cursor=%+v,%v", got, err)
	}
	for _, test := range []struct {
		name, token, owner, mode string
		codec                    *browseCursorCodec
	}{
		{"other session", token, "session-b", "browse", codec},
		{"other tool", token, "session-a", "export", codec},
		{"tampered", token + "x", "session-a", "browse", codec},
		{"restart", token, "session-a", "browse", newBrowseCursorCodec()},
		{"malformed", "not-a-token", "session-a", "browse", codec},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := test.codec.decode(test.token, test.owner, test.mode); err == nil {
				t.Fatal("invalid cursor accepted")
			}
		})
	}
	state.Expires = time.Now().Add(-time.Second).Unix()
	token, err = codec.encode(state)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := codec.decode(token, "session-a", "browse"); err == nil {
		t.Fatal("expired cursor accepted")
	}
}
