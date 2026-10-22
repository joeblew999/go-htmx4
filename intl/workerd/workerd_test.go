package workerd_test

import (
	"context"
	"testing"

	"github.com/joeblew999/go-htmx4/intl"
	"github.com/joeblew999/go-htmx4/intl/workerd"
)

// TestStart: what an ordinary Go server gets: intl formats through workerd's Intl, with the app's catalogs (here one
// message compiled the way go tool intl gen writes them), and stop ends it.
func TestStart(t *testing.T) {
	shipped := intl.Locales{intl.NewLocale("en"), intl.NewLocale("de")}
	intl.Use(shipped, []bool{true, true})
	stop, err := workerd.Start([]byte(`export default { "en": ["Hello"], "de": ["Hallo"] };` + "\n"))
	if err != nil {
		t.Fatal(err)
	}
	de := intl.WithRequest(context.Background(), intl.Request{Locale: shipped[1], TimeZone: "Europe/Berlin"})
	if got := intl.Text(de, intl.Int(1234567)); got != "1.234.567" {
		t.Errorf("Int in German = %q, want 1.234.567", got)
	}
	if got := intl.Text(de, intl.M(0)); got != "Hallo" {
		t.Errorf("message 0 in German = %q, want Hallo", got)
	}
	stop()
	if got := intl.Text(de, intl.Int(1)); got != "1" {
		t.Errorf("after stop: %q, want the fallback 1", got)
	}
}
