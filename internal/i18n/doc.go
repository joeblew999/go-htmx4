// Package i18n is the server half of the app's internationalization: which locales ship (I18N_LOCALES in
// mise.toml), the request's locale, translated messages (ICU MessageFormat, generated into typed accessors by
// internal/i18n/msggen) with their plural rules, and the plain numbers inside them.
//
// One Intl, run twice. Dates, times, zones, relative times, durations, numbers, currencies, units and lists go
// through worker/intl.mjs:
//
//   - On the server, Locale.Format calls it with the Worker's own V8 Intl, so pages arrive finished (emails and
//     PDFs can use the same call). The wasm carries no CLDR formatting data.
//   - In the browser, static/intl.js imports the same module (served as /static/intl-core.mjs) and re-formats only
//     what the server couldn't know: the viewer's own time zone, and relative times as time passes.
//   - Markup carries the raw value and its options (<time datetime data-intl="datetime" data-opts='{…}'>), so either
//     side can word it. Under go run and go test there is no JS runtime: Format reports !ok and the caller writes a
//     plain fallback (2026-07-04 15:30 UTC). The smoke test checks the server's output, e2e the browser's.
//   - Forms send canonical values back (decimal strings, UTC instants); ParseNumber reads a number typed the
//     language's way ("1.234,5") back to "1234.5".
//
// internal/i18n/cldr is the generated locale table (names, direction, plural rules, digits: a few KB). Never edit it:
// `mise run i18n:generate` writes it, and testdata/numbers.json, with the generator at the git tag i18n-generator.
package i18n
