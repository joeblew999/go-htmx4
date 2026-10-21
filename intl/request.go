package intl

import "context"

type ctxKey struct{}

// Request is the request-scoped state an app puts in the context: the resolved locale, the request
// path without its locale prefix (so views can link the same page in another locale), and the viewer's
// date and time preferences.
type Request struct {
	Locale *Locale
	Path   string // e.g. "/about?x=1", never locale-prefixed
	// Origin is the request's scheme and host, e.g. "https://example.com", for absolute URLs: canonical links,
	// hreflang alternates and sitemaps must not be relative.
	Origin string

	// TimeZone is the viewer's IANA time zone ("Europe/Berlin"), "" for unknown: the browser formats dates in it.
	TimeZone string
	// TimeZoneChosen is true when TimeZone is the viewer's explicit choice rather than a guess (e.g. from
	// the connection), so browser code must not replace it with the browser's zone.
	TimeZoneChosen bool
	// HourCycle is the viewer's explicit clock, "h12" or "h23"; "" keeps the locale's.
	HourCycle string
}

// WithRequest returns ctx carrying r. The app's locale middleware calls it once per request; the renderers (Value,
// Msg) and components read it through the render context with [FromContext].
func WithRequest(ctx context.Context, r Request) context.Context {
	return context.WithValue(ctx, ctxKey{}, r)
}

// Current is the request's locale: <html lang={ intl.Current(ctx).ID() } dir={ intl.Current(ctx).Dir() }>. A render
// without a request is in the source locale.
func Current(ctx context.Context) *Locale {
	if r, ok := FromContext(ctx); ok {
		return r.Locale
	}
	return Default()
}

// FromContext returns the request's i18n state; ok is false when none was set.
func FromContext(ctx context.Context) (r Request, ok bool) {
	r, ok = ctx.Value(ctxKey{}).(Request)
	return r, ok && r.Locale != nil
}
