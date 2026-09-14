// Command wsload checks the shared board's live updates over real WebSockets: flags for kit/wsload,
// which documents the checks. It exits 1 if any check fails.
//
//	go run ./cmd/wsload -n 2                                   # smoke (local workerd)
//	go run ./cmd/wsload -n 1000                                # design size
//	go run ./cmd/wsload -n 1000 -writes 50                     # burst: coalescing
//	go run ./cmd/wsload -base https://….workers.dev -n 1000    # deployed
//	go run ./cmd/wsload -n 200 -hold 90s                       # keep sockets open, report drops (e.g. during a redeploy)
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/joeblew999/go-htmx4/kit/wsload"
)

func main() {
	log.SetFlags(0)
	var o wsload.Options
	flag.StringVar(&o.Base, "base", "http://127.0.0.1:8913", "app base URL")
	flag.StringVar(&o.Topic, "topic", "load", "board topic")
	flag.IntVar(&o.N, "n", 2, "number of WebSockets")
	flag.IntVar(&o.Dialers, "dialers", 50, "concurrent dials")
	flag.IntVar(&o.Writes, "writes", 1, "concurrent POST /board/add requests")
	flag.DurationVar(&o.Timeout, "timeout", 15*time.Second, "how long to wait for delivery")
	flag.DurationVar(&o.Hold, "hold", 0, "after the checks, keep the sockets open this long and report drops")
	locales := flag.String("locales", "", "comma-separated ?locale= values spread over the sockets (each must get only its own fragments)")
	flag.Parse()
	if *locales != "" {
		o.Locales = strings.Split(*locales, ",")
	}
	o.Out = os.Stdout

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	res, err := wsload.Run(ctx, o)
	if err != nil {
		log.Fatal(err)
	}
	if !res.OK() {
		os.Exit(1)
	}
}
