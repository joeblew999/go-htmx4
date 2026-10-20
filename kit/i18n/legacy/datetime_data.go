package i18n

// DateTimeData is a locale's Gregorian calendar and time zone display data for [DateTimeFormat],
// resolved through the CLDR parent chain at generate time.
type DateTimeData struct {
	// Months[context][width][month-1]: context 0 format, 1 stand-alone; width 0 abbreviated, 1 wide,
	// 2 narrow.
	Months [2][3][12]string
	// Days[context][width][weekday]: width 0 abbreviated, 1 wide, 2 narrow, 3 short; Sunday is 0.
	Days [2][4][7]string
	Eras [3][2]string // [abbreviated, wide, narrow][BC, AD]
	AmPm [3][2]string // format context [abbreviated, wide, narrow][AM, PM]
	// DayPeriods[width][period] (format context, abbreviated/wide/narrow) in ICU order: midnight, noon,
	// morning1, afternoon1, evening1, night1, morning2, afternoon2, evening2, night2; "" = none.
	DayPeriods     [3][10]string
	DayPeriodRules DayPeriodRules

	DateFormats     [4]string // dateFormats full, long, medium, short
	TimeFormats     [4]string // timeFormats full, long, medium, short
	DateTimeFormats [4]string // dateTimeFormats (standard), {1} = date, {0} = time
	AtTimeFormats   [4]string // dateTimeFormats-atTime

	// Skeletons is ICU's DateTimePatternGenerator pattern map in its iteration order (canonical items,
	// the standard patterns, then availableFormats child-first), see [BuildSkeletonPatterns].
	Skeletons   []SkeletonPattern
	AppendItems [16]string // appendItems by pattern generator field
	FieldNames  [16]string // field display names (dateFields displayName) by pattern generator field

	IntervalFallback string            // intervalFormatFallback, e.g. "{0} – {1}"
	Intervals        []IntervalFormats // intervalFormats, sorted by Skeleton

	Zone ZoneStrings
}

// DayPeriodRules are a locale's CLDR dayPeriodRules: the flexible day period of each hour (index into
// DateTimeData.DayPeriods, -1 = no rules) and whether midnight and noon are named.
type DayPeriodRules struct {
	Hours    [24]int8
	Midnight bool
	Noon     bool
}

// SkeletonPattern is one DateTimePatternGenerator entry: a skeleton and its pattern.
type SkeletonPattern struct {
	Skeleton  string // canonical skeleton, e.g. "yMMMd"
	Pattern   string // e.g. "MMM d, y"; "" for count-dependent entries ICU keeps without a pattern
	Specified bool   // the skeleton came from availableFormats (not derived from the pattern)
	Std       bool   // derived from the locale's standard date or time patterns only
}

// IntervalFormats are the interval patterns of one skeleton by greatest difference, in ICU's index
// order: era, year, month, day, am/pm, hour, minute ("" = none).
type IntervalFormats struct {
	Skeleton    string
	PatternList string // the seven patterns separated by U+001F (trailing empty ones omitted)
}

// ZoneStrings are a locale's time zone display strings (CLDR timeZoneNames).
type ZoneStrings struct {
	HourFormat     string      // "+HH:mm;-HH:mm"
	GMTFormat      string      // "GMT{0}"
	GMTZeroFormat  string      // "GMT"
	RegionFormat   string      // "{0} Time"
	FallbackFormat string      // "{1} ({0})"
	MetaZones      []ZoneNames // by metazone id, sorted by ID
	Zones          []ZoneNames // zone-specific names and exemplar cities by CLDR canonical zone id, sorted by ID
	// RegionNames are the region display names generic location names use, as "CODE\x1eName" entries
	// separated by U+001F, sorted by code.
	RegionNames string
}

// ZoneNames are the display names of a metazone or zone.
type ZoneNames struct {
	ID string
	// Names are the long generic, long standard, long daylight, short generic, short standard and short
	// daylight names and the exemplar city (zones only), separated by U+001F; "" = none.
	Names string
}

// TimeZoneData is the shared time zone data: every zone id Intl accepts with its CLDR canonical id,
// region and metazone history, and the metazones' reference (golden) zones.
type TimeZoneData struct {
	Zones     []ZoneInfo     // sorted by lower-case ID
	MetaZones []MetaZoneInfo // sorted by ID
}

// ZoneInfo is one time zone id.
type ZoneInfo struct {
	ID        string // e.g. "Asia/Kolkata"
	Canonical string // CLDR canonical id, e.g. "Asia/Calcutta"; "" when ID is canonical
	Region    string // ISO 3166 region of a canonical zone, "" = none
	Primary   bool   // the region's only or primary zone
	MetaZones []MetaZoneSpan
}

// MetaZoneSpan is a period in which a zone uses a metazone, [From, To) in Unix seconds.
type MetaZoneSpan struct {
	MetaZone string
	From, To int64
}

// MetaZoneInfo is a metazone's reference zone per region ("001" = the golden zone).
type MetaZoneInfo struct {
	ID    string
	Zones []RegionZone
}

// RegionZone is a region and a zone id.
type RegionZone struct{ Region, Zone string }
