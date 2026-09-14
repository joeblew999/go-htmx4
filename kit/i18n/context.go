package i18n

import "context"

type ctxKey struct{}

// Request is the request-scoped i18n state an app puts in the context: the resolved locale and the
// request path without its locale prefix (so views can link the same page in another locale).
type Request struct {
	Locale *Locale
	Path   string // e.g. "/about?x=1", never locale-prefixed
}

// WithRequest returns ctx carrying r. Middleware calls it once per request; gsx components read it
// through the ambient ctx with [FromContext].
func WithRequest(ctx context.Context, r Request) context.Context {
	return context.WithValue(ctx, ctxKey{}, r)
}

// FromContext returns the request's i18n state; ok is false when none was set.
func FromContext(ctx context.Context) (r Request, ok bool) {
	r, ok = ctx.Value(ctxKey{}).(Request)
	return r, ok && r.Locale != nil
}
