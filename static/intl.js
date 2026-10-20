// The browser half of the app's Intl (worker/intl.mjs, served as /static/intl-core.mjs). The Worker already formatted
// every [data-intl] element with the same V8 Intl (kit/i18n Format), in the server's best guess of the viewer's
// zone. This script re-formats only what the server couldn't know: the viewer's own time zone (unless they chose
// one, <html data-tz>) and relative times as time passes. Most of the time the text doesn't change at all.
//
//   <time datetime="…" data-intl="datetime" data-opts='{"dateStyle":"full"}'>Samstag, 4. Juli 2026</time>
//
// The value is data-value, else datetime, else value; data-opts is a JSON object of Intl options; data-end and
// data-unit are extra inputs (worker/intl.mjs lists the kinds). data-template, if set, is a translated message with
// the result's place marked by U+E000 (views/intl.go slot). The language is the nearest lang attribute; a browser
// without full data for it (trimmed ICU) keeps the server's text. <select data-intl-zones> gets every zone this
// browser knows, each named in the page's language.
//
// Elements are formatted on load and whenever they're added or changed: htmx swaps, hx-ws pushes and morphs all go
// through one MutationObserver. window.intl exposes format and parseNumber, so forms can send plain values back.
import { format, fullySupports, parseNumber, zoneName, elapsedUnit } from "./intl-core.mjs";

const SLOT = "";
const root = document.documentElement;

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

const supported = new Map();
function supports(lang) {
  if (!supported.has(lang)) supported.set(lang, fullySupports(lang));
  return supported.get(lang);
}

function formatEl(el) {
  const kind = el.dataset.intl;
  const lang = el.closest("[lang]")?.lang || root.lang || "en";
  let text;
  if (kind === "browser-zone") {
    text = browserZone();
  } else {
    if (!supports(lang)) return; // keep the server's text
    const value = el.dataset.value ?? el.getAttribute("datetime") ?? el.getAttribute("value");
    const extra = { end: el.dataset.end, unit: el.dataset.unit, tz: viewerZone(), hc: root.dataset.hc || undefined };
    text = format(lang, kind, value, el.dataset.opts || "", extra);
    if (kind === "relative" && el.tagName === "TIME" && !el.dataset.unit) {
      const title = format(lang, "datetime", value, { dateStyle: "medium", timeStyle: "short" }, extra);
      if (title && el.title !== title) el.title = title;
    }
  }
  if (!text) return; // this browser can't: keep the server's text
  const tpl = el.dataset.template;
  if (tpl && tpl.includes(SLOT)) text = tpl.replace(SLOT, text);
  if (el.textContent !== text) el.textContent = text;
  if (!el.hasAttribute("data-intl-done")) el.setAttribute("data-intl-done", ""); // formatted by this browser
}

function fillZones(select) {
  if (select.dataset.intlZonesDone || !Intl.supportedValuesOf) return;
  select.dataset.intlZonesDone = "1";
  const lang = select.closest("[lang]")?.lang || root.lang || "en";
  const have = new Set([...select.options].map((o) => o.value));
  const frag = document.createDocumentFragment();
  for (const id of Intl.supportedValuesOf("timeZone")) {
    if (!have.has(id)) frag.append(new Option(zoneName(lang, id), id));
  }
  select.append(frag);
}

function update(node = document) {
  if (node.nodeType !== 1 && node !== document) return;
  if (node.nodeType === 1 && node.matches("[data-intl]")) formatEl(node);
  for (const el of node.querySelectorAll("[data-intl]")) formatEl(el);
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

update();
observer.observe(document.body, {
  childList: true,
  subtree: true,
  attributes: true,
  attributeFilter: ["data-intl", "data-value", "data-opts", "datetime", "value", "lang"],
});
setInterval(() => document.querySelectorAll('[data-intl="relative"]').forEach(formatEl), 30000);

window.intl = { update, format, parseNumber, elapsedUnit };
