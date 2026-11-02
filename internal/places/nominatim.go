package places

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Query is a localized Nominatim search constrained to one address level.
type Query struct {
	Text, CountryCode, FeatureType string
}

// Result is a localized Nominatim result and its OpenStreetMap link.
type Result struct {
	Name, Country, Region, City, Street string
	CountryCode                         string
	Latitude, Longitude                 string
	OpenURL                             string
}

// Nominatim searches an explicitly configured Nominatim endpoint.
type Nominatim struct {
	Endpoint   string
	HTTPClient *http.Client
}

type searchResult struct {
	Name    string            `json:"display_name"`
	Lat     string            `json:"lat"`
	Lon     string            `json:"lon"`
	Address map[string]string `json:"address"`
}

// Search is deliberately submit-driven; the public Nominatim service does not support autocomplete.
func (n Nominatim) Search(ctx context.Context, q Query, locale string) ([]Result, error) {
	endpoint, err := url.Parse(n.Endpoint)
	if err != nil || (endpoint.Scheme != "http" && endpoint.Scheme != "https") || endpoint.Host == "" {
		return nil, fmt.Errorf("invalid Nominatim endpoint %q", n.Endpoint)
	}
	if strings.TrimSpace(q.Text) == "" {
		return nil, fmt.Errorf("Nominatim search text is empty")
	}
	if q.FeatureType != "" && q.FeatureType != "country" && q.FeatureType != "state" && q.FeatureType != "city" && q.FeatureType != "settlement" {
		return nil, fmt.Errorf("invalid Nominatim feature type %q", q.FeatureType)
	}
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/search"
	params := endpoint.Query()
	params.Set("format", "jsonv2")
	params.Set("addressdetails", "1")
	params.Set("limit", "5")
	params.Set("q", q.Text)
	params.Set("accept-language", locale)
	if q.CountryCode != "" {
		params.Set("countrycodes", strings.ToLower(q.CountryCode))
	}
	if q.FeatureType != "" {
		params.Set("featuretype", q.FeatureType)
	}
	endpoint.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create Nominatim request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", locale)
	req.Header.Set("User-Agent", "go-htmx4/1.0 (+https://github.com/joeblew999/go-htmx4)")
	client := n.HTTPClient
	if client == nil {
		client = nominatimHTTPClient(8 * time.Second)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Nominatim search: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Nominatim search: HTTP %d", resp.StatusCode)
	}
	var found []searchResult
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&found); err != nil {
		return nil, fmt.Errorf("decode Nominatim response: %w", err)
	}
	results := make([]Result, 0, len(found))
	for _, p := range found {
		if p.Name == "" {
			continue
		}
		lat, err := strconv.ParseFloat(p.Lat, 64)
		if err != nil {
			return nil, fmt.Errorf("parse Nominatim latitude %q: %w", p.Lat, err)
		}
		lon, err := strconv.ParseFloat(p.Lon, 64)
		if err != nil {
			return nil, fmt.Errorf("parse Nominatim longitude %q: %w", p.Lon, err)
		}
		if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
			return nil, fmt.Errorf("Nominatim returned out-of-range coordinates (%s, %s)", p.Lat, p.Lon)
		}
		mapURL := url.URL{Scheme: "https", Host: "www.openstreetmap.org", Path: "/"}
		mapQuery := mapURL.Query()
		mapQuery.Set("mlat", p.Lat)
		mapQuery.Set("mlon", p.Lon)
		mapQuery.Set("zoom", "16")
		mapURL.RawQuery = mapQuery.Encode()
		results = append(results, Result{
			Name: p.Name, Country: p.Address["country"], Region: p.Address["state"], CountryCode: p.Address["country_code"],
			City:     first(p.Address, "city", "town", "village", "municipality", "hamlet"),
			Street:   first(p.Address, "road", "pedestrian", "footway", "path"),
			Latitude: strconv.FormatFloat(lat, 'f', -1, 64), Longitude: strconv.FormatFloat(lon, 'f', -1, 64),
			OpenURL: mapURL.String(),
		})
	}
	return results, nil
}

func first(values map[string]string, keys ...string) string {
	for _, key := range keys {
		if value := values[key]; value != "" {
			return value
		}
	}
	return ""
}
