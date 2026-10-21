// Package i18n is the app's internationalization over the runtime's own Intl. Which locales ship is
// I18N_LOCALES in mise.toml (the generated locales.Shipped); the request's locale comes from its URL; UI text
// comes from ICU MessageFormat catalogs, generated into typed accessors by internal/i18n/msggen.
//
// Everything a language decides goes through worker/intl.mjs, the same module with the same V8 Intl on the
// Worker (workerd locally) and in the browser:
//
//   - On the Worker, Go calls it through Locale.Format: dates, times, zones, relative times, durations, numbers,
//     currencies, units, lists; a message's plural category (Intl.PluralRules) and the numbers in it
//     (Intl.NumberFormat); a locale's name, direction and likely subtags (Intl.DisplayNames, Intl.Locale). Pages
//     arrive finished, and emails or PDFs can use the same call.
//   - In the browser, static/intl.js imports the same module (served as /static/intl-core.mjs) and re-formats only
//     what the server couldn't know: the viewer's own time zone, and relative times as time passes.
//   - Markup carries the raw value and its options (<time datetime data-intl="datetime" data-opts='{…}'>), so either
//     side can word it.
//   - Forms send canonical values back (decimal strings, UTC instants); ParseNumber reads a number typed the
//     language's way ("1.234,5") back to "1234.5".
//
// There is no Intl without a JS runtime: under go test, Format reports !ok, a plural takes its "other" case and a
// number its plain digits. What a language says is tested on workerd: the smoke test (e2e/smoke) and the catalogs'
// plural check (locales).
package i18n
