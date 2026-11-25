//go:build !(js && wasm)

package places

import (
	"net/http"
	"time"
)

func nominatimHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout}
}
