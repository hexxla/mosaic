package ollama

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestDigestPinRejectsSameNameRetagging(t *testing.T) {
	expected := strings.Repeat("a", 64)
	digest := expected
	retagDuringInference := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			_, _ = fmt.Fprintf(w, `{"models":[{"name":"model:latest","digest":%q}]}`, digest)
		case "/api/embeddings":
			if retagDuringInference {
				digest = strings.Repeat("b", 64)
			}
			_, _ = fmt.Fprint(w, `{"embedding":[1,0]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	base, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	client := NewClient(base, "model")
	client.ExpectedDigest = expected
	if client.ModelIdentity() != "model@sha256:"+expected {
		t.Fatal("digest missing from durable identity")
	}
	if _, err := client.Embed(t.Context(), "text", 2); err != nil {
		t.Fatal(err)
	}
	digest = strings.Repeat("b", 64)
	if _, err := client.Embed(t.Context(), "text", 2); err == nil {
		t.Fatal("same-name model swap accepted")
	}
	digest = expected
	retagDuringInference = true
	if _, err := client.Embed(t.Context(), "text", 2); err == nil {
		t.Fatal("retagged inference result accepted")
	}
}
