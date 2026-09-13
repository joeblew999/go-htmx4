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
		"about":           {method: "GET", path: "/about", want: []string{"About this demo", "hx-live"}},
		"greet":           {method: "POST", path: "/greet", body: "name=Ada&flavour=gsxui", want: []string{"Hello, Ada.", `hx-swap-oob="beforeend:#gsxui-toaster"`, "data-gsxui-slot-toast"}},
		"greet shout":     {method: "POST", path: "/greet", body: "name=Ada&shout=on", want: []string{"HELLO, ADA!"}},
		"greet no name":   {method: "POST", path: "/greet", body: "name=", want: []string{"Please enter a name."}},
		"server info":     {method: "GET", path: "/fragments/server-info", want: []string{"Uptime"}},
		"stats":           {method: "GET", path: "/fragments/stats", want: []string{"<ul"}},
		"healthz":         {method: "GET", path: "/healthz", want: []string{"ok"}},
		"htmx":            {method: "GET", path: "/static/htmx.min.js", want: []string{"htmx"}},
		"hx-live":         {method: "GET", path: "/static/hx-live.js", want: []string{"hx-live"}},
		"gsxui js":        {method: "GET", path: "/gsxui/index.js", want: []string{"gsxui"}},
		"font via assets": {method: "GET", path: "/assets/geist-latin-wght-normal.woff2"},
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
