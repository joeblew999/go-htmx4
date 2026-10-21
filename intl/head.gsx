package intl

import (
	"context"

	"github.com/gsxhq/gsx"
)

// HTML is the <html> element's language attributes for the request: lang and dir from its locale, and the viewer's
// chosen time zone and clock (data-tz, data-hc) for js/intl-browser.mjs: <html { intl.HTML(ctx)... }>.
func HTML(ctx context.Context) gsx.Attrs {
	l := Current(ctx)
	return gsx.Attrs{
		{Key: "lang", Value: l.ID()},
		{Key: "dir", Value: l.Dir()},
		{Key: "data-tz", Value: ChosenTimeZone(ctx)},
		{Key: "data-hc", Value: HourCycle(ctx)},
	}
}

// Head is what <head> says about the page's languages: its canonical URL, the page in every shipped locale (hreflang,
// reciprocal, x-default the source locale's), and Open Graph's og:url, og:locale and og:locale:alternate. page is the
// path and the query parameters that make it a different page, without a locale prefix.
component Head(page string) {
	<link rel="canonical" href={Canonical(ctx, page)}/>
	{ for _, a := range Alternates(ctx, page) {
		<link rel="alternate" hreflang={a.Lang} href={a.Href}/>
	} }
	<meta property="og:url" content={Canonical(ctx, page)}/>
	<meta property="og:locale" content={OGLocale(Current(ctx))}/>
	{ for _, l := range shipped {
		{ if l != Current(ctx) {
			<meta property="og:locale:alternate" content={OGLocale(l)}/>
		} }
	} }
}
