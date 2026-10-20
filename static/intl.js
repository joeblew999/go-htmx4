// Formats dates, times, durations and rich numbers in the viewer's browser with its own Intl (views/intl.go). The
// server writes the raw value, the Intl options and a readable fallback; this script puts the browser's wording in
// the page's language and the viewer's time zone in its place:
//
//   <time datetime="…" data-intl="datetime" data-opts='{"dateStyle":"full"}'>2026-07-04 15:30 UTC</time>
//
// data-intl kinds (the value is data-value, else datetime, else value; data-opts is a JSON object of options):
//   datetime       Intl.DateTimeFormat                 value: an instant
//   range          Intl.DateTimeFormat formatRange     value: the start, data-end: the end
//   relative       Intl.RelativeTimeFormat             value: an instant, relative to now (refreshed every 30 s);
//                                                      or a number with data-unit ("day", "hour"…)
//   number         Intl.NumberFormat                   value: a decimal string
//   number-range   Intl.NumberFormat formatRange       value: the start, data-end: the end
//   plural         Intl.PluralRules select             value: a number (with data-end: selectRange)
//   list           Intl.ListFormat                     value: a JSON array of strings
//   duration       Intl.DurationFormat                 value: a JSON object ({"days":1,"hours":2,…})
//   display-name   Intl.DisplayNames                   value: a code; opts.type: language, region, script,
//                                                      currency, calendar, dateTimeField
//   collate        Intl.Collator                       value: a JSON array of strings, shown sorted
//   segment        Intl.Segmenter                      value: text; shows the count and the segments
//   locale         Intl.Locale                         value: maximize, minimize, calendars, collations,
//                                                      hourCycles, numberingSystems, timeZones, direction,
//                                                      firstDay, weekend
//   supported      Intl.supportedValuesOf              value: calendar, collation, currency, numberingSystem,
//                                                      timeZone, unit; shows the count and the first few
//   zone           a time zone's name                  value: an IANA id
//   browser-zone   the browser's own time zone id
//
// data-template, if set, is a translated message with the result's place marked by U+E000 (views/intl.go slot).
// The language is the nearest lang attribute. Dates use <html data-tz> (the viewer's chosen zone) or the browser's
// own, and <html data-hc> (the viewer's chosen clock). A value or option this browser's Intl rejects keeps the
// fallback. <select data-intl-zones> gets every zone this browser knows, each named in the page's language.
//
// Elements are formatted on load and whenever they're added or changed: htmx swaps, hx-ws pushes and morphs all
// go through one MutationObserver.
(() => {
  const SLOT = "";
  const root = document.documentElement;

  // Relative time units: the largest unit that keeps the number readable ("3 minutes ago", not "180 seconds ago").
  const SECONDS_MAX = 45, MINUTES_MAX = 45 * 60, HOURS_MAX = 22 * 3600, DAYS_MAX = 26 * 86400, MONTHS_MAX = 320 * 86400;
  function elapsedUnit(seconds) {
    const abs = Math.abs(seconds);
    if (abs < SECONDS_MAX) return [Math.round(seconds), "second"];
    if (abs < MINUTES_MAX) return [Math.round(seconds / 60), "minute"];
    if (abs < HOURS_MAX) return [Math.round(seconds / 3600), "hour"];
    if (abs < DAYS_MAX) return [Math.round(seconds / 86400), "day"];
    if (abs < MONTHS_MAX) return [Math.round(seconds / (30 * 86400)), "month"];
    return [Math.round(seconds / (365 * 86400)), "year"];
  }

  function browserZone() {
    try {
      return new Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC";
    } catch {
      return "UTC";
    }
  }

  // viewerZone: the zone the viewer chose, if this browser knows it; else the browser's own.
  function viewerZone() {
    const chosen = root.dataset.tz;
    if (chosen) {
      try {
        new Intl.DateTimeFormat("en", { timeZone: chosen });
        return chosen;
      } catch {}
    }
    return browserZone();
  }

  // dateOptions adds the viewer's zone and clock to options that don't set their own.
  function dateOptions(opts) {
    const o = { ...opts };
    if (!o.timeZone) o.timeZone = viewerZone();
    const hc = root.dataset.hc;
    if (hc && o.hour12 === undefined && !o.hourCycle && (o.hour || o.timeStyle)) o.hourCycle = hc;
    return o;
  }

  function parse(json, fallback) {
    if (!json) return fallback;
    try {
      return JSON.parse(json);
    } catch {
      return fallback;
    }
  }

  function instant(s) {
    const t = Date.parse(s);
    if (Number.isNaN(t)) throw new RangeError("bad instant " + s);
    return t;
  }

  // zoneName: "Europe/Berlin · Central European Time (GMT+2)", in lang.
  function zoneName(lang, id) {
    const part = (style) =>
      new Intl.DateTimeFormat(lang, { timeZone: id, timeZoneName: style }).formatToParts(Date.now()).find((p) => p.type === "timeZoneName")?.value;
    let label = id.replaceAll("_", " ");
    try {
      label += " · " + part("longGeneric") + " (" + part("shortOffset") + ")";
    } catch {}
    return label;
  }

  const kinds = {
    datetime: (lang, v, o) => new Intl.DateTimeFormat(lang, dateOptions(o)).format(instant(v)),
    range: (lang, v, o, el) => new Intl.DateTimeFormat(lang, dateOptions(o)).formatRange(instant(v), instant(el.dataset.end)),
    relative(lang, v, o, el) {
      if (el.dataset.unit) return new Intl.RelativeTimeFormat(lang, o).format(Number(v), el.dataset.unit);
      const rtf = new Intl.RelativeTimeFormat(lang, { numeric: "auto", ...o });
      const when = instant(v);
      if (el.tagName === "TIME") el.title = new Intl.DateTimeFormat(lang, dateOptions({ dateStyle: "medium", timeStyle: "short" })).format(when);
      const [value, unit] = elapsedUnit((when - Date.now()) / 1000);
      return rtf.format(value, unit);
    },
    // A decimal string, not a float: Intl.NumberFormat formats strings exactly ("1234567.891" keeps its digits).
    number: (lang, v, o) => new Intl.NumberFormat(lang, o).format(v),
    list: (lang, v, o) => new Intl.ListFormat(lang, o).format(JSON.parse(v)),
    duration(lang, v, o) {
      if (!Intl.DurationFormat) throw new TypeError("no Intl.DurationFormat");
      const d = JSON.parse(v);
      const text = new Intl.DurationFormat(lang, o).format(d);
      // Display "auto" hides zero units, so an all-zero duration would be "": show "0 sec" instead.
      return text || new Intl.DurationFormat(lang, { ...o, secondsDisplay: "always" }).format(d);
    },
    "number-range": (lang, v, o, el) => new Intl.NumberFormat(lang, o).formatRange(v, el.dataset.end),
    plural(lang, v, o, el) {
      const pr = new Intl.PluralRules(lang, o);
      return el.dataset.end ? pr.selectRange(Number(v), Number(el.dataset.end)) : pr.select(Number(v));
    },
    "display-name": (lang, v, o) => new Intl.DisplayNames(lang, { type: "language", ...o }).of(v),
    collate: (lang, v, o) => JSON.parse(v).sort(new Intl.Collator(lang, o).compare).join(" · "),
    segment(lang, v, o) {
      const parts = [...new Intl.Segmenter(lang, o).segment(v)].filter((s) => s.isWordLike ?? s.segment.trim() !== "");
      return parts.length + " · " + parts.map((s) => s.segment.trim()).join(" | ");
    },
    locale(lang, v) {
      const loc = new Intl.Locale(lang);
      const get = (name) => {
        const m = loc["get" + name[0].toUpperCase() + name.slice(1)];
        const x = typeof m === "function" ? m.call(loc) : loc[name];
        if (x === undefined) throw new TypeError("no " + name);
        return x;
      };
      switch (v) {
        case "maximize":
        case "minimize":
          return loc[v]().toString();
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
    zone: (lang, v) => zoneName(lang, v),
    "browser-zone": () => browserZone(),
  };

  function format(el) {
    const kind = kinds[el.dataset.intl];
    if (!kind) return;
    const lang = el.closest("[lang]")?.lang || root.lang || undefined;
    const value = el.dataset.value ?? el.getAttribute("datetime") ?? el.getAttribute("value");
    let text;
    try {
      text = kind(lang, value, parse(el.dataset.opts, {}), el);
    } catch {
      return; // this browser can't: keep the server's fallback
    }
    if (!text) return; // nothing to show: keep the fallback
    const tpl = el.dataset.template;
    if (tpl && tpl.includes(SLOT)) text = tpl.replace(SLOT, text);
    if (el.textContent !== text) el.textContent = text;
  }

  function fillZones(select) {
    if (select.dataset.intlZonesDone || !Intl.supportedValuesOf) return;
    select.dataset.intlZonesDone = "1";
    const lang = select.closest("[lang]")?.lang || root.lang || undefined;
    const have = new Set([...select.options].map((o) => o.value));
    const frag = document.createDocumentFragment();
    for (const id of Intl.supportedValuesOf("timeZone")) {
      if (have.has(id)) continue;
      frag.append(new Option(zoneName(lang, id), id));
    }
    select.append(frag);
  }

  function update(node = document) {
    if (node.nodeType !== 1 && node !== document) return;
    if (node.nodeType === 1 && node.matches("[data-intl]")) format(node);
    for (const el of node.querySelectorAll("[data-intl]")) format(el);
    if (node.nodeType === 1 && node.matches("select[data-intl-zones]")) fillZones(node);
    for (const el of node.querySelectorAll("select[data-intl-zones]")) fillZones(el);
  }

  // Our own text changes are childList mutations of a [data-intl] element that add only text nodes: skipped.
  const observer = new MutationObserver((records) => {
    const todo = new Set();
    for (const r of records) {
      if (r.type === "attributes") todo.add(r.target);
      for (const n of r.addedNodes) if (n.nodeType === 1) todo.add(n);
    }
    for (const n of todo) if (n.isConnected) update(n);
  });

  function start() {
    update();
    observer.observe(document.body, {
      childList: true,
      subtree: true,
      attributes: true,
      attributeFilter: ["data-intl", "data-value", "data-opts", "datetime", "value", "lang"],
    });
    setInterval(() => document.querySelectorAll('[data-intl="relative"]').forEach(format), 30000);
  }

  window.intl = { update, elapsedUnit };
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", start);
  else start();
})();
