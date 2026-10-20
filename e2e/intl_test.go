package e2e

import (
	"testing"
	"time"
)

// TestIntlKinds: every data-intl kind on /de/formats is formatted by Chrome's Intl in German (none left at its
// fallback), and a few results are the German wording.
func TestIntlKinds(t *testing.T) {
	b := newTab(t, "browser")
	b.open("/de/formats")
	b.waitJS(5*time.Second, `document.readyState === "complete"`)
	left := b.str(`(() => {
		const stuck = [];
		for (const el of document.querySelectorAll("[data-intl]")) {
			if (!el.hasAttribute("data-intl-done")) stuck.push(el.dataset.intl + ":" + el.dataset.value);
		}
		return stuck.join(", ");
	})()`)
	check(t, "every kind is formatted", left == "", " unformatted: ", left)
	for kind, want := range map[string]string{
		"display-name": "Japan",
		"number-range": "3–5 kg",
		"plural":       "one",
		"collate":      "10 · 9",
		"segment":      "Hello | world",
		"locale":       "de-Latn-DE",
		"supported":    "buddhist",
	} {
		js := `[...document.querySelectorAll('[data-intl="` + kind + `"]')].some(e => e.textContent.includes(` + "`" + want + "`" + `))`
		check(t, kind+" in German", b.is(js), " want ", want)
	}
	b.noProblems()
}
