package i18n

import (
	"context"

	"github.com/joeblew999/go-htmx4/locales"
)

// Messages is how Translate reads the catalogs. The views package sets it to its own accessor (views.M), so gsxui's
// text follows everything the views do, the tests' pseudo-locale included; this package can't import views (cycle).
var Messages = locales.For

// Translate renders a gsxui component's own text (registered for T in gsx.toml [renderers]) from the app's message
// catalogs in the request's locale. This file is the app's, not gsxui's: `gsxui add --overwrite` leaves it alone.
func Translate(ctx context.Context, m T) string {
	msgs := Messages(ctx)
	switch m {
	case "Close":
		return msgs.GsxuiClose()
	case "Notifications":
		return msgs.GsxuiNotifications()
	}
	return string(m)
}
