// Package httpx holds net/http helpers that work the same under standard Go and TinyGo on Cloudflare
// Workers (workers-go).
//
// TinyGo 0.43's net/http ships the pre-Go 1.22 ServeMux: no "GET /path" method patterns and no {$}.
// Register plain paths and check the method in the handler with [Allow]. Render HTML with [Render]:
// workers-go streams unbuffered responses chunked, which broke htmx history restore under workerd, so
// responses are buffered and carry a Content-Length.
package httpx

import (
	"bytes"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/gsxhq/gsx"
)

// Allow reports whether r uses one of methods (GET also allows HEAD). Otherwise it replies 405 with an
// Allow header and returns false.
func Allow(w http.ResponseWriter, r *http.Request, methods ...string) bool {
	if slices.Contains(methods, r.Method) || (r.Method == http.MethodHead && slices.Contains(methods, http.MethodGet)) {
		return true
	}
	w.Header().Set("Allow", strings.Join(methods, ", "))
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	return false
}

// pages are the buffers pages render into, kept between requests. A page is 50–170 KB: one buffer
// grown from empty for each copies it over and over (the home page allocated 560 KB, mostly that).
// WriteHTML hands the bytes to the ResponseWriter before the buffer goes back.
var pages = sync.Pool{New: func() any { return bytes.NewBuffer(make([]byte, 0, 256<<10)) }}

// Render renders n into a buffer and writes it as HTML with a Content-Length. If rendering fails it
// replies 500 instead and returns the error, so a half-rendered page is never sent. Keep the buffer: a streamed
// (chunked) response broke htmx's history restore (Back) under local workerd.
func Render(w http.ResponseWriter, r *http.Request, n gsx.Node) error {
	buf := pages.Get().(*bytes.Buffer)
	buf.Reset()
	defer pages.Put(buf)
	if err := n.Render(r.Context(), buf); err != nil {
		http.Error(w, "render failed", http.StatusInternalServerError)
		return err
	}
	WriteHTML(w, buf.Bytes())
	return nil
}

// WriteHTML writes b as text/html with a Content-Length.
func WriteHTML(w http.ResponseWriter, b []byte) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(b)))
	w.Write(b)
}

// GetOnly wraps h so it only answers GET and HEAD (e.g. an http.FileServer, which serves any method).
func GetOnly(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if Allow(w, r, http.MethodGet) {
			h.ServeHTTP(w, r)
		}
	})
}

// Origin is the scheme and host the request was made to, e.g. "https://app.example.workers.dev" or
// "http://localhost:9913", for absolute URLs (sitemaps, canonical links) without configuring a site URL. On Workers
// the request URL is absolute; under standard Go the scheme comes from TLS and X-Forwarded-Proto.
func Origin(r *http.Request) string {
	scheme, host := r.URL.Scheme, r.URL.Host
	if host == "" {
		host = r.Host
	}
	if scheme == "" {
		scheme = "http"
		if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
			scheme = "https"
		}
	}
	return scheme + "://" + host
}
