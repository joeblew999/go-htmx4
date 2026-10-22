package main

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// TestNative: off Workers the same handler runs on charter's stand-ins: D1 is SQLite with the migrations, and the
// rate limit counts charter.toml's limit (3 a minute).
func TestNative(t *testing.T) {
	h := handler()
	post := func() int {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("POST", "/visit", nil))
		return rec.Code
	}
	for i := range 3 {
		if code := post(); code != 200 {
			t.Fatalf("visit %d: %d", i+1, code)
		}
	}
	if code := post(); code != 429 {
		t.Errorf("a 4th visit in a minute: %d, want 429", code)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if !strings.Contains(rec.Body.String(), "3 visits") {
		t.Errorf("page:\n%s", rec.Body.String())
	}
}
