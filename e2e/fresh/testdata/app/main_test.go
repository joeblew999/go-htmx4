package main

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/joeblew999/go-htmx4/intl/intltest"
)

func TestMain(m *testing.M) { os.Exit(intltest.Main(m)) }

// TestHome: the home page renders, with workerd's Intl, in a test as intl/README.md says to write one.
func TestHome(t *testing.T) {
	var b bytes.Buffer
	if err := Home(Order{Placed: time.Now(), Total: 1}, nil, Note{Created: time.Now()}).Render(context.Background(), &b); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), "<h1>Welcome</h1>") {
		t.Errorf("no heading in\n%s", b.String())
	}
}
