// Package e2e drives the running app in headless Chrome (chromedp, no Node): pages, the live board across
// two browsers, boosted navigation, presence and the write limit. It's its own module so chromedp stays out
// of the app's dependencies.
//
//	mise run e2e                               # builds, starts local workerd, runs these tests
//	E2E_BASE=https://….workers.dev mise run e2e # against a deployed Worker
package e2e

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chromedp/cdproto/cdp"
	cdplog "github.com/chromedp/cdproto/log"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

// BASE is the Worker's URL: go tool charter test e2e sets it (wrangler dev, or -base https://… for a deployed one).
var base = strings.TrimRight(os.Getenv("BASE"), "/")

func TestMain(m *testing.M) {
	if base == "" {
		fmt.Println("e2e: no BASE, the app's URL: go tool charter test e2e")
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
	problems []string // console errors, exceptions, CSP violations, HTTP ≥ 400 (except favicon)
	wsOpened int
	wsFrames int
	allow4xx bool     // tests that expect 4xx responses
	known    []string // problems a test expects (an upstream bug, named with its report), matched by substring
}

func newTab(t *testing.T, name string) *tab {
	t.Helper()
	return newTabWith(t, name)
}

// newTabWith is newTab with Chrome started with extra options (the live page's fake camera).
func newTabWith(t *testing.T, name string, extra ...chromedp.ExecAllocatorOption) *tab {
	t.Helper()
	// No QUIC: some networks break Chrome's HTTP/3 to Cloudflare (net::ERR_QUIC_PROTOCOL_ERROR); HTTP/2 tests the same app.
	// WSURLReadTimeout: chromedp waits 20 s for Chrome to start by default; the first cold start on a CI runner took longer
	// ("websocket url timeout reached").
	opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.WindowSize(1100, 1000), chromedp.Flag("disable-quic", true),
		chromedp.WSURLReadTimeout(90*time.Second))
	opts = append(opts, extra...)
	if os.Getenv("CI") != "" {
		opts = append(opts, chromedp.NoSandbox)
	}
	// chromedp finds the machine's Chrome; CHROME names another.
	if exe := os.Getenv("CHROME"); exe != "" {
		opts = append(opts, chromedp.ExecPath(exe))
	}
	alloc, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, cancelCtx := chromedp.NewContext(alloc)
	ctx, cancelTimeout := context.WithTimeout(ctx, 2*time.Minute)
	t.Cleanup(func() { cancelTimeout(); cancelCtx(); cancelAlloc() })
	tb := &tab{t: t, name: name, ctx: ctx}
	listen := func(ev any) {
		tb.mu.Lock()
		defer tb.mu.Unlock()
		switch e := ev.(type) {
		case *runtime.EventConsoleAPICalled:
			if e.Type == runtime.ConsoleAPICalledTypeError || e.Type == runtime.ConsoleAPICalledTypeWarning {
				var parts []string
				for _, a := range e.Args {
					parts = append(parts, strings.Trim(string(a.Value), `"`)+a.Description)
				}
				tb.problems = append(tb.problems, string(e.Type)+": "+strings.Join(parts, " "))
			}
		case *cdplog.EventEntryAdded:
			// Chrome reports what the page's Content Security Policy blocked here, not on the console API.
			if e.Entry.Source == cdplog.EntrySourceSecurity || strings.Contains(e.Entry.Text, "Content Security Policy") {
				tb.problems = append(tb.problems, "CSP: "+e.Entry.Text)
			}
		case *runtime.EventExceptionThrown:
			tb.problems = append(tb.problems, "exception: "+(&chromedp.ExceptionError{ExceptionDetails: e.ExceptionDetails}).Error())
		case *network.EventResponseReceived:
			if e.Response.Status >= 400 && !strings.HasSuffix(e.Response.URL, "/favicon.ico") && !(tb.allow4xx && e.Response.Status < 500) {
				tb.problems = append(tb.problems, fmt.Sprintf("HTTP %d %s", e.Response.Status, e.Response.URL))
			}
		case *network.EventWebSocketHandshakeResponseReceived:
			tb.wsOpened++
		case *network.EventWebSocketFrameReceived:
			tb.wsFrames++
		}
	}
	if err := chromedp.Listen(ctx, listen); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	call(tb, network.Enable, network.EnableParams{})
	call(tb, cdplog.Enable, cdp.Empty{})
	return tb
}

// call sends one protocol command to the tab and returns its result.
func call[P, R any](tb *tab, cmd cdp.Command[P, R], params P) R {
	tb.t.Helper()
	res, err := chromedp.Call(tb.ctx, cmd, params)
	if err != nil {
		tb.t.Fatalf("%s: %s: %v", tb.name, cmd.Method, err)
	}
	return res
}

func (tb *tab) run(actions ...chromedp.Action[chromedp.Void]) {
	tb.t.Helper()
	if err := chromedp.Do(tb.ctx, actions...); err != nil {
		tb.t.Fatalf("%s: %v", tb.name, err)
	}
}

func (tb *tab) open(path string) {
	tb.t.Helper()
	tb.run(chromedp.Navigate(base+path), chromedp.WaitReady(chromedp.CSS("main")))
}

// evalAs runs JavaScript (awaiting promises) and returns its result as a T.
func evalAs[T any](tb *tab, js string) T {
	tb.t.Helper()
	v, err := chromedp.Run(tb.ctx, chromedp.Evaluate[T](js, chromedp.EvalAwaitPromise))
	if err != nil {
		tb.t.Fatalf("%s: eval %.100q: %v", tb.name, js, err)
	}
	return v
}

// eval runs JavaScript (awaiting promises) for what it does: its result, whatever it is, is dropped.
func (tb *tab) eval(js string) {
	tb.t.Helper()
	if _, err := chromedp.Run(tb.ctx, chromedp.Evaluate[chromedp.Void](js, chromedp.EvalAwaitPromise)); err != nil {
		tb.t.Fatalf("%s: eval %.100q: %v", tb.name, js, err)
	}
}

func (tb *tab) str(js string) string { return evalAs[string](tb, js) }
func (tb *tab) num(js string) int    { return evalAs[int](tb, js) }
func (tb *tab) is(js string) bool    { return evalAs[bool](tb, js) }

// waitJS polls a JavaScript condition. An exception counts as "not yet": during a full navigation the document
// being parsed can briefly have no <html> or <body>.
func (tb *tab) waitJS(d time.Duration, js string) bool {
	tb.t.Helper()
	return waitFor(d, func() bool {
		ok, err := chromedp.Run(tb.ctx, chromedp.Evaluate[bool](js, chromedp.EvalAwaitPromise))
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

// presence is the online badge's text, e.g. "2 online": the Room pushes the count into #presence and the
// page renders the label next to it.
func (tb *tab) presence() string {
	return tb.str(`(() => { const p = document.getElementById("presence")?.parentElement; return p ? [...p.childNodes].map((n) => n.textContent.trim()).filter(Boolean).join(" ") : "(none)"; })()`)
}

func (tb *tab) version() int {
	return tb.num(`Number(document.getElementById("board")?.dataset.version ?? -1)`)
}

func (tb *tab) noProblems() {
	tb.t.Helper()
	tb.mu.Lock()
	defer tb.mu.Unlock()
	var problems []string
	for _, p := range tb.problems {
		if !slices.ContainsFunc(tb.known, func(k string) bool { return strings.Contains(p, k) }) {
			problems = append(problems, p)
		}
	}
	if len(problems) > 0 {
		tb.t.Errorf("%s: console errors, exceptions or HTTP errors: %v", tb.name, problems)
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
