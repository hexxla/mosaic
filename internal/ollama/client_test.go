package ollama

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestClient_Embed_expectedDim(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"embedding":[` + repeatJSONFloats(384) + `]}`)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embeddings" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(payload)
	}))
	t.Cleanup(ts.Close)

	u, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	c := NewClient(u, "all-minilm")
	c.HTTP = ts.Client()

	vec, err := c.Embed(t.Context(), "hello", 384)
	if err != nil {
		t.Fatal(err)
	}
	if len(vec) != 384 {
		t.Fatalf("len %d", len(vec))
	}
}

func repeatJSONFloats(n int) string {
	var b strings.Builder
	b.Grow(n * 5)
	for i := range n {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString("1.0")
	}
	return b.String()
}
