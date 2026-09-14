package i18n_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/joeblew999/go-htmx4/kit/i18n"
	"github.com/joeblew999/go-htmx4/kit/i18n/cldr"
)

func init() {
	// Time zone offsets come from Go's tzdata. Pin the copy shipped with the Go toolchain (the same data
	// time/tzdata embeds for TinyGo Workers) so results don't depend on the host's /usr/share/zoneinfo.
	if os.Getenv("ZONEINFO") == "" {
		//lint:ignore SA1019 the build-time GOROOT is what we want: its lib/time/zoneinfo.zip
		zip := filepath.Join(runtime.GOROOT(), "lib", "time", "zoneinfo.zip")
		if _, err := os.Stat(zip); err == nil {
			os.Setenv("ZONEINFO", zip)
		}
	}
}

func TestDateTimeFormatAPI(t *testing.T) {
	de := cldr.Data.MustLocale("de")
	berlin, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Skip("no tzdata:", err)
	}
	at := time.Date(2026, 7, 4, 15, 30, 45, 6e6, time.UTC)

	byName := de.MustDateTimeFormat(i18n.DateTimeOptions{DateStyle: i18n.FullStyle, TimeStyle: i18n.LongStyle, TimeZone: "europe/berlin"})
	byLoc := de.MustDateTimeFormat(i18n.DateTimeOptions{DateStyle: i18n.FullStyle, TimeStyle: i18n.LongStyle, Location: berlin})
	want := "Samstag, 4. Juli 2026 um 17:30:45 MESZ"
	if got := byName.Format(at); got != want {
		t.Errorf("TimeZone: got %q, want %q", got, want)
	}
	if got := byLoc.Format(at); got != want {
		t.Errorf("Location: got %q, want %q", got, want)
	}
	if got := byLoc.FormatMillis(at.UnixMilli()); got != want {
		t.Errorf("FormatMillis: got %q, want %q", got, want)
	}
	if r := byName.ResolvedOptions(); r.TimeZone != "Europe/Berlin" || r.HourCycle != i18n.H23 || r.Calendar != "gregory" {
		t.Errorf("resolvedOptions: %+v", r)
	}

	fixed := de.MustDateTimeFormat(i18n.DateTimeOptions{Hour: i18n.FieldNumeric, Minute: i18n.FieldTwoDigit, TimeZoneName: i18n.ZoneLong, Location: time.FixedZone("", 5*3600+1800)})
	if got, r := fixed.Format(at), fixed.ResolvedOptions(); got != "21:00 GMT+05:30" || r.TimeZone != "+05:30" {
		t.Errorf("fixed zone: got %q, timeZone %q", got, r.TimeZone)
	}

	utc := de.MustDateTimeFormat(i18n.DateTimeOptions{})
	if got := utc.Format(time.Date(2026, 12, 31, 23, 30, 0, 0, time.UTC)); got != "31.12.2026" {
		t.Errorf("zero options (UTC): got %q", got)
	}

	for name, o := range map[string]struct {
		opts i18n.DateTimeOptions
		err  error
	}{
		"style+component": {i18n.DateTimeOptions{DateStyle: i18n.ShortStyle, Hour: i18n.FieldNumeric}, i18n.ErrDateTimeOptions},
		"zone":            {i18n.DateTimeOptions{TimeZone: "Mars/Olympus_Mons"}, i18n.ErrTimeZone},
		"offset":          {i18n.DateTimeOptions{TimeZone: "+24:00"}, i18n.ErrTimeZone},
		"fsd":             {i18n.DateTimeOptions{FractionalSecondDigits: 4}, i18n.ErrRange},
		"year text":       {i18n.DateTimeOptions{Year: i18n.FieldShort}, i18n.ErrRange},
	} {
		if _, err := de.DateTimeFormat(o.opts); !errors.Is(err, o.err) {
			t.Errorf("%s: got error %v, want %v", name, err, o.err)
		}
	}
	defer func() {
		if recover() == nil {
			t.Error("MustDateTimeFormat did not panic")
		}
	}()
	de.MustDateTimeFormat(i18n.DateTimeOptions{TimeZone: "Nowhere/Land"})
}

func TestDateTimeFormatRange(t *testing.T) {
	en := cldr.Data.MustLocale("en")
	f := en.MustDateTimeFormat(i18n.DateTimeOptions{Year: i18n.FieldNumeric, Month: i18n.FieldShort, Day: i18n.FieldNumeric, TimeZone: "UTC"})
	a := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	for _, c := range []struct {
		end  time.Time
		want string
	}{
		{a.Add(time.Hour), "Jan 15, 2026"},
		{a.AddDate(0, 0, 2), "Jan 15 – 17, 2026"},
		{a.AddDate(1, 1, 5), "Jan 15, 2026 – Feb 20, 2027"},
	} {
		if got := f.FormatRange(a, c.end); got != c.want {
			t.Errorf("FormatRange(%v): got %q, want %q", c.end, got, c.want)
		}
	}
}

func ExampleLocale_DateTimeFormat() {
	at := time.Date(2026, 9, 14, 13, 5, 0, 0, time.UTC)
	for _, id := range []string{"en", "de", "ja", "ar"} {
		f := cldr.Data.MustLocale(id).MustDateTimeFormat(i18n.DateTimeOptions{
			DateStyle: i18n.MediumStyle, TimeStyle: i18n.ShortStyle, TimeZone: "UTC",
		})
		fmt.Println(f.Format(at))
	}
	// Output:
	// Sep 14, 2026, 1:05 PM
	// 14.09.2026, 13:05
	// 2026/09/14 13:05
	// 14‏/09‏/2026، 1:05 م
}
