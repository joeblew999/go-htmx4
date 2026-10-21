//go:build js && wasm

package live

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/syumai/workers-go/cloudflare"
)

// Publish hands one version of a fragment, rendered per locale, to topic's Room through the Durable Object
// namespace bound as binding (e.g. "ROOM"), which pushes each connected browser its locale's fragment. A
// failed publish only delays other tabs: the Room's cache and the next publish catch them up.
func Publish(binding, topic string, version int64, fragments Localized) error {
	ns, err := cloudflare.NewDurableObjectNamespace(binding)
	if err != nil {
		return err
	}
	room, err := ns.Get(ns.IdFromName(topic))
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, "https://room/publish", strings.NewReader(string(fragments.JSON())))
	if err != nil {
		return err
	}
	req.Header.Set(VersionHeader, strconv.FormatInt(version, 10))
	res, err := room.Fetch(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("live: room %s: HTTP %d: %s", topic, res.StatusCode, body)
	}
	return nil
}
