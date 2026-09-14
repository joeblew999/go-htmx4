// Command msggen generates typed message accessors from ICU MessageFormat catalogs (kit/i18n/msggen).
//
//	go run ./cmd/msggen -dir locales -out locales -locales en,de,ar
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/joeblew999/go-htmx4/kit/i18n/msggen"
)

func main() {
	var cfg msggen.Config
	var locales string
	flag.StringVar(&cfg.Dir, "dir", "locales", "catalog directory (<locale>.toml)")
	flag.StringVar(&cfg.Out, "out", "locales", "output package directory")
	flag.StringVar(&cfg.Package, "pkg", "", "package name (default: base of -out)")
	flag.StringVar(&cfg.Source, "source", "", "source locale (default: the first of -locales)")
	flag.StringVar(&locales, "locales", "", "comma-separated shipped locale ids (required)")
	flag.Parse()
	if locales == "" {
		flag.Usage()
		os.Exit(2)
	}
	cfg.Locales = strings.Split(locales, ",")
	cfg.Log = os.Stderr
	if err := msggen.Generate(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
