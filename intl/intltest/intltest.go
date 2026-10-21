// Package intltest gives go test the Intl a Worker has: intl's js/intl-worker.mjs, with the app's catalogs, on
// workerd, set as intl.Runtime. There is no other Intl: a package whose tests render or format anything runs them
// under it from TestMain:
//
//	func TestMain(m *testing.M) { os.Exit(intltest.Main(m)) }
//
// workerd must be on PATH (mise installs it); the catalogs are the ones go tool intl gen wrote, found through the
// app's intl.toml (none without one).
package intltest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/joeblew999/go-htmx4/intl"
	"github.com/joeblew999/go-htmx4/intl/catalog"
	"github.com/joeblew999/go-htmx4/intl/config"
	intljs "github.com/joeblew999/go-htmx4/intl/js"
)

// CompatibilityDate is the Workers compatibility date the test Worker runs on.
const CompatibilityDate = "2026-09-11"

// Workerd is the workerd release intl is tested with: go tool intl init pins it in the app's mise.toml, and intl's
// mise tasks pin it for themselves.
const Workerd = "1.20261019.1"

// noWorkerd says how to get workerd.
const noWorkerd = "workerd, the Workers runtime whose Intl the tests use, is not installed: add it to mise.toml's [tools] (workerd = \"" + Workerd + "\"; go tool intl init does) and run the tests through mise"

// Main starts workerd, runs the tests with it as intl.Runtime, and stops it.
func Main(m *testing.M) int {
	stop, err := Start()
	if err != nil {
		fmt.Fprintln(os.Stderr, "intltest:", err)
		return 1
	}
	defer stop()
	return m.Run()
}

var base string

// Start starts workerd serving js/intl-worker.mjs with the app's catalogs and sets intl.Runtime to it.
func Start() (stop func(), err error) {
	catalogs := []byte("export default {};\n")
	if c, err := config.Find("."); err == nil {
		if catalogs, err = readCatalogs(c); err != nil {
			return nil, err
		}
	}
	return start(catalogs)
}

// StartIntl starts workerd's Intl without the app's catalogs (go tool intl, before they are compiled).
func StartIntl() (stop func(), err error) { return start([]byte("export default {};\n")) }

func start(catalogs []byte) (stop func(), err error) {
	workerd, err := exec.LookPath("workerd")
	if err != nil {
		return nil, errors.New(noWorkerd)
	}
	dir, err := os.MkdirTemp("", "intltest-")
	if err != nil {
		return nil, err
	}
	if err := os.CopyFS(dir, intljs.FS); err != nil {
		return nil, err
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	addr := l.Addr().String()
	l.Close()
	files := map[string]string{
		"intl-messages.mjs": string(catalogs),
		"main.mjs": `import { goIntl } from "./intl-worker.mjs";
import * as intl from "./intl.mjs";
export default {
  async fetch(req) {
    if (req.method !== "POST") return new Response("ok");
    const { method, args, tree } = await req.json();
    if (method === "tree") return new Response(intl.message(args[0], tree, args.slice(1)));
    if (typeof goIntl[method] !== "function") return new Response("no method " + method, { status: 400 });
    return new Response(String(goIntl[method](...args)));
  },
};
`,
		"config.capnp": `using Workerd = import "/workerd/workerd.capnp";
const config :Workerd.Config = (
  services = [(name = "main", worker = (
    modules = [
      (name = "main.mjs", esModule = embed "main.mjs"),
      (name = "intl-worker.mjs", esModule = embed "intl-worker.mjs"),
      (name = "intl.mjs", esModule = embed "intl.mjs"),
      (name = "intl-messages.mjs", esModule = embed "intl-messages.mjs"),
    ],
    compatibilityDate = "` + CompatibilityDate + `"))],
  sockets = [(name = "http", address = "` + addr + `", http = (), service = "main")],
);
`,
	}
	for name, s := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(s), 0o644); err != nil {
			return nil, err
		}
	}
	var errs bytes.Buffer
	reapOrphans()
	cmd := exec.Command(workerd, "serve", filepath.Join(dir, "config.capnp"))
	cmd.Stderr = io.MultiWriter(os.Stderr, &errs)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	// Who owns this workerd: a test process that dies without stopping it (a panic) leaves it to the next run's
	// reapOrphans.
	os.WriteFile(filepath.Join(dir, "owner"), []byte(fmt.Sprintf("%d %d", os.Getpid(), cmd.Process.Pid)), 0o644)
	exited := make(chan struct{})
	go func() { cmd.Wait(); close(exited) }()
	stop = func() {
		cmd.Process.Kill()
		<-exited
		os.RemoveAll(dir)
	}
	base = "http://" + addr
	for deadline := time.Now().Add(20 * time.Second); ; time.Sleep(50 * time.Millisecond) {
		if resp, err := client.Get(base); err == nil {
			resp.Body.Close()
			break
		}
		select {
		case <-exited: // it didn't start: a mise shim without a version, a bad config
			os.RemoveAll(dir)
			if strings.Contains(errs.String(), "No version is set") {
				return nil, errors.New(noWorkerd)
			}
			return nil, fmt.Errorf("workerd exited: %s", strings.TrimSpace(errs.String()))
		default:
		}
		if time.Now().After(deadline) {
			stop()
			return nil, fmt.Errorf("workerd didn't answer on %s", addr)
		}
	}
	// As on a Worker, an Intl that can't answer gives "" (the value's machine form shows); the reason goes to stderr.
	intl.Runtime = func(method string, args ...string) string {
		s, err := post(map[string]any{"method": method, "args": args})
		if err != nil {
			fmt.Fprintln(os.Stderr, "intltest:", method, err)
			return ""
		}
		return s
	}
	return stop, nil
}

// Plurals checks that every plural and selectordinal in the app's catalogs has a case for each category its
// language's Intl.PluralRules uses (Russian needs one, few, many and other):
//
//	func TestPlurals(t *testing.T) { intltest.Plurals(t) }
func Plurals(t testing.TB) {
	t.Helper()
	c, err := config.Find(".")
	if err != nil {
		t.Fatal(err)
	}
	p, err := catalog.Read(c)
	if err != nil {
		t.Fatal(err)
	}
	problems, err := PluralProblems(p)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range problems {
		t.Error(s)
	}
}

// Categories are the plural categories Intl.PluralRules uses for locale id: cardinal ("one", "few", "many",
// "other" for Russian) and ordinal.
func Categories(id string) (cardinal, ordinal []string, err error) {
	get := func(typ string) ([]string, error) {
		s, ok := intl.NewLocale(id).Intl(intl.Call{API: "PluralRules", Method: "resolvedOptions", Opts: `{"type":"` + typ + `"}`}, "", "")
		var o struct{ PluralCategories []string }
		if !ok || json.Unmarshal([]byte(s), &o) != nil || len(o.PluralCategories) == 0 {
			return nil, fmt.Errorf("intl: Intl has no %s plural categories for %s", typ, id)
		}
		return o.PluralCategories, nil
	}
	if cardinal, err = get("cardinal"); err != nil {
		return nil, nil, err
	}
	ordinal, err = get("ordinal")
	return cardinal, ordinal, err
}

// PluralProblems are the plurals and selectordinals in p's own translations (and the source) missing a case for a
// category their language uses. Intl must be running (Start, StartIntl).
func PluralProblems(p *catalog.Project) ([]string, error) {
	var out []string
	for _, l := range p.Locales {
		cardinal, ordinal, err := Categories(l.ID)
		if err != nil {
			return nil, err
		}
		for i, e := range l.Entries {
			if e.From != l.ID {
				continue // missing, or an ancestor's: checked there
			}
			for _, pl := range e.Plurals() {
				need, typ := cardinal, "plural"
				if pl.Ordinal {
					need, typ = ordinal, "selectordinal"
				}
				for _, cat := range need {
					if !slices.Contains(pl.Cases, cat) {
						out = append(out, fmt.Sprintf("%s %s: a %s without a %q case (%s needs %s)",
							l.File, p.Messages[i].Key, typ, cat, l.ID, strings.Join(need, ", ")))
					}
				}
			}
		}
	}
	return out, nil
}

func readCatalogs(c config.Config) ([]byte, error) {
	b, err := os.ReadFile(filepath.Join(c.CatalogDir(), "intl-messages.mjs"))
	if err != nil {
		return nil, fmt.Errorf("%w: run go tool intl gen", err)
	}
	return b, nil
}

// reapOrphans stops the workerds of earlier test processes that died without stopping them.
func reapOrphans() {
	dirs, _ := filepath.Glob(filepath.Join(os.TempDir(), "intltest-*"))
	for _, d := range dirs {
		b, err := os.ReadFile(filepath.Join(d, "owner"))
		if err != nil {
			continue
		}
		var owner, worker int
		if _, err := fmt.Sscan(string(b), &owner, &worker); err != nil || alive(owner) {
			continue
		}
		if p, err := os.FindProcess(worker); err == nil && alive(worker) {
			p.Kill()
		}
		os.RemoveAll(d)
	}
}

// alive reports whether process pid is running.
func alive(pid int) bool {
	p, err := os.FindProcess(pid)
	return err == nil && p.Signal(syscall.Signal(0)) == nil
}

// Tree formats a message tree as go tool intl gen writes it (js/intl.mjs message) in lang with args.
func Tree(lang string, tree json.RawMessage, args ...string) (string, error) {
	return post(map[string]any{"method": "tree", "args": append([]string{lang}, args...), "tree": tree})
}

var client = &http.Client{Transport: &http.Transport{MaxIdleConnsPerHost: 32}, Timeout: 10 * time.Second}

func post(body map[string]any) (string, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	resp, err := client.Post(base, "application/json", bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("workerd: %d %s", resp.StatusCode, out)
	}
	return string(out), nil
}
