package e2e

import (
	"strings"
	"testing"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
)

// TestViewerTimeZone: dates follow the viewer. Local workerd has no Cloudflare cf, so the server's automatic zone
// is UTC; a browser in Tokyo re-formats those times itself (static/relative-time.js). Choosing a zone in the
// header dialog stores it, the page re-renders in it, and the browser leaves chosen times alone.
func TestViewerTimeZone(t *testing.T) {
	b := newTab(t, "browser")
	b.run(emulation.SetTimezoneOverride("Asia/Tokyo"))
	b.open("/")
	b.waitJS(5*time.Second, `document.readyState === "complete" && typeof htmx === "object"`)

	// Automatic zone: the server renders UTC, the browser corrects it to Tokyo.
	b.clickText("button", "Show server info")
	ok := b.waitJS(5*time.Second, `document.querySelector("#server-info time")?.getAttribute("data-time-zone") === "Asia/Tokyo"`)
	text := b.str(`document.querySelector("#server-info time")?.textContent ?? ""`)
	check(t, "browser re-formats an automatic time in its own zone", ok && strings.Contains(text, "GMT+9"), " ", text)
	b.run(chromedp.KeyEvent("\x1b"))

	// Calibration: the browser corrects a time only when it reproduces the server's text in the server's zone.
	guard := b.str(`(() => {
		const opts = '{"dateStyle":"full","timeStyle":"long"}', at = "2026-07-04T15:30:45Z";
		const make = (text) => {
			const el = document.createElement("time");
			el.setAttribute("datetime", at);
			el.setAttribute("data-local-time", opts);
			el.setAttribute("data-time-zone", "UTC");
			el.textContent = text;
			document.body.append(el);
			return el;
		};
		const same = make(new Intl.DateTimeFormat("en", { dateStyle: "full", timeStyle: "long", timeZone: "UTC" }).format(Date.parse(at)));
		const other = make("Saturday the 4th of July, 3:30 in the afternoon");
		relativeTime.update();
		return JSON.stringify({ same: same.textContent, other: other.textContent });
	})()`)
	check(t, "calibrated: matching server text is corrected to Tokyo, foreign wording left alone",
		strings.Contains(guard, `"same":"Sunday, July 5, 2026 at 12:30:45 AM GMT+9"`) && strings.Contains(guard, `"other":"Saturday the 4th of July, 3:30 in the afternoon"`), " ", guard)

	// Choose America/Los_Angeles and a 24-hour clock in the header dialog.
	b.eval(`document.querySelector('header button[hx-get$="/fragments/preferences"]').click()`, nil)
	check(t, "preferences dialog loads the zone list", b.waitJS(5*time.Second, `document.querySelectorAll("#pref-tz option").length > 400`))
	b.eval(`(() => {
		document.getElementById("pref-tz").value = "America/Los_Angeles";
		document.getElementById("pref-hc").value = "h23";
		document.querySelector("#preferences button[type=submit]").click();
	})()`, nil)
	ok = b.waitJS(8*time.Second, `document.cookie.includes("tz=America/Los_Angeles") && !document.querySelector("header dialog[open]")`)
	check(t, "saving stores the zone and closes the dialog", ok, " ", b.str(`document.cookie`))

	b.open("/formats")
	body := b.str(`document.body.textContent`)
	check(t, "pages render in the chosen zone and clock", strings.Contains(body, "time zone America/Los_Angeles") && strings.Contains(body, "08:30 Pacific Daylight Time"))

	b.open("/")
	b.waitJS(5*time.Second, `document.readyState === "complete" && typeof htmx === "object"`)
	b.clickText("button", "Show server info")
	b.waitJS(5*time.Second, `!!document.querySelector("#server-info time")`)
	time.Sleep(300 * time.Millisecond) // give the browser script a chance to (wrongly) re-format
	text = b.str(`document.querySelector("#server-info time").textContent`)
	check(t, "a chosen zone isn't replaced by the browser's", strings.Contains(text, "PDT") && !b.is(`document.querySelector("#server-info time").hasAttribute("data-local-time")`), " ", text)

	b.noProblems()
}
