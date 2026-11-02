package app

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/joeblew999/charter/bindings"
	"github.com/joeblew999/charter/d1"
	"github.com/joeblew999/go-htmx4/internal/places"
	"github.com/joeblew999/go-htmx4/intl"
	"github.com/joeblew999/go-htmx4/views"
	"github.com/joeblew999/go-htmx4/views/gsxkit"
)

const defaultNominatimURL = "https://nominatim.openstreetmap.org"

type cachedPlaces struct {
	Payload string `json:"payload"`
}

type nominatimGate struct {
	LastRequest int64 `json:"last_request_ms"`
}

// PlacesSearch is the server-side action for the submit-driven places picker. D1 provides the shared one-request-per-
// second gate and a 24-hour query cache across Worker isolates; the locale in the active Intl request is passed through.
func (s *server) PlacesSearch(w http.ResponseWriter, r *http.Request) {
	q := places.Query{
		Country: strings.TrimSpace(r.URL.Query().Get("country")),
		State:   strings.TrimSpace(r.URL.Query().Get("state")),
		City:    strings.TrimSpace(r.URL.Query().Get("city")),
		Street:  strings.TrimSpace(r.URL.Query().Get("street")),
	}
	state := gsxkit.PlaceSearchState{Country: q.Country, Region: q.State, City: q.City, Street: q.Street}
	if r.URL.Query().Get("selected") == "true" {
		state.Selected = &gsxkit.PlaceResult{
			Name:      r.URL.Query().Get("selected-name"),
			Country:   q.Country,
			Region:    q.State,
			City:      q.City,
			Street:    q.Street,
			Latitude:  r.URL.Query().Get("latitude"),
			Longitude: r.URL.Query().Get("longitude"),
		}
		s.render(w, r, "places-search", views.PlacesPicker(state))
		return
	}
	if q.Country == "" && q.State == "" && q.City == "" && q.Street == "" {
		state.Status = "empty"
		s.render(w, r, "places-search", views.PlacesPicker(state))
		return
	}
	if !validPlaceQuery(q) {
		state.Status = "invalid"
		w.WriteHeader(http.StatusBadRequest)
		s.render(w, r, "places-search", views.PlacesPicker(state))
		return
	}
	results, limited, err := s.searchPlaces(r.Context(), q, intl.Current(r.Context()).ID())
	if limited {
		w.Header().Set("Retry-After", "1")
		state.Status = "limited"
		w.WriteHeader(http.StatusTooManyRequests)
	} else if err != nil {
		log.Printf("places search: %v", err)
		state.Status = "unavailable"
		w.WriteHeader(http.StatusBadGateway)
	} else if len(results) == 0 {
		state.Status = "none"
	}
	state.Results = results
	s.render(w, r, "places-search", views.PlacesPicker(state))
}

func validPlaceQuery(q places.Query) bool {
	return len(q.Country) <= 120 && len(q.State) <= 120 && len(q.City) <= 120 && len(q.Street) <= 200
}

func (s *server) searchPlaces(ctx context.Context, q places.Query, locale string) ([]gsxkit.PlaceResult, bool, error) {
	db, err := d1.Open("DB")
	if err != nil {
		return nil, false, err
	}
	keyInput, err := json.Marshal(struct {
		Query  places.Query `json:"query"`
		Locale string       `json:"locale"`
	}{q, locale})
	if err != nil {
		return nil, false, err
	}
	sum := sha256.Sum256(keyInput)
	key := hex.EncodeToString(sum[:])
	if row, err := d1.One[cachedPlaces](db,
		`SELECT payload FROM places_cache WHERE cache_key = ? AND expires_at > datetime('now')`, key); err == nil {
		var results []gsxkit.PlaceResult
		if err := json.Unmarshal([]byte(row.Payload), &results); err != nil {
			return nil, false, err
		}
		return results, false, nil
	} else if !errors.Is(err, d1.ErrNoRows) {
		return nil, false, err
	}
	if _, err := d1.One[nominatimGate](db, `UPDATE places_rate
SET last_request_ms = CAST((julianday('now') - 2440587.5) * 86400000 AS INTEGER)
WHERE id = 1 AND last_request_ms <= CAST((julianday('now') - 2440587.5) * 86400000 AS INTEGER) - 1000
RETURNING last_request_ms`); errors.Is(err, d1.ErrNoRows) {
		return nil, true, nil
	} else if err != nil {
		return nil, false, err
	}

	found, err := (places.Nominatim{Endpoint: nominatimEndpoint()}).Search(ctx, q, locale)
	if err != nil {
		return nil, false, err
	}
	results := make([]gsxkit.PlaceResult, 0, len(found))
	for _, p := range found {
		results = append(results, gsxkit.PlaceResult{
			Name: p.Name, Country: p.Country, Region: p.Region, City: p.City, Street: p.Street,
			Latitude: p.Latitude, Longitude: p.Longitude, OpenURL: p.OpenURL,
		})
	}
	payload, err := json.Marshal(results)
	if err != nil {
		return nil, false, err
	}
	if err := d1.Exec(db, `INSERT INTO places_cache (cache_key, payload, expires_at)
VALUES (?, ?, datetime('now', '+1 day'))
ON CONFLICT (cache_key) DO UPDATE SET payload = excluded.payload, expires_at = excluded.expires_at`,
		key, string(payload)); err != nil {
		return nil, false, err
	}
	if err := d1.Exec(db, `DELETE FROM places_cache WHERE cache_key NOT IN
(SELECT cache_key FROM places_cache ORDER BY expires_at DESC LIMIT 500)`); err != nil {
		return nil, false, err
	}
	return results, false, nil
}

func nominatimEndpoint() string {
	return strings.TrimRight(cmp.Or(bindings.Var("NOMINATIM_URL"), defaultNominatimURL), "/")
}
