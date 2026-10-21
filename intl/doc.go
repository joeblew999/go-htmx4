// Package intl is a gsx app's language: the runtime's own Intl, written in markup through gsx type renderers. Go
// decides nothing about a language; it routes the request to a locale (WithRequest) and passes values.
//
// Two types render in .gsx (gsx.toml [renderers], the gsx Translation keys and Package renderers patterns):
//
//   - Msg, a message from the app's catalogs (ICU MessageFormat): { locales.BoardNotes(n) }, in text and attributes.
//     go tool intl gen writes a constructor per key, so a missing key is a compile error.
//   - Value, any Intl call: { intl.Date(t, `{"dateStyle":"full"}`) }, { intl.Of("DisplayNames", "of",
//     `{"type":"region"}`, intl.JSON("JP")) }. It renders as Intl's result inside an element js/intl-browser.mjs
//     redoes in the viewer's own zone; v.Plain() is the text alone, for attributes.
//
// Both are formatted by js/intl.mjs, the same module in the Worker (where Go calls it through js/intl-worker.mjs) and
// in the browser. Outside a Worker there is no Intl but workerd's: tests run under intltest. intl/README.md is the
// whole setup.
package intl
