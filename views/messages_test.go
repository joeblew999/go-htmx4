package views

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/joeblew999/go-htmx4/internal/i18n"
	"github.com/joeblew999/go-htmx4/internal/i18n/i18ntest"
	"github.com/joeblew999/go-htmx4/locales"
)

// TestMain runs the tests with the Worker's Intl: worker/gointl.mjs on workerd (internal/i18n/i18ntest).
func TestMain(m *testing.M) { os.Exit(i18ntest.Main(m)) }

func TestElapsed(t *testing.T) {
	for _, tc := range []struct {
		d          time.Duration
		json, text string
	}{
		{0, `{"days":0,"hours":0,"minutes":0,"seconds":0}`, "0s"},
		{900 * time.Millisecond, `{"days":0,"hours":0,"minutes":0,"seconds":0}`, "0s"},
		{26*time.Hour + 5*time.Minute + 30*time.Second, `{"days":1,"hours":2,"minutes":5,"seconds":30}`, "1d 2h 5m 30s"},
		{90 * time.Second, `{"days":0,"hours":0,"minutes":1,"seconds":30}`, "1m 30s"},
	} {
		if got := durationJSON(tc.d); got != tc.json {
			t.Errorf("durationJSON(%v) = %s, want %s", tc.d, got, tc.json)
		}
		if got := durationText(tc.d); got != tc.text {
			t.Errorf("durationText(%v) = %q, want %q", tc.d, got, tc.text)
		}
	}
}

func TestNum(t *testing.T) {
	for _, tc := range []struct {
		locale string
		v      int64
		want   string
	}{{"en", 1234567, "1,234,567"}, {"de", -1234, "-1.234"}, {"en-IN", 1234567, "12,34,567"}, {"es", 1234, "1234"}} {
		ctx := i18n.WithRequest(context.Background(), i18n.Request{Locale: locales.Shipped.Find(tc.locale)})
		if got := Num(ctx, tc.v); got != tc.want {
			t.Errorf("Num(%s, %d) = %q, want %q", tc.locale, tc.v, got, tc.want)
		}
	}
}

func TestJSOptions(t *testing.T) {
	for in, want := range map[string]string{
		"": "{}",
		`{"dateStyle":"full","timeStyle":"long"}`:                        `{dateStyle: "full", timeStyle: "long"}`,
		`{"maximumFractionDigits":0}`:                                    `{maximumFractionDigits: 0}`,
		`{"hour":"numeric","timeZone":"America/New_York","hour12":true}`: `{hour: "numeric", timeZone: "America/New_York", hour12: true}`,
		`{"style":"currency","currency":"EUR","currencyDisplay":"name"}`: `{style: "currency", currency: "EUR", currencyDisplay: "name"}`,
	} {
		if got := jsOptions(in); got != want {
			t.Errorf("jsOptions(%s) = %s, want %s", in, got, want)
		}
	}
}

func TestValidTimeZone(t *testing.T) {
	for id, want := range map[string]bool{"Europe/Berlin": true, "America/Argentina/Buenos_Aires": true, "UTC": true, "Etc/GMT+5": true,
		"": false, "../etc/passwd": false, "Europe/Berlin\"": false, "<script>": false, "a/b/c/d": false} {
		if got := ValidTimeZone(id); got != want {
			t.Errorf("ValidTimeZone(%q) = %v, want %v", id, got, want)
		}
	}
}

func TestIsolate(t *testing.T) {
	for _, tc := range []struct{ dir, s, want string }{
		{"ltr", "Ada", "Ada"},
		{"ltr", "سارة", "⁨سارة⁩"},
		{"rtl", "سارة", "سارة"},
		{"rtl", "Ada", "⁨Ada⁩"},
		{"ltr", "123", "⁨123⁩"},
	} {
		if got := Isolate(tc.dir, tc.s); got != tc.want {
			t.Errorf("Isolate(%s, %q) = %q, want %q", tc.dir, tc.s, got, tc.want)
		}
	}
}
