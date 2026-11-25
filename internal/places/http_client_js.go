//go:build js && wasm

package places

import (
	"net/http"
	"time"

	"github.com/syumai/workers-go/cloudflare/fetch"
)

func nominatimHTTPClient(timeout time.Duration) *http.Client {
	client := fetch.NewClient().HTTPClient(fetch.RedirectModeFollow)
	client.Timeout = timeout
	return client
}
