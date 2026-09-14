// Package e2e drives the running app in headless Chrome (chromedp, no Node): pages, the live board across
// two browsers, boosted navigation, presence and the write limit. It's its own module so chromedp stays out
// of the app's and kit/'s dependencies.
//
//	mise run e2e                               # builds, starts local workerd, runs these tests
//	E2E_BASE=https://….workers.dev mise run e2e # against a deployed Worker
package e2e

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

var base = strings.TrimRight(os.Getenv("E2E_BASE"), "/")

func TestMain(m *testing.M) {
	if base == "" {
		fmt.Println("e2e: set E2E_BASE to the app's URL (mise run e2e does)")
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// tab is one browser (its own Chrome process: hx-ws closes the socket of a hidden tab, so tabs in one
// browser would disconnect each other).
type tab struct {
	t    *testing.T
	name string
	ctx  context.Context

	mu       sync.Mutex
	problems []string // console errors, exceptions, HTTP ≥ 400 (except favicon)
	wsOpened int
	wsFrames int
	allow4xx bool // tests that expect 4xx responses
}

func newTab(t *testing.T, name string) *tab {
	t.Helper()
	// No QUIC: some networks break Chrome's HTTP/3 to Cloudflare (net::ERR_QUIC_PROTOCOL_ERROR); HTTP/2 tests the same app.
	opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.WindowSize(1100, 1000), chromedp.Flag("disable-quic", true))
	if os.Getenv("CI") != "" {
		opts = append(opts, chromedp.NoSandbox)
	}
	alloc, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, cancelCtx := chromedp.NewContext(alloc)
	ctx, cancelTimeout := context.WithTimeout(ctx, 2*time.Minute)
	t.Cleanup(func() { cancelTimeout(); cancelCtx(); cancelAlloc() })
	tb := &tab{t: t, name: name, ctx: ctx}
	chromedp.ListenTarget(ctx, func(ev any) {
		tb.mu.Lock()
		defer tb.mu.Unlock()
		switch e := ev.(type) {
		case *runtime.EventConsoleAPICalled:
			if e.Type == runtime.APITypeError || e.Type == runtime.APITypeWarning {
				var parts []string
				for _, a := range e.Args {
					parts = append(parts, strings.Trim(string(a.Value), `"`)+a.Description)
				}
				tb.problems = append(tb.problems, string(e.Type)+": "+strings.Join(parts, " "))
			}
		case *runtime.EventExceptionThrown:
			tb.problems = append(tb.problems, "exception: "+e.ExceptionDetails.Error())
		case *network.EventResponseReceived:
			if e.Response.Status >= 400 && !strings.HasSuffix(e.Response.URL, "/favicon.ico") && !(tb.allow4xx && e.Response.Status < 500) {
				tb.problems = append(tb.problems, fmt.Sprintf("HTTP %d %s", e.Response.Status, e.Response.URL))
			}
		case *network.EventWebSocketHandshakeResponseReceived:
			tb.wsOpened++
		case *network.EventWebSocketFrameReceived:
			tb.wsFrames++
		}
	})
	tb.run(network.Enable())
	return tb
}

func (tb *tab) run(actions ...chromedp.Action) {
	tb.t.Helper()
	if err := chromedp.Run(tb.ctx, actions...); err != nil {
		tb.t.Fatalf("%s: %v", tb.name, err)
	}
}

func (tb *tab) open(path string) {
	tb.t.Helper()
	tb.run(chromedp.Navigate(base+path), chromedp.WaitReady("main", chromedp.ByQuery))
}

// eval runs JavaScript (awaiting promises) and decodes the result into out (nil to ignore it).
func (tb *tab) eval(js string, out any) {
	tb.t.Helper()
	var ignore any
	if out == nil {
		out = &ignore
	}
	if err := chromedp.Run(tb.ctx, chromedp.Evaluate(js, out, func(p *runtime.EvaluateParams) *runtime.EvaluateParams {
		return p.WithAwaitPromise(true)
	})); err != nil {
		tb.t.Fatalf("%s: eval %.100q: %v", tb.name, js, err)
	}
}

func (tb *tab) str(js string) string { var s string; tb.eval(js, &s); return s }
func (tb *tab) num(js string) int    { var n int; tb.eval(js, &n); return n }
func (tb *tab) is(js string) bool    { var b bool; tb.eval(js, &b); return b }

// waitJS polls a JavaScript condition. An exception counts as "not yet": during a full navigation the document
// being parsed can briefly have no <html> or <body>.
func (tb *tab) waitJS(d time.Duration, js string) bool {
	tb.t.Helper()
	return waitFor(d, func() bool {
		var ok bool
		err := chromedp.Run(tb.ctx, chromedp.Evaluate(js, &ok, func(p *runtime.EvaluateParams) *runtime.EvaluateParams {
			return p.WithAwaitPromise(true)
		}))
		return err == nil && ok
	})
}

// clickText clicks the first element matching selector whose trimmed text is text.
func (tb *tab) clickText(selector, text string) {
	tb.t.Helper()
	if !tb.is(fmt.Sprintf(`(() => { const el = [...document.querySelectorAll(%q)].find(e => e.textContent.trim() === %q); if (!el) return false; el.click(); return true; })()`, selector, text)) {
		tb.t.Fatalf("%s: no %s with text %q", tb.name, selector, text)
	}
}

func (tb *tab) presence() string {
	return tb.str(`document.getElementById("presence")?.textContent ?? "(none)"`)
}

func (tb *tab) version() int {
	return tb.num(`Number(document.getElementById("board")?.dataset.version ?? -1)`)
}

func (tb *tab) noProblems() {
	tb.t.Helper()
	tb.mu.Lock()
	defer tb.mu.Unlock()
	if len(tb.problems) > 0 {
		tb.t.Errorf("%s: console errors, exceptions or HTTP errors: %v", tb.name, tb.problems)
	}
}

func waitFor(d time.Duration, cond func() bool) bool {
	for end := time.Now().Add(d); time.Now().Before(end); time.Sleep(50 * time.Millisecond) {
		if cond() {
			return true
		}
	}
	return cond()
}

func check(t *testing.T, name string, ok bool, detail ...any) {
	t.Helper()
	if ok {
		t.Logf("✓ %s %s", name, fmt.Sprint(detail...))
	} else {
		t.Errorf("✗ %s %s", name, fmt.Sprint(detail...))
	}
}

func topic(prefix string) string { return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano()%1e9) }
