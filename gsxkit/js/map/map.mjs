// /gsxkit/map.js and /gsxkit/map.css: Leaflet (bundled by internal/jsbundle from npm-lock.json, its stylesheet beside
// it: mise run gsxkit:bundle), and the glue between it and the markup of gsxkit's map component
// (gsxkit/ui/map.gsx.tmpl), which go tool gsxkit add copies into an app (#164).
//
// Everything a reader sees is in the markup, translated and made of gsxui components: the zoom and location buttons,
// each marker's pin and popup (a <template> the component renders), the attribution. The glue only hands them to
// Leaflet, wires the buttons, and writes data-* state (data-locate, data-zoom) that Tailwind's group-data variants
// read. Leaflet's own controls are off: their labels would be English.
import L from "leaflet";
import "leaflet/dist/leaflet.css";

// The pin's size and the point of it that marks the place: gsxui's icons are 24 by 24, drawn at size-8 (32 px), and a
// map pin's point is the middle of its bottom edge.
const PIN = 32;

// Where the reader's own location centres the map, when it was further out.
const LOCATED_ZOOM = 14;

// Every map on the page, and every one a boosted navigation or a swap brings in later: the module runs once per
// document, however many maps load it. A map that leaves the page is taken down.
const maps = new Map();
function start() {
	for (const section of document.querySelectorAll("[data-map]")) {
		if (!maps.has(section)) maps.set(section, draw(section));
	}
	for (const [section, map] of maps) {
		if (!section.isConnected) {
			map.remove();
			maps.delete(section);
		}
	}
}
start();
new MutationObserver(start).observe(document.documentElement, { childList: true, subtree: true });

function draw(section) {
	const { center, zoom, tiles, maxZoom } = JSON.parse(section.dataset.map);
	const canvas = section.querySelector("[data-map-canvas]");
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
	for (const template of section.querySelectorAll("template[data-map-marker]")) marker(template).addTo(map);
	// The map is drawn for its box: when the box changes (a card opening, a phone turning), Leaflet measures again.
	new ResizeObserver(() => map.invalidateSize()).observe(canvas);

	let you;
	const state = () => {
		section.dataset.zoom = String(map.getZoom());
		section.dataset.atMin = String(map.getZoom() <= map.getMinZoom());
		section.dataset.atMax = String(map.getZoom() >= map.getMaxZoom());
	};
	map.on("zoomend", state);
	state();
	section.addEventListener("click", (event) => {
		const action = event.target.closest("[data-map-action]")?.dataset.mapAction;
		if (action === "zoom-in") map.zoomIn();
		else if (action === "zoom-out") map.zoomOut();
		else if (action === "locate") locate();
	});

	// locate asks the browser where the reader is and marks it with the component's own pin for them.
	function locate() {
		if (!navigator.geolocation) {
			section.dataset.locate = "unavailable";
			return;
		}
		section.dataset.locate = "locating";
		navigator.geolocation.getCurrentPosition(
			(position) => {
				const at = [position.coords.latitude, position.coords.longitude];
				you?.remove();
				you = marker(section.querySelector("template[data-map-you]"), at).addTo(map);
				map.setView(at, Math.max(map.getZoom(), LOCATED_ZOOM));
				section.dataset.locate = "found";
			},
			(error) => {
				section.dataset.locate = error.code === error.PERMISSION_DENIED ? "denied" : "unavailable";
			},
			{ enableHighAccuracy: true, timeout: 15_000, maximumAge: 60_000 },
		);
	}
	section.dataset.ready = "true";
	return map;
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
