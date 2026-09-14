package views

import (
	"context"
	"testing"
	"time"

	"github.com/joeblew999/go-htmx4/kit/i18n"
	"github.com/joeblew999/go-htmx4/kit/i18n/cldr"
)

func TestElapsed(t *testing.T) {
	for _, tc := range []struct {
		locale string
		d      time.Duration
		want   string
	}{
		{"en", 0, "0 sec"},
		{"en", 900 * time.Millisecond, "0 sec"},
		{"en", 26*time.Hour + 5*time.Minute + 30*time.Second, "1 day, 2 hr, 5 min, 30 sec"},
		{"de", 90 * time.Second, "1 Min., 30 Sek."},
	} {
		ctx := i18n.WithRequest(context.Background(), i18n.Request{Locale: cldr.Data.MustLocale(tc.locale)})
		if got := Elapsed(ctx, tc.d); got != tc.want {
			t.Errorf("Elapsed(%s, %v) = %q, want %q", tc.locale, tc.d, got, tc.want)
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
