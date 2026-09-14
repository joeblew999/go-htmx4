package locales

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joeblew999/go-htmx4/kit/i18n/cldr"
	"github.com/joeblew999/go-htmx4/kit/i18n/msggen"
)

// TestGeneratedUpToDate fails when *_msg_gen.go doesn't match the catalogs: run mise run i18n:messages.
func TestGeneratedUpToDate(t *testing.T) {
	var ids []string
	for _, ld := range cldr.Data.Locales {
		ids = append(ids, ld.ID)
	}
	out := t.TempDir()
	if err := msggen.Generate(msggen.Config{Dir: ".", Out: out, Package: "locales", Locales: ids}); err != nil {
		t.Fatal(err)
	}
	gen, _ := filepath.Glob(filepath.Join(out, "*_msg_gen.go"))
	have, _ := filepath.Glob("*_msg_gen.go")
	if len(gen) != len(have) {
		t.Fatalf("generated %d files, the package has %d: run mise run i18n:messages", len(gen), len(have))
	}
	for _, f := range gen {
		want, _ := os.ReadFile(f)
		got, err := os.ReadFile(filepath.Base(f))
		if err != nil || string(got) != string(want) {
			t.Errorf("%s is stale: run mise run i18n:messages", filepath.Base(f))
		}
	}
}

// TestShippedLocalesComplete keeps every shipped locale indexable: a partial catalog serves noindex pages.
func TestShippedLocalesComplete(t *testing.T) {
	for _, ld := range cldr.Data.Locales {
		if !Complete(ld.ID) {
			t.Errorf("%s: catalog incomplete (pages get X-Robots-Tag: noindex)", ld.ID)
		}
	}
}
