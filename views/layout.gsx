package views

import (
	"github.com/gsxhq/gsx"
	"github.com/joeblew999/go-htmx4/intl"
	"github.com/joeblew999/go-htmx4/locales"
	"github.com/joeblew999/go-htmx4/ui"
	"github.com/joeblew999/go-htmx4/ui/icon"
)

// Layout is the document shell, following gsxui's site layout: gsxui theme CSS, the theme toggle, a header of
// gsxui Buttons and the gsxui toaster.
//
// Script order matters. The htmx-config meta comes before htmx, which reads it when it loads (hx-ws spreads
// reconnects over 1–3 s: a deploy drops every socket at once, and hx-ws's defaults would bring 1,000 tabs back
// within ~0.3 s, over a Room's ~1,000 requests/s soft limit). Then htmx with its extensions hx-live and hx-ws, one
// deferred script (the assets task joins them, so they register before htmx starts), the board's version guard, and
// the gsxui behaviours as native ES modules.
// Everything loads on every page, so a boosted navigation to /board has hx-ws ready.
//
// htmx 4 inheritance is explicit: only the nav links are boosted (morph + view transition), so the pages'
// own hx-* requests keep their default swaps. The nav wraps: translated labels don't fit one row on a phone
// (e2e TestNoOverflow).
//
// <html lang dir> come from the request's locale (intl.HTML), and every UI string from locales/*.toml (intl.Msg).
// <html data-tz data-hc> are the viewer's chosen time zone and clock ("" = the browser's own and the language's), for
// intl-browser.mjs, which redoes dates, times and rich numbers in the browser. The
// head carries a translated description, the canonical URL, the page in every locale and Open Graph (intl.Head, with
// pageKey in views.go), the site's icon, and on the home page the site's name for Google (WebSite structured data). The
// language links sit in the footer, outside the boosted nav: switching locale must be a full page load, since a
// boosted swap keeps the old <html lang dir>.
component Layout(title intl.Msg, description intl.Msg, path string, children gsx.Node) {
	<!DOCTYPE html>
	<html { intl.HTML(ctx)... }>
		<head>
			<meta charset="utf-8"/>
			<meta name="viewport" content="width=device-width, initial-scale=1"/>
			<title>{ locales.PageTitle(title) }</title>
			<meta name="description" content={description}/>
			<intl.Head page={pageKey(ctx)}/>
			<meta property="og:type" content="website"/>
			<meta property="og:site_name" content="go-htmx4"/>
			<meta property="og:title" content={locales.PageTitle(title)}/>
			<meta property="og:description" content={description}/>
			<meta name="twitter:card" content="summary"/>
			<link rel="icon" href="/static/favicon.svg" type="image/svg+xml"/>
			{ if path == "/" {
				<script type="application/ld+json">@{ webSite(ctx) }</script>
			} }
			<ThemeScript/>
			<link rel="stylesheet" href="/assets/gsxui.css"/>
			// The body font (gsxui.preset.json's fontSans), fetched with the stylesheet: no reflow when it arrives late.
			<link
				rel="preload"
				href="/assets/geist-latin-wght-normal.woff2"
				as="font"
				type="font/woff2"
				crossorigin="anonymous"
			/>
			<meta name="htmx-config" content="ws.reconnectDelay:2s ws.reconnectJitter:0.5"/>
			<script src="/static/htmx.js" defer></script>
			<script type="module" src="/static/intl-browser.mjs"></script>
			<VersionGuard/>
			<script type="module" src="/gsxui/index.js"></script>
		</head>
		<body class="min-h-svh bg-background font-sans text-foreground antialiased">
			<header class="border-b">
				<nav
					class="mx-auto flex max-w-3xl flex-wrap items-center gap-2 p-4"
					hx-boost:inherited="true"
					hx-swap:inherited="outerMorph transition:true"
				>
					<a href={"/" |> localize} class="me-auto font-semibold" translate="no">go-htmx4</a>
					<ui.Button variant={navVariant(path, "/")} size="sm" href={"/" |> localize}>{ locales.NavHome }</ui.Button>
					<ui.Button variant={navVariant(path, "/board")} size="sm" href={"/board" |> localize}>
						{ locales.NavBoard }
					</ui.Button>
					<ui.Button variant={navVariant(path, "/formats")} size="sm" href={"/formats" |> localize}>
						{ locales.NavFormats }
					</ui.Button>
					<ui.Button variant={navVariant(path, "/components")} size="sm" href={"/components" |> localize}>
						{ locales.NavComponents }
					</ui.Button>
					<ui.Button variant={navVariant(path, "/about")} size="sm" href={"/about" |> localize}>
						{ locales.NavAbout }
					</ui.Button>
					<ui.Button
						variant="ghost"
						size="icon-sm"
						href="#languages"
						hx-boost:inherited="false"
						aria-label={locales.NavLanguages}
						title={locales.NavLanguages}
					>
						<icon.Languages/>
					</ui.Button>
					<PreferencesDialog/>
					<ThemeToggle/>
				</nav>
			</header>
			<main class="mx-auto flex max-w-3xl flex-col gap-6 p-4">
				<LanguageSuggestion/>
				{ children }
			</main>
			<LanguageFooter/>
			<ui.Toaster/>
		</body>
	</html>
}

// LanguageSuggestion offers this page in the reader's own language, written in that language, on every page until
// they choose one: the language they chose, or the one their browser asks for, when the site has it and the page is
// in another (intl.Suggested). A link, never a redirect, so Google still reaches every language version. "No thanks"
// (intl.Decline) keeps the page's language as their choice.
component LanguageSuggestion() {
	{ if s, ok := intl.Suggested(ctx); ok {
		<ui.Alert lang={s.Lang} dir="auto">
			<icon.Languages/>
			<ui.AlertTitle>
				<a href={s.Href} hreflang={s.Lang} class="underline underline-offset-4">{ locales.NavAlsoIn(s.Name).In(s.Lang) }</a>
			</ui.AlertTitle>
			<ui.AlertDescription>
				<a href={intl.Decline(ctx)} rel="nofollow">{ locales.NavNoThanks.In(s.Lang) }</a>
			</ui.AlertDescription>
		</ui.Alert>
	} }
}

// LanguageFooter lists the current page in every shipped locale: plain links (crawlable, no JS), each named in its
// own language with lang and hreflang (and translate="no": a locale's own name must not be machine-translated).
component LanguageFooter() {
	<footer id="languages" class="border-t">
		<nav aria-label={locales.NavLanguages} class="mx-auto flex max-w-3xl flex-wrap items-center gap-1 p-4">
			<icon.Languages class="me-1 size-4 text-muted-foreground"/>
			{ for _, l := range intl.Links(ctx) {
				<ui.Button
					variant={activeVariant(l.Current)}
					size="sm"
					href={l.Href}
					hreflang={l.Lang}
					lang={l.Lang}
					translate="no"
					{ if l.Current {
						aria-current="page"
					} }
				>
					{ l.Name }
				</ui.Button>
			} }
		</nav>
	</footer>
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
