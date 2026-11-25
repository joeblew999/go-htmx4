// Package assets loads the browser's files Vite builds from web/ (vite.config.ts).
package assets

import "github.com/gsxhq/vite"

// Entry loads one of Vite's entries (vite.config.ts's input): its stylesheets, the chunks it imports, and its module.
// With npm run dev they come from Vite's dev server, which also reloads the page.
component Entry(name string) {
	{{ b := vite.FromContext(ctx).Entry(name) }}
	// Vite's files (dist/assets, served before the Worker), not routes: no page in the tree names them.
	{ for _, href := range b.CSS {
		// structpages:lint:ignore url-attr
		<link rel="stylesheet" href={href}/>
	} }
	{ for _, href := range b.Preloads {
		// structpages:lint:ignore url-attr
		<link rel="modulepreload" href={href}/>
	} }
	{ for _, src := range b.JS {
		<script type="module" src={src}></script>
	} }
}
