package i18n_test

import (
	"context"
	"flag"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/joeblew999/go-htmx4/kit/i18n/cldr"
	"github.com/joeblew999/go-htmx4/kit/i18n/cldrgen"
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

// TestGoldenReproduces (under `mise run i18n:verify`) records Intl's output again with the pinned workerd and
// requires the committed golden: the goldens are exactly what that runtime produces for the current cases.
func TestGoldenReproduces(t *testing.T) {
	if os.Getenv("I18N_VERIFY") == "" {
		t.Skip("golden reproduction: run `mise run i18n:verify` (needs workerd from mise)")
	}
	committed, err := intltest.Load("testdata/golden/workerd.json")
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := intltest.Oracle{Port: 8947}.Run(context.Background(), intltest.All())
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Runtime != committed.Runtime {
		t.Errorf("golden recorded with %q, this workerd is %q", committed.Runtime, fresh.Runtime)
	}
	diffs := 0
	for id, want := range fresh.Results {
		if got, ok := committed.Results[id]; !ok || got != want {
			if diffs++; diffs <= 20 {
				t.Errorf("%s: committed golden %q, workerd now %q", id, got, want)
			}
		}
	}
	for id := range committed.Results {
		if _, ok := fresh.Results[id]; !ok {
			if diffs++; diffs <= 20 {
				t.Errorf("%s: in the committed golden but no longer a case", id)
			}
		}
	}
	if diffs > 0 {
		t.Errorf("%d golden differences: review them, then mise run i18n:golden", diffs)
	}
}

// evidence is an upstream fact a known entry rests on: a pinned file (cldrgen.Reference) that has, or lacks, a
// piece of text. "chromium-icu" is the ICU data Intl ships (cldrgen.ChromiumICU), "cldr-json" what kit/i18n is
// generated from (cldrgen.DefaultTag).
type evidence struct{ source, file, has, lacks string }

const (
	chromiumLocales = "source/data/locales/"
	cldrGregorian   = "cldr-dates-full/main/%s/ca-gregorian.json"
)

// knownEvidence backs every data-difference entry in known with checkable upstream text; `mise run i18n:verify`
// fetches the pinned files and checks it (TestKnownEvidence). Entries for unimplemented features need none.
var knownEvidence = map[string][]evidence{
	"es/datetime/more-yMMMMdv/": {
		{source: "chromium-icu", file: chromiumLocales + "es.txt", has: `Hv{"H 'h' v"}`},
		{source: "cldr-json", file: fmt.Sprintf(cldrGregorian, "es"), has: `"Hv": "H v"`},
	},
	"he/datetime/more-yMMMMdv/": {
		{source: "chromium-icu", file: chromiumLocales + "he.txt", lacks: `Hv{"`},
		{source: "chromium-icu", file: chromiumLocales + "root.txt", has: `Hv{"HH'h' v"}`},
		{source: "cldr-json", file: fmt.Sprintf(cldrGregorian, "he"), has: `"Hv": "H v"`},
	},
	"pt-BR/datetime/more-yMMMMdv/": {
		{source: "chromium-icu", file: chromiumLocales + "pt.txt", has: `Hv{"HH'h', v"}`},
		{source: "cldr-json", file: fmt.Sprintf(cldrGregorian, "pt"), has: `"Hv": "HH v"`},
	},
	"pt-PT/datetime/more-yMMMMdv/": {
		{source: "chromium-icu", file: chromiumLocales + "pt.txt", has: `Hv{"HH'h', v"}`},
		{source: "chromium-icu", file: chromiumLocales + "pt_PT.txt", lacks: `Hv{"`},
		{source: "cldr-json", file: fmt.Sprintf(cldrGregorian, "pt-PT"), has: `"Hv": "HH v"`},
	},
	"ru/datetime/more-yMMMMdv/": {
		{source: "chromium-icu", file: chromiumLocales + "ru.txt", has: `Hv{"HH 'ч'. v"}`},
		{source: "cldr-json", file: fmt.Sprintf(cldrGregorian, "ru"), has: `"Hv": "HH v"`},
	},
	"de/datetime/morerange-dateFull/era": {
		{source: "chromium-icu", file: chromiumLocales + "de.txt", has: "E E, d. MMM y G"},
		{source: "cldr-json", file: fmt.Sprintf(cldrGregorian, "de"), lacks: "E E, d. MMM y G"},
	},
}

// TestKnownEvidence: known entries without evidence must be unimplemented features, and (under
// `mise run i18n:verify`) every piece of evidence must hold in the pinned upstream files.
func TestKnownEvidence(t *testing.T) {
	for prefix, reason := range known {
		if _, ok := knownEvidence[prefix]; !ok && reason != noISOCalendar {
			t.Errorf("known %q has no knownEvidence (only unimplemented features may go without)", prefix)
		}
	}
	for prefix := range knownEvidence {
		if _, ok := known[prefix]; !ok {
			t.Errorf("knownEvidence %q has no known entry", prefix)
		}
	}
	if os.Getenv("I18N_VERIFY") == "" {
		t.Skip("upstream evidence: run `mise run i18n:verify`")
	}
	for prefix, evs := range knownEvidence {
		for _, ev := range evs {
			b, err := cldrgen.Reference(ev.source, ev.file)
			if err != nil {
				t.Errorf("%s: %s %s: %v", prefix, ev.source, ev.file, err)
				continue
			}
			if ev.has != "" && !strings.Contains(string(b), ev.has) {
				t.Errorf("%s: %s %s lacks %q: the difference may be gone (re-record goldens, drop the known entry)", prefix, ev.source, ev.file, ev.has)
			}
			if ev.lacks != "" && strings.Contains(string(b), ev.lacks) {
				t.Errorf("%s: %s %s has %q", prefix, ev.source, ev.file, ev.lacks)
			}
		}
	}
}

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
