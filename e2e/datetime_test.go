package e2e

import (
	"testing"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
)

// testViewerTimeZone: dates follow the viewer. The browser formats them with its own Intl (static/intl.js) in its own
// zone; choosing a zone and clock in the header dialog stores them, and every page formats in them from then on.
func testViewerTimeZone(t *testing.T) {
	b := newTab(t, "browser")
	b.run(emulation.SetTimezoneOverride("Asia/Tokyo"))
	b.open("/")
	b.waitJS(5*time.Second, `document.readyState === "complete" && typeof htmx === "object"`)

	// No choice: the browser's own zone.
	b.clickText("button", "Show server info")
	ok := b.waitJS(5*time.Second, `(document.querySelector("#server-info time")?.textContent ?? "").includes("GMT+9")`)
	text := b.str(`document.querySelector("#server-info time")?.textContent ?? ""`)
	check(t, "the browser formats a time in its own zone", ok, " ", text)
	check(t, "durations are formatted too", b.waitJS(5*time.Second, `/\bsec\b/.test(document.querySelector("#server-info data[data-intl=duration]")?.textContent ?? "")`),
		" ", b.str(`document.querySelector("#server-info data[data-intl=duration]")?.textContent ?? ""`))
	b.run(chromedp.KeyEvent("\x1b"))

	b.open("/formats")
	check(t, "the formats page is in Tokyo time", b.waitJS(5*time.Second, `document.body.textContent.includes("Sunday, July 5, 2026")`))

	// A value this browser's Intl rejects keeps the server's fallback.
	kept := b.str(`(() => {
		const el = document.createElement("time");
		el.setAttribute("datetime", "2026-07-04T15:30:45Z");
		el.dataset.intl = "datetime";
		el.dataset.opts = '{"dateStyle":"bogus"}';
		el.textContent = "2026-07-04 15:30 UTC";
		document.body.append(el);
		intl.update(el);
		return el.textContent;
	})()`)
	check(t, "rejected options keep the fallback", kept == "2026-07-04 15:30 UTC", " ", kept)

	// Choose America/Los_Angeles and a 24-hour clock in the header dialog. The zone list is the browser's own.
	b.open("/")
	b.waitJS(5*time.Second, `document.readyState === "complete" && typeof htmx === "object"`)
	b.eval(`document.querySelector('header button[hx-get$="/fragments/preferences"]').click()`, nil)
	check(t, "preferences dialog lists the browser's zones", b.waitJS(5*time.Second, `document.querySelectorAll("#pref-tz option").length > 300`))
	check(t, "zones are named", b.is(`[...document.querySelectorAll("#pref-tz option")].some(o => o.value === "Europe/Berlin" && o.textContent.includes("Central European"))`))
	check(t, "automatic names the browser's zone", b.is(`document.querySelector('#pref-tz option[value=""]').textContent.includes("Asia/Tokyo")`))
	b.eval(`(() => {
		document.getElementById("pref-tz").value = "America/Los_Angeles";
		document.getElementById("pref-hc").value = "h23";
		document.querySelector("#preferences button[type=submit]").click();
	})()`, nil)
	ok = b.waitJS(8*time.Second, `document.cookie.includes("tz=America/Los_Angeles") && !document.querySelector("header dialog[open]")`)
	check(t, "saving stores the zone and closes the dialog", ok, " ", b.str(`document.cookie`))

	b.open("/formats")
	check(t, "pages format in the chosen zone and clock", b.waitJS(5*time.Second, `document.body.textContent.includes("08:30:45 Pacific Daylight Time")`))

	b.open("/")
	b.waitJS(5*time.Second, `document.readyState === "complete" && typeof htmx === "object"`)
	b.clickText("button", "Show server info")
	ok = b.waitJS(5*time.Second, `(document.querySelector("#server-info time")?.textContent ?? "").includes("PDT")`)
	check(t, "a chosen zone beats the browser's", ok, " ", b.str(`document.querySelector("#server-info time")?.textContent ?? ""`))

	b.noProblems()
}
