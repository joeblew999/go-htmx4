// Keeps <time datetime data-relative-time> text fresh in the viewer's browser (plan decision 6).
//
// Go renders the text on the server (kit/i18n RelativeTimeFormat, numeric "auto"), so the page is complete
// without this script. Pushed and cached fragments go stale, though, so every 30 s and after each htmx swap
// this re-formats them with Intl.RelativeTimeFormat in the page's own language — never the browser's
// default. It leaves the server text alone when the browser lacks full data for that language (Chrome and
// Firefox trim ICU data for many locales), so it can only refresh the text, never degrade it.
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

  let cached = null;
  function formatter() {
    const lang = document.documentElement.lang;
    if (cached && cached.lang === lang) return cached.rtf;
    let rtf = null;
    try {
      const supported = Intl.RelativeTimeFormat.supportedLocalesOf([lang]).length > 0;
      const candidate = new Intl.RelativeTimeFormat(lang, { numeric: "auto" });
      // Trimmed ICU data resolves to another language (often en): keep the server's text then.
      if (supported && candidate.resolvedOptions().locale.split("-")[0] === lang.split("-")[0]) rtf = candidate;
    } catch {
      rtf = null;
    }
    cached = { lang, rtf };
    return rtf;
  }

  function update(root = document) {
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

  window.relativeTime = { update, elapsedUnit };
  document.addEventListener("htmx:after:swap", () => update());
  document.addEventListener("DOMContentLoaded", () => update());
  setInterval(update, 30000);
})();
