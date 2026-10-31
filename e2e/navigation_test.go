package e2e

import (
	"fmt"
	"testing"
	"time"
)

// pageShape is what makes a page that page: its title, main's elements and its headings.
const pageShape = `(() => {
	const m = document.querySelector("main");
	if (!m) return "";
	return [document.title, [...m.children].map((e) => e.tagName).join(","),
		[...m.querySelectorAll("h1,h2,h3")].map((h) => h.textContent.trim()).join(" / ")].join(" | ");
})()`

// testBoostedNavigation: a click in the nav gives the page a full load gives (#168: a re-minified htmx kept the old
// page's text, so Live → Map still said "Live video"), and leaving the map logs nothing.
func testBoostedNavigation(t *testing.T) {
	b := newTab(t, "navigation")
	for _, hop := range [][2]string{{"/live", "/map"}, {"/", "/formats"}, {"/map", "/about"}} {
		from, to := hop[0], hop[1]
		b.open(to)
		want := b.str(pageShape)
		b.open(from)
		b.eval(fmt.Sprintf(`document.querySelector('header nav a[href=%q]').click()`, to), nil)
		same := b.waitJS(10*time.Second, fmt.Sprintf(`location.pathname === %q && %s === %q`, to, pageShape, want))
		check(t, from+" → "+to, same, b.str(pageShape))
	}
	b.noProblems()
}
