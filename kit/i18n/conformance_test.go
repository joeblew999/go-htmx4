package i18n_test

import (
	"flag"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/joeblew999/go-htmx4/kit/i18n/cldr"
	"github.com/joeblew999/go-htmx4/kit/i18n/intltest"
)

var dump = flag.String("dump", "", "write every conformance mismatch to this file (id, want, got)")

// known lists accepted differences from the Intl oracle: case ID prefix → reason. Every entry needs a
// reason a reviewer can check.
var known = map[string]string{
	// DateTimeFormat: Chrome's ICU 78.2 data (chromium/deps/icu source/data/locales) predates fixes in
	// CLDR 48.2, which kit/i18n is generated from. Compare the "Hv" and "GyMMMEd" entries there with
	// cldr-json 48.2.1 ca-gregorian.json.
	"es/datetime/more-yMMMMdv/":          "ICU 78.2 es availableFormats Hv is \"H 'h' v\"; CLDR 48.2 has \"H v\"",
	"he/datetime/more-yMMMMdv/":          "ICU 78.2 he Hv falls back to root \"HH'h' v\"; CLDR 48.2 he has \"H v\"",
	"pt-BR/datetime/more-yMMMMdv/":       "ICU 78.2 pt Hv is \"HH'h', v\"; CLDR 48.2 has \"HH v\"",
	"pt-PT/datetime/more-yMMMMdv/":       "ICU 78.2 pt Hv is \"HH'h', v\"; CLDR 48.2 has \"HH v\"",
	"ru/datetime/more-yMMMMdv/":          "ICU 78.2 ru Hv is \"HH 'ч'. v\"; CLDR 48.2 has \"HH v\"",
	"de/datetime/morerange-dateFull/era": "ICU 78.2 de intervalFormats GyMMMEd/G has the CLDR 48.0 typo \"E E, d. MMM y G\" (\"Freitag Freitag\"); CLDR 48.2 fixed it",
	// Only the Gregorian calendar is implemented (plan Phase 9): -u-ca/calendar iso8601 formats as gregory.
	"ar/datetime/resolved-ca-iso8601/":      noISOCalendar,
	"de/datetime/resolved-ca-iso8601/":      noISOCalendar,
	"en/datetime/resolved-ca-iso8601/":      noISOCalendar,
	"en-IN/datetime/resolved-ca-iso8601/":   noISOCalendar,
	"es/datetime/resolved-ca-iso8601/":      noISOCalendar,
	"fr/datetime/resolved-ca-iso8601/":      noISOCalendar,
	"he/datetime/resolved-ca-iso8601/":      noISOCalendar,
	"hi/datetime/resolved-ca-iso8601/":      noISOCalendar,
	"ja/datetime/resolved-ca-iso8601/":      noISOCalendar,
	"pt-BR/datetime/resolved-ca-iso8601/":   noISOCalendar,
	"pt-PT/datetime/resolved-ca-iso8601/":   noISOCalendar,
	"ru/datetime/resolved-ca-iso8601/":      noISOCalendar,
	"zh-Hans/datetime/resolved-ca-iso8601/": noISOCalendar,
	"zh-Hant/datetime/resolved-ca-iso8601/": noISOCalendar,
}

const noISOCalendar = "calendar iso8601 is not implemented (plan Phase 9): kit/i18n formats and reports gregory, Intl uses CLDR's ISO 8601 calendar patterns"

// TestConformance compares kit/i18n with what Intl (workerd = Chrome's V8/ICU) produced for every case
// in kit/i18n/intltest. Regenerate the golden file with `go run ./cmd/intloracle`.
func TestConformance(t *testing.T) {
	g, err := intltest.Load("testdata/golden/workerd.json")
	if err != nil {
		t.Fatal(err)
	}
	type miss struct{ id, want, got string }
	byArea := map[string][]miss{}
	total, pass := 0, 0
	for _, c := range intltest.All() {
		want, ok := g.Results[c.ID]
		if !ok {
			t.Fatalf("case %s has no golden result: run go run ./cmd/intloracle", c.ID)
		}
		if c.API == "Locale" {
			want = intltest.LocaleExpected(want)
		}
		got, err := intltest.Eval(cldr.Data, c)
		if err != nil {
			got = "error: " + err.Error()
		}
		total++
		if got == want || knownReason(c.ID) != "" {
			pass++
			continue
		}
		parts := strings.Split(c.ID, "/")
		area := parts[0]
		if len(parts) > 2 {
			area = parts[1] + "/" + strings.SplitN(parts[2], "-", 2)[0]
		}
		byArea[area] = append(byArea[area], miss{c.ID, want, got})
	}
	t.Logf("conformance: %d/%d cases match Intl (%s)", pass, total, g.Runtime)
	if *dump != "" {
		var b strings.Builder
		for _, ms := range byArea {
			for _, m := range ms {
				fmt.Fprintf(&b, "%s\t%q\t%q\n", m.id, m.want, m.got)
			}
		}
		if err := os.WriteFile(*dump, []byte(b.String()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	areas := make([]string, 0, len(byArea))
	for a := range byArea {
		areas = append(areas, a)
	}
	slices.Sort(areas)
	for _, a := range areas {
		ms := byArea[a]
		var b strings.Builder
		for i, m := range ms {
			if i == 8 {
				fmt.Fprintf(&b, "\n    … %d more", len(ms)-i)
				break
			}
			fmt.Fprintf(&b, "\n    %s\n      want %q\n      got  %q", m.id, m.want, m.got)
		}
		t.Errorf("%s: %d mismatches%s", a, len(ms), b.String())
	}
}

func knownReason(id string) string {
	for prefix, reason := range known {
		if strings.HasPrefix(id, prefix) {
			return reason
		}
	}
	return ""
}
