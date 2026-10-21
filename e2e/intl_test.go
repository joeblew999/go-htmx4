package e2e

import (
	"testing"
	"time"
)

// testIntlKinds: every Intl call on /de/formats is redone by Chrome's Intl in German (none left at the server's
// text), and a few results are the German wording.
func testIntlKinds(t *testing.T) {
	b := newTab(t, "browser")
	b.open("/de/formats")
	b.waitJS(5*time.Second, `document.readyState === "complete"`)
	left := b.str(`(() => {
		const stuck = [];
		for (const el of document.querySelectorAll("[data-intl]")) {
			if (!el.hasAttribute("data-intl-done")) stuck.push(el.dataset.intl + "." + (el.dataset.method || "format"));
		}
		return stuck.join(", ");
	})()`)
	check(t, "every kind is formatted", left == "", " unformatted: ", left)
	for call, want := range map[string]string{
		`[data-intl="DisplayNames"]`:                              "Japan",
		`[data-intl="NumberFormat"][data-method="formatRange"]`:   "3–5 kg",
		`[data-intl="PluralRules"][data-method="select"]`:         "one",
		`[data-intl="Collator"][data-method="compare"]`:           "-1",
		`[data-intl="Segmenter"][data-method="segment"]`:          `"segment":"Smith arrived. "`,
		`[data-intl="Locale"][data-method="maximize"]`:            "de-Latn-DE",
		`[data-intl="Intl"][data-method="supportedValuesOf"]`:     "buddhist",
		`[data-intl="DateTimeFormat"][data-method="formatRange"]`: "4. Juli – 20. August",
	} {
		js := `[...document.querySelectorAll('` + call + `')].some(e => e.textContent.includes(` + "`" + want + "`" + `))`
		check(t, call+" in German", b.is(js), " want ", want)
	}
	b.noProblems()
}
