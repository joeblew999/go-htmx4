package views

import (
	"context"
	"os"
	"testing"

	"github.com/joeblew999/go-htmx4/intl"
	"github.com/joeblew999/go-htmx4/intl/intltest"
	"github.com/joeblew999/go-htmx4/locales"
)

// TestMain runs the tests with the Worker's Intl, on workerd (intl/intltest).
func TestMain(m *testing.M) { os.Exit(intltest.Main(m)) }

// TestInt: counts in the language's digits and grouping (en-IN groups in lakhs, Spanish leaves four digits alone).
func TestInt(t *testing.T) {
	for _, tc := range []struct {
		locale string
		v      int64
		want   string
	}{{"en", 1234567, "1,234,567"}, {"de", -1234, "-1.234"}, {"en-IN", 1234567, "12,34,567"}, {"es", 1234, "1234"}} {
		ctx := intl.WithRequest(context.Background(), intl.Request{Locale: locales.Shipped.Find(tc.locale)})
		if got := intl.Text(ctx, intl.Int(tc.v)); got != tc.want {
			t.Errorf("Int(%s, %d) = %q, want %q", tc.locale, tc.v, got, tc.want)
		}
	}
}
