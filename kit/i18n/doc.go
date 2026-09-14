// Package i18n is internationalization for Go on Cloudflare Workers (TinyGo) and standard Go: locale
// negotiation and CLDR-exact formatting with the options and output of JavaScript's Intl, computed on
// the server so every page is complete for users, crawlers and AI fetchers alike.
//
// Data comes from Unicode CLDR as generated Go tables for exactly the locales an app ships
// (kit/i18n/cldrgen, `go run ./cmd/cldrgen`); [cldr.Data] is a ready set of 14 locales. No reflection,
// no runtime JSON, no maps in the tables: TinyGo lays them out at compile time.
//
// Locales:
//
//	loc := cldr.Data.Match(r.Header.Get("Accept-Language")) // or Data.Locale(tag)
//	loc.Lang(), loc.Dir()                                  // <html lang dir>
//
// Numbers ([Locale.NumberFormat], Intl.NumberFormat's options): decimal, percent and currency styles;
// symbol, narrow symbol, code and name currency displays; accounting; standard, compact, scientific and
// engineering notation; significant and fraction digits with rounding priority, increments and all nine
// rounding modes; sign display; grouping; numbering systems (-u-nu). Money is exact: use [Minor].
//
// Plurals: cardinal, ordinal and ranges ([PluralRules], [LocaleData.SelectRange]).
//
// Conformance: kit/i18n/intltest records what Intl (workerd, identical to Chrome) outputs for thousands
// of cases, and the tests require the same output from this package, byte for byte.
//
// [cldr.Data]: https://pkg.go.dev/github.com/joeblew999/go-htmx4/kit/i18n/cldr#Data
package i18n
