package i18n

// This file is the schema of the generated CLDR tables (see kit/i18n/cldrgen). Generated packages build
// values of these types as static composite literals: sorted slices, no maps, so TinyGo can lay them
// out at compile time instead of running initializers on every Worker request.

// Data is one generated CLDR data set: the locales an app ships plus the shared supplemental tables.
type Data struct {
	CLDRVersion     string // e.g. "48"
	CLDRJSONVersion string // cldr-json package tag, e.g. "48.2.1"
	Locales         []*LocaleData

	Likely          []Likely         // likelySubtags, sorted by From
	Parents         []Parent         // parentLocales (explicit parents only), sorted by Child
	LanguageAliases []Alias          // e.g. iw → he, sorted by From
	RegionAliases   []Alias          // e.g. UK → GB (first replacement), sorted by From
	Currencies      []CurrencyInfo   // ISO 4217 fraction data, sorted by Code
	RegionCurrency  []RegionCurrency // current tender per region, sorted by Region
	RegionHourCycle []RegionHours    // timeData, sorted by Region ("001" = world default)
	// NumberingSystems are all numeric (decimal-digit) systems, for -u-nu values a locale has no
	// symbols of its own for (they borrow the locale's latn symbols and patterns), sorted by ID.
	NumberingSystems []NumberingSystem
	// RootSymbols are CLDR root's explicit symbols for numbering systems that don't alias to the
	// locale's latn symbols (arab, arabext), sorted by ID.
	RootSymbols []SystemSymbols
	TimeZones   TimeZoneData
	Weeks       []RegionWeek // weekData, sorted by Region ("001" = world default)
}

// RegionWeek is a region's week data: first day (0 = Sunday … 6 = Saturday), minimal days in the first
// week, and weekend days (Intl.Locale getWeekInfo).
type RegionWeek struct {
	Region   string
	FirstDay int8
	MinDays  int8
	Weekend  []int8
}

// SystemSymbols are one numbering system's symbols.
type SystemSymbols struct {
	ID      string
	Symbols NumberSymbols
}

// NumberingSystem is a numeric numbering system's ten digits, e.g. {"thai", "๐๑๒๓๔๕๖๗๘๙"}.
type NumberingSystem struct{ ID, Digits string }

// Likely maps a partial locale id ("und-Arab", "zh-TW") to its maximal form ("ar-Arab-EG").
type Likely struct{ From, To string }

// Parent is an explicit parentLocales entry, e.g. {"en-IN", "en-001"}.
type Parent struct{ Child, Parent string }

// Alias is a deprecated or legacy code and its replacement.
type Alias struct{ From, To string }

// CurrencyInfo is supplemental currencyData fractions for one code (DEFAULT for unlisted codes).
type CurrencyInfo struct {
	Code         string
	Digits       int8
	Rounding     int8 // rounding increment in units of the last digit (0 = none)
	CashDigits   int8
	CashRounding int8
}

// RegionCurrency is the currency in current use in a region.
type RegionCurrency struct{ Region, Currency string }

// RegionHours is a region's preferred and allowed hour cycles from timeData ("h", "H", "hb"…).
type RegionHours struct{ Region, Preferred, Allowed string }

// LocaleData is one locale, fully resolved through its CLDR parent chain at generate time.
type LocaleData struct {
	ID          string // canonical id as shipped, e.g. "pt-BR"
	DataID      string // cldr-json directory the data came from, e.g. "pt"
	Maximal     string // likely-subtags maximized id, e.g. "pt-Latn-BR"
	RTL         bool   // layout characterOrder is right-to-left
	NativeName  string // the locale's name in its own language, e.g. "português (Brasil)"
	Cardinal    PluralRules
	Ordinal     PluralRules
	Ranges      []PluralRange
	Numbers     NumbersData
	LangNames   []Name // language display names (allowlisted at generate time), sorted by Code
	RegionNames []Name
	ScriptNames []Name
	DateTime    DateTimeData
	Relative    RelativeData
	Lists       ListData
	Duration    DurationData
}

// Name is a display name for a code, e.g. {"de", "Deutsch"}.
type Name struct{ Code, Name string }

// PluralRange is a pluralRanges entry: a range from Start to End takes Result.
type PluralRange struct{ Start, End, Result PluralCat }

// NumbersData is the locale's numbers.json + currencies.json.
type NumbersData struct {
	DefaultSystem string
	NativeSystem  string
	MinGrouping   int8
	Systems       []NumberSystem // default system first; "latn" and the native system when different
	// ExtraSymbols are resolved symbols for other numbering systems where the locale's CLDR XML
	// overrides what root and latn would give (e.g. zh-Hant arab's minus sign), sorted by ID.
	ExtraSymbols []SystemSymbols
	Currencies   []CurrencyNames
	// currencyFormats unitPattern-count-*: "{0} {1}" by PluralCat, for currencyDisplay "name".
	CurrencyUnitPattern [PluralCatCount]string
}

// NumberSystem is the symbols and patterns of one numbering system in a locale.
type NumberSystem struct {
	ID      string
	Digits  string // ten digit runes, e.g. "٠١٢٣٤٥٦٧٨٩"
	Symbols NumberSymbols

	Decimal    NumPattern
	Percent    NumPattern
	Scientific NumPattern
	Currency   NumPattern
	Accounting NumPattern
	// currencyFormats standard-alphaNextToNumber / accounting-alphaNextToNumber: used when a currency
	// symbol with letters touches the number (zero value = not present).
	CurrencyAlpha   NumPattern
	AccountingAlpha NumPattern

	CompactShort    []CompactPattern
	CompactLong     []CompactPattern
	CompactCurrency []CompactPattern // currencyFormats short standard

	// currencySpacing insertBetween (currencyMatch/surroundingMatch are evaluated per symbol at
	// generate time into CurrencyNames.Spacing).
	SpaceBeforeCurrency string // number ¤
	SpaceAfterCurrency  string // ¤ number
}

// NumberSymbols are the locale symbols for one numbering system.
type NumberSymbols struct {
	Decimal, Group, Percent, PerMille, Minus, Plus, Exponential, Infinity, NaN, ApproximatelySign string
	CurrencyDecimal, CurrencyGroup                                                                string // "" = same as Decimal/Group
}

// NumPattern is a parsed LDML number pattern. Affixes use private-use runes for pattern specials.
type NumPattern struct {
	Set            bool
	PosPre, PosSuf string
	NegPre, NegSuf string
	HasNeg         bool
	MinInt         int8
	MinFrac        int8
	MaxFrac        int8
	Group1         int8 // primary grouping size, 0 = none
	Group2         int8 // secondary grouping size (0 = same as primary)
	MinExp         int8 // scientific: minimum exponent digits
}

// Affix special runes (the generator replaces unquoted pattern specials with these).
const (
	AffixCurrency = '\uE000'
	AffixPercent  = '\uE001'
	AffixMinus    = '\uE002'
	AffixPlus     = '\uE003'
	AffixPerMille = '\uE004'
	AffixApprox   = '\uE005'
)

// CompactPattern is one decimalFormats short/long entry, e.g. 10000-count-one → "00K".
type CompactPattern struct {
	Magnitude int8 // log10 of the type (3 for 1000)
	Cat       PluralCat
	Exact     string // explicit-value entry ("1000-count-1" → "1"): used only when the shown number is exactly this
	Zeros     int8   // number of '0's: integer digits shown; 0 = "0" pattern (no compaction)
	Pre, Suf  string
}

// CurrencyNames are one currency's display strings in a locale.
type CurrencyNames struct {
	Code        string
	Symbol      string // "" = code
	Narrow      string // "" = Symbol
	DisplayName string
	Names       [PluralCatCount]string // displayName-count-*; "" = fall back to Other, then DisplayName
	// Spacing bits (LDML currencySpacing): bit 0 symbol's first rune matches beforeCurrency (number ¤),
	// bit 1 symbol's last rune matches afterCurrency (¤ number); bits 2-3 the same for Narrow, 4-5 for Code.
	Spacing uint8
}
