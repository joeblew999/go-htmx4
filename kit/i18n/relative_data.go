package i18n

// WidthCount is the number of display widths (Short, Long, Narrow), for data arrays indexed by Width.
const WidthCount = 3

// RelUnit identifies an Intl.RelativeTimeFormat unit.
type RelUnit uint8

// Relative time units.
const (
	RelYear RelUnit = iota
	RelQuarter
	RelMonth
	RelWeek
	RelDay
	RelHour
	RelMinute
	RelSecond
	RelUnitCount
)

var relUnitNames = [RelUnitCount]string{"year", "quarter", "month", "week", "day", "hour", "minute", "second"}

func (u RelUnit) String() string {
	if u < RelUnitCount {
		return relUnitNames[u]
	}
	return ""
}

// ParseRelUnit maps "day" or "days" (Intl accepts both) to a RelUnit.
func ParseRelUnit(s string) (RelUnit, bool) {
	for i, n := range relUnitNames {
		if s == n || s == n+"s" {
			return RelUnit(i), true
		}
	}
	return 0, false
}

// RelativeData is a locale's dateFields relative-time data, indexed [unit][Width].
type RelativeData struct {
	Units [RelUnitCount][WidthCount]RelativeUnit
}

// RelativeUnit is one unit at one width: numeric-auto phrases ("yesterday") and "in {0} days" patterns
// by plural category.
type RelativeUnit struct {
	Phrases []RelPhrase // relative-type-N, sorted by Offset
	Future  [PluralCatCount]string
	Past    [PluralCatCount]string
}

// RelPhrase is a relative-type-N phrase, e.g. {-1, "yesterday"}.
type RelPhrase struct {
	Offset int8
	Text   string
}

// ListType is Intl.ListFormat's type.
type ListType uint8

// List types.
const (
	ListConjunction ListType = iota
	ListDisjunction
	ListUnit
	ListTypeCount
)

// ListData is a locale's listPatterns indexed [ListType][Width], plus the contextual rule ICU applies in
// code (not CLDR data) for Spanish and Hebrew.
type ListData struct {
	Patterns [ListTypeCount][WidthCount]ListPattern
	Rule     uint8 // ListRuleNone, ListRuleEs, ListRuleHe
}

// List contextual rules.
const (
	ListRuleNone uint8 = iota
	ListRuleEs
	ListRuleHe
)

// ListPattern is a CLDR listPattern: "{0} and {1}" for two items; start, middle and end for more.
type ListPattern struct{ Two, Start, Middle, End string }

// UnitData is a locale's unit display patterns for Intl.NumberFormat style "unit".
type UnitData struct {
	Units []UnitPatterns // sorted by ID
	// compoundUnitPattern "per" by Width, e.g. "{0}/{1}", "{0} per {1}"
	Per [WidthCount]string
}

// UnitPatterns are one unit's patterns ("{0} km") by [Width][PluralCat], and its perUnitPattern by Width.
type UnitPatterns struct {
	ID       string // ECMA-402 identifier, e.g. "kilometer", "kilometer-per-hour"
	Patterns [WidthCount][PluralCatCount]string
	PerUnit  [WidthCount]string // e.g. "{0}/h"; "" when CLDR has none
}

// DurationData is a locale's digital duration patterns (CLDR durationUnit hm, hms, ms).
type DurationData struct {
	HM, HMS, MS string // e.g. "h:mm", "h:mm:ss", "m:ss"
}
