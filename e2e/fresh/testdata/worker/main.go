// The fresh Worker (TestFreshWorker): a gsx page and a visit counter on D1 behind a rate limit, written as charter's
// docs say. One main package: transport.Run serves it as the Worker, and off Workers as a plain HTTP server; the
// bindings are charter's, which stand in for themselves off Workers, so it has no file per platform.
package main

import (
	"net/http"

	"github.com/joeblew999/charter/bindings"
	"github.com/joeblew999/charter/d1"
	"github.com/joeblew999/charter/transport"
	"github.com/joeblew999/charter/web"
)

func main() { transport.Run(handler()) }

func handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if !web.Allow(w, r, http.MethodGet) {
			return
		}
		n, err := visits()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		web.Render(w, r, Page(n, bindings.Var("APP_ENV")))
	})
	mux.HandleFunc("/visit", func(w http.ResponseWriter, r *http.Request) {
		if !web.Allow(w, r, http.MethodPost) {
			return
		}
		if ok, _ := bindings.Allow("VISITS", bindings.ClientKey(r)); !ok {
			http.Error(w, "slow down", http.StatusTooManyRequests)
			return
		}
		db, err := d1.Open("DB")
		if err == nil {
			err = d1.Exec(db, "INSERT INTO visits DEFAULT VALUES")
		}
		n, verr := visits()
		if err != nil || verr != nil {
			http.Error(w, "the database: "+firstError(err, verr), http.StatusInternalServerError)
			return
		}
		web.Render(w, r, Count(n))
	})
	return mux
}

// visits is how many visits the database has.
func visits() (int, error) {
	db, err := d1.Open("DB")
	if err != nil {
		return 0, err
	}
	c, err := d1.One[struct {
		N int `json:"n"`
	}](db, "SELECT count(*) AS n FROM visits")
	return c.N, err
}

// firstError is the first of errs that isn't nil, as text.
func firstError(errs ...error) string {
	for _, err := range errs {
		if err != nil {
			return err.Error()
		}
	}
	return ""
}
