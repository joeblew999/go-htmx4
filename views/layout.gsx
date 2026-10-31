package views

import (
	"github.com/gsxhq/gsx"
	"github.com/joeblew999/go-htmx4/intl"
	"github.com/joeblew999/go-htmx4/locales"
	"github.com/joeblew999/go-htmx4/seo"
	"github.com/joeblew999/go-htmx4/ui"
	"github.com/joeblew999/go-htmx4/ui/icon"
	intlui "github.com/joeblew999/go-htmx4/views/intl"
)

// Layout is the document shell, following gsxui's site layout: gsxui theme CSS, the theme toggle, a header of
// gsxui Buttons and the gsxui toaster.
//
// Script order matters. The htmx-config meta comes before htmx, which reads it when it loads (hx-ws spreads
// reconnects over 1–3 s: a deploy drops every socket at once, and hx-ws's defaults would bring 1,000 tabs back
// within ~0.3 s, over a Room's ~1,000 requests/s soft limit). Then htmx with its extensions hx-live and hx-ws, one
// deferred script (the assets task joins them, so they register before htmx starts), the board's version guard, and
// what every document loads (documentAssets): the theme, gsxui's CSS, and intl's and gsxui's modules, which run
// after htmx, in that order.
// Everything loads on every page, so a boosted navigation to /board has hx-ws ready.
//
// htmx 4 inheritance is explicit: only the nav links are boosted (morph + view transition), so the pages'
// own hx-* requests keep their default swaps. The nav wraps: translated labels don't fit one row on a phone
// (e2e TestNoOverflow).
//
// <html lang dir> come from the request's locale (intl.HTML), and every UI string from locales/*.toml (intl.Msg).
// <html data-tz data-hc> are the viewer's chosen time zone and clock ("" = the browser's own and the language's), for
// intl-browser.mjs, which redoes dates, times and rich numbers in the browser. The head carries what search engines
// and link previews read (seo.Head: the translated title and description, Open Graph, the site's icon, and on the home
// page the site's name for Google) and the page's language versions (intl.Head, with the page table's Key). intl's
// screens (go tool intl add, in views/intl) offer the page in the reader's language above the content, list it in
// every language in the footer, outside the boosted nav (switching locale must be a full page load, since a boosted
// swap keeps the old <html lang dir>), and set the time zone and clock from the header.
component Layout(title intl.Msg, description intl.Msg, children gsx.Node) {
	<!DOCTYPE html>
	<html { intl.HTML(ctx)... }>
		<head>
			<meta charset="utf-8"/>
			<meta name="viewport" content="width=device-width, initial-scale=1"/>
			<seo.Head
				title={locales.PageTitle(title) |> text}
				description={description |> text}
				siteName="go-htmx4"
				icon="/static/favicon.svg"
				home={homeURL(ctx)}
				lang={intl.Current(ctx).ID()}
			/>
			<intl.Head page={Key(ctx)}/>
			<meta name="htmx-config" content={htmxConfig(ctx)}/>
			<script src="/static/htmx.js" defer></script>
			<VersionGuard/>
			<documentAssets/>
		</head>
		<body class="min-h-svh bg-background font-sans text-foreground antialiased">
			<header class="border-b">
				<nav
					class="mx-auto flex max-w-3xl flex-wrap items-center gap-2 p-4"
					hx-boost:inherited="true"
					hx-swap:inherited="outerMorph transition:true"
				>
					<a href={Home{} |> url} class="me-auto font-semibold" translate="no">go-htmx4</a>
					<ui.Button variant={navVariant(ctx, Home{})} size="sm" href={Home{} |> url}>
						{ locales.NavHome }
					</ui.Button>
					<ui.Button variant={navVariant(ctx, Board{})} size="sm" href={Board{} |> url}>
						{ locales.NavBoard }
					</ui.Button>
					<ui.Button variant={navVariant(ctx, Formats{})} size="sm" href={Formats{} |> url}>
						{ locales.NavFormats }
					</ui.Button>
					<ui.Button variant={navVariant(ctx, Components{})} size="sm" href={Components{} |> url}>
						{ locales.NavComponents }
					</ui.Button>
					// A full page load, not a boosted swap: the page's own module (the MoQ elements) must run.
					<ui.Button
						variant={navVariant(ctx, Live{})}
						size="sm"
						href={Live{} |> url}
						hx-boost:inherited="false"
					>
						{ locales.NavLive }
					</ui.Button>
					<ui.Button variant={navVariant(ctx, Map{})} size="sm" href={Map{} |> url}>
						{ locales.NavMap }
					</ui.Button>
					<ui.Button variant={navVariant(ctx, About{})} size="sm" href={About{} |> url}>
						{ locales.NavAbout }
					</ui.Button>
					<ui.Button
						variant="ghost"
						size="icon-sm"
						href="#languages"
						hx-boost:inherited="false"
						aria-label={locales.IntlLanguages}
						title={locales.IntlLanguages}
					>
						<icon.Languages/>
					</ui.Button>
					<intlui.PreferencesDialog/>
					<ThemeToggle/>
				</nav>
			</header>
			<main class="mx-auto flex max-w-3xl flex-col gap-6 p-4">
				<intlui.LanguageOffer/>
				{ children }
			</main>
			<footer id="languages" class="border-t">
				<intlui.LanguageList class="mx-auto max-w-3xl p-4"/>
			</footer>
			<ui.Toaster/>
		</body>
	</html>
}

// documentAssets is what every document of the app loads, the page (Layout) and the /components sidebar's frame
// alike: the theme before the first paint, gsxui's CSS and its body font, then as modules the browser's Intl
// (intl-browser.mjs) and gsxui's behaviours.
component documentAssets() {
	<ThemeScript/>
	// Static files (dist/site, served before the Worker), not routes: no page in the tree names them.
	// structpages:lint:ignore url-attr
	<link rel="stylesheet" href="/assets/gsxui.css"/>
	// The body font (gsxui.preset.json's fontSans), fetched with the stylesheet: no reflow when it arrives late.
	<link
		rel="preload"
		// structpages:lint:ignore url-attr
		href="/assets/geist-latin-wght-normal.woff2"
		as="font"
		type="font/woff2"
		crossorigin="anonymous"
	/>
	<script type="module" src="/static/intl-browser.mjs"></script>
	<script type="module" src="/gsxui/index.js"></script>
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
