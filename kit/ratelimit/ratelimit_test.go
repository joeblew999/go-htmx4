package ratelimit_test

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/joeblew999/go-htmx4/kit/ratelimit"
)

func TestClientKey(t *testing.T) {
	r := httptest.NewRequest("POST", "/board/add", nil)
	if got := ratelimit.ClientKey(r); got != "local" {
		t.Errorf("no header: key = %q, want local", got)
	}
	r.Header.Set("CF-Connecting-IP", "203.0.113.7")
	if got := ratelimit.ClientKey(r); got != "203.0.113.7" {
		t.Errorf("key = %q, want the CF-Connecting-IP", got)
	}
}

func TestAllowOutsideWorkers(t *testing.T) {
	ok, err := ratelimit.Allow("WRITES", "k")
	if !ok || !errors.Is(err, ratelimit.ErrNoBinding) {
		t.Errorf("Allow = %v, %v; want fail-open with ErrNoBinding", ok, err)
	}
}
