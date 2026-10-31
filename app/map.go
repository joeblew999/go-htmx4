package app

import (
	"cmp"
	"net/http"
	"strconv"

	"github.com/joeblew999/charter/transport"
	"github.com/joeblew999/go-htmx4/views"
	"github.com/joeblew999/go-htmx4/views/gsxkit"
)

// The map page (#164): gsxkit's map component around Cloudflare's guess of where the request came from.

// The map's view of a place Cloudflare names (a town), and of the whole world when it names none.
const (
	placeZoom = 10
	worldZoom = 2
)

// world is where the map opens without a place: the middle of the inhabited latitudes.
var world = gsxkit.LatLng{20, 0}

// Map is views.App's: request.cf's latitude, longitude and names, the map centred on them; without them (go test,
// local workerd) the whole world.
func (s *server) Map(r *http.Request) views.MapProps {
	cf := transport.CFOf(r)
	p := views.MapProps{Center: world, Zoom: worldZoom, Place: cmp.Or(cf.City, cf.Region, cf.Country), Country: cf.Country, Colo: cf.Colo}
	lat, errLat := strconv.ParseFloat(cf.Latitude, 64)
	lng, errLng := strconv.ParseFloat(cf.Longitude, 64)
	if errLat == nil && errLng == nil {
		p.Found, p.At, p.Center, p.Zoom = true, gsxkit.LatLng{lat, lng}, gsxkit.LatLng{lat, lng}, placeZoom
	}
	return p
}
