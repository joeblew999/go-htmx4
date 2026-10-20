package intltest

import "strings"

func init() {
	RegisterCases(DateTime)
	RegisterCases(DateTimeMore)
	Register("DateTimeFormat", evalDateTime)
}

// Instants the DateTimeFormat cases format (JavaScript Date arguments, millisecond precision).
const (
	dtWinter   = "2026-01-15T08:05:09.123Z"
	dtSummer   = "2026-07-04T15:30:45.006Z"
	dtMidnight = "2026-03-01T00:00:00.000Z"
	dtNoon     = "2026-06-21T12:00:00.000Z"
	dtEvening  = "2026-11-30T19:45:00.000Z"
	dtYearEnd  = "2026-12-31T23:59:59.999Z"
	dtBC       = "-000043-03-15T12:00:00.000Z"
)

// dtStyles are Intl.DateTimeFormat's four style values.
var dtStyles = []string{"full", "long", "medium", "short"}

// dtComponents are component option sets (formatMatcher "best fit" → ICU skeletons). Sets whose name
// starts with "y" or "G" are also formatted for a BC date.
var dtComponents = []struct {
	name string
	opts opts
}{
	{"default", opts{}},
	{"y", opts{"year": "numeric"}},
	{"yy", opts{"year": "2-digit"}},
	{"M", opts{"month": "numeric"}},
	{"MM", opts{"month": "2-digit"}},
	{"MMM", opts{"month": "short"}},
	{"MMMM", opts{"month": "long"}},
	{"MMMMM", opts{"month": "narrow"}},
	{"d", opts{"day": "numeric"}},
	{"dd", opts{"day": "2-digit"}},
	{"E", opts{"weekday": "short"}},
	{"EEEE", opts{"weekday": "long"}},
	{"EEEEE", opts{"weekday": "narrow"}},
	{"Gy", opts{"era": "short", "year": "numeric"}},
	{"GGGGy", opts{"era": "long", "year": "numeric"}},
	{"GGGGGyMd", opts{"era": "narrow", "year": "numeric", "month": "numeric", "day": "numeric"}},
	{"GyMMMd", opts{"era": "short", "year": "numeric", "month": "short", "day": "numeric"}},
	{"GyMMMEd", opts{"era": "long", "year": "numeric", "month": "short", "day": "numeric", "weekday": "short"}},
	{"yM", opts{"year": "numeric", "month": "numeric"}},
	{"yMd", opts{"year": "numeric", "month": "numeric", "day": "numeric"}},
	{"yyMMdd", opts{"year": "2-digit", "month": "2-digit", "day": "2-digit"}},
	{"yMMM", opts{"year": "numeric", "month": "short"}},
	{"yMMMM", opts{"year": "numeric", "month": "long"}},
	{"yMMMd", opts{"year": "numeric", "month": "short", "day": "numeric"}},
	{"yMMMMd", opts{"year": "numeric", "month": "long", "day": "numeric"}},
	{"yMMMEd", opts{"year": "numeric", "month": "short", "day": "numeric", "weekday": "short"}},
	{"yMMMMEEEEd", opts{"year": "numeric", "month": "long", "day": "numeric", "weekday": "long"}},
	{"yMMMMMd", opts{"year": "numeric", "month": "narrow", "day": "numeric"}},
	{"yE", opts{"year": "numeric", "weekday": "long"}},
	{"yd", opts{"year": "numeric", "day": "numeric"}},
	{"MMMd", opts{"month": "short", "day": "numeric"}},
	{"MMMMd", opts{"month": "long", "day": "numeric"}},
	{"Md", opts{"month": "numeric", "day": "numeric"}},
	{"MEd", opts{"month": "numeric", "day": "numeric", "weekday": "short"}},
	{"MMMEd", opts{"month": "short", "day": "numeric", "weekday": "short"}},
	{"MMMMEEEEd", opts{"month": "long", "day": "numeric", "weekday": "long"}},
	{"Ed", opts{"day": "numeric", "weekday": "short"}},
	{"EEEEd", opts{"day": "numeric", "weekday": "long"}},
	{"h", opts{"hour": "numeric"}},
	{"hh", opts{"hour": "2-digit"}},
	{"hm", opts{"hour": "numeric", "minute": "2-digit"}},
	{"hmNumeric", opts{"hour": "numeric", "minute": "numeric"}},
	{"hms", opts{"hour": "numeric", "minute": "2-digit", "second": "2-digit"}},
	{"hhmmss", opts{"hour": "2-digit", "minute": "2-digit", "second": "2-digit"}},
	{"m", opts{"minute": "numeric"}},
	{"s", opts{"second": "numeric"}},
	{"ms", opts{"minute": "2-digit", "second": "2-digit"}},
	{"dh", opts{"day": "numeric", "hour": "numeric"}},
	{"Ehm", opts{"weekday": "short", "hour": "numeric", "minute": "2-digit"}},
	{"yMdhm", opts{"year": "numeric", "month": "numeric", "day": "numeric", "hour": "numeric", "minute": "2-digit"}},
	{"yMMMdhm", opts{"year": "numeric", "month": "short", "day": "numeric", "hour": "numeric", "minute": "2-digit"}},
	{"yMMMMdhms", opts{"year": "numeric", "month": "long", "day": "numeric", "hour": "numeric", "minute": "2-digit", "second": "2-digit"}},
	{"yMMMMEEEEdhm", opts{"year": "numeric", "month": "long", "day": "numeric", "weekday": "long", "hour": "numeric", "minute": "2-digit"}},
	{"MMMdhm", opts{"month": "short", "day": "numeric", "hour": "numeric", "minute": "2-digit"}},
	{"yMdHms", opts{"year": "numeric", "month": "numeric", "day": "numeric", "hour": "numeric", "minute": "2-digit", "second": "2-digit", "hour12": false}},
	{"GyMMMdhm", opts{"era": "short", "year": "numeric", "month": "short", "day": "numeric", "hour": "numeric", "minute": "2-digit"}},
	{"hmz", opts{"year": "numeric", "month": "short", "day": "numeric", "hour": "numeric", "minute": "2-digit", "timeZoneName": "short"}},
	{"tzOnly", opts{"timeZoneName": "long"}},
}

// dtZones are the time zones the zone-name cases cover. Instants avoid Africa/Casablanca and El_Aaiun
// after 2026-09-20, where workerd's tzdata predates IANA 2026c.
var dtZones = []string{"UTC", "America/New_York", "America/Los_Angeles", "Europe/London", "Europe/Berlin", "Asia/Kolkata",
	"Asia/Tokyo", "Asia/Kathmandu", "Australia/Lord_Howe", "America/St_Johns", "Asia/Jerusalem", "America/Sao_Paulo",
	"Pacific/Apia", "+05:30", "America/Phoenix", "Europe/Dublin", "Pacific/Honolulu"}

var dtZoneNames = []string{"short", "long", "shortOffset", "longOffset", "shortGeneric", "longGeneric"}

// dtPeriodInstants are local times (zone UTC) around CLDR day period boundaries.
var dtPeriodInstants = []string{
	"2026-05-10T00:00:00.000Z", "2026-05-10T03:07:00.000Z", "2026-05-10T05:59:00.000Z", "2026-05-10T06:00:00.000Z",
	"2026-05-10T08:05:00.000Z", "2026-05-10T11:59:59.000Z", "2026-05-10T12:00:00.000Z", "2026-05-10T12:30:00.000Z",
	"2026-05-10T15:30:00.000Z", "2026-05-10T18:00:00.000Z", "2026-05-10T19:45:00.000Z", "2026-05-10T21:00:00.000Z",
	"2026-05-10T23:59:00.000Z",
}

// dtRanges are formatRange start/end pairs (zone UTC).
var dtRanges = []struct{ name, start, end string }{
	{"same", "2026-01-15T08:05:09.000Z", "2026-01-15T08:05:09.000Z"},
	{"seconds", "2026-01-15T08:05:09.000Z", "2026-01-15T08:05:40.000Z"},
	{"minutes", "2026-01-15T08:05:00.000Z", "2026-01-15T08:35:00.000Z"},
	{"hours", "2026-01-15T08:05:00.000Z", "2026-01-15T11:35:00.000Z"},
	{"ampm", "2026-01-15T08:05:00.000Z", "2026-01-15T15:30:00.000Z"},
	{"days", "2026-01-15T10:00:00.000Z", "2026-01-17T09:00:00.000Z"},
	{"months", "2026-01-15T10:00:00.000Z", "2026-03-03T10:00:00.000Z"},
	{"years", "2026-01-15T10:00:00.000Z", "2027-02-20T10:00:00.000Z"},
}

var dtRangeSets = []struct {
	name string
	opts opts
}{
	{"default", opts{}},
	{"yMMMd", opts{"year": "numeric", "month": "short", "day": "numeric"}},
	{"yMMMMEEEEd", opts{"year": "numeric", "month": "long", "day": "numeric", "weekday": "long"}},
	{"MMMd", opts{"month": "short", "day": "numeric"}},
	{"yMMM", opts{"year": "numeric", "month": "short"}},
	{"MMMM", opts{"month": "long"}},
	{"y", opts{"year": "numeric"}},
	{"hm", opts{"hour": "numeric", "minute": "2-digit"}},
	{"Hm", opts{"hour": "numeric", "minute": "2-digit", "hour12": false}},
	{"h", opts{"hour": "numeric"}},
	{"hmv", opts{"hour": "numeric", "minute": "2-digit", "timeZoneName": "short"}},
	{"yMMMdhm", opts{"year": "numeric", "month": "short", "day": "numeric", "hour": "numeric", "minute": "2-digit"}},
	{"dateMedium", opts{"dateStyle": "medium"}},
	{"dateLongTimeShort", opts{"dateStyle": "long", "timeStyle": "short"}},
	{"timeShort", opts{"timeStyle": "short"}},
	{"Bh", opts{"hour": "numeric", "dayPeriod": "short"}},
}

// DateTime returns the Intl.DateTimeFormat cases: styles, component skeletons, hour cycles, day periods,
// fractional seconds, time zone names, numbering systems, formatRange and resolvedOptions.
func DateTime() []Case {
	var out []Case
	format := func(loc, name string, o opts, zone, in string) {
		o = withZone(o, zone)
		out = append(out, Case{ID: loc + "/datetime/" + name + "/" + in, Ctor: "DateTimeFormat", Locale: loc, Options: o,
			Method: "format", Args: []any{"@date:" + in}})
	}
	for _, loc := range Locales {
		// dateStyle / timeStyle and every combination
		for _, ds := range append([]string{""}, dtStyles...) {
			for _, ts := range append([]string{""}, dtStyles...) {
				if ds == "" && ts == "" {
					continue
				}
				o := opts{}
				name := "style"
				if ds != "" {
					o["dateStyle"] = ds
					name += "-d" + ds
				}
				if ts != "" {
					o["timeStyle"] = ts
					name += "-t" + ts
				}
				for _, in := range []string{dtWinter, dtSummer} {
					format(loc, name, o, "America/Los_Angeles", in)
				}
			}
		}
		// components → best pattern
		for _, set := range dtComponents {
			ins := []string{dtWinter, dtSummer}
			if strings.HasPrefix(set.name, "y") || strings.HasPrefix(set.name, "G") || set.name == "default" {
				ins = append(ins, dtBC, dtYearEnd)
			}
			for _, in := range ins {
				format(loc, "comp-"+set.name, set.opts, "UTC", in)
			}
		}
		// hour cycles: hour12, hourCycle, -u-hc
		hcSets := []struct {
			name string
			o    opts
		}{
			{"hm-hour12", opts{"hour": "numeric", "minute": "2-digit", "hour12": true}},
			{"hm-hour24", opts{"hour": "numeric", "minute": "2-digit", "hour12": false}},
			{"hm-h11", opts{"hour": "numeric", "minute": "2-digit", "hourCycle": "h11"}},
			{"hm-h12", opts{"hour": "numeric", "minute": "2-digit", "hourCycle": "h12"}},
			{"hm-h23", opts{"hour": "numeric", "minute": "2-digit", "hourCycle": "h23"}},
			{"hm-h24", opts{"hour": "numeric", "minute": "2-digit", "hourCycle": "h24"}},
			{"hhmm-h11", opts{"hour": "2-digit", "minute": "2-digit", "hourCycle": "h11"}},
			{"h-h24", opts{"hour": "numeric", "hourCycle": "h24"}},
			{"hm-hour12-h23", opts{"hour": "numeric", "minute": "2-digit", "hour12": true, "hourCycle": "h23"}},
			{"tshort-hour12", opts{"timeStyle": "short", "hour12": true}},
			{"tshort-hour24", opts{"timeStyle": "short", "hour12": false}},
			{"tshort-h11", opts{"timeStyle": "short", "hourCycle": "h11"}},
			{"tshort-h24", opts{"timeStyle": "short", "hourCycle": "h24"}},
			{"dmedium-tmedium-h23", opts{"dateStyle": "medium", "timeStyle": "medium", "hourCycle": "h23"}},
			{"dmedium-tmedium-h12", opts{"dateStyle": "medium", "timeStyle": "medium", "hourCycle": "h12"}},
			// the app applies the viewer's clock preference to styles (views.LocalTime, /formats)
			{"dfull-tlong-h12", opts{"dateStyle": "full", "timeStyle": "long", "hourCycle": "h12"}},
			{"dfull-tlong-h23", opts{"dateStyle": "full", "timeStyle": "long", "hourCycle": "h23"}},
			{"dlong-tshort-h12", opts{"dateStyle": "long", "timeStyle": "short", "hourCycle": "h12"}},
			{"dshort-tfull-h23", opts{"dateStyle": "short", "timeStyle": "full", "hourCycle": "h23"}},
		}
		hcInstants := []string{"2026-05-10T00:05:00.000Z", "2026-05-10T12:30:00.000Z", "2026-05-10T18:45:00.000Z"}
		for _, set := range hcSets {
			for _, in := range hcInstants {
				format(loc, "hc-"+set.name, set.o, "UTC", in)
			}
		}
		for _, hc := range []string{"h11", "h12", "h23", "h24"} {
			ext := loc + "-u-hc-" + hc
			for _, set := range []struct {
				name string
				o    opts
			}{
				{"uhc-hm", opts{"hour": "numeric", "minute": "2-digit"}},
				{"uhc-tshort", opts{"timeStyle": "short"}},
				{"uhc-hm-hour12", opts{"hour": "numeric", "minute": "2-digit", "hour12": true}},
			} {
				for _, in := range hcInstants[:2] {
					format(ext, set.name, set.o, "UTC", in)
				}
			}
		}
		// day periods
		for _, w := range []string{"narrow", "short", "long"} {
			for _, in := range dtPeriodInstants {
				format(loc, "period-"+w, opts{"dayPeriod": w}, "UTC", in)
				format(loc, "period-"+w+"-h", opts{"dayPeriod": w, "hour": "numeric"}, "UTC", in)
			}
		}
		for _, in := range dtPeriodInstants {
			format(loc, "period-short-hm", opts{"dayPeriod": "short", "hour": "numeric", "minute": "2-digit"}, "UTC", in)
			format(loc, "period-long-hm-h23", opts{"dayPeriod": "long", "hour": "numeric", "minute": "2-digit", "hourCycle": "h23"}, "UTC", in)
		}
		// fractional seconds
		for _, set := range []struct {
			name string
			o    opts
		}{
			{"fsd1", opts{"hour": "numeric", "minute": "2-digit", "second": "2-digit", "fractionalSecondDigits": 1}},
			{"fsd2", opts{"hour": "numeric", "minute": "2-digit", "second": "2-digit", "fractionalSecondDigits": 2}},
			{"fsd3", opts{"hour": "numeric", "minute": "2-digit", "second": "2-digit", "fractionalSecondDigits": 3}},
			{"fsd3-s", opts{"second": "numeric", "fractionalSecondDigits": 3}},
			{"fsd2-only", opts{"fractionalSecondDigits": 2}},
			{"fsd3-m", opts{"minute": "2-digit", "fractionalSecondDigits": 3}},
		} {
			for _, in := range []string{dtWinter, dtSummer} {
				format(loc, set.name, set.o, "UTC", in)
			}
		}
		// time zone names
		for _, zone := range dtZones {
			zid := strings.NewReplacer("/", "_", "+", "plus", ":", "").Replace(zone)
			for _, tzn := range dtZoneNames {
				for _, in := range []string{dtWinter, dtSummer} {
					format(loc, "zone-"+zid+"-"+tzn, opts{"hour": "numeric", "minute": "2-digit", "timeZoneName": tzn}, zone, in)
				}
			}
			format(loc, "zone-"+zid+"-tfull", opts{"timeStyle": "full"}, zone, dtSummer)
			format(loc, "zone-"+zid+"-dshort", opts{"dateStyle": "short"}, zone, dtYearEnd)
		}
		// numbering systems
		for _, nu := range []string{"arab", "deva", "hanidec", "latn"} {
			format(loc, "nu-"+nu+"-dmedium-tmedium", opts{"dateStyle": "medium", "timeStyle": "medium", "numberingSystem": nu}, "Asia/Kolkata", dtWinter)
			format(loc, "nu-"+nu+"-offset", opts{"hour": "2-digit", "minute": "2-digit", "timeZoneName": "longOffset", "numberingSystem": nu, "fractionalSecondDigits": 2}, "Asia/Kolkata", dtSummer)
		}
		format(loc+"-u-nu-arab", "nu-ext-yMMMd", opts{"year": "numeric", "month": "short", "day": "numeric"}, "UTC", dtWinter)
		format(loc+"-u-nu-deva", "nu-ext-hms", opts{"hour": "numeric", "minute": "2-digit", "second": "2-digit"}, "UTC", dtSummer)
		// formatRange
		for _, set := range dtRangeSets {
			for _, r := range dtRanges {
				out = append(out, Case{ID: loc + "/datetime/range-" + set.name + "/" + r.name, Ctor: "DateTimeFormat", Locale: loc,
					Options: withZone(set.opts, "UTC"), Method: "formatRange", Args: []any{"@date:" + r.start, "@date:" + r.end}})
			}
		}
		// resolvedOptions (whole object: locale, calendar, numberingSystem, timeZone, hourCycle, components, styles)
		resolved := []struct {
			name string
			o    opts
		}{
			{"default", opts{"timeZone": "UTC"}},
			{"dfull", opts{"dateStyle": "full", "timeZone": "UTC"}},
			{"dshort-tlong", opts{"dateStyle": "short", "timeStyle": "long", "timeZone": "UTC"}},
			{"tshort-hour12", opts{"timeStyle": "short", "hour12": true, "timeZone": "UTC"}},
			{"hm", opts{"hour": "numeric", "minute": "2-digit", "timeZone": "UTC"}},
			{"hm-hour12", opts{"hour": "2-digit", "minute": "numeric", "hour12": true, "timeZone": "UTC"}},
			{"hm-hour24", opts{"hour": "numeric", "minute": "2-digit", "hour12": false, "timeZone": "UTC"}},
			{"hm-h24", opts{"hour": "numeric", "hourCycle": "h24", "timeZone": "UTC"}},
			{"ymd-hour12", opts{"year": "numeric", "hour12": true, "timeZone": "UTC"}},
			{"components", opts{"weekday": "long", "era": "short", "year": "2-digit", "month": "long", "day": "2-digit", "timeZone": "UTC"}},
			{"time-all", opts{"dayPeriod": "narrow", "hour": "numeric", "minute": "numeric", "second": "numeric", "fractionalSecondDigits": 2, "timeZoneName": "shortGeneric", "timeZone": "UTC"}},
			{"month-narrow", opts{"month": "narrow", "timeZone": "UTC"}},
			{"zone-lower", opts{"timeZone": "america/new_york"}},
			{"zone-etc-utc", opts{"timeZone": "Etc/UTC"}},
			{"zone-gmt", opts{"timeZone": "GMT"}},
			{"zone-calcutta", opts{"timeZone": "Asia/Calcutta"}},
			{"zone-offset", opts{"timeZone": "+05:30"}},
			{"zone-offset-compact", opts{"timeZone": "-0800"}},
			{"zone-etc-gmt", opts{"timeZone": "Etc/GMT+5"}},
			{"nu-arab", opts{"numberingSystem": "arab", "timeZone": "UTC"}},
			{"nu-bogus", opts{"numberingSystem": "abcd", "timeZone": "UTC"}},
			{"ca-iso8601", opts{"calendar": "iso8601", "timeZone": "UTC"}},
		}
		for _, r := range resolved {
			out = append(out, Case{ID: loc + "/datetime/resolved-" + r.name + "/object", Ctor: "DateTimeFormat", Locale: loc,
				Options: r.o, Method: "resolvedOptions"})
		}
		for _, ext := range []string{"-u-hc-h11", "-u-hc-h24", "-u-nu-arab", "-u-ca-gregory"} {
			out = append(out, Case{ID: loc + ext + "/datetime/resolved-ext/hm", Ctor: "DateTimeFormat", Locale: loc + ext,
				Options: opts{"hour": "numeric", "minute": "2-digit", "timeZone": "UTC"}, Method: "resolvedOptions"})
			out = append(out, Case{ID: loc + ext + "/datetime/resolved-ext/hm-hour24", Ctor: "DateTimeFormat", Locale: loc + ext,
				Options: opts{"hour": "numeric", "minute": "2-digit", "hour12": false, "timeZone": "UTC"}, Method: "resolvedOptions"})
		}
		for _, f := range []string{"hourCycle", "hour12", "timeZone", "numberingSystem", "calendar"} {
			out = append(out, Case{ID: loc + "/datetime/resolved-field/" + f, Ctor: "DateTimeFormat", Locale: loc,
				Options: opts{"hour": "numeric", "timeZone": "Asia/Tokyo"}, Method: "resolvedOptions", Field: f})
		}
		// errors
		for _, e := range []struct {
			name string
			o    opts
		}{
			{"style-and-component", opts{"dateStyle": "short", "hour": "numeric", "timeZone": "UTC"}},
			{"bad-zone", opts{"timeZone": "Mars/Olympus_Mons"}},
			{"bad-offset", opts{"timeZone": "+25:00"}},
			{"fsd-range", opts{"fractionalSecondDigits": 4, "timeZone": "UTC"}},
			{"bad-month", opts{"month": "tiny", "timeZone": "UTC"}},
		} {
			out = append(out, Case{ID: loc + "/datetime/error-" + e.name + "/x", Ctor: "DateTimeFormat", Locale: loc,
				Options: e.o, Method: "format", Args: []any{"@date:" + dtWinter}})
		}
	}
	return out
}

// dtMoreComponents are further skeletons: odd combinations that exercise appended fields, date + time
// composition by month width, hour cycles with dates and fractional seconds.
var dtMoreComponents = []struct {
	name string
	opts opts
}{
	{"MMMMEEEE", opts{"month": "long", "weekday": "long"}},
	{"MMME", opts{"month": "short", "weekday": "short"}},
	{"GGGG", opts{"era": "long"}},
	{"GMMM", opts{"era": "short", "month": "short"}},
	{"yMMMMM", opts{"year": "numeric", "month": "narrow"}},
	{"Ehms", opts{"weekday": "long", "hour": "numeric", "minute": "2-digit", "second": "2-digit"}},
	{"dhm", opts{"day": "numeric", "hour": "numeric", "minute": "2-digit"}},
	{"MMdd", opts{"month": "2-digit", "day": "2-digit"}},
	{"yyMMM", opts{"year": "2-digit", "month": "short"}},
	{"MMMMdhm", opts{"month": "long", "day": "numeric", "hour": "numeric", "minute": "2-digit"}},
	{"full-all", opts{"weekday": "long", "year": "numeric", "month": "long", "day": "numeric", "hour": "numeric", "minute": "2-digit", "second": "2-digit", "timeZoneName": "long"}},
	{"BEh", opts{"dayPeriod": "long", "weekday": "long", "hour": "numeric"}},
	{"HHmmssSSS", opts{"hour": "2-digit", "minute": "2-digit", "second": "2-digit", "fractionalSecondDigits": 3, "hour12": false}},
	{"yMdh-h11", opts{"year": "numeric", "month": "numeric", "day": "numeric", "hour": "numeric", "hourCycle": "h11"}},
	{"ms-mixed", opts{"minute": "2-digit", "second": "numeric"}},
	{"hs", opts{"hour": "numeric", "second": "2-digit"}},
	{"yh", opts{"year": "numeric", "hour": "numeric"}},
	{"MMMMMd", opts{"month": "narrow", "day": "numeric"}},
	{"GGGGG", opts{"era": "narrow"}},
	{"yMMMEd-short", opts{"year": "numeric", "month": "short", "day": "numeric", "weekday": "narrow"}},
	{"O", opts{"timeZoneName": "shortOffset"}},
	{"dz", opts{"day": "numeric", "timeZoneName": "short"}},
	{"yMMMMdv", opts{"year": "numeric", "month": "long", "day": "numeric", "hour": "numeric", "timeZoneName": "longGeneric"}},
	{"Bhms", opts{"dayPeriod": "short", "hour": "numeric", "minute": "2-digit", "second": "2-digit"}},
	{"hmm-h24-date", opts{"month": "short", "day": "numeric", "hour": "2-digit", "minute": "2-digit", "hourCycle": "h24"}},
}

// dtMoreZones are zones for partial location names, non-primary and aliased zones, negative DST in
// vanguard tzdata, southern-hemisphere DST, unusual offsets and metazone history.
var dtMoreZones = []string{"America/Indiana/Knox", "America/Detroit", "Europe/Busingen", "America/Boise", "Asia/Urumqi",
	"America/Argentina/Buenos_Aires", "Asia/Calcutta", "US/Eastern", "Antarctica/Troll", "Africa/Casablanca",
	"Australia/Sydney", "America/Santiago", "Pacific/Chatham", "Pacific/Kiritimati", "Etc/GMT+5", "Europe/Kyiv",
	"Asia/Tehran", "Europe/Moscow", "Asia/Hong_Kong", "Africa/Windhoek", "Europe/Minsk"}

var dtMoreRanges = []struct{ name, start, end string }{
	{"seconds", "2026-01-15T08:05:09.000Z", "2026-01-15T08:05:40.000Z"},
	{"ampm", "2026-01-15T11:05:00.000Z", "2026-01-15T13:30:00.000Z"},
	{"days", "2026-01-15T10:00:00.000Z", "2026-01-17T09:00:00.000Z"},
	{"months", "2026-01-31T10:00:00.000Z", "2026-02-01T10:00:00.000Z"},
	{"years", "2026-12-31T22:00:00.000Z", "2027-01-01T02:00:00.000Z"},
	{"era", "-000001-06-01T00:00:00.000Z", "0001-06-01T00:00:00.000Z"},
}

var dtMoreRangeSets = []struct {
	name string
	opts opts
}{
	{"yMMMMd", opts{"year": "numeric", "month": "long", "day": "numeric"}},
	{"MMMEd", opts{"month": "short", "day": "numeric", "weekday": "short"}},
	{"yMdHm", opts{"year": "numeric", "month": "numeric", "day": "numeric", "hour": "numeric", "minute": "2-digit", "hour12": false}},
	{"hms", opts{"hour": "numeric", "minute": "2-digit", "second": "2-digit"}},
	{"BBBBh", opts{"dayPeriod": "long", "hour": "numeric"}},
	{"GyMMMd", opts{"era": "short", "year": "numeric", "month": "short", "day": "numeric"}},
	{"Md", opts{"month": "numeric", "day": "numeric"}},
	{"EEEE", opts{"weekday": "long"}},
	{"Kmm", opts{"hour": "numeric", "minute": "2-digit", "hourCycle": "h11"}},
	{"timeMedium", opts{"timeStyle": "medium"}},
	{"dateFull", opts{"dateStyle": "full"}},
	{"dateShortTimeFull", opts{"dateStyle": "short", "timeStyle": "full"}},
	{"hmvvvv", opts{"hour": "numeric", "minute": "2-digit", "timeZoneName": "longGeneric"}},
}

// DateTimeMore returns further Intl.DateTimeFormat cases (odd skeletons, zone edge cases, historical
// metazones and ranges).
func DateTimeMore() []Case {
	var out []Case
	format := func(loc, name string, o opts, zone, in string) {
		out = append(out, Case{ID: loc + "/datetime/" + name + "/" + in, Ctor: "DateTimeFormat", Locale: loc,
			Options: withZone(o, zone), Method: "format", Args: []any{"@date:" + in}})
	}
	for _, loc := range Locales {
		for _, set := range dtMoreComponents {
			for _, in := range []string{dtWinter, dtSummer} {
				format(loc, "more-"+set.name, set.opts, "America/Los_Angeles", in)
			}
		}
		for _, set := range dtMoreRangeSets {
			for _, r := range dtMoreRanges {
				out = append(out, Case{ID: loc + "/datetime/morerange-" + set.name + "/" + r.name, Ctor: "DateTimeFormat", Locale: loc,
					Options: withZone(set.opts, "Europe/Berlin"), Method: "formatRange", Args: []any{"@date:" + r.start, "@date:" + r.end}})
			}
		}
		for hc, ds := range map[string]string{"h11": "full", "h24": "short"} {
			format(loc+"-u-hc-"+hc, "uhc-tfull-d"+ds, opts{"dateStyle": ds, "timeStyle": "full"}, "Asia/Tokyo", dtWinter)
		}
	}
	for _, loc := range []string{"en", "de", "ja", "ar", "es", "fr"} {
		for _, zone := range dtMoreZones {
			zid := strings.NewReplacer("/", "_", "+", "plus").Replace(zone)
			for _, tzn := range []string{"short", "long", "shortGeneric", "longGeneric"} {
				for _, in := range []string{dtWinter, dtSummer, "1990-07-01T12:00:00.000Z"} {
					format(loc, "morezone-"+zid+"-"+tzn, opts{"hour": "numeric", "minute": "2-digit", "timeZoneName": tzn}, zone, in)
				}
			}
		}
	}
	return out
}

// withZone returns a copy of o with timeZone set.
func withZone(o opts, zone string) opts {
	c := make(opts, len(o)+1)
	for k, v := range o {
		c[k] = v
	}
	c["timeZone"] = zone
	return c
}
