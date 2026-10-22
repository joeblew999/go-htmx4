// Package fresh is the readiness test (#65): a gsx app made from nothing, outside this repo, the way intl/README.md
// and seo/README.md say, with gsxui and intl's screens (go tool intl add), served by an ordinary Go server (not a
// Worker), answering in two languages, passing its own go test, seo's audit and intl:check. Its code is
// testdata/app, and every go and gsx block in those READMEs is in it as they write it (TestREADMECode). If an outside
// developer couldn't get this far by following the READMEs, neither can this test.
//
// intl, seo and badge come from this checkout; FRESH_VERSIONS=published takes the versions published on the Go
// proxy instead (mise run fresh:published, after a release).
package fresh

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"testing"
	"time"
	"unicode"
)

const module = "github.com/joeblew999/go-htmx4"

func TestFreshApp(t *testing.T) {
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	app := t.TempDir()
	// mise runs the app's own tasks (intl init writes them) without asking to trust its config.
	env := append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=mod", "MISE_TRUSTED_CONFIG_PATHS="+app, "MISE_YES=1")
	gsxui := "gsxui"
	if bin := gsxuiBin(t, repo); bin != "" {
		env = append(env, "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"))
		gsxui = filepath.Join(bin, "gsxui")
	}
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = app
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s: %v\n%s", strings.Join(args, " "), err, out)
		}
		return string(out)
	}
	write := func(name, content string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(filepath.Join(app, name)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(app, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// A new module with gsx, intl and seo as tools, as the READMEs' "Set up an app" starts. gsx at the version this repo
	// builds with; Go from this repo's toolchain, so mise (which reads go.mod's toolchain) wants a Go that's installed.
	gomod := read(t, filepath.Join(repo, "go.mod"))
	gsxVersion := regexp.MustCompile(`github.com/gsxhq/gsx (v\S+)`).FindStringSubmatch(gomod)[1]
	goVersion := regexp.MustCompile(`(?m)^go (\S+)`).FindStringSubmatch(gomod)[1]
	toolchain := regexp.MustCompile(`(?m)^toolchain (\S+)`).FindStringSubmatch(gomod)[1]
	write("go.mod", "module example.com/fresh\n\ngo "+goVersion+"\n\ntoolchain "+toolchain+"\n")
	run("go", "mod", "edit", "-require=github.com/gsxhq/gsx@"+gsxVersion, "-tool=github.com/gsxhq/gsx/cmd/gsx")
	if os.Getenv("FRESH_VERSIONS") == "published" {
		run("go", "get", "-tool", module+"/intl/cmd/intl@latest", module+"/seo/cmd/seo@latest")
	} else {
		run("go", "mod", "edit",
			"-require="+module+"/intl@v0.0.0", "-replace="+module+"/intl="+filepath.Join(repo, "intl"), "-tool="+module+"/intl/cmd/intl",
			"-require="+module+"/seo@v0.0.0", "-replace="+module+"/seo="+filepath.Join(repo, "seo"), "-tool="+module+"/seo/cmd/seo",
			"-replace="+module+"/badge="+filepath.Join(repo, "badge"))
	}
	run("go", "mod", "tidy")

	// gsxui the npm-free way, with a component that brings its icon package (gsxui's own code, which intl:check must
	// leave alone) and the README's slider; go tool intl init; intl's screens (go tool intl add runs gsxui add for the
	// components they need, and writes their text in both languages); the app's own messages and code
	// (testdata/app); the browser's Intl (go tool intl assets).
	run(gsxui, "init", "-rtl")
	run(gsxui, "add", "accordion", "slider")
	run("go", "tool", "intl", "init", "-locales", "en,de")
	run("go", "tool", "intl", "add", "language-list", "language-offer", "preferences")
	for _, l := range []string{"en", "de"} {
		write("locales/"+l+".toml", read(t, filepath.Join(app, "locales", l+".toml"))+read(t, filepath.Join("testdata", l+".toml")))
	}
	for _, f := range []string{"main.go", "page.gsx", "main_test.go"} {
		write(f, read(t, filepath.Join("testdata", "app", f)))
	}
	write("static/favicon.svg", read(t, filepath.Join("testdata", "app", "favicon.svg")))
	run("go", "tool", "intl", "assets", "-browser", "static")
	run("go", "tool", "intl", "gen")
	run("go", "tool", "gsx", "generate")
	run("go", "build", "-o", "fresh", ".")
	run("go", "test", "./...")

	// Serve it, and read the pages in both languages.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()
	srv := exec.Command(filepath.Join(app, "fresh"))
	srv.Dir, srv.Env, srv.Stderr = app, append(env, "ADDR="+addr), os.Stderr
	if err := srv.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { srv.Process.Signal(syscall.SIGTERM); srv.Wait() }()
	base := "http://" + addr
	for deadline := time.Now().Add(60 * time.Second); ; {
		if resp, err := http.Get(base + "/robots.txt"); err == nil {
			resp.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the fresh app didn't start")
		}
		time.Sleep(100 * time.Millisecond)
	}

	// What intl/README.md says the home page renders for a German request, and intl's screens in each language: the
	// language list, the dialog's button, its form, and the offer of the page in the language the browser asks for.
	for _, tc := range []struct {
		path, accept string
		wants        []string
	}{
		{"/", "", []string{`lang="en"`, "<h1>Welcome</h1>", "1 note", "July 4, 2026 · €1,234.50", "3 minutes ago · Japan", "42 kilometers",
			`<meta name="description" content="Notes and orders, in every language."`, `hreflang="de"`, ">Deutsch<", `aria-label="Time zone and clock"`}},
		{"/de/", "", []string{`lang="de"`, "<h1>Willkommen</h1>", "1 Notiz", "4. Juli 2026 · 1.234,50 €", "vor 3 Minuten · Japan",
			`"@type":"WebSite"`, `aria-current="page"`, `aria-label="Zeitzone und Uhrzeitformat"`}},
		{"/about", "", []string{"<title>About · Shop</title>", `<link rel="canonical" href="` + base + `/about">`}},
		{"/", "de", []string{"Diese Seite gibt es auch auf \u2068Deutsch\u2069.", "Nein danke"}},
		{"/de/fragments/preferences", "", []string{`action="/de/preferences"`, "Zeitzone", "Uhrzeitformat", "Speichern", "Schließen"}},
		{"/static/intl-browser.mjs", "", []string{"import"}},
	} {
		body := get(t, base+tc.path, tc.accept)
		text := visible(body)
		for _, want := range tc.wants {
			if !strings.Contains(body, want) && !strings.Contains(text, want) {
				t.Errorf("GET %s (Accept-Language %q): no %q in\n%s", tc.path, tc.accept, want, body)
			}
		}
		// A // comment after an element's > on the same line is text in gsx: none may reach a page.
		if strings.HasSuffix(tc.path, "/") && strings.Contains(text, " // ") {
			t.Errorf("GET %s: a comment rendered as text:\n%s", tc.path, text)
		}
	}

	// seo's audit of the running app, as Googlebot reads it: robots.txt, the sitemap, every page's head.
	run("go", "tool", "seo", "init", "-base", base)
	run("mise", "run", "seo:audit")

	// What the README says CI runs: the catalogs, then gsx's own checks on the app's markup, not gsxui's.
	run("mise", "run", "intl:check")
}

// TestREADMECode: every go and gsx block in intl's and seo's READMEs is the fresh app's code, as the README writes it,
// so the fresh-app test builds and runs it. A block's "// …" line stands for code it leaves out: the pieces around it
// come in that order. Whitespace doesn't count (gsx fmt and gofmt lay the app's code out), and neither do comments in
// a go block; a gsx block's do, since in markup some are text.
func TestREADMECode(t *testing.T) {
	var src strings.Builder
	for _, f := range []string{"main.go", "page.gsx", "main_test.go"} {
		src.WriteString(squeeze(stripComments(read(t, filepath.Join("testdata", "app", f)), strings.HasSuffix(f, ".go"))))
	}
	code := src.String()
	n := 0
	for _, readme := range []string{"../../intl/README.md", "../../seo/README.md"} {
		for _, b := range blocks(read(t, readme)) {
			n++
			at := 0
			for i, piece := range strings.Split(b.code, "// …") {
				p := squeeze(stripComments(piece, b.lang == "go"))
				if p == "" {
					continue
				}
				j := strings.Index(code[at:], p)
				if j < 0 {
					t.Errorf("%s:%d: the %s block isn't the fresh app's code (testdata/app) as written; piece %d:\n%s", readme, b.line, b.lang, i+1, piece)
					break
				}
				at += j + len(p)
			}
		}
	}
	if n < 9 {
		t.Errorf("found %d go and gsx blocks in the READMEs, want at least 9", n)
	}
}

type block struct {
	lang, code string
	line       int
}

// fence is a fenced code block's opening or closing line, indented as in a list item.
var fence = regexp.MustCompile("^\\s*```(\\w*)\\s*$")

// blocks are a Markdown file's go and gsx code blocks.
func blocks(md string) []block {
	var out []block
	var cur *block
	for i, line := range strings.Split(md, "\n") {
		m := fence.FindStringSubmatch(line)
		switch {
		case m != nil && cur == nil:
			cur = &block{lang: m[1], line: i + 1}
		case m != nil:
			if cur.lang == "go" || cur.lang == "gsx" {
				out = append(out, *cur)
			}
			cur = nil
		case cur != nil:
			cur.code += line + "\n"
		}
	}
	return out
}

// stripComments drops Go's // comments, outside strings, when goCode.
func stripComments(s string, goCode bool) string {
	if !goCode {
		return s
	}
	var out strings.Builder
	for _, line := range strings.Split(s, "\n") {
		var quote byte
		cut := len(line)
		for i := 0; i < len(line) && cut == len(line); i++ {
			switch c := line[i]; {
			case quote != 0:
				if c == '\\' && quote != '`' {
					i++
				} else if c == quote {
					quote = 0
				}
			case c == '"' || c == '`' || c == '\'':
				quote = c
			case c == '/' && strings.HasPrefix(line[i:], "//"):
				cut = i
			}
		}
		out.WriteString(line[:cut] + "\n")
	}
	return out.String()
}

// squeeze is s without whitespace.
func squeeze(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, s)
}

// visible is a page's text, its tags removed and no-break spaces made spaces, as a reader sees it.
func visible(html string) string {
	text := regexp.MustCompile(`<[^>]*>`).ReplaceAllString(html, "")
	return strings.NewReplacer("\u00a0", " ", "\u202f", " ").Replace(text)
}

// gsxuiBin is "" when the gsxui command is installed (mise's), else a directory to put first on PATH with a gsxui that
// is go run at the commit mise.toml pins: CI installs only the tools it builds with, and go tool intl add runs gsxui
// itself.
func gsxuiBin(t *testing.T, repo string) string {
	if _, err := exec.LookPath("gsxui"); err == nil {
		return ""
	}
	pin := regexp.MustCompile(`"go:(github.com/gsxhq/gsxui/cmd/gsxui)" = "([^"]+)"`).FindStringSubmatch(read(t, filepath.Join(repo, "mise.toml")))
	if pin == nil {
		t.Fatal("mise.toml pins no gsxui")
	}
	bin := t.TempDir()
	script := "#!/bin/sh\nexec go run " + pin[1] + "@" + pin[2] + " \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "gsxui"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return bin
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// get is the body of url, asking for the language accept when it isn't "".
func get(t *testing.T, url, accept string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	if accept != "" {
		req.Header.Set("Accept-Language", accept)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		t.Fatalf("GET %s: %d\n%s", url, resp.StatusCode, b)
	}
	return string(b)
}
