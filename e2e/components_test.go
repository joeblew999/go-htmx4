package e2e

import (
	"testing"
	"time"
)

// testComponents: /de/components in Chrome. Every Intl call on the page is redone by the browser, the gsxui
// components that write words (the calendar) write German ones from Intl, and the demos that change a format change
// it through the browser's Intl.
func testComponents(t *testing.T) {
	b := newTab(t, "browser")
	b.open("/de/components")
	b.waitJS(5*time.Second, `document.readyState === "complete" && typeof htmx === "object"`)

	left := b.str(`[...document.querySelectorAll("[data-intl]")].filter((el) => !el.hasAttribute("data-intl-done")).map((el) => el.dataset.intl).join(", ")`)
	check(t, "every Intl call is redone", left == "", " not redone: ", left)

	// The calendar: German words from Intl, the week starting on Monday, and calendar.js navigating in German.
	cal := `document.querySelector("[data-gsxui-slot-popover-content] [data-gsxui-slot-calendar-caption]")`
	check(t, "calendar caption in German", b.is(cal+`.textContent.trim() === "Juli 2026"`), " ", b.str(cal+`.textContent`))
	check(t, "the week starts on Monday", b.is(`document.querySelector("[data-gsxui-slot-popover-content] [data-gsxui-slot-calendar-weekday]").textContent.trim() === "M"`))
	b.eval(`document.querySelector("[data-gsxui-slot-popover-content] [data-gsxui-slot-calendar-next]").click()`, nil)
	check(t, "next month in German", b.waitJS(3*time.Second, cal+`.textContent.trim() === "August 2026"`), " ", b.str(cal+`.textContent`))

	// Formats changed in the page, redone by the browser's Intl.
	b.eval(`document.querySelector('[data-value="disjunction"]').click()`, nil)
	check(t, "the list joined with oder", b.waitJS(3*time.Second, `document.getElementById("list-result").textContent.includes(" oder ")`),
		" ", b.str(`document.getElementById("list-result").textContent`))
	b.eval(`(() => { const s = document.getElementById("components-slider"); s.value = "7"; s.dispatchEvent(new Event("input", {bubbles: true})) })()`, nil)
	check(t, "the slider in German units", b.waitJS(3*time.Second, `document.getElementById("slider-result").textContent.trim() === "7 Kilometer"`),
		" ", b.str(`document.getElementById("slider-result").textContent`))
	// The calendar system is chosen as a reader would: open the select, pick the item (select.js keeps the form's
	// hidden select in step with it).
	b.eval(`(() => { const root = document.querySelector('[data-gsxui-slot-select]:has(select[name="calendar"])'); root.querySelector("[data-gsxui-slot-select-trigger]").click(); root.querySelector('[data-value="hebrew"]').click() })()`, nil)
	check(t, "the date in the Hebrew calendar", b.waitJS(3*time.Second, `document.getElementById("calendar-result").textContent.includes("Tammus")`),
		" ", b.str(`document.getElementById("calendar-result").textContent`))
	b.eval(`(() => { const i = document.getElementById("components-amount"); i.value = "1.234,5"; i.dispatchEvent(new Event("input", {bubbles: true})) })()`, nil)
	check(t, "the amount read back by Intl", b.waitJS(3*time.Second, `document.querySelector("#amount-result code").textContent === "1234.5"`))

	// Counted on the Worker by Intl.Segmenter.
	b.eval(`(() => { const t = document.getElementById("components-text"); t.value = "Hallo Welt. Zwei 👨‍👩‍👧!"; t.dispatchEvent(new Event("input", {bubbles: true})) })()`, nil)
	check(t, "graphemes, words and sentences counted", b.waitJS(5*time.Second, `/^19\D.*\b3\D.*\b2\D/.test(document.getElementById("count-result").textContent.trim())`),
		" ", b.str(`document.getElementById("count-result").textContent`))

	// The week, loaded behind a skeleton, starts on Monday; initials are graphemes.
	check(t, "the week loads and starts on Monday", b.waitJS(5*time.Second, `document.body.textContent.includes("Montag, 29. Juni")`))
	check(t, "initials", b.is(`[...document.querySelectorAll("[data-gsxui-slot-avatar-fallback]")].some((el) => el.textContent.trim() === "ZÅ")`))

	// The sidebar's own document, in German.
	b.eval(`document.querySelector("iframe").scrollIntoView()`, nil) // it loads lazily
	check(t, "the sidebar frame is German", b.waitJS(5*time.Second, `document.querySelector("iframe")?.contentDocument?.documentElement.lang === "de"`))
	// The header's theme toggle reaches the frame too, both ways (gsxui's iframe sync, views/theme.gsx).
	frameDark := `document.querySelector("iframe").contentDocument.documentElement.classList.contains("dark")`
	pageDark := b.is(`document.documentElement.classList.contains("dark")`)
	b.eval(`document.querySelector("[data-site-theme-toggle]").click()`, nil)
	check(t, "the theme toggle reaches the frame", b.waitJS(3*time.Second, frameDark+` === `+map[bool]string{true: "false", false: "true"}[pageDark]))
	b.eval(`document.querySelector("[data-site-theme-toggle]").click()`, nil)
	check(t, "and back", b.waitJS(3*time.Second, frameDark+` === `+map[bool]string{true: "true", false: "false"}[pageDark]))
	b.noProblems()
}
