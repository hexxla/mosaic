package ollama

import (
	"errors"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type countingBody struct {
	io.Reader
	read int
}

func (b *countingBody) Read(p []byte) (int, error) {
	n, err := b.Reader.Read(p)
	b.read += n
	return n, err
}

func (*countingBody) Close() error { return nil }

func TestEmbedBoundsResponseBodies(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		status int
		limit  int
		want   string
	}{
		{"success", http.StatusOK, maxEmbeddingResponseBytes, "response exceeds"},
		{"error", http.StatusBadGateway, maxEmbeddingErrorBytes, "[truncated]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := &countingBody{Reader: strings.NewReader(strings.Repeat("x", tc.limit+1024))}
			c := testClient(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: tc.status, Status: http.StatusText(tc.status), Body: body}, nil
			})
			_, err := c.Embed(t.Context(), "hello", 1)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Embed error = %v, want %q", err, tc.want)
			}
			if body.read != tc.limit+1 {
				t.Fatalf("read %d bytes, want %d", body.read, tc.limit+1)
			}
			if tc.status != http.StatusOK && len(err.Error()) > tc.limit+128 {
				t.Fatalf("error diagnostic is not bounded: %d bytes", len(err.Error()))
			}
		})
	}
}

func TestEmbedValidatesFloat32Range(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, value string
		wantErr     bool
	}{
		{"finite", "3.4028234663852886e38", false},
		{"positive overflow", "1e100", true},
		{"negative overflow", "-1e100", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := testClient(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"embedding":[` + tc.value + `]}`))}, nil
			})
			vec, err := c.Embed(t.Context(), "hello", 1)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Embed = %v, %v", vec, err)
			}
			if err == nil && (len(vec) != 1 || math.IsInf(float64(vec[0]), 0) || math.IsNaN(float64(vec[0]))) {
				t.Fatalf("Embed returned a nonfinite vector: %v", vec)
			}
		})
	}
}

func TestEmbedPreservesURLComponentsWithoutMutatingBase(t *testing.T) {
	t.Parallel()
	// #nosec G101 -- Fixed fake credentials exercise URL redaction.
	const original = "https://alice:secret@example.test/prefix?token=private#fragment"
	u, err := url.Parse(original)
	if err != nil {
		t.Fatal(err)
	}
	c := testClient(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/prefix/api/embeddings" || r.URL.RawQuery != "token=private" || r.URL.Fragment != "" {
			t.Fatalf("embedding endpoint = %s", DiagnosticURL(r.URL))
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"embedding":[1]}`))}, nil
	})
	c.Base = u
	if _, err := c.Embed(t.Context(), "hello", 1); err != nil {
		t.Fatal(err)
	}
	if u.String() != original {
		t.Fatalf("Base changed: %s", DiagnosticURL(u))
	}
}

func TestClientDiagnosticsRemoveURLSecretsAndPreserveCause(t *testing.T) {
	t.Parallel()
	u, err := url.Parse("https://alice:secret@example.test/prefix?token=private#fragment")
	if err != nil {
		t.Fatal(err)
	}
	if got := DiagnosticURL(u); got != "https://example.test/prefix" {
		t.Fatalf("DiagnosticURL = %q", got)
	}
	failure := errors.New("transport unavailable")
	c := testClient(func(*http.Request) (*http.Response, error) { return nil, failure })
	c.Base = u
	for _, operation := range []func() error{
		func() error { return c.Ping(t.Context()) },
		func() error { _, err := c.Embed(t.Context(), "hello", 1); return err },
	} {
		err := operation()
		if !errors.Is(err, failure) {
			t.Fatalf("lost transport cause: %v", err)
		}
		for _, secret := range []string{"alice", "secret", "token", "private", "fragment"} {
			if strings.Contains(err.Error(), secret) {
				t.Fatalf("diagnostic contains %q: %v", secret, err)
			}
		}
	}
}

func testClient(transport roundTripFunc) *Client {
	return &Client{Base: &url.URL{Scheme: "http", Host: "localhost"}, Model: "test", HTTP: &http.Client{Transport: transport}}
}
