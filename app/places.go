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

// PlacesSearch serves each submit-driven step of the Places picker. D1 provides the shared one-request-per-second
// gate and a 24-hour query cache across Worker isolates; the active Intl locale is passed through to Nominatim.
func (s *server) PlacesSearch(w http.ResponseWriter, r *http.Request) {
	state := gsxkit.PlaceSearchState{
		Step:        r.URL.Query().Get("step"),
		Term:        strings.TrimSpace(r.URL.Query().Get("term")),
		Country:     strings.TrimSpace(r.URL.Query().Get("country")),
		CountryCode: strings.TrimSpace(r.URL.Query().Get("country-code")),
		Region:      strings.TrimSpace(r.URL.Query().Get("region")),
		City:        strings.TrimSpace(r.URL.Query().Get("city")),
		Street:      strings.TrimSpace(r.URL.Query().Get("street")),
	}
	if state.Step == "" {
		state.Step = "country"
	}
	if !validPlaceStep(state.Step) {
		state.Step = "country"
		state.Status = "invalid"
		w.WriteHeader(http.StatusBadRequest)
		s.renderPlacesPicker(w, r, state)
		return
	}
	if !validPlaceState(state) {
		state.Status = "invalid"
		w.WriteHeader(http.StatusBadRequest)
		s.renderPlacesPicker(w, r, state)
		return
	}
	if r.URL.Query().Get("back") == "true" {
		switch state.Step {
		case "country":
			state.Country, state.CountryCode, state.Region, state.City, state.Street = "", "", "", "", ""
		case "region":
			state.Region, state.City, state.Street = "", "", ""
		case "city":
			state.City, state.Street = "", ""
		case "street":
			state.Street = ""
		}
		state.Selected = nil
	}
	if r.URL.Query().Get("choose") == "true" {
		choice := gsxkit.PlaceResult{
			Name:        strings.TrimSpace(r.URL.Query().Get("place-name")),
			Country:     strings.TrimSpace(r.URL.Query().Get("place-country")),
			CountryCode: strings.TrimSpace(r.URL.Query().Get("place-country-code")),
			Region:      strings.TrimSpace(r.URL.Query().Get("place-region")),
			City:        strings.TrimSpace(r.URL.Query().Get("place-city")),
			Street:      strings.TrimSpace(r.URL.Query().Get("place-street")),
			Latitude:    strings.TrimSpace(r.URL.Query().Get("latitude")),
			Longitude:   strings.TrimSpace(r.URL.Query().Get("longitude")),
			OpenURL:     strings.TrimSpace(r.URL.Query().Get("open-url")),
		}
		next := state
		if !validPlaceResult(choice) || !choosePlace(&next, choice) || !validPlaceState(next) {
			state.Status = "invalid"
			w.WriteHeader(http.StatusBadRequest)
		} else {
			state = next
		}
		s.renderPlacesPicker(w, r, state)
		return
	}
	if r.URL.Query().Get("search") != "true" {
		s.renderPlacesPicker(w, r, state)
		return
	}
	if state.Step == "done" {
		state.Status = "invalid"
		w.WriteHeader(http.StatusBadRequest)
		s.renderPlacesPicker(w, r, state)
		return
	}
	if state.Term == "" {
		state.Status = "empty"
		s.renderPlacesPicker(w, r, state)
		return
	}
	if !validPlaceState(state) {
		state.Status = "invalid"
		w.WriteHeader(http.StatusBadRequest)
		s.renderPlacesPicker(w, r, state)
		return
	}
	q := places.Query{
		Text:        placeSearchText(state),
		CountryCode: state.CountryCode,
		FeatureType: placeFeatureType(state.Step),
	}
	if state.Step == "country" {
		q.CountryCode = ""
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
	} else {
		results = placesInCountry(results, state.CountryCode, state.Country)
		if len(results) == 0 {
			state.Status = "none"
		}
	}
	state.Results = results
	s.renderPlacesPicker(w, r, state)
}

func (s *server) renderPlacesPicker(w http.ResponseWriter, r *http.Request, state gsxkit.PlaceSearchState) {
	state.BackStep = previousPlaceStep(state.Step)
	s.render(w, r, "places-search", views.PlacesPicker(state))
}

func validPlaceStep(step string) bool {
	return step == "country" || step == "region" || step == "city" || step == "street" || step == "done"
}

func validPlaceState(state gsxkit.PlaceSearchState) bool {
	if len(state.Term) > 200 || len(state.Country) > 120 || len(state.CountryCode) > 2 ||
		len(state.Region) > 120 || len(state.City) > 120 || len(state.Street) > 200 {
		return false
	}
	if state.CountryCode != "" && !validPlaceCountryCode(state.CountryCode) {
		return false
	}
	switch state.Step {
	case "country":
		return true
	case "region":
		return state.Country != ""
	case "city":
		return state.Country != "" && state.Region != ""
	case "street":
		return state.Country != "" && state.Region != "" && state.City != ""
	case "done":
		return true
	}
	return false
}

func validPlaceCountryCode(code string) bool {
	if len(code) != 2 {
		return false
	}
	for _, c := range code {
		if c < 'a' || c > 'z' {
			if c < 'A' || c > 'Z' {
				return false
			}
		}
	}
	return true
}

func validPlaceResult(place gsxkit.PlaceResult) bool {
	return len(place.Name) <= 500 && len(place.Country) <= 120 && len(place.CountryCode) <= 2 &&
		len(place.Region) <= 120 && len(place.City) <= 120 && len(place.Street) <= 200 &&
		len(place.Latitude) <= 32 && len(place.Longitude) <= 32 && len(place.OpenURL) <= 256
}

func previousPlaceStep(step string) string {
	switch step {
	case "region":
		return "country"
	case "city":
		return "region"
	case "street":
		return "city"
	case "done":
		return "street"
	default:
		return ""
	}
}

func placeSearchText(state gsxkit.PlaceSearchState) string {
	parts := make([]string, 0, 4)
	parts = append(parts, state.Term)
	switch state.Step {
	case "region":
		parts = append(parts, state.Country)
	case "city":
		parts = append(parts, state.Region, state.Country)
	case "street":
		parts = append(parts, state.City, state.Region, state.Country)
	}
	return strings.Join(nonEmptyPlaceParts(parts), ", ")
}

func nonEmptyPlaceParts(parts []string) []string {
	out := parts[:0]
	for _, part := range parts {
		if part == "" {
			continue
		}
		duplicate := false
		for _, existing := range out {
			if strings.EqualFold(part, existing) {
				duplicate = true
				break
			}
		}
		if !duplicate {
			out = append(out, part)
		}
	}
	return out
}

func placeFeatureType(step string) string {
	switch step {
	case "country":
		return "country"
	case "region":
		return "state"
	case "city":
		return "city"
	default:
		return ""
	}
}

func choosePlace(state *gsxkit.PlaceSearchState, place gsxkit.PlaceResult) bool {
	if place.Name == "" {
		return false
	}
	if state.Step != "country" && !placeMatchesCountry(state.CountryCode, state.Country, place) {
		return false
	}
	switch state.Step {
	case "country":
		state.Country = firstNonEmpty(place.Country, place.Name)
		state.CountryCode = strings.ToLower(place.CountryCode)
		if state.CountryCode != "" && !validPlaceCountryCode(state.CountryCode) {
			return false
		}
		state.Region, state.City, state.Street = "", "", ""
		state.Step = "region"
	case "region":
		state.Region = firstNonEmpty(place.Region, place.City, place.Name)
		state.City, state.Street = "", ""
		state.Step = "city"
	case "city":
		state.City = firstNonEmpty(place.City, place.Name)
		state.Street = ""
		state.Step = "street"
	case "street":
		state.Street = firstNonEmpty(place.Street, place.Name)
		state.Selected = &place
		state.Step = "done"
	default:
		return false
	}
	state.Term = ""
	state.Results = nil
	state.Status = ""
	return true
}

func placesInCountry(results []gsxkit.PlaceResult, countryCode, country string) []gsxkit.PlaceResult {
	if countryCode == "" && country == "" {
		return results
	}
	filtered := results[:0]
	for _, place := range results {
		if placeMatchesCountry(countryCode, country, place) {
			filtered = append(filtered, place)
		}
	}
	return filtered
}

func placeMatchesCountry(countryCode, country string, place gsxkit.PlaceResult) bool {
	if countryCode != "" {
		return place.CountryCode != "" && strings.EqualFold(countryCode, place.CountryCode)
	}
	return country != "" && place.Country != "" && strings.EqualFold(country, place.Country)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
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
			CountryCode: p.CountryCode, Latitude: p.Latitude, Longitude: p.Longitude, OpenURL: p.OpenURL,
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
