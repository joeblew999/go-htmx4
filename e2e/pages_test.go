package e2e

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

// TestPages: the home page's gsxui components driven by htmx 4 and hx-live, boosted navigation, the theme.
func TestPages(t *testing.T) {
	b := newTab(t, "browser")
	b.run(chromedp.Navigate(base+"/"), chromedp.WaitReady("#greeting", chromedp.ByQuery))
	marker := b.num(`window.__marker = Math.floor(Math.random() * 1e9)`)

	check(t, "htmx, hx-live and the toaster loaded", b.is(`typeof htmx === "object" && !!document.querySelector("[hx-live]") && !!document.getElementById("gsxui-toaster")`))
	check(t, "web fonts loaded from /assets/", b.is(`document.fonts.ready.then(() => [...document.fonts].some(f => f.status === "loaded"))`))

	// hx-live: counter, menu, toggle, filter.
	b.clickText("button", "Increment")
	b.clickText("button", "Increment")
	check(t, "hx-live counter", b.waitJS(2*time.Second, `document.querySelector("[data-count]").dataset.count === "2"`))
	b.clickText("button", "Menu")
	check(t, "hx-live menu opens", b.waitJS(2*time.Second, `!document.querySelector('[data-open] > div').hidden`))
	b.eval(`document.querySelector("h1").dispatchEvent(new MouseEvent("click", {bubbles: true}))`, nil)
	check(t, "hx-live menu closes on click outside", b.waitJS(2*time.Second, `document.querySelector('[data-open] > div').hidden`))
	b.clickText("button", "Bold")
	check(t, "hx-live aria.pressed + :class", b.waitJS(2*time.Second, `(() => { const e = [...document.querySelectorAll("button")].find(e => e.textContent.trim() === "Bold"); return e.getAttribute("aria-pressed") === "true" && e.classList.contains("font-bold"); })()`))
	b.run(chromedp.SendKeys(`input[aria-label="Filter components"]`, "dia", chromedp.ByQuery))
	check(t, "hx-live filter", b.waitJS(2*time.Second, `[...document.querySelectorAll("[hx-live] li")].filter(li => !li.hidden).map(li => li.textContent.trim()).join() === "dialog"`))

	// Server round-trip: fragment + out-of-band toast, escaped.
	b.run(chromedp.SendKeys("#name", "Ada <b>", chromedp.ByID))
	b.clickText("button", "Say hello")
	ok := b.waitJS(5*time.Second, `document.getElementById("greeting").textContent.includes("Hello, Ada <b>.")`)
	check(t, "POST /greet swaps #greeting, escaped", ok && b.num(`document.querySelectorAll("#greeting b").length`) == 0)
	check(t, "OOB success toast", b.waitJS(3*time.Second, `[...document.querySelectorAll("#gsxui-toaster [data-gsxui-slot-toast]")].some(t => t.textContent.includes("Greeting rendered"))`))
	b.clickText("button", "Clear")
	check(t, "DELETE /greet via hx-action + hx-method", b.waitJS(5*time.Second, `document.getElementById("greeting").textContent.includes("Cleared.")`))

	// Dialog body via hx-get; tabs load lazily.
	b.clickText("button", "Show server info")
	check(t, "dialog opens", b.waitJS(3*time.Second, `(() => { const d = document.querySelector("dialog[data-gsxui-slot-dialog-content]"); return d.open && d.dataset.state === "open"; })()`))
	check(t, "dialog body from /fragments/server-info", b.waitJS(5*time.Second, `document.getElementById("server-info").textContent.includes("Uptime")`),
		" ", strings.Join(strings.Fields(b.str(`document.getElementById("server-info").textContent`)), " "))
	b.run(chromedp.KeyEvent("\x1b"))
	check(t, "dialog closes on Escape", b.waitJS(3*time.Second, `!document.querySelector("dialog[data-gsxui-slot-dialog-content]").open`))
	b.clickText("[role=tab]", "Stats")
	check(t, "Stats tab loads lazily", b.waitJS(5*time.Second, `!!document.querySelector("#tab-stats ul")`))

	// Boosted navigation with morph: same document; Back restores it.
	b.clickText("nav a", "About")
	ok = b.waitJS(5*time.Second, `location.pathname === "/about" && document.body.textContent.includes("About go-htmx4")`)
	check(t, "boosted nav to /about (no reload)", ok && b.is(fmt.Sprintf(`window.__marker === %d`, marker)))
	b.eval(`history.back()`, nil)
	ok = b.waitJS(5*time.Second, `location.pathname === "/" && !!document.getElementById("greeting")`)
	check(t, "Back to / in the same document", ok && b.is(fmt.Sprintf(`window.__marker === %d`, marker)))

	// Theme toggle (gsxui site pattern): flips, survives boosted nav and reload.
	dark0 := b.is(`document.documentElement.classList.contains("dark")`)
	bg0 := b.str(`getComputedStyle(document.body).backgroundColor`)
	// Element click, not a mouse click at its coordinates: the toasts from above may still cover the header corner.
	b.eval(`document.querySelector("[data-site-theme-toggle]").click()`, nil)
	ok = b.waitJS(2*time.Second, fmt.Sprintf(`document.documentElement.classList.contains("dark") === %v`, !dark0))
	check(t, "theme toggle flips dark and colours", ok && b.str(`getComputedStyle(document.body).backgroundColor`) != bg0)
	b.clickText("nav a", "About")
	b.waitJS(5*time.Second, `location.pathname === "/about"`)
	check(t, "theme kept across boosted nav", b.is(fmt.Sprintf(`document.documentElement.classList.contains("dark") === %v`, !dark0)))
	b.run(chromedp.Reload(), chromedp.WaitReady("main", chromedp.ByQuery))
	check(t, "theme persists across reload", b.is(fmt.Sprintf(`document.documentElement.classList.contains("dark") === %v`, !dark0)))

	b.noProblems()
}
