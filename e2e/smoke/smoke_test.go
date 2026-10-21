// Package smoke checks a running Worker over plain HTTP: every route, the TinyGo build, the server's Intl, locales,
// SEO, the board through D1 and the Room, and live pushes. One list of checks for both places it runs:
//
//	mise run smoke          local workerd serving the production build (SMOKE_LOCAL=1: local-only checks too)
//	SMOKE_BASE=$(cat .deploy-url) mise run smoke   the deployed Worker
//
// SMOKE_BASE is the URL. go test runs the handlers on standard Go, which hides TinyGo runtime gaps; this runs the
// real wasm.
package smoke

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	base  = strings.TrimSuffix(os.Getenv("SMOKE_BASE"), "/")
	local = os.Getenv("SMOKE_LOCAL") == "1"
	// compiler is the Go compiler the Worker was built with (mise.toml WORKER_COMPILER: tinygo, or go → "gc").
	compiler = map[string]string{"": "tinygo", "tinygo": "tinygo", "go": "gc"}[os.Getenv("WORKER_COMPILER")]
	// No redirects followed: some checks are about the redirect itself.
	client = &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
)

type res struct {
	status int
	header http.Header
	body   string
}

func do(t *testing.T, method, path, form string, cookies ...string) res {
	t.Helper()
	var body io.Reader
	if form != "" {
		body = strings.NewReader(form)
	}
	req, err := http.NewRequest(method, base+path, body)
	if err != nil {
		t.Fatal(err)
	}
	if form != "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if len(cookies) > 0 {
		req.Header.Set("Cookie", strings.Join(cookies, "; "))
	}
	r, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer r.Body.Close()
	b, _ := io.ReadAll(r.Body)
	return res{r.StatusCode, r.Header, string(b)}
}

func get(t *testing.T, path string, cookies ...string) res {
	t.Helper()
	return do(t, "GET", path, "", cookies...)
}

func post(t *testing.T, path, form string) res { t.Helper(); return do(t, "POST", path, form) }

func has(t *testing.T, name, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		if len(got) > 200 {
			got = got[:200] + "…"
		}
		t.Errorf("✗ %s: want %q in %q", name, want, got)
	}
}

func TestMain(m *testing.M) {
	if base == "" {
		os.Stderr.WriteString("SMOKE_BASE is required (mise run smoke sets it)\n")
		os.Exit(2)
	}
	// A brand-new workers.dev hostname can take a little while to answer.
	for i := 0; i < 60; i++ {
		if r, err := client.Get(base + "/healthz"); err == nil {
			b, _ := io.ReadAll(r.Body)
			r.Body.Close()
			if string(b) == "ok" {
				break
			}
		}
		time.Sleep(2 * time.Second)
	}
	os.Exit(m.Run())
}

func TestPages(t *testing.T) {
	page := get(t, "/").body
	// runtime.Compiler of the build being checked: "tinygo", or "gc" for WORKER_COMPILER=go.
	has(t, "page runs on "+compiler+" wasm", page, `<span id="target" translate="no">`+compiler+` js/wasm</span>`)
	if local {
		has(t, "text binding APP_ENV", page, `<span id="env" translate="no">workerd (local)</span>`)
	}
	has(t, "home: boost + morph", page, `hx-boost:inherited="true"`)
	has(t, "home: hx-live counter", page, `hx-on:click="data.count++"`)
	has(t, "theme toggle in header", page, "data-site-theme-toggle")
	has(t, "about page", get(t, "/about").body, "About go-htmx4")
	has(t, "POST form, escaped", post(t, "/greet", "name="+url.QueryEscape("<b>Ada")).body, "Hello, &lt;b&gt;Ada.")
	has(t, "POST form, OOB toast", post(t, "/greet", "name=Ada").body, `hx-swap-oob="beforeend:#gsxui-toaster"`)
	has(t, "POST form empty", post(t, "/greet", "name=").body, "Please enter a name.")
	has(t, "DELETE /greet", do(t, "DELETE", "/greet", "").body, "Cleared")
	info := get(t, "/fragments/server-info").body
	has(t, "server-info names the compiler", info, compiler)
	has(t, "server-info says a runtime serves many requests", info, "serves many requests")
	has(t, "stats fragment", get(t, "/fragments/stats").body, "<ul")
	if r := get(t, "/greet"); r.status != 405 {
		t.Errorf("✗ 405 on wrong method: got %d", r.status)
	}
	if r := get(t, "/nope.txt"); r.status != 404 {
		t.Errorf("✗ 404 for missing file: got %d", r.status)
	}
}

func TestStaticAssets(t *testing.T) {
	for path, typ := range map[string]string{
		"/static/htmx.min.js":                   "text/javascript",
		"/static/hx-live.js":                    "text/javascript",
		"/static/hx-ws.js":                      "text/javascript",
		"/static/intl.js":                       "text/javascript",
		"/static/intl-core.mjs":                 "text/javascript", // a module: anything else and the browser refuses it
		"/gsxui/index.js":                       "text/javascript",
		"/assets/gsxui.css":                     "text/css",
		"/assets/geist-latin-wght-normal.woff2": "font/woff2",
	} {
		r := get(t, path)
		if r.status != 200 || !strings.HasPrefix(r.header.Get("Content-Type"), typ) {
			t.Errorf("✗ %s: %d %s, want 200 %s", path, r.status, r.header.Get("Content-Type"), typ)
		}
	}
}

// TestServerIntl: the Worker formats with its own V8 Intl (worker/intl.mjs via internal/i18n Format), so pages arrive
// finished, and picks each message's plural form with Intl.PluralRules.
func TestServerIntl(t *testing.T) {
	de := get(t, "/de/formats").body
	has(t, "Intl markup", de, `&#34;currency&#34;:&#34;EUR&#34;`)
	has(t, "dates", de, "Samstag, 4. Juli 2026")
	has(t, "currency", de, "1.234,50 €")
	has(t, "numbers, the locale's own digits", get(t, "/ar/formats").body, "١٬٢٣٤٬٥٦٧٫٨٩١")
	has(t, "zone names", get(t, "/formats").body, "11:30 AM Eastern Time")
	has(t, "display names", get(t, "/ja/formats").body, "日本")
	has(t, "plural rules", get(t, "/ru/formats").body, "2, 3, 4, 22, 23")
	has(t, "round trip: 1.234,5 → 1234.5", post(t, "/de/formats/try", "number="+url.QueryEscape("1.234,5")+"&text=Hallo").body, "1234.5")
	has(t, "chosen zone and clock reach the browser", get(t, "/formats", "tz=America/Los_Angeles", "hc=h23").body, `data-tz="America/Los_Angeles" data-hc="h23"`)
	has(t, "chosen zone on the server", get(t, "/fragments/server-info", "tz=America/Los_Angeles").body, "PDT")
	start := time.Now()
	if r := get(t, "/de/fragments/preferences"); r.status != 200 {
		t.Errorf("✗ preferences: %d", r.status)
	}
	t.Logf("· /de/fragments/preferences took %v", time.Since(start).Round(time.Millisecond))
}

func TestLocales(t *testing.T) {
	has(t, "locale page /de/", get(t, "/de/").body, `<html lang="de" dir="ltr"`)
	has(t, "rtl page /ar/formats", get(t, "/ar/formats").body, `<html lang="ar" dir="rtl"`)
	has(t, "rtl page /he/about", get(t, "/he/about").body, `<html lang="he" dir="rtl"`)
	about := get(t, "/fr/about").body
	has(t, "language names, each in its own language", about, "português (Brasil)")
	has(t, "language names, right to left", about, "العربية")
	has(t, "locale facts from Intl", get(t, "/zh-hant/formats").body, "zh-Hant-TW")
	has(t, "UI catalog", get(t, "/de/").body, "Go + htmx 4 + gsxui auf Cloudflare Workers")
	has(t, "ICU plural", post(t, "/ru/board/note?topic=smoke", "body=").body, "до 280 символов")
	has(t, "localized hx-post", get(t, "/de/board?topic=smoke").body, `hx-post="/de/board/add?topic=smoke`)
	// A preview or staging Worker (APP_PREVIEW) is noindex everywhere; otherwise a translated page must be indexable.
	if get(t, "/").header.Get("X-Robots-Tag") == "" {
		if r := get(t, "/de/about"); r.header.Get("X-Robots-Tag") != "" {
			t.Errorf("✗ translated page indexable: X-Robots-Tag %q", r.header.Get("X-Robots-Tag"))
		}
	}
	if r := get(t, "/en/about"); r.status != 301 {
		t.Errorf("✗ /en/ prefix → 301: got %d", r.status)
	}
	if r := get(t, "/xx/"); r.status != 404 {
		t.Errorf("✗ unknown locale → 404: got %d", r.status)
	}
}

func TestSEO(t *testing.T) {
	about := get(t, "/de/about").body
	has(t, "canonical + hreflang", about, `<link rel="canonical" href="`+base+`/de/about">`)
	has(t, "hreflang x-default", about, `<link rel="alternate" hreflang="x-default" href="`+base+`/about">`)
	has(t, "Open Graph url + locale", about, `<meta property="og:url" content="`+base+`/de/about"><meta property="og:locale" content="de_DE">`)
	has(t, "Open Graph locale: script → territory", get(t, "/zh-hant/").body, `<meta property="og:locale" content="zh_TW">`)
	has(t, "Open Graph alternates", about, `<meta property="og:locale:alternate" content="pt_BR">`)
	has(t, "sitemap with alternates", get(t, "/sitemap.xml").body, `<xhtml:link rel="alternate" hreflang="zh-Hant" href="`+base+`/zh-hant/formats"/>`)
	has(t, "robots.txt is ours", get(t, "/robots.txt").body, "Sitemap: "+base+"/sitemap.xml")
	has(t, "fragment noindex", get(t, "/fragments/stats").header.Get("X-Robots-Tag"), "noindex")
}

// TestBoard: Go (TinyGo) → database/sql d1 driver → D1 → publish → Room Durable Object → hx-ws, and live pushes to
// WebSockets in three locales (cmd/wsload).
func TestBoard(t *testing.T) {
	has(t, "board page with hx-ws", get(t, "/board?topic=smoke").body, `hx-ws:connect="/live/smoke?locale=en"`)
	has(t, "board write via D1", post(t, "/board/add?topic=smoke", "delta=1").body, `hx-swap-oob="true" data-version="`)
	if r := get(t, "/live/smoke"); r.status != 426 {
		t.Errorf("✗ /live needs an upgrade: got %d", r.status)
	}
	root, _ := filepath.Abs("../..")
	cmd := exec.Command("go", "run", "./cmd/wsload", "-base", base, "-n", "3", "-locales", "en,de,ar", "-topic", "smoke")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("✗ live push to 3 WebSockets in 3 locales (wsload): %v\n%s", err, out)
	}
}

// TestWriteLimit: over the write limit (60 per 10 s per client) the board answers 429 with a toast. Local only: it
// uses up this client's budget. The writes go out in parallel: one after another they took over 10 s on a CI runner,
// so the window reset and nothing was refused.
func TestWriteLimit(t *testing.T) {
	if !local {
		t.Skip("local only: it would use up the deployed Worker's write budget for this client")
	}
	var mu sync.Mutex
	codes := map[int]int{}
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)
	for i := 0; i < 70; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			r := post(t, "/board/add?topic=smoke-limit", "delta=1")
			mu.Lock()
			codes[r.status]++
			mu.Unlock()
		}()
	}
	wg.Wait()
	if codes[429] == 0 {
		t.Errorf("✗ write limit → 429: got %v", codes)
	}
	has(t, "429 carries a toast", post(t, "/board/add?topic=smoke-limit", "delta=1").body, "Slow down")
}
