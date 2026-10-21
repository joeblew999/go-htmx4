package i18n_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/joeblew999/go-htmx4/kit/i18n"
	"github.com/joeblew999/go-htmx4/kit/i18n/cldr"
)

// TestFormatNumberGolden: numbers in messages match what a full Intl-conformant formatter outputs
// (testdata/numbers.json, written with the locale table by `mise run i18n:generate`).
func TestFormatNumberGolden(t *testing.T) {
	data, err := os.ReadFile("testdata/numbers.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Locale, Input string
		MaxFrac       int
		Want          string
	}
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		got := cldr.Data.MustLocale(r.Locale).FormatNumber(i18n.MustDecimal(r.Input), r.MaxFrac)
		if got != r.Want {
			t.Errorf("%s FormatNumber(%s, %d) = %q, want %q", r.Locale, r.Input, r.MaxFrac, got, r.Want)
		}
	}
}
