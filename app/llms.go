package app

import (
	"context"

	"github.com/joeblew999/go-htmx4/intl"
	"github.com/joeblew999/go-htmx4/locales"
	"github.com/joeblew999/go-htmx4/seo"
	"github.com/joeblew999/go-htmx4/views"
)

// repo is the app's repository on GitHub (charter.toml's repo, TestCharterToml): where llms.txt's docs are.
const repo = "joeblew999/go-htmx4"

// pageText is each page's title and description, the ones it shows (TestLLMs checks), for llms.txt.
var pageText = map[any][2]intl.Msg{
	views.Home{}:       {locales.HomeHeading, locales.HomeDescription},
	views.Board{}:      {locales.BoardTitle, locales.BoardIntro},
	views.Formats{}:    {locales.FormatsTitle, locales.FormatsDescription},
	views.Components{}: {locales.ComponentsTitle, locales.ComponentsDescription},
	views.About{}:      {locales.AboutTitle, locales.AboutDescription},
}

// llms is /llms.txt (seo.Site's LLMs), in the source language: what the site is, its pages, and the docs as Markdown
// an agent can read (#144).
func llms(origin string) string {
	ctx := context.Background() // no request: the source locale
	text := func(m intl.Msg) string { return intl.Text(ctx, m) }
	var pages []seo.LLMsLink
	for i, p := range views.All {
		t := pageText[p]
		pages = append(pages, seo.LLMsLink{Title: text(t[0]), URL: origin + views.Sitemap()[i], Note: text(t[1])})
	}
	raw := "https://raw.githubusercontent.com/" + repo + "/main/"
	docs := []seo.LLMsLink{
		{Title: "README", URL: raw + "README.md", Note: "what this is, and how to start an app of your own from its pieces"},
		{Title: "intl", URL: raw + "intl/README.md", Note: "every language of a gsx app through the runtime's Intl"},
		{Title: "seo", URL: raw + "seo/README.md", Note: "what Google and agents read, and the audit that checks it"},
		{Title: "gsxkit", URL: raw + "gsxkit/README.md", Note: "gsx's and gsxui's tasks and skills for any app"},
		{Title: "Architecture", URL: raw + "docs/architecture.md", Note: "how the app works"},
		{Title: "Deploy", URL: raw + "docs/deploy.md", Note: "how it ships to Cloudflare Workers"},
		{Title: "Commands", URL: raw + "docs/commands.md", Note: "every mise task"},
		{Title: "AGENTS.md", URL: raw + "AGENTS.md", Note: "the rules for agents working in the repository"},
	}
	return seo.LLMsTxt("go-htmx4", text(locales.HomeDescription), "",
		seo.LLMsSection{Name: "Pages", Links: pages}, seo.LLMsSection{Name: "Docs", Links: docs})
}
