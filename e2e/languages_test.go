package e2e

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

// testNoOverflow: translated text (German, Russian, French and Portuguese run long; Arabic is RTL; Hindi and Japanese
// use other scripts) never makes a page scroll sideways, on a phone and on a desktop. Wide tables scroll inside their
// own container, which is allowed.
func testNoOverflow(t *testing.T) {
	b := newTab(t, "browser")
	for _, size := range [][2]int64{{390, 844}, {1100, 1000}} {
		b.run(chromedp.EmulateViewport(size[0], size[1]))
		var bad []string
		for _, prefix := range []string{"", "/de", "/ru", "/fr", "/pt-pt", "/ar", "/hi", "/ja"} {
			for _, page := range []string{"/", "/formats", "/components", "/board?topic=" + topic("overflow"), "/about"} {
				b.open(prefix + page)
				over := b.str(`(() => {
					const w = document.documentElement.clientWidth;
					if (document.documentElement.scrollWidth <= w + 1) return "";
					const out = [];
					for (const el of document.querySelectorAll("body *")) {
						const r = el.getBoundingClientRect();
						if ((r.right > w + 1 || r.left < -1) && !el.closest("[data-gsxui-slot-table-container]")) {
							out.push(el.tagName.toLowerCase() + (el.id ? "#" + el.id : "") + " " + (el.textContent || "").trim().slice(0, 40));
							if (out.length === 3) break;
						}
					}
					return document.documentElement.scrollWidth + "px: " + out.join(" | ");
				})()`)
				if over != "" {
					bad = append(bad, fmt.Sprintf("%s%s at %dpx (%s)", prefix, page, size[0], over))
				}
			}
		}
		check(t, fmt.Sprintf("no horizontal overflow at %dpx in 8 locales × 4 pages", size[0]), len(bad) == 0, strings.Join(bad, "; "))
	}
	b.noProblems()
}

// testArabicDigitsAndMirroring: /ar/formats is right-to-left, and the browser's Intl shows the native
// Arabic-Indic digits row next to CLDR's default Latin digits.
func testArabicDigitsAndMirroring(t *testing.T) {
	b := newTab(t, "browser")
	b.open("/ar/formats")
	check(t, "Arabic page is rtl", b.is(`document.documentElement.dir === "rtl" && getComputedStyle(document.querySelector("main")).direction === "rtl"`))
	check(t, "native Arabic-Indic digits (numberingSystem arab)", b.waitJS(5*time.Second, `document.body.textContent.includes("١٬٢٣٤٬٥٦٧٫٨٩١")`))
	check(t, "CLDR default digits for ar are Latin", b.is(`document.body.textContent.includes("1,234,567.891")`))
	check(t, "examples mirror: labels start at the right edge, results to their left", b.is(`(() => {
		const row = document.querySelector("#numberformat li"), dt = row.querySelector("dt"), dd = row.querySelector("dd");
		const r = row.getBoundingClientRect(), t = dt.getBoundingClientRect(), d = dd.getBoundingClientRect();
		return getComputedStyle(row).direction === "rtl" && r.right - t.right < 40 && d.right <= t.left + 1;
	})()`))
	b.noProblems()
}

// testLocaleSwitchKeepsPreferences: the time zone chosen in one language applies after switching language, and back.
func testLocaleSwitchKeepsPreferences(t *testing.T) {
	b := newTab(t, "browser")
	b.open("/de/formats")
	b.waitJS(5*time.Second, `document.readyState === "complete" && typeof htmx === "object"`)
	b.eval(`document.querySelector('header button[hx-get$="/fragments/preferences"]').click()`, nil)
	check(t, "German preferences dialog loads", b.waitJS(5*time.Second, `document.querySelectorAll("#pref-tz option").length > 300`))
	b.eval(`(() => {
		document.getElementById("pref-tz").value = "Asia/Tokyo";
		document.querySelector("#preferences button[type=submit]").click();
	})()`, nil)
	check(t, "zone saved from the German page", b.waitJS(8*time.Second, `document.cookie.includes("tz=Asia/Tokyo") && location.pathname === "/de/formats"`))

	b.eval(`document.querySelector('#languages a[hreflang="ar"]').click()`, nil)
	ok := b.waitJS(8*time.Second, `location.pathname === "/ar/formats" && document.documentElement.lang === "ar" && document.documentElement.dataset.tz === "Asia/Tokyo"`)
	check(t, "switching to Arabic keeps the zone", ok)
	b.waitJS(5*time.Second, `document.readyState === "complete" && typeof htmx === "object"`)
	b.eval(`document.querySelector('header button[hx-get$="/fragments/preferences"]').click()`, nil)
	check(t, "Arabic dialog shows the chosen zone", b.waitJS(5*time.Second, `document.getElementById("pref-tz")?.value === "Asia/Tokyo"`))

	b.eval(`document.querySelector('#languages a[hreflang="en"]').click()`, nil)
	b.waitJS(8*time.Second, `location.pathname === "/" && document.documentElement.lang === "en"`)
	b.open("/formats")
	check(t, "back in English, still the chosen zone", b.is(`document.documentElement.dataset.tz === "Asia/Tokyo" && document.body.textContent.includes("time zone Asia/Tokyo")`))
	b.noProblems()
}

// testMixedLocalePush: one topic, a German and an Arabic browser. A write in either pushes each browser the board in
// its own language (the Room keeps a fragment per locale), and relative times tick in the page's language.
func testMixedLocalePush(t *testing.T) {
	top := topic("e2e-locales")
	de, ar := newTab(t, "de"), newTab(t, "ar")
	de.run(chromedp.Navigate(base+"/de/board?topic="+top), chromedp.WaitVisible("#board", chromedp.ByQuery))
	ar.run(chromedp.Navigate(base+"/ar/board?topic="+top), chromedp.WaitVisible("#board", chromedp.ByQuery))
	check(t, "both connected (presence 2)", waitFor(8*time.Second, func() bool {
		return strings.HasPrefix(de.presence(), "2") && strings.HasPrefix(ar.presence(), "2")
	}), fmt.Sprintf("(de %q, ar %q)", de.presence(), ar.presence()))

	ar.run(chromedp.SendKeys(`input[name="body"]`, "مرحبا من الرياض", chromedp.ByQuery), chromedp.Click(`#board ~ * form button, form[hx-post*="/board/note"] button`, chromedp.ByQuery))
	pushed := waitFor(6*time.Second, func() bool {
		return strings.Contains(de.str(`document.querySelector("#board")?.textContent ?? ""`), "مرحبا من الرياض")
	})
	check(t, "Arabic note pushed to the German browser", pushed)
	check(t, "German browser got the German fragment", de.is(`document.getElementById("board").lang === "de" && document.getElementById("board").textContent.includes("Version")`),
		de.str(`document.getElementById("board").lang`))
	check(t, "Arabic browser has the Arabic fragment", waitFor(5*time.Second, func() bool {
		return ar.is(`document.getElementById("board").lang === "ar" && document.getElementById("board").textContent.includes("الإصدار") && document.getElementById("board").textContent.includes("مرحبا من الرياض")`)
	}))
	check(t, "user text isolated with dir=auto", de.is(`[...document.querySelectorAll("#board [dir=auto]")].some(e => e.textContent.includes("مرحبا"))`))

	// Relative times are worded by each browser in its page's language, and follow the instant: pretend the note is
	// 5 minutes old (intl-browser.mjs redoes it on the attribute change).
	const age = `document.querySelector("#board time[data-intl=since]").setAttribute("datetime", new Date(Date.now() - 5 * 60 * 1000).toISOString())`
	de.eval(age, nil)
	ar.eval(age, nil)
	rel := `(document.querySelector("#board time[data-intl=since]")?.textContent ?? "")`
	check(t, "German relative time", de.waitJS(5*time.Second, rel+` === "vor 5 Minuten"`), " ", de.str(rel))
	check(t, "Arabic relative time", ar.waitJS(5*time.Second, rel+` === "قبل 5 دقائق"`), " ", ar.str(rel))
	de.noProblems()
	ar.noProblems()
}
