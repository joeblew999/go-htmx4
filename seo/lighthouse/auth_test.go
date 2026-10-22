package lighthouse_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/joeblew999/go-htmx4/seo/lighthouse"
)

// TestAuth: an API key goes as ?key=, a token as Authorization: Bearer, and neither when there is none.
func TestAuth(t *testing.T) {
	var key, header string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key, header = r.URL.Query().Get("key"), r.Header.Get("Authorization")
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	defer func(e string) { lighthouse.Endpoint = e }(lighthouse.Endpoint)
	lighthouse.Endpoint = srv.URL
	for _, tc := range []struct {
		auth        lighthouse.Auth
		key, header string
	}{
		{lighthouse.Auth{Key: "k123"}, "k123", ""},
		{lighthouse.Auth{Token: "t456"}, "", "Bearer t456"},
		{lighthouse.Auth{}, "", ""},
	} {
		lighthouse.Run(context.Background(), srv.Client(), tc.auth, "https://example.com/", "mobile")
		if key != tc.key || header != tc.header {
			t.Errorf("%+v: key %q, Authorization %q; want %q, %q", tc.auth, key, header, tc.key, tc.header)
		}
	}
}
