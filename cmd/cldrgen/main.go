// Command cldrgen generates kit/i18n CLDR data tables for the locales an app ships (kit/i18n/cldrgen).
//
//	go run ./cmd/cldrgen -locales en,de,ar -out locales/cldr
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/joeblew999/go-htmx4/kit/i18n/cldrgen"
)

func main() {
	var cfg cldrgen.Config
	var locales, currencies string
	flag.StringVar(&locales, "locales", "", "comma-separated BCP 47 ids to ship; the first is the default (required)")
	flag.StringVar(&cfg.Out, "out", "", "output package directory (required)")
	flag.StringVar(&cfg.Package, "pkg", "", "package name (default: base of -out)")
	flag.StringVar(&cfg.Tag, "tag", cldrgen.DefaultTag, "cldr-json release tag")
	flag.StringVar(&cfg.CacheDir, "cache", "", "cldr-json cache directory (default: user cache dir)")
	flag.StringVar(&currencies, "currencies", "", "currency codes to include names for (default: all)")
	quiet := flag.Bool("q", false, "no per-locale log")
	flag.Parse()
	if locales == "" || cfg.Out == "" {
		flag.Usage()
		os.Exit(2)
	}
	cfg.Locales = strings.Split(locales, ",")
	if currencies != "" {
		cfg.Currencies = strings.Split(currencies, ",")
	}
	if !*quiet {
		cfg.Log = os.Stderr
	}
	if err := cldrgen.Generate(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
