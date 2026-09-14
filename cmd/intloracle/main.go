// Command intloracle records what JavaScript's Intl (workerd = Chrome's V8/ICU) outputs for the
// kit/i18n conformance cases (kit/i18n/intltest), as the golden file kit/i18n's tests compare against.
//
//	go run ./cmd/intloracle -out kit/i18n/testdata/golden/workerd.json
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/joeblew999/go-htmx4/kit/i18n/intltest"
)

func main() {
	out := flag.String("out", "kit/i18n/testdata/golden/workerd.json", "golden file to write")
	var o intltest.Oracle
	flag.StringVar(&o.Workerd, "workerd", "workerd", "workerd binary")
	flag.IntVar(&o.Port, "port", 8946, "loopback port for the oracle")
	flag.Parse()
	cases := intltest.All()
	g, err := o.Run(context.Background(), cases)
	if err != nil {
		fmt.Fprintln(os.Stderr, "intloracle:", err)
		os.Exit(1)
	}
	if err := g.Save(*out); err != nil {
		fmt.Fprintln(os.Stderr, "intloracle:", err)
		os.Exit(1)
	}
	fmt.Printf("✓ %s: %d cases from %s\n", *out, len(g.Results), g.Runtime)
}
