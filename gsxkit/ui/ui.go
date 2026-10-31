// Package ui is gsxkit's components: a registry of gsx components gsxui doesn't have, each made of gsxui components,
// with its text translated into every language intl ships and, for one that runs in the browser, its script and
// styles bundled from npm (no Node). go tool gsxkit add copies them into an app the way gsxui add copies gsxui's: the
// gsx (*.gsx.tmpl here, which gsx generate leaves alone) into the app's views/gsxkit, the browser files (assets/) into
// its web/gsxkit, served at /gsxkit/, and the text (locales/) into its catalogs under [gsxkit.<component>]. The app
// owns its copies. The copying is intl/registry's, as intl's own screens' is (#164).
//
// The browser files are built from js/<component> by the repository's internal/jsbundle (mise run gsxkit:bundle) and
// committed, so an app needs neither npm nor esbuild.
package ui

import (
	"embed"

	"github.com/joeblew999/go-htmx4/intl/registry"
)

//go:embed *.gsx.tmpl locales/*.toml assets/*
var files embed.FS

// Components are gsxkit's components.
var Components = &registry.Registry{
	Command: "go tool gsxkit add",
	Table:   "gsxkit",
	Files:   files,
	Locales: "github.com/joeblew999/go-htmx4/gsxkit/ui/locales",
	Components: []registry.Component{
		{Name: "map", Gsxui: []string{"alert", "button", "button-group", "spinner"}, Assets: []string{"map.js", "map.css"},
			Use: "  <gsxkit.Map center={gsxkit.LatLng{48.137, 11.575}} zoom={12} label={…} locate={true}>\n" +
				"    <gsxkit.MapMarker at={gsxkit.LatLng{48.137, 11.575}} title=\"Munich\">popup</gsxkit.MapMarker>\n" +
				"  </gsxkit.Map>\n" +
				"    serve web/gsxkit at /gsxkit/ and gsxui's behaviours at /gsxui/ (its core is shared); the Content Security Policy's img-src takes https://tile.openstreetmap.org"},
		{Name: "moq", Gsxui: []string{"alert", "badge", "button", "card", "collapsible", "field", "input-group"}, Assets: []string{"moq.js"},
			Use: "  <gsxkit.MoqBroadcast url={relay} name={broadcast} link={address that watches it}/>\n" +
				"  <gsxkit.MoqWatch url={relay} name={broadcast}/>\n" +
				"    serve web/gsxkit at /gsxkit/ and gsxui's behaviours at /gsxui/ (its core is shared); the Content Security Policy takes the relay in connect-src, blob: in script-src and worker-src"},
	},
}
