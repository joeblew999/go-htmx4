// Command intlbrowsers runs the kit/i18n conformance cases through Intl in real browsers and reports how their
// output drifts from the golden file (workerd, the pinned Chrome V8/ICU). kit/i18n matches the golden byte for
// byte; the drift shows what browser-side formatting (static/relative-time.js) would change on screen.
//
//	go run ./cmd/intlbrowsers -browsers chrome,firefox,safari -out build/i18n-browsers
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joeblew999/go-htmx4/kit/i18n/intltest"
)

func main() {
	browsers := flag.String("browsers", "chrome,firefox", "comma-separated: chrome, firefox, safari (safaridriver --enable once)")
	golden := flag.String("golden", "kit/i18n/testdata/golden/workerd.json", "golden file to compare with")
	out := flag.String("out", "build/i18n-browsers", "directory for <browser>.json drift reports")
	examples := flag.Int("examples", 3, "example differences shown per area")
	flag.Parse()

	want, err := intltest.Load(*golden)
	if err != nil {
		fail(err)
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fail(err)
	}
	cases := intltest.All()
	ran := 0
	for _, name := range strings.Split(*browsers, ",") {
		got, err := intltest.Browser{Name: name}.Run(context.Background(), cases)
		if err != nil {
			fmt.Fprintf(os.Stderr, "· %s skipped: %v\n", name, err)
			continue
		}
		ran++
		drift := intltest.Compare(want, got)
		fmt.Print(drift.Summary(*examples))
		b, err := json.MarshalIndent(drift, "", " ")
		if err != nil {
			fail(err)
		}
		path := filepath.Join(*out, name+".json")
		if err := os.WriteFile(path, b, 0o644); err != nil {
			fail(err)
		}
		fmt.Printf("  → %s\n\n", path)
	}
	if ran == 0 {
		fail(fmt.Errorf("no browser ran"))
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "intlbrowsers:", err)
	os.Exit(1)
}
