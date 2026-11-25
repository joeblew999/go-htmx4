// What every page loads. Vite bundles it (npm run build) and serves it while developing (npm run dev).
//
// htmx first: its module sets window.htmx, which the extensions after it register with, before htmx starts
// (DOMContentLoaded).
import "virtual:gsx-devpanel"; // gsx's dev panel: Cmd-D / Ctrl-D
import "htmx.org";
import "htmx.org/dist/ext/hx-live.js";
import "htmx.org/dist/ext/hx-ws.js";
// The browser's half of intl (npm run generate writes web/intl).
import "./intl/intl-browser.mjs";
import "./gsxui/index.js";
import "./gsxui/index.css";
