//go:build !js

package intltest

import (
	"strings"
	"testing"
)

func TestCompare(t *testing.T) {
	want := &Golden{Runtime: "workerd 2026-09-11", Results: map[string]string{
		"de/number/a/1": "1.234", "de/number/b/1": "5", "ja/datetime/x/1": "午後", "locale/und-Arab": "{}", "en/list/l/1": "a, b",
	}}
	got := &Golden{Runtime: "Firefox 155.0", Results: map[string]string{
		"de/number/a/1": "1.234", "de/number/b/1": "6", "ja/datetime/x/1": "午後", "locale/und-Arab": "{\"x\":1}",
	}}
	d := Compare(want, got)
	if len(d.Diffs) != 2 || d.Diffs["de/number/b/1"] != [2]string{"5", "6"} || len(d.Missing) != 1 || d.Missing[0] != "en/list/l/1" {
		t.Fatalf("Compare = %+v", d)
	}
	var areas []string
	for _, a := range d.Areas {
		areas = append(areas, a.Area)
	}
	if strings.Join(areas, ",") != "datetime,list,locale,number" || d.Areas[3].Cases != 2 || d.Areas[3].Diffs != 1 {
		t.Errorf("areas = %+v", d.Areas)
	}
	if s := d.Summary(1); !strings.Contains(s, "Firefox 155.0 vs workerd 2026-09-11: 2 of 5 cases differ") || !strings.Contains(s, `golden "5"`) {
		t.Errorf("Summary:\n%s", s)
	}
}

func TestBrowserVersion(t *testing.T) {
	for ua, want := range map[string]string{
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/152.0.0.0 Safari/537.36": "Chrome 152.0.0.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:155.0) Gecko/20100101 Firefox/155.0":                                          "Firefox 155.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.0 Safari/605.1.15":         "Safari 26.0",
	} {
		if got := browserVersion(ua); got != want {
			t.Errorf("browserVersion(%q) = %q, want %q", ua, got, want)
		}
	}
}
