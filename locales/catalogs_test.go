package locales

import (
	"os"
	"testing"

	"github.com/joeblew999/go-htmx4/intl"
	"github.com/joeblew999/go-htmx4/intl/intltest"
)

func TestMain(m *testing.M) { os.Exit(intltest.Main(m)) }

// TestShippedLocalesComplete keeps every shipped locale indexable: a partial catalog serves noindex pages.
func TestShippedLocalesComplete(t *testing.T) {
	for _, l := range Shipped {
		if !intl.Complete(l.ID()) {
			t.Errorf("%s: catalog incomplete (pages get X-Robots-Tag: noindex)", l.ID())
		}
	}
}

func TestPlurals(t *testing.T) { intltest.Plurals(t) }
