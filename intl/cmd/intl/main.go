// Command intl is go tool intl, an app's intl tooling (its settings: intl.toml, found up from the working
// directory):
//
//	go tool intl gen                     compile the catalogs: intl-messages.mjs, intl_gen.go, the gsxui translator
//	go tool intl assets -worker build    the JS a Worker imports: intl.mjs, intl-worker.mjs, intl-messages.mjs
//	go tool intl assets -browser static  the JS a page imports: intl.mjs, intl-browser.mjs
//	go tool intl locales                 the shipped locales, comma-separated (for scripts)
package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/joeblew999/go-htmx4/intl/config"
	"github.com/joeblew999/go-htmx4/intl/gen"
	intljs "github.com/joeblew999/go-htmx4/intl/js"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	c, err := config.Find(".")
	check(err)
	switch os.Args[1] {
	case "gen":
		check(gen.Generate(c, os.Stderr))
	case "assets":
		f := flag.NewFlagSet("assets", flag.ExitOnError)
		worker := f.String("worker", "", "directory for the Worker's modules")
		browser := f.String("browser", "", "directory for the page's modules")
		f.Parse(os.Args[2:])
		if *worker == "" && *browser == "" {
			usage()
		}
		if *worker != "" {
			check(copyJS(*worker, "intl.mjs", "intl-worker.mjs"))
			b, err := os.ReadFile(filepath.Join(c.CatalogDir(), "intl-messages.mjs"))
			check(err)
			check(os.WriteFile(filepath.Join(*worker, "intl-messages.mjs"), b, 0o644))
		}
		if *browser != "" {
			check(copyJS(*browser, "intl.mjs", "intl-browser.mjs"))
		}
	case "locales":
		fmt.Println(strings.Join(c.Locales, ","))
	default:
		usage()
	}
}

func copyJS(dir string, names ...string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, n := range names {
		b, err := fs.ReadFile(intljs.FS, n)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, n), b, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: go tool intl gen | assets [-worker dir] [-browser dir] | locales")
	os.Exit(2)
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "intl:", err)
		os.Exit(1)
	}
}
