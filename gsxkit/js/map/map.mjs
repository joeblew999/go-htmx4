// /gsxkit/map.js and /gsxkit/map.css: Leaflet (bundled by internal/jsbundle from npm-lock.json, its stylesheet beside
// it: mise run gsxkit:bundle), and the glue between it and the markup of gsxkit's map component
// (gsxkit/ui/map.gsx.tmpl), which go tool gsxkit add copies into an app (#164).
//
// Everything a reader sees is in the markup, translated and made of gsxui components: the zoom and location buttons,
// each marker's pin and popup (a <template> the component renders), the attribution. The glue only hands them to
// Leaflet, reflects its state (data-zoom, data-locate, the zoom buttons' disabled), and is written the gsxui way, on
// gsxui's core (the page's own /gsxui/gsxui.js, which the bundle shares): on() for the buttons, init() and once() to
// draw each map whenever it appears, on a full load, a boosted navigation or a swap. Leaflet's own controls are off:
// their labels would be English.
import L from "leaflet";
import "leaflet/dist/leaflet.css";
import { init, on, once } from "/gsxui/gsxui.js";

// The pin's size and the point of it that marks the place: gsxui's icons are 24 by 24, drawn at size-8 (32 px), and a
// map pin's point is the middle of its bottom edge.
const PIN = 32;

// Where the reader's own location centres the map, when it was further out.
const LOCATED_ZOOM = 14;

// Each map's Leaflet map, by the component's root.
const maps = new WeakMap();

on("click", "[data-map] [data-map-action]", (event, button) => {
	const root = button.closest("[data-map]");
	const map = maps.get(root);
	if (!map) return;
	const action = button.dataset.mapAction;
	if (action === "zoom-in") map.zoomIn();
	else if (action === "zoom-out") map.zoomOut();
	else if (action === "locate") locate(root, map);
});

// draw makes the root's Leaflet map, once, when its stylesheet applies: Leaflet lays its panes and tiles out by it. A
// full load runs this module after the stylesheet; a boosted navigation brings the component's <link> in with the
// page, still loading.
const draw = once((root) => {
	styled(root).then(() => {
		if (!root.isConnected) return;
		const map = create(root);
		maps.set(root, map);
		live.add(root);
		reflect(root, map);
	});
});

function styled(root) {
	const link = root.querySelector('link[rel="stylesheet"]');
	if (!link || link.sheet) return Promise.resolve();
	return new Promise((resolve) => {
		link.addEventListener("load", resolve, { once: true });
		link.addEventListener("error", resolve, { once: true });
	});
}

function create(root) {
	const { center, zoom, tiles, maxZoom } = JSON.parse(root.dataset.map);
	const canvas = root.querySelector("[data-map-canvas]");
	const map = L.map(canvas, {
		center,
		zoom,
		maxZoom,
		zoomControl: false,
		attributionControl: false,
		// The page scrolls past the map; the buttons, a pinch or a double click zoom it.
		scrollWheelZoom: false,
	});
	L.tileLayer(tiles, { maxZoom }).addTo(map);
	for (const template of root.querySelectorAll("template[data-map-marker]")) marker(template).addTo(map);
	// The map is drawn for its box: when the box changes (a card opening, a phone turning), Leaflet measures again.
	new ResizeObserver(() => map.invalidateSize()).observe(canvas);
	map.on("zoomend", () => reflect(root, map));
	return map;
}

// reflect writes the map's state where the markup reads it. It runs again whenever init() does (a morph resets the
// root's attributes to the server's), so it is cheap and only writes what changed.
function reflect(root, map) {
	const zoom = String(map.getZoom());
	if (root.dataset.zoom !== zoom) root.dataset.zoom = zoom;
	if (root.dataset.ready !== "true") root.dataset.ready = "true";
	for (const [action, at] of [["zoom-in", map.getZoom() >= map.getMaxZoom()], ["zoom-out", map.getZoom() <= map.getMinZoom()]]) {
		const button = root.querySelector(`[data-map-action="${action}"]`);
		if (button && button.disabled !== at) button.disabled = at;
	}
}

// locate asks the browser where the reader is and marks it with the component's own pin for them.
const you = new WeakMap();
function locate(root, map) {
	if (!navigator.geolocation) {
		root.dataset.locate = "unavailable";
		return;
	}
	root.dataset.locate = "locating";
	navigator.geolocation.getCurrentPosition(
		(position) => {
			const at = [position.coords.latitude, position.coords.longitude];
			you.get(root)?.remove();
			you.set(root, marker(root.querySelector("template[data-map-you]"), at).addTo(map));
			map.setView(at, Math.max(map.getZoom(), LOCATED_ZOOM));
			root.dataset.locate = "found";
		},
		(error) => {
			root.dataset.locate = error.code === error.PERMISSION_DENIED ? "denied" : "unavailable";
		},
		{ enableHighAccuracy: true, timeout: 15_000, maximumAge: 60_000 },
	);
}

// marker is a template's marker: its pin, its popup (when it has one), at the place it names or at.
function marker(template, at = JSON.parse(template.dataset.mapMarker)) {
	const content = template.content.cloneNode(true);
	const pin = content.querySelector("[data-map-pin]");
	const popup = content.querySelector("[data-map-popup]");
	const m = L.marker(at, {
		icon: L.divIcon({ html: pin, className: "", iconSize: [PIN, PIN], iconAnchor: [PIN / 2, PIN], popupAnchor: [0, -PIN] }),
		// A pin is a button for the keyboard and for screen readers, named by its title.
		title: template.dataset.title ?? "",
		keyboard: true,
		riseOnHover: true,
	});
	// No close button: its label would be Leaflet's English. A click elsewhere, or Escape, closes the popup.
	if (popup) m.bindPopup(popup, { closeButton: false });
	return m;
}

// gsxui's core has no hook for an element leaving the page: a map whose root has gone is taken down here, so Leaflet
// lets go of its listeners and the window's resize.
const live = new Set();
new MutationObserver(() => {
	for (const root of live) {
		if (root.isConnected) continue;
		maps.get(root)?.remove();
		maps.delete(root);
		live.delete(root);
	}
}).observe(document.documentElement, { childList: true, subtree: true });

// --- init ------------------------------------------------------------------

function initRoot(root) {
	draw(root);
	const map = maps.get(root);
	if (map) reflect(root, map);
}

init("[data-map]", initRoot);
