package views

import (
	"github.com/gsxhq/gsx"
	"github.com/joeblew999/go-htmx4/ui"
)

// Layout is the document shell, following gsxui's site layout: gsxui theme CSS, the theme toggle, a header of
// gsxui Buttons and the gsxui toaster.
//
// Script order matters. The htmx-config meta comes before htmx, which reads it when it loads (hx-ws spreads
// reconnects over 1–3 s: a deploy drops every socket at once, and hx-ws's defaults would bring 1,000 tabs back
// within ~0.3 s, over a Room's ~1,000 requests/s soft limit). Then htmx (classic script, sets window.htmx), its
// extensions hx-live and hx-ws, the board's version guard, and the gsxui behaviours as native ES modules.
// Everything loads on every page, so a boosted navigation to /board has hx-ws ready.
//
// htmx 4 inheritance is explicit: only the nav links are boosted (morph + view transition), so the pages'
// own hx-* requests keep their default swaps.
component Layout(title string, path string, children gsx.Node) {
	<!DOCTYPE html>
	<html lang="en">
		<head>
			<meta charset="utf-8"/>
			<meta name="viewport" content="width=device-width, initial-scale=1"/>
			<title>{ title } · go-htmx4</title>
			<ThemeScript/>
			<link rel="stylesheet" href="/assets/gsxui.css"/>
			<meta name="htmx-config" content="ws.reconnectDelay:2s ws.reconnectJitter:0.5"/>
			<script src="/static/htmx.min.js"></script>
			<script src="/static/hx-live.js"></script>
			<script src="/static/hx-ws.js"></script>
			<VersionGuard/>
			<script type="module" src="/gsxui/index.js"></script>
		</head>
		<body class="min-h-svh bg-background font-sans text-foreground antialiased">
			<header class="border-b">
				<nav
					class="mx-auto flex max-w-3xl items-center gap-2 p-4"
					hx-boost:inherited="true"
					hx-swap:inherited="outerMorph transition:true"
				>
					<a href="/" class="mr-auto font-semibold">go-htmx4</a>
					<ui.Button variant={navVariant(path, "/")} size="sm" href="/">Home</ui.Button>
					<ui.Button variant={navVariant(path, "/board")} size="sm" href="/board">Board</ui.Button>
					<ui.Button variant={navVariant(path, "/about")} size="sm" href="/about">About</ui.Button>
					<ThemeToggle/>
				</nav>
			</header>
			<main class="mx-auto flex max-w-3xl flex-col gap-6 p-4">{ children }</main>
			<ui.Toaster/>
		</body>
	</html>
}

// VersionGuard keeps the board monotonic: HTTP responses and hx-ws pushes both go through htmx.swap and can
// arrive out of order, so drop any swap whose board version is older than the one on screen. Pages without
// #board are unaffected.
component VersionGuard() {
	<script>
		document.addEventListener("htmx:before:swap", (event) => {
			const incoming = /data-version="(\d+)"/.exec(event.detail.ctx.text ?? "");
			const current = document.getElementById("board")?.dataset.version;
			if (incoming && current && Number(incoming[1]) < Number(current)) event.preventDefault();
		});
	</script>
}
