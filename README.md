# go-htmx4

A minimal starter for building server-rendered web apps with [Go](https://go.dev) and [htmx 4](https://htmx.org).

- Standard library only — no external Go dependencies
- Uses Go 1.22+ `net/http` routing patterns (`GET /{$}`, `POST /count`)
- Templates and static assets are embedded with `go:embed`, so the result is a single self-contained binary
- htmx 4.0.0 is vendored into `static/`, so there's no CDN dependency at runtime

## Requirements

- Go 1.26 or newer

## Quick start

```sh
git clone https://github.com/joeblew999/go-htmx4.git
cd go-htmx4
go run .
```

Open <http://localhost:8080>.

To use a different address:

```sh
go run . -addr :3000
```

## Build

```sh
go build -o go-htmx4 .
./go-htmx4
```

## Project layout

```
.
├── main.go              # HTTP server, routes and handlers
├── templates/
│   └── index.html       # Page template + named fragment templates
└── static/
    └── htmx.min.js      # Vendored htmx 4.0.0
```

## How it works

The page is rendered in full by `GET /`. htmx attributes on elements then call
small endpoints that return **HTML fragments**, which htmx swaps into the page:

| Route         | Returns                   | Used by                                             |
| ------------- | ------------------------- | --------------------------------------------------- |
| `GET /`       | Full page (`index.html`)  | Browser                                             |
| `POST /count` | `count` fragment          | `<button hx-post="/count" hx-target="#count">`      |
| `GET /time`   | `time` fragment           | `<p hx-get="/time" hx-trigger="load, every 1s">`    |
| `GET /static/`| Embedded static files     | `<script src="/static/htmx.min.js">`                |

Fragments are declared with `{{define "name"}}` in the templates and rendered
with `tmpl.ExecuteTemplate`, so the full page and the partial updates share the
same markup.

## Adding a feature

1. Add a named fragment to `templates/index.html` (or a new file in `templates/`):
   ```html
   {{define "greeting"}}<p>Hello, {{.}}!</p>{{end}}
   ```
2. Add a handler in `main.go`:
   ```go
   mux.HandleFunc("POST /greet", func(w http.ResponseWriter, r *http.Request) {
       render(w, "greeting", r.FormValue("name"))
   })
   ```
3. Wire it up in the page:
   ```html
   <form hx-post="/greet" hx-target="#greeting">
     <input name="name" placeholder="Your name">
     <button>Greet</button>
   </form>
   <div id="greeting"></div>
   ```

## Upgrading htmx

```sh
curl -sfL -o static/htmx.min.js https://cdn.jsdelivr.net/npm/htmx.org@<version>/dist/htmx.min.js
```

See the [htmx 2 → 4 migration notes](https://four.htmx.org/migration-guide-htmx-4/) if you're coming from htmx 2.
