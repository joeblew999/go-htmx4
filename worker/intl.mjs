// The app's one Intl formatter, run in two places with the same V8 Intl:
//   - on the Worker: worker/index.mjs puts it on globalThis.goIntl and Go calls it (kit/i18n Format), so every
//     page, email or PDF the server writes is already formatted;
//   - in the browser: static/intl.js imports it (as /static/intl-core.mjs) and re-formats only what the server
//     couldn't know: the viewer's own time zone and "3 minutes ago" as time passes.
// Pure functions, no DOM. Intl formats but doesn't parse: parseNumber reads a number typed in a language's
// own way back to a plain decimal, so values make the round trip.
//
// format(lang, kind, value, opts, extra) → string ("" when this Intl can't: the caller keeps its fallback)
//   kinds and their values:
//   datetime       Intl.DateTimeFormat                 an instant
//   range          Intl.DateTimeFormat formatRange     the start; extra.end: the end
//   relative       Intl.RelativeTimeFormat             an instant, relative to extra.now (default: now);
//                                                      or a number with extra.unit ("day", "hour"…)
//   number         Intl.NumberFormat                   a decimal string
//   number-range   Intl.NumberFormat formatRange       the start; extra.end: the end
//   plural         Intl.PluralRules select             a number (with extra.end: selectRange)
//   list           Intl.ListFormat                     a JSON array of strings
//   duration       Intl.DurationFormat                 a JSON object ({"days":1,"hours":2,…})
//   display-name   Intl.DisplayNames                   a code; opts.type: language, region, script, currency,
//                                                      calendar, dateTimeField
//   collate        Intl.Collator                       a JSON array of strings, shown sorted
//   segment        Intl.Segmenter                      text; shows the count and the segments
//   locale         Intl.Locale                         maximize, minimize, calendars, collations, hourCycles,
//                                                      numberingSystems, timeZones, direction, firstDay, weekend
//   supported      Intl.supportedValuesOf              calendar, collation, currency, numberingSystem, timeZone,
//                                                      unit; shows the count and the first few
//   zone           a time zone's name                  an IANA id
// extra: { end, unit, tz (dates' zone, default UTC), hc ("h12"/"h23": the viewer's clock), now (ms) }

// Relative time units: the largest unit that keeps the number readable ("3 minutes ago", not "180 seconds ago").
const SECONDS_MAX = 45, MINUTES_MAX = 45 * 60, HOURS_MAX = 22 * 3600, DAYS_MAX = 26 * 86400, MONTHS_MAX = 320 * 86400;
export function elapsedUnit(seconds) {
  const abs = Math.abs(seconds);
  if (abs < SECONDS_MAX) return [Math.round(seconds), "second"];
  if (abs < MINUTES_MAX) return [Math.round(seconds / 60), "minute"];
  if (abs < HOURS_MAX) return [Math.round(seconds / 3600), "hour"];
  if (abs < DAYS_MAX) return [Math.round(seconds / 86400), "day"];
  if (abs < MONTHS_MAX) return [Math.round(seconds / (30 * 86400)), "month"];
  return [Math.round(seconds / (365 * 86400)), "year"];
}

// dateOptions adds the zone and clock to options that don't set their own.
// "viewerClock": false keeps the language's own clock (e.g. to show what the default looks like).
function dateOptions(opts, extra) {
  const { viewerClock, ...o } = opts;
  if (!o.timeZone) o.timeZone = extra.tz || "UTC";
  if (extra.hc && viewerClock !== false && o.hour12 === undefined && !o.hourCycle && (o.hour || o.timeStyle)) o.hourCycle = extra.hc;
  return o;
}

function instant(s) {
  const t = Date.parse(s);
  if (Number.isNaN(t)) throw new RangeError("bad instant " + s);
  return t;
}

// zoneName: "Europe/Berlin · Central European Time (GMT+2)", in lang.
export function zoneName(lang, id, now = Date.now()) {
  const part = (style) =>
    new Intl.DateTimeFormat(lang, { timeZone: id, timeZoneName: style }).formatToParts(now).find((p) => p.type === "timeZoneName")?.value;
  let label = id.replaceAll("_", " ");
  try {
    label += " · " + part("longGeneric") + " (" + part("shortOffset") + ")";
  } catch {}
  return label;
}

const kinds = {
  datetime: (lang, v, o, x) => new Intl.DateTimeFormat(lang, dateOptions(o, x)).format(instant(v)),
  range: (lang, v, o, x) => new Intl.DateTimeFormat(lang, dateOptions(o, x)).formatRange(instant(v), instant(x.end)),
  relative(lang, v, o, x) {
    if (x.unit) return new Intl.RelativeTimeFormat(lang, o).format(Number(v), x.unit);
    const [value, unit] = elapsedUnit((instant(v) - (x.now ?? Date.now())) / 1000);
    return new Intl.RelativeTimeFormat(lang, { numeric: "auto", ...o }).format(value, unit);
  },
  // A decimal string, not a float: Intl.NumberFormat formats strings exactly ("1234567.891" keeps its digits).
  number: (lang, v, o) => new Intl.NumberFormat(lang, o).format(v),
  "number-range": (lang, v, o, x) => new Intl.NumberFormat(lang, o).formatRange(v, x.end),
  plural(lang, v, o, x) {
    const pr = new Intl.PluralRules(lang, o);
    return x.end ? pr.selectRange(Number(v), Number(x.end)) : pr.select(Number(v));
  },
  list: (lang, v, o) => new Intl.ListFormat(lang, o).format(JSON.parse(v)),
  duration(lang, v, o) {
    if (!Intl.DurationFormat) throw new TypeError("no Intl.DurationFormat");
    const d = JSON.parse(v);
    // Display "auto" hides zero units, so an all-zero duration would be "": show "0 sec" instead.
    return new Intl.DurationFormat(lang, o).format(d) || new Intl.DurationFormat(lang, { ...o, secondsDisplay: "always" }).format(d);
  },
  "display-name": (lang, v, o) => new Intl.DisplayNames(lang, { type: "language", ...o }).of(v),
  collate: (lang, v, o) => JSON.parse(v).sort(new Intl.Collator(lang, o).compare).join(" · "),
  segment(lang, v, o) {
    const parts = [...new Intl.Segmenter(lang, o).segment(v)].filter((s) => s.isWordLike ?? s.segment.trim() !== "");
    return parts.length + " · " + parts.map((s) => s.segment.trim()).join(" | ");
  },
  locale(lang, v) {
    const base = new Intl.Locale(lang);
    if (v === "maximize" || v === "minimize") return base[v]().toString();
    // Region-dependent answers (time zones, week, clocks) need the region: "de" → de-Latn-DE.
    const loc = base.maximize();
    const get = (name) => {
      const m = loc["get" + name[0].toUpperCase() + name.slice(1)];
      const x = typeof m === "function" ? m.call(loc) : loc[name];
      if (x === undefined) throw new TypeError("no " + name);
      return x;
    };
    switch (v) {
      case "direction":
        return get("textInfo").direction;
      case "firstDay":
      case "weekend":
      case "minimalDays": {
        const x = get("weekInfo")[v];
        if (x === undefined) throw new TypeError("no " + v);
        return Array.isArray(x) ? x.join(", ") : String(x);
      }
      default:
        return get(v).join(", ");
    }
  },
  supported(lang, v) {
    const all = Intl.supportedValuesOf(v);
    return all.length + " · " + all.slice(0, 6).join(", ") + (all.length > 6 ? ", …" : "");
  },
  zone: (lang, v, o, x) => zoneName(lang, v, x.now),
};

// format is "" when this Intl can't (unknown kind, value or option): the caller keeps its fallback.
export function format(lang, kind, value, opts = {}, extra = {}) {
  const f = kinds[kind];
  if (!f) return "";
  try {
    return String(f(lang, value, typeof opts === "string" ? (opts ? JSON.parse(opts) : {}) : opts, extra) ?? "");
  } catch {
    return "";
  }
}

// fullySupports: this Intl has data for lang itself, not a fallback to another language (trimmed ICU data in some
// browsers resolves e.g. hi to en). Callers keep the server's text when it doesn't.
export function fullySupports(lang) {
  try {
    return Intl.DateTimeFormat.supportedLocalesOf([lang]).length > 0 &&
      new Intl.NumberFormat(lang).resolvedOptions().locale.split("-")[0] === lang.split("-")[0];
  } catch {
    return false;
  }
}

// parseNumber reads a number written the language's way ("1.234,5" in German, "١٬٢٣٤٫٥" in Arabic) back to a plain
// decimal string ("1234.5"), using the separators and digits this Intl formats with. "" when it isn't a number.
export function parseNumber(lang, text) {
  let parts;
  try {
    parts = new Intl.NumberFormat(lang).formatToParts(-12345.6);
  } catch {
    return "";
  }
  const sym = (type) => parts.find((p) => p.type === type)?.value ?? "";
  const group = sym("group"), decimal = sym("decimal") || ".";
  const digits = new Intl.NumberFormat(lang, { useGrouping: false }).format(9876543210).split("").reverse(); // 0…9
  let s = text.trim().replace(/[‎‏؜  \s]/g, "");
  if (group) s = s.split(group.replace(/[  ]/g, "")).join("");
  let out = "";
  for (const ch of s) {
    const d = digits.indexOf(ch);
    if (d >= 0) out += d;
    else if (/[0-9]/.test(ch)) out += ch;
    else if (ch === decimal) out += ".";
    else if (ch === "-" || ch === "−") out += "-";
    else if (ch === "+") continue;
    else return "";
  }
  return /^-?(\d+(\.\d+)?|\.\d+)$/.test(out) ? out : "";
}
