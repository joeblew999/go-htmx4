// Package i18n is the server half of the app's internationalization: which locales ship, the request's locale,
// and translated messages (ICU MessageFormat, generated into typed accessors by kit/i18n/msggen) with their
// plural rules and the numbers inside them.
//
// Everything else (dates, times, time zones, relative times, durations, currencies, units, lists) is
// formatted in the browser with its own Intl by static/intl.js: the server writes the raw value and a readable
// fallback, e.g. <time datetime="…" data-intl="datetime">, and the viewer's browser words it in the page's
// language and the viewer's own time zone.
//
// The earlier server-side Intl implementation (CLDR tables, conformance goldens, browser drift checks) is kept,
// untouched and out of the build, as its own module in kit/i18n/legacy (`mise run i18n:legacy` tests it). Its
// generator also writes this package's locale table: `mise run i18n:generate`.
package i18n
