package app

import (
	"bytes"
	"encoding/json"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/joeblew999/charter/transport"
	gsxkitui "github.com/joeblew999/go-htmx4/gsxkit/ui"
	"github.com/joeblew999/go-htmx4/intl/catalog"
	"github.com/joeblew999/go-htmx4/intl/config"
	"github.com/joeblew999/go-htmx4/intl/registry"
	gsxkitcomponent "github.com/joeblew999/go-htmx4/views/gsxkit"
)

// TestMap: the map page (#164) opens on Cloudflare's place for the request, with its pin and popup, and on the whole
// world without one; its tiles are allowed by the Content Security Policy, and it loads gsxkit's map files.
func TestMap(t *testing.T) {
	h := newServer().routes()
	req := transport.WithCF(httptest.NewRequest("GET", "/map", nil),
		transport.CF{City: "Munich", Region: "Bavaria", Country: "DE", Latitude: "48.13743", Longitude: "11.57549", Colo: "MUC"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	body := rec.Body.String()
	if got := mapView(t, body); got.Center != [2]float64{48.13743, 11.57549} || got.Zoom != 10 || got.MaxZoom != 19 ||
		got.Tiles != "https://tile.openstreetmap.org/{z}/{x}/{y}.png" {
		t.Errorf("/map with Cloudflare's place: the map is %+v", got)
	}
	for _, want := range []string{
		"data-map-marker=\"[48.13743,11.57549]\" data-title=\"Cloudflare&#39;s guess: \u2068Munich\u2069\"",
		"Through Cloudflare&#39;s \u2068MUC\u2069 data center",
		"Germany",
		`data-map-you`,
		`<link rel="stylesheet" href="/gsxkit/map.css"`,
		`src="/gsxkit/map.js"`,
		`data-gsxkit-places`,
		`id="map-places-country-term"`,
		`name="country"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("/map with Cloudflare's place: no %s", want)
		}
	}
	if csp := rec.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "img-src 'self' data: https://tile.openstreetmap.org;") {
		t.Errorf("/map: Content-Security-Policy %q doesn't allow the tiles", csp)
	}
	world := get(t, h, "/map").Body.String()
	if got := mapView(t, world); got.Center != [2]float64{20, 0} || got.Zoom != 2 || strings.Contains(world, "data-map-marker") {
		t.Errorf("/map without Cloudflare's place: the map is %+v, or it has a pin", got)
	}
}

func TestPlacesPickerDrillsDownByLocationLevel(t *testing.T) {
	var language string
	requests := 0
	nominatim := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		language = r.URL.Query().Get("accept-language")
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Query().Get("featuretype") {
		case "country":
			if got := r.URL.Query().Get("q"); got != "Germany" {
				t.Errorf("country query = %q", got)
			}
			w.Write([]byte(`[{"display_name":"Deutschland","lat":"51","lon":"10","country_code":"de","address":{"country":"Deutschland"}}]`))
		case "state":
			if got := r.URL.Query().Get("q"); got != "Bavaria, Deutschland" || r.URL.Query().Get("countrycodes") != "de" {
				t.Errorf("region query = %q, countrycodes = %q", got, r.URL.Query().Get("countrycodes"))
			}
			w.Write([]byte(`[{"display_name":"Bayern, Deutschland","lat":"48.9","lon":"11.4","country_code":"de","address":{"country":"Deutschland","state":"Bayern"}}]`))
		case "city":
			if got := r.URL.Query().Get("q"); got != "Munich, Bayern, Deutschland" || r.URL.Query().Get("countrycodes") != "de" {
				t.Errorf("city query = %q, countrycodes = %q", got, r.URL.Query().Get("countrycodes"))
			}
			w.Write([]byte(`[{"display_name":"München, Bayern, Deutschland","lat":"48.13743","lon":"11.57549","country_code":"de","address":{"country":"Deutschland","state":"Bayern","city":"München"}}]`))
		default:
			if got := r.URL.Query().Get("q"); got != "Marienplatz, München, Bayern, Deutschland" || r.URL.Query().Get("countrycodes") != "de" {
				t.Errorf("street query = %q, countrycodes = %q", got, r.URL.Query().Get("countrycodes"))
			}
			w.Write([]byte(`[{"display_name":"Marienplatz, München, Bayern, Deutschland","lat":"48.13743","lon":"11.57549","country_code":"de","address":{"country":"Deutschland","state":"Bayern","city":"München","road":"Marienplatz"}}]`))
		}
	}))
	defer nominatim.Close()
	t.Setenv("NOMINATIM_URL", nominatim.URL)

	h := newServer().routes()
	search := func(step, term, country, code, region, city string) *httptest.ResponseRecorder {
		values := url.Values{
			"step": {step}, "term": {term}, "search": {"true"},
			"country": {country}, "country-code": {code}, "region": {region}, "city": {city},
		}
		return get(t, h, "/de/places/search?"+values.Encode())
	}
	choose := func(step string, place gsxkitcomponent.PlaceResult, country, code, region, city string) *httptest.ResponseRecorder {
		values := url.Values{
			"step": {step}, "choose": {"true"},
			"country": {country}, "country-code": {code}, "region": {region}, "city": {city},
			"place-name": {place.Name}, "place-country": {place.Country}, "place-country-code": {place.CountryCode},
			"place-region": {place.Region}, "place-city": {place.City}, "place-street": {place.Street},
			"latitude": {place.Latitude}, "longitude": {place.Longitude}, "open-url": {place.OpenURL},
		}
		return get(t, h, "/de/places/search?"+values.Encode())
	}

	rec := search("country", "Germany", "", "", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if language != "de" {
		t.Errorf("Nominatim accept-language = %q, want active Intl locale de", language)
	}
	second := search("country", "Germany", "", "", "", "")
	if second.Code != http.StatusOK || requests != 1 {
		t.Errorf("repeat search: status = %d, Nominatim requests = %d; want 200 and cached response", second.Code, requests)
	}
	if !strings.Contains(rec.Body.String(), `data-step="country"`) || !strings.Contains(rec.Body.String(), "Land auswählen") {
		t.Errorf("country results did not render as the active stage: %s", rec.Body.String())
	}

	country := gsxkitcomponent.PlaceResult{Name: "Deutschland", Country: "Deutschland", CountryCode: "de"}
	rec = choose("country", country, "", "", "", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `data-step="region"`) ||
		!strings.Contains(rec.Body.String(), `id="map-places-region-term"`) || !strings.Contains(rec.Body.String(), "Deutschland") {
		t.Fatalf("country selection did not advance to region: %d, %s", rec.Code, rec.Body.String())
	}
	time.Sleep(1100 * time.Millisecond)
	rec = search("region", "Bavaria", "Deutschland", "de", "", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Region auswählen") {
		t.Fatalf("region search: %d, %s", rec.Code, rec.Body.String())
	}
	region := gsxkitcomponent.PlaceResult{Name: "Bayern, Deutschland", Country: "Deutschland", CountryCode: "de", Region: "Bayern"}
	rec = choose("region", region, "Deutschland", "de", "", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `data-step="city"`) || !strings.Contains(rec.Body.String(), "Bayern") {
		t.Fatalf("region selection did not advance to city: %d, %s", rec.Code, rec.Body.String())
	}
	time.Sleep(1100 * time.Millisecond)
	rec = search("city", "Munich", "Deutschland", "de", "Bayern", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "München, Bayern, Deutschland") {
		t.Fatalf("city search: %d, %s", rec.Code, rec.Body.String())
	}
	city := gsxkitcomponent.PlaceResult{Name: "München", Country: "Deutschland", CountryCode: "de", Region: "Bayern", City: "München"}
	rec = choose("city", city, "Deutschland", "de", "Bayern", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `data-step="street"`) || !strings.Contains(rec.Body.String(), "München") {
		t.Fatalf("city selection did not advance to street: %d, %s", rec.Code, rec.Body.String())
	}
	time.Sleep(1100 * time.Millisecond)
	rec = search("street", "Marienplatz", "Deutschland", "de", "Bayern", "München")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Marienplatz") {
		t.Fatalf("street search: %d, %s", rec.Code, rec.Body.String())
	}
	street := gsxkitcomponent.PlaceResult{
		Name: "Marienplatz, München, Bayern, Deutschland", Country: "Deutschland", CountryCode: "de",
		Region: "Bayern", City: "München", Street: "Marienplatz", Latitude: "48.13743", Longitude: "11.57549",
		OpenURL: "https://www.openstreetmap.org/?mlat=48.13743&mlon=11.57549&zoom=16",
	}
	rec = choose("street", street, "Deutschland", "de", "Bayern", "München")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `data-step="done"`) ||
		!strings.Contains(rec.Body.String(), "Ausgewählter Ort") || !strings.Contains(rec.Body.String(), "Marienplatz") || requests != 4 {
		t.Errorf("street selection did not complete the picker: status=%d requests=%d body=%s", rec.Code, requests, rec.Body.String())
	}
}

func TestPlacesSearchRejectsOversizedTerm(t *testing.T) {
	rec := get(t, newServer().routes(), "/places/search?step=country&search=true&term="+strings.Repeat("x", 201))
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "Please shorten your place search terms.") {
		t.Errorf("oversized search: status = %d, body = %q", rec.Code, rec.Body.String())
	}
}

func TestPlaceSearchTextDeduplicatesRepeatedLevels(t *testing.T) {
	state := gsxkitcomponent.PlaceSearchState{Step: "region", Country: "Thailand", CountryCode: "th"}
	region := gsxkitcomponent.PlaceResult{Name: "Bangkok, Thailand", Country: "Thailand", City: "Bangkok", CountryCode: "th"}
	if !choosePlace(&state, region) || state.Region != "Bangkok" {
		t.Fatalf("region selection = %+v, want Bangkok", state)
	}
	state.Step = "street"
	state.Term = "Sukhumvit Road"
	state.City = "Bangkok"
	if got, want := placeSearchText(state), "Sukhumvit Road, Bangkok, Thailand"; got != want {
		t.Errorf("street search text = %q, want %q", got, want)
	}
}

// mapView is the view the page's map opens on: its data-map, which map.js reads.
func mapView(t *testing.T, body string) (v struct {
	Center  [2]float64
	Zoom    int
	Tiles   string
	MaxZoom int
}) {
	t.Helper()
	m := regexp.MustCompile(`data-map="([^"]*)"`).FindStringSubmatch(body)
	if m == nil {
		t.Fatal("no data-map")
	}
	if err := json.Unmarshal([]byte(html.UnescapeString(m[1])), &v); err != nil {
		t.Fatalf("data-map: %v", err)
	}
	return v
}

// TestGsxkitComponents: the app's gsxkit components are gsxkit's, gsx and browser files alike (as TestIntlScreens for
// intl's): change gsxkit/ui or gsxkit/js, then mise run gsxkit:add.
func TestGsxkitComponents(t *testing.T) {
	c, err := config.Find(".")
	if err != nil {
		t.Fatal(err)
	}
	var paths registry.Paths
	if paths.Catalogs, err = catalog.PackagePath(c.CatalogDir()); err != nil {
		t.Fatal(err)
	}
	if paths.Gsxui, err = catalog.PackagePath(filepath.Join(c.Root, c.Gsxui)); err != nil {
		t.Fatal(err)
	}
	r := gsxkitui.Components
	for _, comp := range r.Components {
		want, err := r.Source(comp, paths)
		if err == nil {
			want, err = registry.Format(c.Root, "views/gsxkit/"+comp.Name+".gsx", want)
		}
		if err != nil {
			t.Fatal(err)
		}
		if got, err := os.ReadFile(filepath.Join(c.Root, "views", "gsxkit", comp.Name+".gsx")); err != nil || !bytes.Equal(got, want) {
			t.Errorf("views/gsxkit/%s.gsx isn't gsxkit's (%v): mise run gsxkit:add", comp.Name, err)
		}
		for _, a := range comp.Assets {
			want, err := r.Asset(a)
			if err != nil {
				t.Fatal(err)
			}
			if got, err := os.ReadFile(filepath.Join(c.Root, "web", "gsxkit", a)); err != nil || !bytes.Equal(got, want) {
				t.Errorf("web/gsxkit/%s isn't gsxkit's (%v): mise run gsxkit:add", a, err)
			}
		}
	}
}
