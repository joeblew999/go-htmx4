package i18n

import (
	"errors"
	"math"
	"slices"
	"strings"
)

// RelativeTimeOptions mirror Intl.RelativeTimeFormat's options. The zero value is
// {style: "long", numeric: "always"}.
type RelativeTimeOptions struct {
	Style           TextStyle
	NumericAuto     bool   // numeric: "auto" ("yesterday" instead of "1 day ago")
	NumberingSystem string // overrides -u-nu
}

// RelativeTimeFormat formats relative times for one locale and set of options (Intl.RelativeTimeFormat).
type RelativeTimeFormat struct {
	loc  *Locale
	opts RelativeTimeOptions
	num  *NumberFormat
	rel  *[RelUnitCount][WidthCount]RelativeUnit
}

// RelativeTimeFormat compiles options for this locale.
func (l *Locale) RelativeTimeFormat(o RelativeTimeOptions) *RelativeTimeFormat {
	return &RelativeTimeFormat{
		loc:  l,
		opts: o,
		num:  l.MustNumberFormat(NumberOptions{NumberingSystem: o.NumberingSystem}),
		rel:  &l.Data.Relative.Units,
	}
}

// ErrNotFinite is returned for NaN or infinite values where Intl throws a RangeError.
var ErrNotFinite = errors.New("i18n: value is not finite")

// Format formats value units relative to now: -1 day is "1 day ago" (or "yesterday" with NumericAuto),
// 2.5 hours is "in 2.5 hours". -0 is in the past, as in Intl.
func (f *RelativeTimeFormat) Format(value float64, unit RelUnit) (string, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return "", ErrNotFinite
	}
	if unit >= RelUnitCount {
		return "", errors.New("i18n: unknown relative time unit")
	}
	u := &f.rel[unit][f.opts.Style.width()]
	if f.opts.NumericAuto && value == math.Trunc(value) && math.Abs(value) <= 127 {
		i, ok := slices.BinarySearchFunc(u.Phrases, int8(value), func(p RelPhrase, o int8) int { return int(p.Offset) - int(o) })
		if ok {
			return u.Phrases[i].Text, nil
		}
	}
	past := value < 0 || value == 0 && math.Signbit(value)
	abs := Float(math.Abs(value))
	// Plural category of the value as Intl.PluralRules sees it by default (at most 3 fraction digits).
	cat := f.loc.Data.Cardinal.Select(abs.roundAt(-3, 1, HalfExpand).plain(0))
	pats := &u.Future
	if past {
		pats = &u.Past
	}
	pat := cmpOr(pats[cat], pats[Other])
	return strings.Replace(pat, "{0}", f.num.Format(abs), 1), nil
}

// MustFormat is Format for finite values.
func (f *RelativeTimeFormat) MustFormat(value float64, unit RelUnit) string {
	s, err := f.Format(value, unit)
	if err != nil {
		panic(err)
	}
	return s
}

// WeekInfo is Intl.Locale's getWeekInfo: ISO day numbers, 1 = Monday … 7 = Sunday.
type WeekInfo struct {
	FirstDay    int
	Weekend     []int
	MinimalDays int
}

// WeekInfo returns the locale's week data, from its (maximized) region and the -u-fw keyword.
func (l *Locale) WeekInfo() WeekInfo {
	var set *Data = l.set
	region := MustParseTag(l.Data.Maximal).Region
	if l.Tag.Region != "" {
		region = l.Tag.Region
	} else if set != nil {
		if r := set.Maximize(l.Tag).Region; r != "" {
			region = r
		}
	}
	var rw *RegionWeek
	if set != nil {
		find := func(r string) *RegionWeek {
			i, ok := slices.BinarySearchFunc(set.Weeks, r, func(w RegionWeek, s string) int { return strings.Compare(w.Region, s) })
			if ok {
				return &set.Weeks[i]
			}
			return nil
		}
		if rw = find(region); rw == nil {
			rw = find("001")
		}
	}
	if rw == nil {
		return WeekInfo{FirstDay: 1, Weekend: []int{6, 7}, MinimalDays: 1}
	}
	iso := func(d int8) int {
		if d == 0 {
			return 7
		}
		return int(d)
	}
	wi := WeekInfo{FirstDay: iso(rw.FirstDay), MinimalDays: int(rw.MinDays)}
	for _, d := range rw.Weekend {
		wi.Weekend = append(wi.Weekend, iso(d))
	}
	slices.Sort(wi.Weekend)
	if fw := l.Tag.Keyword("fw"); fw != "" {
		for d, name := range []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"} {
			if fw == name {
				wi.FirstDay = d + 1
			}
		}
	}
	return wi
}

// Thresholds for choosing a relative time unit from an elapsed time (ElapsedUnit). The browser behaviour
// that keeps <time data-relative-time> fresh (static/relative-time.js) uses the same values, so server and
// client text agree; a test in the app checks the two stay in sync.
const (
	RelativeSecondsMax = 45      // below 45 s: seconds
	RelativeMinutesMax = 45 * 60 // below 45 min: minutes
	RelativeHoursMax   = 22 * 3600
	RelativeDaysMax    = 26 * 86400
	RelativeMonthsMax  = 320 * 86400
)

// ElapsedUnit picks the unit and rounded value for a relative time: seconds is from - now (negative in
// the past), e.g. -150 → (-3, RelMinute). Months are 30 days and years 365 days.
func ElapsedUnit(seconds float64) (float64, RelUnit) {
	abs := math.Abs(seconds)
	switch {
	case abs < RelativeSecondsMax:
		return math.Round(seconds), RelSecond
	case abs < RelativeMinutesMax:
		return math.Round(seconds / 60), RelMinute
	case abs < RelativeHoursMax:
		return math.Round(seconds / 3600), RelHour
	case abs < RelativeDaysMax:
		return math.Round(seconds / 86400), RelDay
	case abs < RelativeMonthsMax:
		return math.Round(seconds / (30 * 86400)), RelMonth
	}
	return math.Round(seconds / (365 * 86400)), RelYear
}
