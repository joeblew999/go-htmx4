package i18n

// This file is the schema of the generated locale table (internal/i18n/cldr, written from CLDR by `mise run
// i18n:generate`). It holds only what the server itself needs: which locales ship, their names and text direction,
// the plural rules that pick a message's wording, and the digits and separators for numbers inside messages.
// Dates, times, currencies, units and the rest go through the runtime's own Intl (Locale.Format), so none of that
// data is here.

// Data is the set of locales an app ships.
type Data struct {
	CLDRVersion     string // e.g. "48"
	CLDRJSONVersion string // cldr-json package tag, e.g. "48.2.1"
	Locales         []*LocaleData
}

// LocaleData is one shipped locale.
type LocaleData struct {
	ID         string // canonical id as shipped, e.g. "pt-BR"
	Maximal    string // likely-subtags maximized id, e.g. "pt-Latn-BR"
	Parent     string // nearest shipped locale on CLDR's parent chain ("en" for en-IN), "" for none
	RTL        bool   // layout characterOrder is right-to-left
	NativeName string // the locale's name in its own language, e.g. "português (Brasil)"
	Currency   string // the maximized region's currency, e.g. "BRL"
	Cardinal   PluralRules
	Ordinal    PluralRules
	Numbers    NumbersData
}

// NumbersData is what numbers inside messages need: the default numbering system's digits and symbols, and
// its decimal pattern's grouping.
type NumbersData struct {
	DefaultSystem string // e.g. "latn", "arab"
	NativeSystem  string // the locale's native digits when they aren't the default, e.g. "deva" for hi
	Digits        string // ten digit runes, e.g. "٠١٢٣٤٥٦٧٨٩"
	Decimal       string
	Group         string
	Minus         string
	Group1        int8 // primary grouping size, 0 = none
	Group2        int8 // secondary grouping size (0 = same as primary): 2 for en-IN's 12,34,567
	MinGrouping   int8 // minimum grouping digits: 2 means 1234 isn't grouped (es, pt-PT)
}
