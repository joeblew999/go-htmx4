// Package fresh is the readiness test (#65): a gsx app made from nothing, outside this repo, with gsxui, set up with
// intl from this checkout the way intl/README.md says, served by an ordinary Go server (not a Worker), answering in
// two languages, and passing intl:check. If an outside developer couldn't get this far by following the README, neither can this test.
package fresh

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"testing"
	"time"
)

const intlModule = "github.com/joeblew999/go-htmx4/intl"

func TestFreshApp(t *testing.T) {
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	app := t.TempDir()
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = app
		// mise runs the app's own tasks (intl init writes them) without asking to trust its config.
		cmd.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=mod", "MISE_TRUSTED_CONFIG_PATHS="+app, "MISE_YES=1")
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
	appendTo := func(name, content string) {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(app, name))
		if err != nil {
			t.Fatal(err)
		}
		write(name, string(b)+content)
	}

	// A new module with gsx and intl as tools, as intl/README.md's "Set up an app" starts. intl comes from this
	// checkout (the replace), gsx at the version this repo builds with.
	gomod := read(t, filepath.Join(repo, "go.mod"))
	gsxVersion := regexp.MustCompile(`github.com/gsxhq/gsx (v\S+)`).FindStringSubmatch(gomod)[1]
	goVersion := regexp.MustCompile(`(?m)^go (\S+)`).FindStringSubmatch(gomod)[1]
	// The Go this repo builds with, so mise (which reads go.mod's toolchain) wants a Go that's installed.
	toolchain := regexp.MustCompile(`(?m)^toolchain (\S+)`).FindStringSubmatch(gomod)[1]
	write("go.mod", "module example.com/fresh\n\ngo "+goVersion+"\n\ntoolchain "+toolchain+"\n")
	run("go", "mod", "edit", "-require="+intlModule+"@v0.0.0", "-replace="+intlModule+"="+filepath.Join(repo, "intl"),
		"-replace=github.com/joeblew999/go-htmx4/seo="+filepath.Join(repo, "seo"),
		"-replace=github.com/joeblew999/go-htmx4/badge="+filepath.Join(repo, "badge"),
		"-require=github.com/gsxhq/gsx@"+gsxVersion, "-tool=github.com/gsxhq/gsx/cmd/gsx", "-tool="+intlModule+"/cmd/intl")
	run("go", "mod", "tidy")

	// gsxui the npm-free way, with a component that brings its icon package (gsxui's own code, which intl:check must
	// leave alone), then go tool intl init, the app's own messages, a page and an ordinary server.
	gsxui := gsxuiCommand(t, repo)
	run(append(gsxui, "init", "-rtl")...)
	run(append(gsxui, "add", "accordion")...)
	run("go", "tool", "intl", "init", "-locales", "en,de")
	appendTo("locales/en.toml", "\n[home]\n# The page's heading.\ntitle = \"Welcome\"\n# How many notes there are.\nnotes = \"{n, plural, one {# note} other {# notes}}\"\n")
	appendTo("locales/de.toml", "\n[home]\ntitle = \"Willkommen\"\nnotes = \"{n, plural, one {# Notiz} other {# Notizen}}\"\n")
	write("page.gsx", `package main

import (
	"example.com/fresh/locales"
	"github.com/joeblew999/go-htmx4/intl"
)

component Page() {
	<!DOCTYPE html>
	<html { intl.HTML(ctx)... }>
		<body>
			<h1>{ locales.HomeTitle }</h1>
			<p>{ locales.HomeNotes(1234) }</p>
			<p>{ 1234567.5 |> number }</p>
		</body>
	</html>
}
`)
	write("main.go", `package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"example.com/fresh/locales"
	"github.com/joeblew999/go-htmx4/intl"
	"github.com/joeblew999/go-htmx4/intl/workerd"
)

func main() {
	stop, err := workerd.Start(locales.Messages)
	if err != nil {
		log.Fatal(err)
	}
	defer stop()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if err := Page().Render(r.Context(), w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("listening", ln.Addr())
	srv := &http.Server{Handler: intl.Site{}.Handler(mux)}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	go func() { <-ctx.Done(); srv.Shutdown(context.Background()) }()
	if err := srv.Serve(ln); err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
`)
	run("go", "tool", "intl", "gen")
	run("go", "tool", "gsx", "generate")
	run("go", "build", "-o", "fresh", ".")

	// Serve it, and read the page in both languages.
	srv := exec.Command(filepath.Join(app, "fresh"))
	srv.Dir = app
	srv.Stderr = os.Stderr
	stdout, err := srv.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := srv.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { srv.Process.Signal(syscall.SIGTERM); srv.Wait() }()
	addr := make(chan string, 1)
	go func() {
		s := bufio.NewScanner(stdout)
		for s.Scan() {
			if a, ok := strings.CutPrefix(s.Text(), "listening "); ok {
				addr <- a
			}
		}
	}()
	var base string
	select {
	case a := <-addr:
		base = "http://" + a
	case <-time.After(60 * time.Second):
		t.Fatal("the fresh app didn't start")
	}
	for path, wants := range map[string][]string{
		"/":    {`lang="en"`, "Welcome", "1,234 notes", "1,234,567.5"},
		"/de/": {`lang="de"`, "Willkommen", "1.234 Notizen", "1.234.567,5"},
	} {
		body := get(t, base+path)
		for _, want := range wants {
			if !strings.Contains(body, want) {
				t.Errorf("GET %s: no %q in\n%s", path, want, body)
			}
		}
	}

	// What the README says CI runs: the catalogs, then gsx's own checks on the app's markup, not gsxui's.
	run("mise", "run", "intl:check")
}

// gsxuiCommand runs the gsxui command: mise's when it's installed, else go run at the commit mise.toml pins (CI
// installs only the tools it builds with, and this compiles beside its build).
func gsxuiCommand(t *testing.T, repo string) []string {
	if p, err := exec.LookPath("gsxui"); err == nil {
		return []string{p}
	}
	pin := regexp.MustCompile(`"go:(github.com/gsxhq/gsxui/cmd/gsxui)" = "([^"]+)"`).FindStringSubmatch(read(t, filepath.Join(repo, "mise.toml")))
	if pin == nil {
		t.Fatal("mise.toml pins no gsxui")
	}
	return []string{"go", "run", pin[1] + "@" + pin[2]}
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func get(t *testing.T, url string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
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
