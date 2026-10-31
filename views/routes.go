package views

import (
	"net/http"
	"net/url"
	"strconv"
)

// The routes the pages call: forms' actions and the fragments htmx loads into a page. They are in the page tree
// (Pages) like the pages, so every link to one comes from the tree ({ GreetAction{} |> url }) and structpages-lint
// checks it, but they are not pages: not in All, so not in the sitemap, not IsPage. Each answers through the app
// (App's Actions).

// GreetAction is the home page's greeting form: the greeting, or the error for an empty name.
type GreetAction struct{}

func (GreetAction) ServeHTTP(w http.ResponseWriter, r *http.Request, app App) { app.Greet(w, r) }

// ClearGreetingAction clears the home page's greeting (DELETE on the greeting's route).
type ClearGreetingAction struct{}

func (ClearGreetingAction) ServeHTTP(w http.ResponseWriter, r *http.Request, app App) {
	app.ClearGreeting(w, r)
}

// ServerInfoFragment is the home page's dialog body: the Go runtime, uptime and requests served.
type ServerInfoFragment struct{}

func (ServerInfoFragment) ServeHTTP(w http.ResponseWriter, r *http.Request, app App) {
	app.ServerInfo(w, r)
}

// StatsFragment is the home page's stats tab, loaded when it opens: what the server has rendered, by name.
type StatsFragment struct{}

func (StatsFragment) ServeHTTP(w http.ResponseWriter, r *http.Request, app App) { app.Stats(w, r) }

// WeekFragment is the components page's week, loaded when its section opens.
type WeekFragment struct{}

func (WeekFragment) ServeHTTP(w http.ResponseWriter, r *http.Request, app App) { app.Week(w, r) }

// SidebarFragment is the components page's sidebar demo, a document of its own.
type SidebarFragment struct{}

func (SidebarFragment) ServeHTTP(w http.ResponseWriter, r *http.Request, app App) { app.Sidebar(w, r) }

// CountAction counts the components page's text box on the Worker: graphemes, words and sentences.
type CountAction struct{}

func (CountAction) ServeHTTP(w http.ResponseWriter, r *http.Request, app App) { app.Count(w, r) }

// TryAction is the formats page's round trip: a number typed the reader's way and some text, formatted by the
// Worker's Intl.
type TryAction struct{}

func (TryAction) ServeHTTP(w http.ResponseWriter, r *http.Request, app App) { app.Try(w, r) }

// BoardAddAction adds Delta (1 or -1) to Topic's board.
type BoardAddAction struct {
	Topic string
	Delta int
}

// Query is the topic and the delta, as the action reads them.
func (b BoardAddAction) Query() string {
	return "?topic=" + url.QueryEscape(b.Topic) + "&delta=" + strconv.Itoa(b.Delta)
}

func (BoardAddAction) ServeHTTP(w http.ResponseWriter, r *http.Request, app App) { app.BoardAdd(w, r) }

// BoardNoteAction adds a note to Topic's board.
type BoardNoteAction struct{ Topic string }

// Query is the topic, as the action reads it.
func (b BoardNoteAction) Query() string { return "?topic=" + url.QueryEscape(b.Topic) }

func (BoardNoteAction) ServeHTTP(w http.ResponseWriter, r *http.Request, app App) {
	app.BoardNote(w, r)
}

// Actions are the routes' answers, App's part for them: each writes the whole response.
type Actions interface {
	Greet(w http.ResponseWriter, r *http.Request)
	ClearGreeting(w http.ResponseWriter, r *http.Request)
	ServerInfo(w http.ResponseWriter, r *http.Request)
	Stats(w http.ResponseWriter, r *http.Request)
	Week(w http.ResponseWriter, r *http.Request)
	Sidebar(w http.ResponseWriter, r *http.Request)
	Count(w http.ResponseWriter, r *http.Request)
	Try(w http.ResponseWriter, r *http.Request)
	BoardAdd(w http.ResponseWriter, r *http.Request)
	BoardNote(w http.ResponseWriter, r *http.Request)
}

// Route is page's path from the tree, for Go code outside a request (the board's fragments rendered for the Room).
func Route(page any) string { return route(page) }
