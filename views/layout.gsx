package views

import (
	"github.com/gsxhq/gsx"
	"github.com/joeblew999/go-htmx4/ui"
)

// Layout is the document shell, mirroring gsxui's site layout: gsxui theme CSS, htmx, the
// gsxui behaviours module, a header of gsxui Buttons, plus the theme toggle. htmxConfig becomes the
// htmx-config meta, which must precede htmx's script (htmx reads it when it loads); head carries
// page-specific tags that need htmx (the board adds hx-ws).
component Layout(title string, path string, htmxConfig string, head gsx.Node, children gsx.Node) {
	<!DOCTYPE html>
	<html lang="en">
		<head>
			<meta charset="utf-8"/>
			<meta name="viewport" content="width=device-width, initial-scale=1"/>
			<title>{ title } · htmx 4 on Workers</title>
			<ThemeScript/>
			<link rel="stylesheet" href="/assets/gsxui.css"/>
			{ if htmxConfig != "" {
				<meta name="htmx-config" content={htmxConfig}/>
			} }
			<script src="/static/htmx.min.js"></script>
			<script type="module" src="/gsxui/index.js"></script>
			{ head }
		</head>
		<body class="min-h-svh bg-background font-sans text-foreground antialiased">
			<header class="border-b">
				<nav class="mx-auto flex max-w-3xl items-center gap-2 p-4">
					<a href="/" class="mr-auto font-semibold">go-htmx4 · Workers demo</a>
					<ui.Button variant={navVariant(path, "/")} size="sm" href="/">Home</ui.Button>
					<ui.Button variant={navVariant(path, "/board")} size="sm" href="/board">Shared board</ui.Button>
					<ThemeToggle/>
				</nav>
			</header>
			<main class="mx-auto flex max-w-3xl flex-col gap-6 p-4">{ children }</main>
		</body>
	</html>
}
