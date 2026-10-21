// What Go calls (internal/i18n): worker/intl.mjs with the app's message catalogs, every argument a string ("" for
// none), so Go in the Worker's wasm and go test (internal/i18n/i18ntest, the same module on workerd) call it the
// same way. worker/index.mjs puts it on globalThis.goIntl.
import * as intl from "./intl.mjs";
import catalogs from "./messages_gen.mjs";

export const goIntl = {
  format(lang, kind, value, opts, end, unit, tz, hc) {
    const extra = {};
    for (const [k, v] of Object.entries({ end, unit, tz, hc })) if (v) extra[k] = v;
    return intl.format(lang, kind, value, opts, extra);
  },
  parseNumber: (lang, text) => intl.parseNumber(lang, text),
  // Message index of catalog (a locale id, or "en-XA": pseudo-localized) in lang.
  message: (lang, catalog, index, ...args) => intl.message(lang, catalogs[catalog]?.[Number(index)], args),
};
