package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestRoutes(t *testing.T) {
	h := newServer().routes()

	tests := map[string]struct {
		method, path, body string
		want               []string
	}{
		"home": {method: "GET", path: "/", want: []string{
			"gsxui + htmx 4", `src="/static/htmx.min.js"`, `src="/static/hx-live.js"`,
			`src="/gsxui/index.js"`, `hx-boost:inherited="true"`, `id="gsxui-toaster"`,
			`hx-post="/greet"`, `hx-on:click="data.count++"`,
		}},
		"about":                {method: "GET", path: "/about", want: []string{"About this demo", "hx-live"}},
		"greet":                {method: "POST", path: "/greet", body: "name=Ada&flavour=gsxui", want: []string{"Hello, Ada.", `hx-swap-oob="beforeend:#gsxui-toaster"`, "data-gsxui-slot-toast"}},
		"greet shout":          {method: "POST", path: "/greet", body: "name=Ada&shout=on", want: []string{"HELLO, ADA!"}},
		"greet no name":        {method: "POST", path: "/greet", body: "name=", want: []string{"Please enter a name."}},
		"greet clear":          {method: "DELETE", path: "/greet", want: []string{`hx-swap-oob="beforeend:#gsxui-toaster"`, "Cleared"}},
		"home live":            {method: "GET", path: "/", want: []string{`hx-action="/greet"`, `hx-method="delete"`, `data-open="false"`, `hx-on="click from:outside -&gt; data.open = false"`, `aria.pressed = !aria.pressed`}},
		"server info":          {method: "GET", path: "/fragments/server-info", want: []string{"Uptime"}},
		"stats":                {method: "GET", path: "/fragments/stats", want: []string{"<ul"}},
		"healthz":              {method: "GET", path: "/healthz", want: []string{"ok"}},
		"htmx":                 {method: "GET", path: "/static/htmx.min.js", want: []string{"htmx"}},
		"hx-live":              {method: "GET", path: "/static/hx-live.js", want: []string{"hx-live"}},
		"gsxui js":             {method: "GET", path: "/gsxui/index.js", want: []string{"gsxui"}},
		"font via assets":      {method: "GET", path: "/assets/geist-latin-wght-normal.woff2"},
		"server info compiler": {method: "GET", path: "/fragments/server-info", want: []string{"gc go1."}},
		"head home":            {method: "HEAD", path: "/"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				_, _ = url.ParseQuery(tc.body)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}
			for _, w := range tc.want {
				if !strings.Contains(rec.Body.String(), w) {
					t.Errorf("body missing %q", w)
				}
			}
		})
	}
}

// Plain-path routes (TinyGo's pre-Go 1.22 ServeMux) still enforce methods and exact paths.
func TestRouteRules(t *testing.T) {
	h := newServer().routes()
	tests := []struct {
		method, path string
		status       int
		allow        string
	}{
		{"GET", "/greet", http.StatusMethodNotAllowed, "POST, DELETE"},
		{"POST", "/", http.StatusMethodNotAllowed, "GET"},
		{"POST", "/about", http.StatusMethodNotAllowed, "GET"},
		{"POST", "/static/htmx.min.js", http.StatusMethodNotAllowed, "GET"},
		{"GET", "/nope", http.StatusNotFound, ""},
		{"GET", "/fragments/nope", http.StatusNotFound, ""},
	}
	for _, tc := range tests {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, nil))
		if rec.Code != tc.status {
			t.Errorf("%s %s: status = %d, want %d", tc.method, tc.path, rec.Code, tc.status)
		}
		if got := rec.Header().Get("Allow"); got != tc.allow {
			t.Errorf("%s %s: Allow = %q, want %q", tc.method, tc.path, got, tc.allow)
		}
	}
}
