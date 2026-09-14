package e2e

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

// TestNoOverflow: translated text (German, Russian, French and Portuguese run long; Arabic is RTL; Hindi and Japanese
// use other scripts) never makes a page scroll sideways, on a phone and on a desktop. Wide tables scroll inside their
// own container, which is allowed.
func TestNoOverflow(t *testing.T) {
	b := newTab(t, "browser")
	for _, size := range [][2]int64{{390, 844}, {1100, 1000}} {
		b.run(chromedp.EmulateViewport(size[0], size[1]))
		var bad []string
		for _, prefix := range []string{"", "/de", "/ru", "/fr", "/pt-pt", "/ar", "/hi", "/ja"} {
			for _, page := range []string{"/", "/formats", "/board?topic=" + topic("overflow"), "/about"} {
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

// TestArabicDigitsAndMirroring: /ar/formats is right-to-left and shows the native Arabic-Indic digits row next to
// CLDR's default Latin digits.
func TestArabicDigitsAndMirroring(t *testing.T) {
	b := newTab(t, "browser")
	b.open("/ar/formats")
	check(t, "Arabic page is rtl", b.is(`document.documentElement.dir === "rtl" && getComputedStyle(document.querySelector("main")).direction === "rtl"`))
	check(t, "native Arabic-Indic digits (numberingSystem arab)", b.is(`document.body.textContent.includes("١٬٢٣٤٬٥٦٧٫٨٩١")`))
	check(t, "CLDR default digits for ar are Latin", b.is(`document.body.textContent.includes("1,234,567.891")`))
	check(t, "table results align to the logical end", b.is(`(() => {
		const cell = document.querySelector("tbody td:last-child");
		const row = cell.closest("tr").getBoundingClientRect(), r = cell.getBoundingClientRect();
		return getComputedStyle(cell).textAlign === "end" && r.left - row.left < 5;
	})()`))
	b.noProblems()
}

// TestLocaleSwitchKeepsPreferences: the time zone chosen in one language applies after switching language, and back.
func TestLocaleSwitchKeepsPreferences(t *testing.T) {
	b := newTab(t, "browser")
	b.open("/de/formats")
	b.waitJS(5*time.Second, `document.readyState === "complete" && typeof htmx === "object"`)
	b.eval(`document.querySelector('header button[hx-get$="/fragments/preferences"]').click()`, nil)
	check(t, "German preferences dialog loads", b.waitJS(5*time.Second, `document.querySelectorAll("#pref-tz option").length > 400`))
	b.eval(`(() => {
		document.getElementById("pref-tz").value = "Asia/Calcutta";
		document.querySelector("#preferences button[type=submit]").click();
	})()`, nil)
	check(t, "zone saved from the German page", b.waitJS(8*time.Second, `document.cookie.includes("tz=Asia/Calcutta") && location.pathname === "/de/formats"`))

	b.eval(`document.querySelector('#languages a[hreflang="ar"]').click()`, nil)
	ok := b.waitJS(8*time.Second, `location.pathname === "/ar/formats" && document.documentElement.lang === "ar" && document.body.textContent.includes("Asia/Calcutta")`)
	check(t, "switching to Arabic keeps the zone", ok)
	b.waitJS(5*time.Second, `document.readyState === "complete" && typeof htmx === "object"`)
	b.eval(`document.querySelector('header button[hx-get$="/fragments/preferences"]').click()`, nil)
	check(t, "Arabic dialog shows the chosen zone", b.waitJS(5*time.Second, `document.getElementById("pref-tz")?.value === "Asia/Calcutta"`))

	b.eval(`document.querySelector('#languages a[hreflang="en"]').click()`, nil)
	b.waitJS(8*time.Second, `location.pathname === "/" && document.documentElement.lang === "en"`)
	b.open("/formats")
	check(t, "back in English, still the chosen zone", b.is(`document.body.textContent.includes("time zone Asia/Calcutta")`))
	b.noProblems()
}

// TestMixedLocalePush: one topic, a German and an Arabic browser. A write in either pushes each browser the board in
// its own language (the Room keeps a fragment per locale), and relative times tick in the page's language.
func TestMixedLocalePush(t *testing.T) {
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

	// Relative time ticks in the page's language: pretend the note is 5 minutes old.
	tick := de.str(`(() => {
		const el = document.querySelector("#board time[data-relative-time]");
		el.setAttribute("datetime", new Date(Date.now() - 5 * 60 * 1000).toISOString());
		relativeTime.update();
		return el.textContent;
	})()`)
	check(t, "German relative time ticks", tick == "vor 5 Minuten", " ", tick)
	tick = ar.str(`(() => {
		const el = document.querySelector("#board time[data-relative-time]");
		el.setAttribute("datetime", new Date(Date.now() - 5 * 60 * 1000).toISOString());
		relativeTime.update();
		return el.textContent;
	})()`)
	check(t, "Arabic relative time ticks", tick == "قبل 5 دقائق", " ", tick)

	// Calibration: server text this browser can't produce means its data differs, so the script leaves the page alone.
	guarded := de.str(`(() => {
		const fake = document.createElement("time");
		fake.setAttribute("datetime", new Date(Date.now() - 3 * 60 * 1000).toISOString());
		fake.setAttribute("data-relative-time", "");
		fake.textContent = "wording this browser never produces";
		document.body.append(fake);
		relativeTime.update();
		return fake.textContent;
	})()`)
	check(t, "script keeps server wording the browser can't reproduce", guarded == "wording this browser never produces", " ", guarded)
	de.noProblems()
	ar.noProblems()
}
