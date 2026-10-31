package e2e

import (
	"fmt"
	"testing"
	"time"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
)

// testMap is gsxkit's map component on the map page (#164): reached by a boosted navigation, Leaflet draws it and its
// tiles once its stylesheet applies; the zoom buttons zoom; the location button asks the browser (here a set position, the permission granted) and
// pins the reader there, whose popup is the component's own; and a full load of the page draws it too.
func testMap(t *testing.T) {
	b := newTab(t, "map")
	b.open("/")
	marker := b.num(`window.__marker = Math.floor(Math.random() * 1e9)`)
	b.run(
		browser.SetPermission(&browser.PermissionDescriptor{Name: "geolocation"}, browser.PermissionSettingGranted).WithOrigin(base),
		emulation.SetGeolocationOverride().WithLatitude(48.13743).WithLongitude(11.57549).WithAccuracy(10),
	)
	b.clickText("nav a", b.str(`document.querySelector("nav a[href$='/map']").textContent.trim()`))
	drawn := b.waitJS(10*time.Second, `location.pathname === "/map" && document.querySelector("[data-map]")?.dataset.ready === "true"`)
	check(t, "a boosted navigation to /map draws the map", drawn && b.is(fmt.Sprintf(`window.__marker === %d`, marker)))
	check(t, "Leaflet draws OpenStreetMap's tiles", b.waitJS(15*time.Second,
		`[...document.querySelectorAll("[data-map] .leaflet-tile")].some((img) => img.src.startsWith("https://tile.openstreetmap.org/") && img.complete && img.naturalWidth > 0)`))
	check(t, "the map's stylesheet applies", b.is(`getComputedStyle(document.querySelector("[data-map] .leaflet-pane")).position === "absolute"`))

	zoom := b.num(`Number(document.querySelector("[data-map]").dataset.zoom)`)
	b.eval(`document.querySelector("[data-map-action=zoom-in]").click()`, nil)
	check(t, "zoom in", b.waitJS(5*time.Second, fmt.Sprintf(`Number(document.querySelector("[data-map]").dataset.zoom) === %d`, zoom+1)))
	b.eval(`document.querySelector("[data-map-action=zoom-out]").click()`, nil)
	check(t, "zoom out", b.waitJS(5*time.Second, fmt.Sprintf(`Number(document.querySelector("[data-map]").dataset.zoom) === %d`, zoom)))

	// On Cloudflare the page has a pin already, at its guess of where the request came from; locally none.
	pins := b.num(`document.querySelectorAll("[data-map] .leaflet-marker-icon").length`)
	b.eval(`document.querySelector("[data-map-action=locate]").click()`, nil)
	found := b.waitJS(10*time.Second, `document.querySelector("[data-map]").dataset.locate === "found"`)
	check(t, "the location button finds the reader", found, b.str(`document.querySelector("[data-map]").dataset.locate`))
	check(t, "and pins them, zoomed in", b.is(fmt.Sprintf(`document.querySelectorAll("[data-map] .leaflet-marker-icon [data-map-pin]").length === %d && Number(document.querySelector("[data-map]").dataset.zoom) >= 14`, pins+1)))
	b.eval(`[...document.querySelectorAll("[data-map] .leaflet-marker-icon")].at(-1).click()`, nil)
	you := b.str(`document.querySelector("[data-map] template[data-map-you]").dataset.title`)
	check(t, "the pin's popup is the component's", b.waitJS(5*time.Second, fmt.Sprintf(`document.querySelector("[data-map] .leaflet-popup-content")?.textContent.trim() === %q`, you)), you)

	b.run(chromedp.Reload(), chromedp.WaitReady("main", chromedp.ByQuery))
	check(t, "a full load of /map draws the map", b.waitJS(10*time.Second, `document.querySelector("[data-map]")?.dataset.ready === "true"`))
	b.noProblems()
}
