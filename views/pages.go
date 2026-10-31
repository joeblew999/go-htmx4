package views

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/jackielii/structpages"
	"github.com/joeblew999/go-htmx4/intl"
	"github.com/joeblew999/go-htmx4/views/gsxkit"
)

// Pages is the app's page tree (#79, #132), as structpages reads it: every page people read is a type here, its
// route in its field's tag, and a Page method (pages.gsx). app mounts it. Everything else takes the routes from it,
// never a path written as a string: the links ({ About{} |> url }, gsx.toml's url filter), the sitemap, intl's and
// seo's IsPage, and each page's canonical address (Key). A new page: a type with a Page method and, when it needs the
// app's data, Props (through App); a field here; an entry in All; a translated title and description to Layout.
//
// TinyGo's reflect can't list or call methods: structpages-gen writes the tables its build reads (mise run generate).
type Pages struct {
	home       Home       `route:"GET /{$} Home"`
	board      Board      `route:"GET /board Board"`
	formats    Formats    `route:"GET /formats Formats"`
	components Components `route:"GET /components Components"`
	live       Live       `route:"GET /live Live"`
	maps       Map        `route:"GET /map Map"`
	about      About      `route:"GET /about About"`
}

// All is every page, each with no parameters: the sitemap's pages (those search engines may index).
var All = []any{Home{}, Board{}, Formats{}, Components{}, Live{}, Map{}, About{}}

// Home is the home page.
type Home struct{}

// DefaultTopic is the board's topic without ?topic=: plain /board.
const DefaultTopic = "lobby"

// Board is the shared board of one topic.
type Board struct{ Topic string }

// Query is what makes a topic's board a different page: ?topic=, unless it's the default topic.
func (b Board) Query() string {
	if b.Topic == "" || b.Topic == DefaultTopic {
		return ""
	}
	return "?topic=" + url.QueryEscape(b.Topic)
}

// Noindex: any topic but the default is someone's scratch space, and anyone can make another: there is no end to them.
func (b Board) Noindex() bool { return b.Query() != "" }

// Formats is what Intl does with dates, numbers and text.
type Formats struct{}

// Components is every gsxui component, with Intl.
type Components struct{}

// Live is live video over MoQ (#156): the reader's own camera broadcast, and the stream ?stream= names, through
// Cloudflare's relay. Every page view gets a stream of its own to broadcast on, and a link to it to send.
type Live struct{ Stream string }

// Query is what makes a stream's page a different page: ?stream=.
func (l Live) Query() string {
	if l.Stream == "" {
		return ""
	}
	return "?stream=" + url.QueryEscape(l.Stream)
}

// Noindex: a stream's page is one person's broadcast for as long as it lasts.
func (l Live) Noindex() bool { return l.Stream != "" }

// Map is gsxkit's map component (#164): where Cloudflare places the request, and the reader's own location on request.
type Map struct{}

// About is the stack.
type About struct{}

// App is what the pages need from the app: package app fills it in, and structpages hands it to their Props
// (WithArgs).
type App interface {
	Home(r *http.Request) HomeProps
	Connection(r *http.Request) Connection
	Stack() []StackItem
	// Board is the request's board, or false when the app has answered the request itself (a bad topic, a store
	// error).
	Board(w http.ResponseWriter, r *http.Request) (BoardProps, bool)
	// Live is where the live video page's broadcast goes: the relay and the broadcast's name.
	Live(r *http.Request) LiveProps
	// Map is where the map page centres: where Cloudflare places the request.
	Map(r *http.Request) MapProps
}

// MapProps is the map page's data: Cloudflare's guess of where the request came from (request.cf), from its IP.
type MapProps struct {
	// Found is whether Cloudflare sent a place with the request (a local run doesn't); At is it.
	Found bool
	At    gsxkit.LatLng
	// Center and Zoom are where the map opens: around At, else the whole world.
	Center gsxkit.LatLng
	Zoom   int
	// Place is the place's name (its town, else its region, else its country's code); Country its country (ISO
	// 3166-1, "DE"); Colo the Cloudflare data center the request came through ("MUC"), "" when unknown.
	Place, Country, Colo string
}

// LiveProps is the live video page's data. The relay URLs carry a relay token when the Worker has one (Cloudflare's
// draft-16 relays take it in the path), so they are never logged.
type LiveProps struct {
	// WatchURL and PublishURL are the relay as a watcher and as the broadcaster reach it.
	WatchURL, PublishURL string
	// Own is this page view's own stream, the one its camera broadcasts on; Watch the stream to play, "" for none.
	Own, Watch string
	// OwnName and WatchName are their broadcasts' paths on the relay; .hang selects the catalog format players read.
	OwnName, WatchName string
	// Origin is the site's origin ("https://…"): the link to send is it and Live{Stream: Own}'s address.
	Origin string
}

// HomeProps is the home page's data.
type HomeProps struct {
	Target, Env          string
	Flavours, Components []string
}

// BoardProps is a board page's data.
type BoardProps struct {
	Topic string
	State BoardState
	Live  bool
}

func (Home) Props(r *http.Request, app App) HomeProps { return app.Home(r) }

func (Board) Props(w http.ResponseWriter, r *http.Request, app App) (BoardProps, error) {
	p, ok := app.Board(w, r)
	if !ok {
		return p, structpages.ErrSkipPageRender
	}
	return p, nil
}

func (Formats) Props(r *http.Request, app App) Connection { return app.Connection(r) }

func (Live) Props(r *http.Request, app App) LiveProps { return app.Live(r) }

func (Map) Props(r *http.Request, app App) MapProps { return app.Map(r) }

func (About) Props(app App) []StackItem { return app.Stack() }

// tree is Pages parsed once: the routes are the same for every request.
var tree = sync.OnceValue(func() *structpages.StructPages {
	sp, err := structpages.Parse(Pages{}, "/", "go-htmx4")
	if err != nil {
		panic(err) // the tree is code: TestPages catches a mistake in it
	}
	return sp
})

// route is page's path without its query, from the tree: "/about".
func route(page any) string {
	r, err := tree().URLFor(page)
	if err != nil {
		panic(err)
	}
	return r
}

// address is page's address without the locale prefix: its route, and its query when that makes a different page.
func address(page any) string {
	if q, ok := page.(interface{ Query() string }); ok {
		return route(page) + q.Query()
	}
	return route(page)
}

// At is the page at path (without the locale prefix) for a request with query, false when path is no page's.
func At(path string, query url.Values) (any, bool) {
	for _, p := range All {
		if route(p) == path {
			if _, ok := p.(Board); ok {
				return Board{Topic: query.Get("topic")}, true
			}
			if _, ok := p.(Live); ok {
				return Live{Stream: query.Get("stream")}, true
			}
			return p, true
		}
	}
	return nil, false
}

// IsPage is intl.Site's IsPage: whether path (without the locale prefix) is a page, so a GET of it is a page view.
func IsPage(path string) bool {
	_, ok := At(path, nil)
	return ok
}

// Indexable is seo.Site's IsPage: whether a request for path ("/de/board?topic=games") is for a page Google may index.
func Indexable(path string) bool {
	path, query, _ := strings.Cut(path, "?")
	q, _ := url.ParseQuery(query)
	p, ok := At(intl.Unprefixed(path), q)
	if b, isBoard := p.(Board); isBoard && b.Noindex() {
		return false
	}
	if l, isLive := p.(Live); isLive && l.Noindex() {
		return false
	}
	return ok
}

// Sitemap is the paths the sitemap lists, in every locale: every page search engines may index.
func Sitemap() []string {
	var out []string
	for _, p := range All {
		out = append(out, address(p))
	}
	return out
}

// Key is the request's page, as its canonical address and its alternates name it (intl.Head): its path without the
// locale prefix or the query parameters that don't make a different page. "/board?topic=lobby&utm_source=x" is
// "/board". What isn't a page is its path.
func Key(ctx context.Context) string {
	path, query := "/", ""
	if r, ok := intl.FromContext(ctx); ok && r.Path != "" {
		path, query, _ = strings.Cut(r.Path, "?")
	}
	q, _ := url.ParseQuery(query)
	if p, ok := At(path, q); ok {
		return address(p)
	}
	return path
}
