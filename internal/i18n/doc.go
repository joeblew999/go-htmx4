// Package i18n is how Go reaches the app's one source of language: the runtime's own Intl, through
// worker/intl.mjs. Go decides nothing about a language itself; it routes the request to a locale and passes values.
//
// worker/intl.mjs runs in two places with the same V8 Intl: on the Worker (workerd locally), where Go calls it
// through worker/gointl.mjs, and in the browser (static/intl.js, served as /static/intl-core.mjs). It does:
//
//   - every message (Locale.Message): ICU MessageFormat from the catalogs msggen writes (worker/messages_gen.mjs), its
//     plural forms by Intl.PluralRules and its numbers by Intl.NumberFormat;
//   - every value (Locale.Format): dates, times, zones, relative times, durations, numbers, currencies, units, lists;
//   - a locale's name, direction and likely subtags (Intl.DisplayNames, Intl.Locale);
//   - reading a number typed the language's way back to a plain decimal (Locale.ParseNumber).
//
// Pages arrive finished; the browser re-formats only what the server couldn't know: the viewer's own time zone, and
// relative times as time passes. Markup carries raw values with their Intl options (<time datetime data-intl=…>).
//
// The shipped locales are I18N_LOCALES in mise.toml (the generated locales.Shipped). Outside the Worker there is no
// Intl but workerd's: go test runs under internal/i18n/i18ntest, which serves worker/gointl.mjs from workerd.
package i18n
