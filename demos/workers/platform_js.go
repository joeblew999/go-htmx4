//go:build js && wasm

package main

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/syumai/workers-go/cloudflare"
	_ "github.com/syumai/workers-go/cloudflare/d1" // registers the "d1" database/sql driver
)

// getenv reads a Worker text binding (Cloudflare vars; locally a workerd text binding).
func getenv(name string) string { return cloudflare.Getenv(name) }

// staticFiles: on Workers, public/ is served by Static Assets before the Worker runs, so a
// request that reaches Go here is a real 404.
func staticFiles() http.Handler { return http.NotFoundHandler() }

// newStore opens the DB binding: D1 on Cloudflare, workerd/local-d1.mjs under workerd.
func newStore() (store, error) {
	db, err := sql.Open("d1", "DB")
	if err != nil {
		return nil, err
	}
	return sqlStore{db}, nil
}

// publish hands a board fragment to the topic's Room Durable Object (room.mjs), which pushes it
// to every connected browser.
func publish(topic string, version int64, fragment string) error {
	ns, err := cloudflare.NewDurableObjectNamespace("ROOM")
	if err != nil {
		return err
	}
	room, err := ns.Get(ns.IdFromName(topic))
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, "https://room/publish", strings.NewReader(fragment))
	if err != nil {
		return err
	}
	req.Header.Set("X-Board-Version", strconv.FormatInt(version, 10))
	res, err := room.Fetch(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("room: HTTP %d: %s", res.StatusCode, body)
	}
	return nil
}
