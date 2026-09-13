package views

import (
	"github.com/gsxhq/gsx"
	"github.com/joeblew999/go-htmx4/demos/gsxui/ui"
)

// Layout is the document shell. Scripts load in the order hx-live requires:
// htmx (classic script, sets window.htmx), then the hx-live extension, then the
// gsxui behaviours as native ES modules (npm-free guide: one module script tag).
//
// htmx 4 inheritance is explicit: only the nav links are boosted, so the page's
// own hx-* requests keep the default innerHTML swap.
component Layout(title string, path string, children gsx.Node) {
	<!DOCTYPE html>
	<html lang="en">
		<head>
			<meta charset="utf-8"/>
			<meta name="viewport" content="width=device-width, initial-scale=1"/>
			<title>{ title } · gsxui + htmx 4</title>
			<link rel="stylesheet" href="/assets/gsxui.css"/>
			<script src="/static/htmx.min.js"></script>
			<script src="/static/hx-live.js"></script>
			<script type="module" src="/gsxui/index.js"></script>
		</head>
		<body class="min-h-svh bg-background font-sans text-foreground antialiased">
			<header class="border-b">
				<nav
					class="mx-auto flex max-w-3xl items-center gap-2 p-4"
					hx-boost:inherited="true"
					hx-swap:inherited="outerMorph transition:true"
				>
					<a href="/" class="mr-auto font-semibold">go-htmx4 · gsxui demo</a>
					<ui.Button variant={navVariant(path, "/")} size="sm" href="/">Home</ui.Button>
					<ui.Button variant={navVariant(path, "/about")} size="sm" href="/about">About</ui.Button>
				</nav>
			</header>
			<main class="mx-auto flex max-w-3xl flex-col gap-6 p-4">{ children }</main>
			<ui.Toaster/>
		</body>
	</html>
}

func navVariant(path, href string) string {
	if path == href {
		return "secondary"
	}
	return "ghost"
}
