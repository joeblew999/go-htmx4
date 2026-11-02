package places

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearchForwardsLocaleAndDrilldownFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("accept-language"); got != "th" {
			t.Errorf("accept-language query = %q", got)
		}
		if got := r.Header.Get("Accept-Language"); got != "th" {
			t.Errorf("Accept-Language header = %q", got)
		}
		if got := r.Header.Get("User-Agent"); got == "" {
			t.Error("missing identifying User-Agent")
		}
		for key, want := range map[string]string{
			"format": "jsonv2", "addressdetails": "1", "limit": "5",
			"q": "Bangkok, Thailand", "countrycodes": "th", "featureType": "city",
		} {
			if got := r.URL.Query().Get(key); got != want {
				t.Errorf("%s = %q, want %q", key, got, want)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"display_name":"กรุงเทพมหานคร","lat":"13.75","lon":"100.5","address":{"country":"ประเทศไทย","country_code":"th","state":"กรุงเทพมหานคร","city":"กรุงเทพมหานคร","road":"ถนนพระรามที่ 1"}}]`))
	}))
	defer server.Close()

	results, err := (Nominatim{Endpoint: server.URL}).Search(context.Background(),
		Query{Text: "Bangkok, Thailand", CountryCode: "TH", FeatureType: "city"}, "th")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Name != "กรุงเทพมหานคร" || results[0].City != "กรุงเทพมหานคร" {
		t.Fatalf("results = %#v", results)
	}
	if results[0].OpenURL == "" {
		t.Fatal("result has no OpenStreetMap link")
	}
	if results[0].CountryCode != "th" {
		t.Errorf("country code = %q, want th", results[0].CountryCode)
	}
}

func TestSearchRejectsInvalidEndpointAndHTTPError(t *testing.T) {
	if _, err := (Nominatim{Endpoint: "not a URL"}).Search(context.Background(), Query{Text: "Bangkok"}, "th"); err == nil {
		t.Fatal("invalid endpoint returned no error")
	}
	if _, err := (Nominatim{Endpoint: "file:///tmp/nominatim"}).Search(context.Background(), Query{Text: "Bangkok"}, "th"); err == nil {
		t.Fatal("non-HTTP endpoint returned no error")
	}
	if _, err := (Nominatim{Endpoint: "https://nominatim.openstreetmap.org"}).Search(context.Background(), Query{Text: "Bangkok", FeatureType: "countrycode"}, "th"); err == nil {
		t.Fatal("invalid feature type returned no error")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	if _, err := (Nominatim{Endpoint: server.URL}).Search(context.Background(), Query{Text: "Bangkok"}, "th"); err == nil {
		t.Fatal("HTTP error returned no error")
	}
}
