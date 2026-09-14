package e2e

import (
	"fmt"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

// TestLocales: locale URLs in a real browser. The language list switches with a full page load (new <html lang
// dir>), boosted navigation stays inside the locale's prefix, RTL mirrors the layout, the formats page shows
// CLDR output, and the bare "/" follows the remembered locale.
func TestLocales(t *testing.T) {
	b := newTab(t, "browser")
	b.run(chromedp.Navigate(base+"/formats"), chromedp.WaitReady("#languages", chromedp.ByQuery))
	marker := b.num(`window.__marker = Math.floor(Math.random() * 1e9)`)
	check(t, "English page", b.is(`document.documentElement.lang === "en" && document.documentElement.dir === "ltr"`))

	// Footer link → Arabic: a full load (no boost), so lang and dir change.
	b.eval(`document.querySelector('#languages a[hreflang="ar"]').click()`, nil)
	ok := b.waitJS(5*time.Second, `location.pathname === "/ar/formats" && document.documentElement.lang === "ar" && !!document.querySelector("#languages")`)
	check(t, "language link loads /ar/formats with lang=ar", ok && b.is(fmt.Sprintf(`window.__marker !== %d`, marker)))
	check(t, "dir=rtl mirrors the layout", b.is(`document.documentElement.dir === "rtl" && getComputedStyle(document.querySelector("header nav")).direction === "rtl"`))
	check(t, "brand sits on the right in RTL", b.is(`(() => { const a = document.querySelector("header nav > a").getBoundingClientRect(), n = document.querySelector("header nav").getBoundingClientRect(); return a.right > n.right - 40; })()`))
	check(t, "CLDR formatting in the page", b.is(`document.body.textContent.includes("1.2 مليون")`))
	check(t, "current language marked", b.is(`document.querySelector('#languages a[aria-current="page"]').getAttribute("hreflang") === "ar"`))

	// Boosted nav inside the locale keeps the prefix and the RTL document (once htmx has processed the page).
	b.waitJS(5*time.Second, `document.readyState === "complete" && typeof htmx === "object"`)
	marker = b.num(`window.__marker = Math.floor(Math.random() * 1e9)`)
	b.eval(`document.querySelector('header nav a[href="/ar/about"]').click()`, nil)
	ok = b.waitJS(5*time.Second, `location.pathname === "/ar/about" && !!document.body?.textContent.includes("حول go-htmx4")`)
	check(t, "boosted nav stays under /ar/ (Arabic UI text)", ok && b.is(fmt.Sprintf(`window.__marker === %d && document.documentElement.dir === "rtl"`, marker)))

	// The remembered locale: the bare / goes to /ar/.
	b.run(chromedp.Navigate(base+"/"), chromedp.WaitReady("main", chromedp.ByQuery))
	check(t, "/ follows the remembered locale", b.waitJS(5*time.Second, `location.pathname === "/ar/" && document.documentElement.lang === "ar" && !!document.querySelector("#languages")`))

	// Back to English via the list: / stays English afterwards.
	b.eval(`document.querySelector('#languages a[hreflang="en"]').click()`, nil)
	b.waitJS(5*time.Second, `location.pathname === "/" && document.documentElement.lang === "en" && !!document.querySelector("#languages")`)
	b.run(chromedp.Navigate(base+"/"), chromedp.WaitReady("main", chromedp.ByQuery))
	check(t, "choosing English sticks", b.waitJS(5*time.Second, `location.pathname === "/" && document.documentElement.lang === "en" && !!document.querySelector("#languages")`))

	b.noProblems()
}
