module github.com/joeblew999/go-htmx4

go 1.27.1

toolchain go1.27.3

require (
	github.com/BurntSushi/toml v1.6.0
	github.com/gsxhq/gsx v0.4.0
	github.com/jackielii/tailwind-merge-go v0.0.0-20260726184505-8a911ed0cf9a
	github.com/joeblew999/charter/go/lib v0.9.2-0.20261021035509-8ca48a3e384b
	github.com/syumai/workers-go v0.36.0
	golang.org/x/net v0.61.0
)

require (
	github.com/coder/websocket v1.8.15 // indirect
	github.com/editorconfig/editorconfig-core-go/v2 v2.6.4 // indirect
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	github.com/tdewolff/minify/v2 v2.24.13 // indirect
	github.com/tdewolff/parse/v2 v2.8.13 // indirect
	golang.org/x/exp/typeparams v0.0.0-20260611194520-c48552f49976 // indirect
	golang.org/x/mod v0.37.0 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/tools v0.46.0 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
)

tool (
	github.com/gsxhq/gsx/cmd/gsx
	github.com/joeblew999/charter/go/lib/cmd/searchconsole
	github.com/joeblew999/charter/go/lib/cmd/workerbench
	github.com/joeblew999/charter/go/lib/cmd/workerbuild
	github.com/joeblew999/charter/go/lib/cmd/workerdeploy
	github.com/joeblew999/charter/go/lib/cmd/workertail
)
