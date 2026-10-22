// Package link is the views' url filter (gsx.toml): href={ About{} |> url } is the page's address in the request's
// language, "/de/about" in German.
package link

import (
	"context"

	"github.com/jackielii/structpages"
	"github.com/joeblew999/go-htmx4/intl"
)

// URL is page's route from structpages' tree (views.Pages), with its query when that makes a different page (a board
// topic: views.Board's Query), in the request's locale (intl.Site).
func URL(ctx context.Context, page any) (string, error) {
	u, err := structpages.URLFor(ctx, page)
	if err != nil {
		return "", err
	}
	if q, ok := page.(interface{ Query() string }); ok {
		u += q.Query()
	}
	return intl.URL(ctx, u), nil
}
