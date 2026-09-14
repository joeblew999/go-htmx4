package i18n

import (
	"errors"
	"slices"
	"strings"
	"sync"
	"time"
)

// DateTimeStyle is Intl.DateTimeFormat's dateStyle and timeStyle.
type DateTimeStyle uint8

// Date and time styles. NoStyle is the zero value (option not set).
const (
	NoStyle DateTimeStyle = iota
	FullStyle
	LongStyle
	MediumStyle
	ShortStyle
)

// DateField is the value of an Intl.DateTimeFormat component option (weekday, era, year, month, day,
// dayPeriod, hour, minute, second). Which values a component accepts is as in ECMA-402: weekday, era
// and dayPeriod take Narrow, Short or Long; year, day, hour, minute and second take Numeric or TwoDigit;
// month takes all five.
type DateField uint8

// Component option values. FieldNone is the zero value (option not set).
const (
	FieldNone DateField = iota
	FieldNumeric
	FieldTwoDigit
	FieldNarrow
	FieldShort
	FieldLong
)

// ZoneName is Intl.DateTimeFormat's timeZoneName.
type ZoneName uint8

// Time zone name styles. ZoneNameNone is the zero value (option not set).
const (
	ZoneNameNone ZoneName = iota
	ZoneShort
	ZoneLong
	ZoneShortOffset
	ZoneLongOffset
	ZoneShortGeneric
	ZoneLongGeneric
)

// HourCycle is Intl.DateTimeFormat's hourCycle.
type HourCycle uint8

// Hour cycles. HourCycleDefault is the zero value (option not set: the locale's, or -u-hc).
const (
	HourCycleDefault HourCycle = iota
	H11
	H12
	H23
	H24
)

// String is the ECMA-402 name, e.g. "h23" ("" for HourCycleDefault).
func (h HourCycle) String() string {
	switch h {
	case H11:
		return "h11"
	case H12:
		return "h12"
	case H23:
		return "h23"
	case H24:
		return "h24"
	}
	return ""
}

// ParseHourCycle parses "h11", "h12", "h23" or "h24".
func ParseHourCycle(s string) (HourCycle, bool) {
	switch s {
	case "h11":
		return H11, true
	case "h12":
		return H12, true
	case "h23":
		return H23, true
	case "h24":
		return H24, true
	}
	return HourCycleDefault, false
}

// DateTimeOptions mirror Intl.DateTimeFormat's options. The zero value formats like
// new Intl.DateTimeFormat(locale, {timeZone: "UTC"}): numeric year, month and day.
type DateTimeOptions struct {
	DateStyle, TimeStyle DateTimeStyle

	Weekday, Era, Year, Month, Day, DayPeriod, Hour, Minute, Second DateField
	FractionalSecondDigits                                          int // 0 (unset) or 1–3
	TimeZoneName                                                    ZoneName

	Hour12    *bool // overrides HourCycle, as in ECMA-402; use [B]
	HourCycle HourCycle

	// TimeZone is an IANA id ("Europe/Berlin", case-insensitive; aliases resolve to CLDR canonical ids),
	// "UTC", or an offset ("+05:30", "-08", "+0100"). Empty means Location, or UTC when Location is nil
	// too (a server has no meaningful system zone).
	TimeZone string
	// Location supplies the zone as a Go location; its name must be a known zone id for localized names.
	Location *time.Location

	NumberingSystem string // overrides the locale's -u-nu
	Calendar        string // overrides -u-ca; only the Gregorian calendar is formatted (see [DateTimeFormat])
}

// B returns a pointer to b, for [DateTimeOptions.Hour12].
func B(b bool) *bool { return &b }

// ErrDateTimeOptions is returned (wrapped) when options conflict, like Intl's TypeError for dateStyle or
// timeStyle combined with component options.
var ErrDateTimeOptions = errors.New("i18n: conflicting date-time options")

// DateTimeFormat formats instants for one locale and set of options, exactly like Intl.DateTimeFormat in
// V8/ICU 78 for Gregorian calendars. Create it once and reuse it.
//
// Calendars other than Gregorian (-u-ca, Calendar) are not implemented yet: such a format still uses
// the Gregorian calendar and reports "gregory".
type DateTimeFormat struct {
	loc  *Locale
	opts DateTimeOptions

	pattern    string
	hourCycle  HourCycle // [[HourCycle]]: set when the format shows hours
	dateStyle  DateTimeStyle
	timeStyle  DateTimeStyle
	components int // explicitly requested components (bits, see comp*)

	zone         *zone
	nu           string // numbering system in use
	sys          *NumberSystem
	digits       [10]string
	locID        string // resolvedOptions().locale
	icuHC        string // -u-hc on the ICU locale ("" = none)
	gen          *patternGenerator
	zf           *zoneFormatter
	interval     *intervalFormat // built on first FormatRange
	intervalOnce sync.Once
}

// component bits (V8's explicit_components_in_options).
const (
	compEra = 1 << iota
	compYear
	compMonth
	compWeekday
	compDay
	compDayPeriod
	compHour
	compMinute
	compSecond
	compTimeZoneName
	compFractionalSecondDigits
)

// DateTimeFormat compiles options for this locale (Intl.DateTimeFormat's constructor).
func (l *Locale) DateTimeFormat(o DateTimeOptions) (*DateTimeFormat, error) {
	f := &DateTimeFormat{loc: l, opts: o}

	if o.Calendar != "" && !wellFormedType(o.Calendar) {
		return nil, rangeErr("calendar")
	}
	if o.NumberingSystem != "" && !wellFormedType(o.NumberingSystem) {
		return nil, rangeErr("numberingSystem")
	}

	// Resolved locale: the -u- keywords relevant to DateTimeFormat (nu, ca, hc) when valid.
	tag := MustParseTag(l.Data.ID)
	extNu := l.Tag.Keyword("nu")
	if extNu != "" && !l.validNumberingSystem(extNu) {
		extNu = ""
	}
	extCa := l.Tag.Keyword("ca")
	extHc := l.Tag.Keyword("hc")
	if _, ok := ParseHourCycle(extHc); !ok {
		extHc = ""
	}
	nu := extNu
	resolvedNu := extNu
	if o.NumberingSystem != "" && l.validNumberingSystem(o.NumberingSystem) {
		if extNu != "" && extNu != o.NumberingSystem {
			resolvedNu = ""
		}
		nu = o.NumberingSystem
	}
	if nu == "" {
		nu = l.Data.Numbers.DefaultSystem
	}
	f.nu = nu
	f.sys, _ = l.numberSystem(nu)
	f.setDigits()

	hour12Set := o.Hour12 != nil
	optHC := o.HourCycle
	if hour12Set {
		optHC = HourCycleDefault
	}
	f.icuHC = extHc
	f.gen = l.newPatternGenerator(extHc, nu, false)
	hcDefault := f.gen.defaultHourCycle()
	hc := optHC
	if hc == HourCycleDefault && extHc != "" {
		hc, _ = ParseHourCycle(extHc)
	}
	if hour12Set {
		if *o.Hour12 {
			hc = H12
			if hcDefault == H11 || hcDefault == H12 {
				hc = hcDefault
			} else if cmpOr(l.Tag.Region, tag.Region) == "JP" { // V8 checks the locale's explicit region
				hc = H11
			}
		} else {
			hc = H23
			if hcDefault == H23 || hcDefault == H24 {
				hc = hcDefault
			}
		}
	} else if hc == HourCycleDefault {
		hc = hcDefault
	}

	zoneID := o.TimeZone
	if zoneID == "" && o.Location == nil {
		zoneID = "UTC"
	}
	z, err := l.set.resolveZone(zoneID, o.Location)
	if err != nil {
		return nil, err
	}
	f.zone = z

	skeleton, hasHour, err := f.componentSkeleton(hc)
	if err != nil {
		return nil, err
	}
	f.dateStyle, f.timeStyle = o.DateStyle, o.TimeStyle
	if o.DateStyle > ShortStyle || o.TimeStyle > ShortStyle {
		return nil, rangeErr("dateStyle/timeStyle")
	}
	dtfHC := HourCycleDefault
	if o.TimeStyle != NoStyle {
		dtfHC = hc
	}
	if o.DateStyle != NoStyle || o.TimeStyle != NoStyle {
		if f.components != 0 {
			return nil, ErrDateTimeOptions
		}
		f.pattern = f.stylePattern(o.DateStyle, o.TimeStyle, dtfHC)
	} else {
		need := f.components&(compWeekday|compYear|compMonth|compDay|compDayPeriod|compHour|compMinute|compSecond|compFractionalSecondDigits) == 0
		if need {
			skeleton += "yMd"
		}
		if hasHour {
			dtfHC = hc
		}
		f.pattern = replaceHourCycleInPattern(f.gen.bestPattern(skeleton, pgMatchHourFieldLength), dtfHC)
	}
	f.hourCycle = dtfHC

	// resolved locale id
	rt := tag
	if resolvedNu != "" {
		rt.setKeyword("nu", resolvedNu)
	}
	// only the Gregorian calendar is formatted, so only -u-ca-gregory stays on the resolved locale
	if extCa == "gregory" && (o.Calendar == "" || o.Calendar == extCa) {
		rt.setKeyword("ca", extCa)
	}
	if extHc != "" {
		keep := true
		if hour12Set || o.HourCycle != HourCycleDefault {
			if ehc, _ := ParseHourCycle(extHc); dtfHC != ehc {
				keep = false
			}
		}
		if keep {
			rt.setKeyword("hc", extHc)
		}
	}
	f.locID = rt.String()
	f.zf = l.newZoneFormatter(f.sys)
	return f, nil
}

// MustDateTimeFormat is DateTimeFormat that panics on invalid options.
func (l *Locale) MustDateTimeFormat(o DateTimeOptions) *DateTimeFormat {
	f, err := l.DateTimeFormat(o)
	if err != nil {
		panic(err)
	}
	return f
}

// wellFormedType checks a Unicode extension type value: one or more 3–8 alphanumeric subtags.
func wellFormedType(s string) bool {
	for _, part := range strings.Split(s, "-") {
		if len(part) < 3 || len(part) > 8 || !isAlnum(part) {
			return false
		}
	}
	return true
}

// validNumberingSystem is V8's IsValidNumberingSystem: a known numeric system.
func (l *Locale) validNumberingSystem(id string) bool {
	switch id {
	case "native", "traditio", "finance":
		return false
	}
	for _, s := range l.Data.Numbers.Systems {
		if s.ID == id {
			return true
		}
	}
	if l.set != nil {
		_, ok := slices.BinarySearchFunc(l.set.NumberingSystems, id, func(n NumberingSystem, s string) int { return strings.Compare(n.ID, s) })
		return ok
	}
	return false
}

func (f *DateTimeFormat) setDigits() {
	i := 0
	for _, r := range f.sys.Digits {
		if i < 10 {
			f.digits[i] = string(r)
		}
		i++
	}
	if i != 10 {
		for k := range f.digits {
			f.digits[k] = string(rune('0' + k))
		}
	}
}

// componentSkeleton builds the skeleton from component options in V8's order (Table 7 of ECMA-402 with
// the hour letter of the resolved hour cycle) and records explicit components.
func (f *DateTimeFormat) componentSkeleton(hc HourCycle) (string, bool, error) {
	o := &f.opts
	var b strings.Builder
	text := func(name string, v DateField, bit int, narrow, long, short string) error {
		switch v {
		case FieldNone:
			return nil
		case FieldNarrow:
			b.WriteString(narrow)
		case FieldLong:
			b.WriteString(long)
		case FieldShort:
			b.WriteString(short)
		default:
			return rangeErr(name)
		}
		f.components |= bit
		return nil
	}
	num := func(name string, v DateField, bit int, twoDigit, numeric string) error {
		switch v {
		case FieldNone:
			return nil
		case FieldTwoDigit:
			b.WriteString(twoDigit)
		case FieldNumeric:
			b.WriteString(numeric)
		default:
			return rangeErr(name)
		}
		f.components |= bit
		return nil
	}
	if err := text("weekday", o.Weekday, compWeekday, "EEEEE", "EEEE", "EEE"); err != nil {
		return "", false, err
	}
	if err := text("era", o.Era, compEra, "GGGGG", "GGGG", "GGG"); err != nil {
		return "", false, err
	}
	if err := num("year", o.Year, compYear, "yy", "y"); err != nil {
		return "", false, err
	}
	switch o.Month {
	case FieldNone:
	case FieldNarrow:
		b.WriteString("MMMMM")
	case FieldLong:
		b.WriteString("MMMM")
	case FieldShort:
		b.WriteString("MMM")
	case FieldTwoDigit:
		b.WriteString("MM")
	case FieldNumeric:
		b.WriteString("M")
	default:
		return "", false, rangeErr("month")
	}
	if o.Month != FieldNone {
		f.components |= compMonth
	}
	if err := num("day", o.Day, compDay, "dd", "d"); err != nil {
		return "", false, err
	}
	if err := text("dayPeriod", o.DayPeriod, compDayPeriod, "BBBBB", "BBBB", "B"); err != nil {
		return "", false, err
	}
	hourCh := byte('j')
	switch hc {
	case H11:
		hourCh = 'K'
	case H12:
		hourCh = 'h'
	case H23:
		hourCh = 'H'
	case H24:
		hourCh = 'k'
	}
	hasHour := o.Hour != FieldNone
	if err := num("hour", o.Hour, compHour, string([]byte{hourCh, hourCh}), string([]byte{hourCh})); err != nil {
		return "", false, err
	}
	if err := num("minute", o.Minute, compMinute, "mm", "m"); err != nil {
		return "", false, err
	}
	if err := num("second", o.Second, compSecond, "ss", "s"); err != nil {
		return "", false, err
	}
	if o.FractionalSecondDigits < 0 || o.FractionalSecondDigits > 3 {
		return "", false, rangeErr("fractionalSecondDigits")
	}
	if o.FractionalSecondDigits > 0 {
		f.components |= compFractionalSecondDigits
		b.WriteString("SSS"[:o.FractionalSecondDigits])
	}
	switch o.TimeZoneName {
	case ZoneNameNone:
	case ZoneLong:
		b.WriteString("zzzz")
	case ZoneShort:
		b.WriteString("z")
	case ZoneLongOffset:
		b.WriteString("OOOO")
	case ZoneShortOffset:
		b.WriteString("O")
	case ZoneLongGeneric:
		b.WriteString("vvvv")
	case ZoneShortGeneric:
		b.WriteString("v")
	default:
		return "", false, rangeErr("timeZoneName")
	}
	if o.TimeZoneName != ZoneNameNone {
		f.components |= compTimeZoneName
	}
	return b.String(), hasHour, nil
}

// replaceHourCycleInPattern is V8's ReplaceHourCycleInPattern.
func replaceHourCycleInPattern(pattern string, hc HourCycle) string {
	var repl byte
	switch hc {
	case H11:
		repl = 'K'
	case H12:
		repl = 'h'
	case H23:
		repl = 'H'
	case H24:
		repl = 'k'
	default:
		return pattern
	}
	var b strings.Builder
	replace := true
	var last byte
	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		switch c {
		case '\'':
			replace = !replace
			b.WriteByte(c)
		case 'H', 'h', 'K', 'k':
			if replace && last == 'd' {
				b.WriteByte(' ')
			}
			if replace {
				b.WriteByte(repl)
			} else {
				b.WriteByte(c)
			}
		default:
			b.WriteByte(c)
		}
		last = c
	}
	return b.String()
}

// hourCycleFromPattern is V8's HourCycleFromPattern.
func hourCycleFromPattern(pattern string) HourCycle {
	inQuote := false
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '\'':
			inQuote = !inQuote
		case 'K':
			if !inQuote {
				return H11
			}
		case 'h':
			if !inQuote {
				return H12
			}
		case 'H':
			if !inQuote {
				return H23
			}
		case 'k':
			if !inQuote {
				return H24
			}
		}
	}
	return HourCycleDefault
}

// replaceSkeletonHour is V8's ReplaceSkeleton: hour letters become the hour cycle's; a, b and B are dropped.
func replaceSkeletonHour(skeleton string, hc HourCycle) string {
	to := byte('H')
	switch hc {
	case H11:
		to = 'K'
	case H12:
		to = 'h'
	case H24:
		to = 'k'
	}
	var b strings.Builder
	for i := 0; i < len(skeleton); i++ {
		switch c := skeleton[i]; c {
		case 'a', 'b', 'B':
		case 'h', 'H', 'K', 'k':
			b.WriteByte(to)
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

var timeStyleSkeletons = [4]string{"jmmsszzzz", "jmmssz", "jmmss", "jmm"}

// stylePattern is V8's DateTimeStylePattern over ICU SimpleDateFormat::construct.
func (f *DateTimeFormat) stylePattern(ds, ts DateTimeStyle, hc HourCycle) string {
	dt := &f.loc.Data.DateTime
	var pattern string
	timePattern := ""
	if ts != NoStyle {
		if f.icuHC != "" || f.loc.Tag.Keyword("rg") != "" {
			g := f.loc.newPatternGenerator(f.icuHC, f.nu, true)
			timePattern = g.bestPattern(timeStyleSkeletons[ts-1], pgMatchNoOptions)
		}
		if timePattern == "" {
			timePattern = dt.TimeFormats[ts-1]
		}
	}
	switch {
	case ds != NoStyle && ts != NoStyle:
		glue := dt.AtTimeFormats[ds-1]
		if glue == "" {
			glue = dt.DateTimeFormats[ds-1]
		}
		pattern = simpleFormat(glue, timePattern, dt.DateFormats[ds-1])
	case ds != NoStyle:
		return dt.DateFormats[ds-1]
	default:
		pattern = timePattern
	}
	if hc == hourCycleFromPattern(pattern) {
		return pattern
	}
	skel := replaceSkeletonHour(patternSkeleton(pattern), hc)
	return replaceHourCycleInPattern(f.gen.bestPattern(skel, pgMatchHourFieldLength), hc)
}

// Pattern is the ICU date pattern in use, e.g. "MMM d, y, h:mm a".
func (f *DateTimeFormat) Pattern() string { return f.pattern }

// Format formats t in the format's time zone.
func (f *DateTimeFormat) Format(t time.Time) string {
	var b strings.Builder
	f.formatPattern(&b, f.pattern, t)
	return replaceUnicodeSpaces(b.String())
}

// FormatMillis formats a JavaScript time value (milliseconds since the Unix epoch).
func (f *DateTimeFormat) FormatMillis(ms int64) string { return f.Format(time.UnixMilli(ms)) }

// replaceUnicodeSpaces reverts ICU's U+202F (before AM/PM) and U+2009 (zh date-time) to U+0020, as V8
// does (crbug.com/1414292).
func replaceUnicodeSpaces(s string) string {
	if !strings.ContainsRune(s, ' ') && !strings.ContainsRune(s, ' ') {
		return s
	}
	return strings.NewReplacer(" ", " ", " ", " ").Replace(s)
}

// DateTimeResolved is Intl.DateTimeFormat's resolvedOptions().
type DateTimeResolved struct {
	Locale, Calendar, NumberingSystem, TimeZone string
	HourCycle                                   HourCycle // HourCycleDefault when the format shows no hours
	Hour12                                      *bool

	// Components are the pattern's fields as ECMA-402 names ("numeric", "2-digit", "short", …), "" when
	// absent or when a date or time style is used.
	Weekday, Era, Year, Month, Day, DayPeriod, Hour, Minute, Second, TimeZoneName string
	FractionalSecondDigits                                                        int

	DateStyle, TimeStyle string
}

var styleNames = [...]string{"", "full", "long", "medium", "short"}

// ResolvedOptions returns the resolved options, derived from the pattern like V8.
func (f *DateTimeFormat) ResolvedOptions() DateTimeResolved {
	r := DateTimeResolved{
		Locale: f.locID, Calendar: "gregory", NumberingSystem: f.nu, TimeZone: f.zone.id,
		HourCycle: f.hourCycle, DateStyle: styleNames[f.dateStyle], TimeStyle: styleNames[f.timeStyle],
	}
	if f.hourCycle != HourCycleDefault {
		r.Hour12 = B(f.hourCycle == H11 || f.hourCycle == H12)
	}
	if f.dateStyle != NoStyle || f.timeStyle != NoStyle {
		return r
	}
	p := f.pattern
	find := func(pairs ...string) string {
		for i := 0; i+1 < len(pairs); i += 2 {
			if strings.Contains(p, pairs[i]) {
				return pairs[i+1]
			}
		}
		return ""
	}
	r.Weekday = find("EEEEE", "narrow", "EEEE", "long", "EEE", "short", "ccccc", "narrow", "cccc", "long", "ccc", "short")
	r.Era = find("GGGGG", "narrow", "GGGG", "long", "GGG", "short")
	r.Year = find("yy", "2-digit", "y", "numeric", "YY", "2-digit", "Y", "numeric")
	r.Month = find("MMMMM", "narrow", "MMMM", "long", "MMM", "short", "MM", "2-digit", "M", "numeric",
		"LLLLL", "narrow", "LLLL", "long", "LLL", "short", "LL", "2-digit", "L", "numeric")
	r.Day = find("dd", "2-digit", "d", "numeric")
	r.DayPeriod = find("BBBBB", "narrow", "bbbbb", "narrow", "BBBB", "long", "bbbb", "long", "B", "short", "b", "short")
	r.Hour = find("HH", "2-digit", "H", "numeric", "hh", "2-digit", "h", "numeric", "kk", "2-digit", "k", "numeric", "KK", "2-digit", "K", "numeric")
	r.Minute = find("mm", "2-digit", "m", "numeric")
	r.Second = find("ss", "2-digit", "s", "numeric")
	for i := 0; i < len(p) && r.FractionalSecondDigits < 3; i++ {
		if p[i] == 'S' {
			r.FractionalSecondDigits++
		}
	}
	r.TimeZoneName = find("zzzz", "long", "z", "short", "OOOO", "longOffset", "O", "shortOffset", "vvvv", "longGeneric", "v", "shortGeneric")
	return r
}
