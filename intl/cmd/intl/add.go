package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joeblew999/go-htmx4/intl/config"
	"github.com/joeblew999/go-htmx4/intl/registry"
	"github.com/joeblew999/go-htmx4/intl/setup"
	intlui "github.com/joeblew999/go-htmx4/intl/ui"
)

// add is go tool intl add: intl's screens, copied into the app the way gsxui add copies components (#77), into
// intl.toml's ui (views/intl, package intlui), with their text, already translated, written into the catalogs. The
// copying is intl/registry's, which gsxkit's components use too (#164).
func add(c config.Config, args []string) int {
	f := flag.NewFlagSet("add", flag.ExitOnError)
	diff := f.Bool("diff", false, "show how intl's screens differ from the app's copies, writing nothing (exit 1 if they do)")
	overwrite := f.Bool("overwrite", false, "replace copies the app changed")
	f.Parse(args)
	if f.NArg() == 0 || *diff && *overwrite {
		fmt.Fprintf(os.Stderr, "usage: go tool intl add [-diff | -overwrite] <screen>…   screens: %s\n", strings.Join(intlui.Screens.Names(), ", "))
		return 2
	}
	if !*diff && c.UI == "" {
		check(setup.UI(c.Root, filepath.ToSlash(c.UIDir()), os.Stdout))
	}
	code, err := intlui.Screens.Add(c, f.Args(), registry.Options{Dir: c.UIDir(), Diff: *diff, Overwrite: *overwrite})
	check(err)
	return code
}
