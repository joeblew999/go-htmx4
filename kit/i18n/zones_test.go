package i18n_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/joeblew999/go-htmx4/kit/i18n"
	"github.com/joeblew999/go-htmx4/kit/i18n/cldr"
)

func TestTimeZone(t *testing.T) {
	for in, want := range map[string]string{
		"Europe/Berlin": "Europe/Berlin", "europe/berlin": "Europe/Berlin", "UTC": "UTC", "Etc/GMT": "UTC",
		"Asia/Kolkata": "Asia/Calcutta", "+05:30": "+05:30", "-08": "-08:00",
	} {
		if got, ok := cldr.Data.TimeZone(in); !ok || got != want {
			t.Errorf("TimeZone(%q) = %q, %v; want %q", in, got, ok, want)
		}
	}
	for _, bad := range []string{"", "Mars/Olympus", "Europe/Berlin\"", "+25:00", "../etc/passwd"} {
		if got, ok := cldr.Data.TimeZone(bad); ok {
			t.Errorf("TimeZone(%q) = %q, want not ok", bad, got)
		}
	}
	ids := cldr.Data.TimeZoneIDs()
	if len(ids) < 400 || !slices.IsSorted(ids) || !slices.Contains(ids, "Europe/Berlin") || slices.Contains(ids, "Asia/Kolkata") {
		t.Errorf("TimeZoneIDs: %d ids, sorted %v", len(ids), slices.IsSorted(ids))
	}
	for _, id := range ids {
		if got, ok := cldr.Data.TimeZone(id); !ok || got != id && !(id == "Etc/UTC" && got == "UTC") && !(id == "Etc/GMT" && got == "UTC") {
			t.Errorf("TimeZoneIDs has %q, which resolves to %q, %v", id, got, ok)
		}
	}
}

func TestDateTimeOptionsIntlJSON(t *testing.T) {
	for _, tc := range []struct {
		o    i18n.DateTimeOptions
		want string
	}{
		{i18n.DateTimeOptions{}, `{}`},
		{i18n.DateTimeOptions{DateStyle: i18n.FullStyle, TimeStyle: i18n.LongStyle, TimeZone: "Asia/Tokyo"}, `{"dateStyle":"full","timeStyle":"long"}`},
		{i18n.DateTimeOptions{Weekday: i18n.FieldLong, Month: i18n.FieldShort, Day: i18n.FieldTwoDigit, Hour: i18n.FieldNumeric, FractionalSecondDigits: 3, TimeZoneName: i18n.ZoneLongGeneric, Hour12: i18n.B(false), HourCycle: i18n.H23},
			`{"weekday":"long","month":"short","day":"2-digit","hour":"numeric","fractionalSecondDigits":3,"timeZoneName":"longGeneric","hour12":false,"hourCycle":"h23"}`},
	} {
		if got := tc.o.IntlJSON(); got != tc.want {
			t.Errorf("IntlJSON = %s, want %s", got, tc.want)
		}
	}
}

// TestTimeZoneName: the name is exactly the zone part of a format with only timeZoneName (which the
// conformance cases check against Intl), for every locale, style and a spread of zones and seasons.
func TestTimeZoneName(t *testing.T) {
	zones := []string{"Europe/Berlin", "America/Los_Angeles", "Asia/Kolkata", "Australia/Lord_Howe", "UTC", "+05:30", "America/Sao_Paulo", "Africa/Casablanca", "Pacific/Chatham"}
	styles := []i18n.ZoneName{i18n.ZoneShort, i18n.ZoneLong, i18n.ZoneShortOffset, i18n.ZoneLongOffset, i18n.ZoneShortGeneric, i18n.ZoneLongGeneric}
	for _, ld := range cldr.Data.Locales {
		loc := cldr.Data.MustLocale(ld.ID)
		for _, at := range []time.Time{time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC), time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)} {
			for _, z := range zones {
				for _, st := range styles {
					name, err := loc.TimeZoneName(z, st, at)
					if err != nil {
						t.Fatalf("%s %s: %v", ld.ID, z, err)
					}
					full := loc.MustDateTimeFormat(i18n.DateTimeOptions{TimeZone: z, TimeZoneName: st}).Format(at)
					if name == "" || !strings.Contains(full, name) {
						t.Errorf("%s %s style %d: name %q not in %q", ld.ID, z, st, name, full)
					}
				}
			}
		}
	}
	if got, _ := cldr.Data.MustLocale("de").TimeZoneName("Europe/Berlin", i18n.ZoneLong, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)); got != "Mitteleuropäische Sommerzeit" {
		t.Errorf("de Europe/Berlin long in July = %q", got)
	}
}
