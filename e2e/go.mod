module github.com/joeblew999/go-htmx4/e2e

go 1.27.1

toolchain go1.27.3

require (
	github.com/chromedp/cdproto v0.160.9
	github.com/chromedp/chromedp v0.24.0
	github.com/joeblew999/go-htmx4/seo v0.0.0
	golang.org/x/net v0.63.0
)

require (
	github.com/go-json-experiment/json v0.0.0-20260623181947-01eb4420fa68 // indirect
	github.com/gsxhq/gsx v0.5.0 // indirect
)

replace github.com/joeblew999/go-htmx4/seo => ../seo

replace github.com/joeblew999/go-htmx4/badge => ../badge
