package httpx_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gsxhq/gsx"
	"github.com/joeblew999/go-htmx4/kit/httpx"
)

func TestAllow(t *testing.T) {
	tests := []struct {
		method  string
		methods []string
		ok      bool
		allow   string
	}{
		{"GET", []string{"GET"}, true, ""},
		{"HEAD", []string{"GET"}, true, ""},
		{"HEAD", []string{"POST"}, false, "POST"},
		{"POST", []string{"GET"}, false, "GET"},
		{"GET", []string{"POST", "DELETE"}, false, "POST, DELETE"},
		{"DELETE", []string{"POST", "DELETE"}, true, ""},
	}
	for _, tc := range tests {
		rec := httptest.NewRecorder()
		ok := httpx.Allow(rec, httptest.NewRequest(tc.method, "/", nil), tc.methods...)
		if ok != tc.ok {
			t.Errorf("%s with %v: ok = %v, want %v", tc.method, tc.methods, ok, tc.ok)
		}
		if !ok && rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s with %v: status = %d, want 405", tc.method, tc.methods, rec.Code)
		}
		if got := rec.Header().Get("Allow"); got != tc.allow {
			t.Errorf("%s with %v: Allow = %q, want %q", tc.method, tc.methods, got, tc.allow)
		}
	}
}

type node func(context.Context, io.Writer) error

func (n node) Render(ctx context.Context, w io.Writer) error { return n(ctx, w) }

var _ gsx.Node = node(nil)

func TestRender(t *testing.T) {
	rec := httptest.NewRecorder()
	err := httpx.Render(rec, httptest.NewRequest("GET", "/", nil), node(func(_ context.Context, w io.Writer) error {
		_, err := io.WriteString(w, "<p>hi</p>")
		return err
	}))
	if err != nil || rec.Code != http.StatusOK || rec.Body.String() != "<p>hi</p>" {
		t.Fatalf("Render = %v, %d %q", err, rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Length"); got != "9" {
		t.Errorf("Content-Length = %q, want 9", got)
	}
	if got := rec.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q", got)
	}
}

func TestRenderError(t *testing.T) {
	boom := errors.New("boom")
	rec := httptest.NewRecorder()
	err := httpx.Render(rec, httptest.NewRequest("GET", "/", nil), node(func(_ context.Context, w io.Writer) error {
		io.WriteString(w, "<p>half")
		return boom
	}))
	if !errors.Is(err, boom) || rec.Code != http.StatusInternalServerError {
		t.Fatalf("Render = %v, %d, want boom and 500", err, rec.Code)
	}
	if body := rec.Body.String(); body != "render failed\n" {
		t.Errorf("body = %q: the half-rendered page must not be sent", body)
	}
}

func TestGetOnly(t *testing.T) {
	h := httpx.GetOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "file") }))
	for method, want := range map[string]int{"GET": 200, "HEAD": 200, "POST": 405, "PUT": 405} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(method, "/x.js", nil))
		if rec.Code != want {
			t.Errorf("%s: status = %d, want %d", method, rec.Code, want)
		}
	}
}

func ExampleAllow() {
	mux := http.NewServeMux()
	// Plain path + method check: TinyGo's ServeMux has no "POST /greet" patterns.
	mux.HandleFunc("/greet", func(w http.ResponseWriter, r *http.Request) {
		if httpx.Allow(w, r, http.MethodPost) {
			io.WriteString(w, "hello")
		}
	})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/greet", nil))
	fmt.Println(rec.Code, rec.Header().Get("Allow"))
	// Output: 405 POST
}

func TestNoIndex(t *testing.T) {
	rec := httptest.NewRecorder()
	httpx.NoIndex(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "fragment") })).
		ServeHTTP(rec, httptest.NewRequest("GET", "/fragments/stats", nil))
	if got := rec.Header().Get("X-Robots-Tag"); got != "noindex" || rec.Body.String() != "fragment" {
		t.Errorf("X-Robots-Tag = %q, body %q", got, rec.Body.String())
	}
}

func TestOrigin(t *testing.T) {
	native := httptest.NewRequest("GET", "/robots.txt", nil) // Host example.com, no scheme in URL
	proxied := httptest.NewRequest("GET", "/robots.txt", nil)
	proxied.Header.Set("X-Forwarded-Proto", "https")
	workers := httptest.NewRequest("GET", "https://app.acme.workers.dev/robots.txt", nil)
	workers.Host = "" // workers-go takes Host from the headers, which a Workers request may not carry
	for _, tc := range []struct {
		r    *http.Request
		want string
	}{
		{native, "http://example.com"},
		{proxied, "https://example.com"},
		{workers, "https://app.acme.workers.dev"},
	} {
		if got := httpx.Origin(tc.r); got != tc.want {
			t.Errorf("Origin(%s) = %q, want %q", tc.r.URL, got, tc.want)
		}
	}
}
