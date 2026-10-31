package app

import (
	"bytes"
	"encoding/json"
	"html"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/joeblew999/charter/transport"
	gsxkitui "github.com/joeblew999/go-htmx4/gsxkit/ui"
	"github.com/joeblew999/go-htmx4/intl/catalog"
	"github.com/joeblew999/go-htmx4/intl/config"
	"github.com/joeblew999/go-htmx4/intl/registry"
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
