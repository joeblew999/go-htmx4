package gsxkit

import (
	"github.com/gsxhq/gsx"
	"github.com/joeblew999/go-htmx4/intl"
	"github.com/joeblew999/go-htmx4/locales"
	"github.com/joeblew999/go-htmx4/ui"
	"github.com/joeblew999/go-htmx4/ui/icon"
)

// A map: Leaflet (https://leafletjs.com) over OpenStreetMap's tiles, with markers whose pins and popups are gsx, and
// zoom and the reader's own location as gsxui buttons. Leaflet and the glue between it and this markup are
// /gsxkit/map.js and /gsxkit/map.css (gsxkit's js/map, bundled from npm with no Node). A page reached by a boosted
// navigation draws its maps too.
//
// Every label is rendered here, translated; map.js only sets the map's data-locate (idle, locating, found, denied,
// unavailable), data-zoom, data-at-min and data-at-max, and data-ready once drawn. The page's Content Security Policy
// must allow the tiles in img-src (https://tile.openstreetmap.org); OpenStreetMap's tile policy asks for the
// attribution the map shows and for light use (https://operations.osmfoundation.org/policies/tiles/): a busy site
// takes its tiles from a provider of its own (change tiles below).
//
// From go tool gsxkit add map: the app's own now, to change as it likes.

// LatLng is a place: its latitude and longitude, in degrees.
type LatLng [2]float64

// The tiles: OpenStreetMap's standard map, as many zoom levels as it draws.
const (
	tiles   = "https://tile.openstreetmap.org/{z}/{x}/{y}.png"
	maxZoom = 19
)

// Map draws a map around center at zoom (0 the whole world, 19 a street), named label for screen readers, with
// children's markers (MapMarker) on it. locate adds a button that asks the browser for the reader's own location and
// marks it with you's pin, named you's title. Its height is h-80; give it another with class.
component Map(center LatLng, zoom int, label intl.Msg, locate bool, children gsx.Node, attrs gsx.Attrs) {
	<div
		class={
			"group/map relative isolate h-80 overflow-hidden rounded-lg border bg-muted",
			// Leaflet's popups in gsxui's popover colours. Leaflet's stylesheet is outside Tailwind's layers, so it would
			// win without the !.
			"[&_.leaflet-popup-content-wrapper]:bg-popover! [&_.leaflet-popup-content-wrapper]:text-popover-foreground! [&_.leaflet-popup-tip]:bg-popover!",
			"[&_.leaflet-popup-content-wrapper]:rounded-lg! [&_.leaflet-popup-content-wrapper]:shadow-md! [&_.leaflet-popup-content]:m-3!",
			"[&_.leaflet-container]:font-sans! [&_.leaflet-container]:bg-muted!"
		}
		data-map=js`{"center": @{center}, "zoom": @{zoom}, "tiles": @{tiles}, "maxZoom": @{maxZoom}}`
		data-locate="idle"
		{ attrs... }
	>
		// The map itself, drawn by Leaflet: never mirrored, whatever the page's direction.
		<div class="size-full" dir="ltr" role="region" aria-label={label} data-map-canvas></div>
		<div class="absolute end-2 top-2 z-1000 flex flex-col gap-2">
			<ui.ButtonGroup orientation="vertical">
				<ui.Button
					variant="outline"
					size="icon-sm"
					class="group-data-[at-max=true]/map:opacity-50"
					aria-label={locales.GsxkitMapZoomIn}
					title={locales.GsxkitMapZoomIn}
					data-map-action="zoom-in"
				>
					<icon.Plus/>
				</ui.Button>
				<ui.Button
					variant="outline"
					size="icon-sm"
					class="group-data-[at-min=true]/map:opacity-50"
					aria-label={locales.GsxkitMapZoomOut}
					title={locales.GsxkitMapZoomOut}
					data-map-action="zoom-out"
				>
					<icon.Minus/>
				</ui.Button>
			</ui.ButtonGroup>
			{ if locate {
				<ui.Button
					variant="outline"
					size="icon-sm"
					aria-label={locales.GsxkitMapLocate}
					title={locales.GsxkitMapLocate}
					data-map-action="locate"
				>
					<icon.LocateFixed class="group-data-[locate=locating]/map:hidden"/>
					<ui.Spinner class="hidden group-data-[locate=locating]/map:block"/>
				</ui.Button>
			} }
		</div>
		{ if locate {
			<p
				role="status"
				class="absolute inset-x-2 bottom-8 z-1000 hidden rounded-md bg-background/90 px-2 py-1 text-sm group-data-[locate=denied]/map:block"
			>
				{ locales.GsxkitMapDenied }
			</p>
			<p
				role="status"
				class="absolute inset-x-2 bottom-8 z-1000 hidden rounded-md bg-background/90 px-2 py-1 text-sm group-data-[locate=unavailable]/map:block"
			>
				{ locales.GsxkitMapUnavailable }
			</p>
			<MapMarker you={true} title={locales.GsxkitMapYou |> text} tone="you">
				<p class="font-medium">{ locales.GsxkitMapYou }</p>
			</MapMarker>
		} }
		<a
			href="https://www.openstreetmap.org/copyright"
			class="absolute bottom-0 end-0 z-1000 rounded-ss-md bg-background/80 px-1.5 py-0.5 text-xs text-muted-foreground hover:underline"
		>
			{ locales.GsxkitMapAttribution }
		</a>
		{ children }
		<link rel="stylesheet" href="/gsxkit/map.css"/>
		<script type="module" src="/gsxkit/map.js"></script>
	</div>
}

// MapMarker is a place on its Map: a pin (MapPin, in tone) named title, and children, when there are any, in the
// popup the pin opens. you is the Map's own marker for the reader, placed where the browser says they are.
component MapMarker(at LatLng, title string, tone string, you bool, children gsx.Node) {
	<template
		{ if you {
			data-map-you
		} else {
			data-map-marker=js`@{at}`
		} }
		data-title={title}
	>
		<span data-map-pin>
			<MapPin tone={tone}/>
		</span>
		{ if children != nil {
			<div class="flex flex-col gap-1 text-sm" data-map-popup>{ children }</div>
		} }
	</template>
}

// MapPin is a marker's pin, for a legend beside the map too: the primary colour, or for tone "you" (the reader's own
// place) its opposite, the page's background outlined in its text colour. Either stands out on the tiles, light or
// dark.
component MapPin(tone string, attrs gsx.Attrs) {
	<icon.MapPin
		class={
			"size-8 drop-shadow-sm",
			switch tone { case "you": "fill-background text-foreground" default: "fill-primary text-primary-foreground" }
		}
		{ attrs... }
	/>
}
