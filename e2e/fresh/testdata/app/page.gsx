package main

import (
	"context"

	"example.com/fresh/locales"
	"example.com/fresh/ui"
	intlui "example.com/fresh/views/intl"
	"github.com/gsxhq/gsx"
	"github.com/joeblew999/go-htmx4/intl"
	"github.com/joeblew999/go-htmx4/seo"
)

// The fresh app's pages, made as intl/README.md and seo/README.md say: every gsx block in those READMEs is here as they
// write it (TestREADMECode).

component Layout(title intl.Msg, description intl.Msg, path string, children gsx.Node) {
	<!DOCTYPE html>
	<html { intl.HTML(ctx)... }>
		<head>
			<meta charset="utf-8"/>
			<seo.Head
				title={locales.PageTitle(title) |> text}
				description={description |> text}
				siteName="Shop"
				icon="/static/favicon.svg"
				home={homeURL(ctx, path)}
				lang={intl.Current(ctx).ID()}
			/>
			<intl.Head page={path}/>
			<script type="module" src="/static/intl-browser.mjs"></script>
		</head>
		<body>
			<header>
				<intlui.PreferencesDialog/>
			</header>
			<main>
				<intlui.LanguageOffer/>
				{ children }
			</main>
			<footer>
				<intlui.LanguageList/>
			</footer>
		</body>
	</html>
}

component Home(order Order, notes []Note, note Note) {
	<Layout title={locales.HomeHeading} description={locales.HomeDescription} path="/">
		<h1>{ locales.HomeHeading }</h1>
		<p title={locales.BoardHint}>{ locales.BoardNotes(len(notes)) }</p>
		<p>
			{ order.Placed |> date("dateStyle", "long") } · { order.Total |> number("style", "currency", "currency", "EUR") }
		</p>
		<p>{ note.Created |> since } · { "JP" |> intl("DisplayNames", "of", "type", "region") }</p>
		<ui.Slider hx-on:input=js`document.querySelector("#km [data-intl]").dataset.args = JSON.stringify([this.value])`/>
		<output id="km">{ 42 |> number("style", "unit", "unit", "kilometer", "unitDisplay", "long") }</output>
	</Layout>
}

component About() {
	<Layout title={locales.AboutTitle} description={locales.AboutDescription} path="/about">
		<h1>{ locales.AboutTitle }</h1>
	</Layout>
}

// homeURL is the home page's absolute URL on the home page itself, "" on every other page: seo.Head puts the site's
// name there for Google.
func homeURL(ctx context.Context, path string) string {
	if path != "/" {
		return ""
	}
	return intl.Canonical(ctx, "/")
}
