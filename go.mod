module github.com/joeblew999/go-htmx4

go 1.27.1

toolchain go1.27.3

require (
	github.com/Masterminds/semver/v3 v3.5.0
	github.com/evanw/esbuild v0.28.2
	github.com/gsxhq/gsx v0.4.0
	github.com/jackielii/structpages v0.10.0
	github.com/jackielii/tailwind-merge-go v0.0.0-20260726184505-8a911ed0cf9a
	github.com/joeblew999/charter v0.20.0
	github.com/joeblew999/go-htmx4/intl v0.0.0
	github.com/joeblew999/go-htmx4/seo v0.3.0
	golang.org/x/net v0.61.0
)

require (
	github.com/BurntSushi/toml v1.6.0 // indirect
	github.com/coder/websocket v1.8.15 // indirect
	github.com/djherbis/atime v1.1.0 // indirect
	github.com/dustin/go-humanize v1.1.0 // indirect
	github.com/editorconfig/editorconfig-core-go/v2 v2.6.4 // indirect
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jackielii/ctxkey v1.0.1 // indirect
	github.com/jmoiron/sqlx v1.4.0 // indirect
	github.com/joeblew999/go-htmx4/badge v0.1.0 // indirect
	github.com/joeblew999/go-htmx4/gsxkit v0.0.0 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/ncruces/go-strftime v1.1.0 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/syumai/workers-go v0.36.0 // indirect
	github.com/tdewolff/argp v0.0.0-20260424074207-decde4f86440 // indirect
	github.com/tdewolff/minify/v2 v2.24.13 // indirect
	github.com/tdewolff/parse/v2 v2.8.13 // indirect
	golang.org/x/exp/typeparams v0.0.0-20260611194520-c48552f49976 // indirect
	golang.org/x/mod v0.42.0 // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/tools v0.52.0 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	modernc.org/libc v1.79.0 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.12.1 // indirect
	modernc.org/sqlite v1.61.0 // indirect
)

tool (
	github.com/gsxhq/gsx/cmd/gsx
	github.com/jackielii/structpages/cmd/structpages-gen
	github.com/joeblew999/go-htmx4/gsxkit/cmd/gsxkit
	github.com/joeblew999/go-htmx4/intl/cmd/intl
	github.com/joeblew999/go-htmx4/seo/cmd/seo
	github.com/tdewolff/minify/v2/cmd/minify
)

replace github.com/joeblew999/go-htmx4/intl => ./intl

replace github.com/joeblew999/go-htmx4/seo => ./seo

replace github.com/joeblew999/go-htmx4/badge => ./badge

replace github.com/joeblew999/go-htmx4/gsxkit => ./gsxkit

replace github.com/jackielii/structpages => github.com/joeblew999/structpages v0.10.1-0.20261022094628-31c2f0e5f405
