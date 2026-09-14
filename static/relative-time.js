// Keeps server-rendered times right in the viewer's browser (plan decision 6). Go renders the finished text on
// the server (kit/i18n), so the page is complete without this script; it only corrects what the server can't know.
//
// <time datetime data-relative-time>: pushed and cached fragments go stale, so every 30 s and after each htmx
// swap the text is re-formatted with Intl.RelativeTimeFormat (numeric "auto").
//
// <time datetime data-local-time='{Intl options}' data-time-zone="Europe/Berlin">: the server guessed the
// viewer's time zone (Cloudflare's, from the connection). When the browser's own zone differs, the text is
// re-formatted with Intl.DateTimeFormat in it. The server omits data-local-time when the viewer chose a zone.
//
// Both always use the page's own language, never the browser's default, and leave the server text alone when
// the browser lacks full data for that language (Chrome and Firefox trim ICU data for many locales), so they
// can only correct the text, never degrade it.
//
// The unit thresholds must match kit/i18n ElapsedUnit (TestRelativeTimeJSMatchesGo).
(() => {
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

  // fullySupports: the browser has data for the page's language (trimmed ICU data resolves to another language,
  // often en: keep the server's text then).
  function fullySupports(Ctor, lang) {
    try {
      return Ctor.supportedLocalesOf([lang]).length > 0 && new Ctor(lang).resolvedOptions().locale.split("-")[0] === lang.split("-")[0];
    } catch {
      return false;
    }
  }

  let cached = null;
  function formatter() {
    const lang = document.documentElement.lang;
    if (cached && cached.lang === lang) return cached.rtf;
    const rtf = fullySupports(Intl.RelativeTimeFormat, lang) ? new Intl.RelativeTimeFormat(lang, { numeric: "auto" }) : null;
    cached = { lang, rtf };
    return rtf;
  }

  function updateRelative(root) {
    const rtf = formatter();
    if (!rtf) return;
    for (const el of root.querySelectorAll("time[data-relative-time]")) {
      const when = Date.parse(el.getAttribute("datetime"));
      if (Number.isNaN(when)) continue;
      const [value, unit] = elapsedUnit((when - Date.now()) / 1000);
      const text = rtf.format(value, unit);
      if (el.textContent !== text) el.textContent = text;
    }
  }

  // canonicalZone spells a zone id the way this browser does ("Asia/Kolkata" and "Asia/Calcutta" compare equal).
  function canonicalZone(id) {
    try {
      return new Intl.DateTimeFormat("en", { timeZone: id }).resolvedOptions().timeZone;
    } catch {
      return "";
    }
  }

  function updateLocal(root) {
    const lang = document.documentElement.lang;
    const zone = new Intl.DateTimeFormat().resolvedOptions().timeZone;
    if (!zone || !fullySupports(Intl.DateTimeFormat, lang)) return;
    for (const el of root.querySelectorAll("time[data-local-time]")) {
      const from = el.getAttribute("data-time-zone");
      if (from === zone || canonicalZone(from) === canonicalZone(zone)) continue;
      const when = Date.parse(el.getAttribute("datetime"));
      if (Number.isNaN(when)) continue;
      try {
        const options = JSON.parse(el.getAttribute("data-local-time"));
        el.textContent = new Intl.DateTimeFormat(lang, { ...options, timeZone: zone }).format(when);
        el.setAttribute("data-time-zone", zone);
      } catch {
        // Unknown options or zone in this browser: keep the server's text.
      }
    }
  }

  function update(root = document) {
    updateRelative(root);
    updateLocal(root);
  }

  window.relativeTime = { update, elapsedUnit };
  document.addEventListener("htmx:after:swap", () => update());
  document.addEventListener("DOMContentLoaded", () => update());
  setInterval(update, 30000);
})();
