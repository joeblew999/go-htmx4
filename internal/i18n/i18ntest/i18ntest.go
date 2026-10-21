// Package i18ntest gives go test the Worker's Intl: worker/gointl.mjs (worker/intl.mjs with the message catalogs) on
// workerd, the runtime the Worker runs on, as i18n.Runtime. There is no other Intl: a package whose tests render or
// format anything runs them under it from TestMain:
//
//	func TestMain(m *testing.M) { os.Exit(i18ntest.Main(m)) }
//
// workerd comes from mise (mise run test:go); the catalogs from mise run i18n:messages.
package i18ntest

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
	"regexp"
	"testing"
	"time"

	"github.com/joeblew999/go-htmx4/internal/i18n"
)

// Main starts workerd, runs the tests with it as i18n.Runtime, and stops it.
func Main(m *testing.M) int {
	stop, err := Start()
	if err != nil {
		fmt.Fprintln(os.Stderr, "i18ntest:", err)
		return 1
	}
	defer stop()
	return m.Run()
}

var base string

// Start starts workerd serving worker/gointl.mjs and sets i18n.Runtime to it.
func Start() (stop func(), err error) {
	workerd, err := exec.LookPath("workerd")
	if err != nil {
		return nil, errors.New("workerd is not on PATH: run the tests through mise (mise run test:go)")
	}
	root, err := repoRoot()
	if err != nil {
		return nil, err
	}
	config, err := os.ReadFile(filepath.Join(root, "workerd", "config.capnp"))
	if err != nil {
		return nil, err
	}
	date := regexp.MustCompile(`compatibilityDate = "([0-9-]+)"`).FindSubmatch(config)
	if date == nil {
		return nil, errors.New("workerd/config.capnp has no compatibilityDate")
	}
	dir, err := os.MkdirTemp("", "i18ntest-")
	if err != nil {
		return nil, err
	}
	for _, f := range []string{"intl.mjs", "gointl.mjs", "messages_gen.mjs"} {
		b, err := os.ReadFile(filepath.Join(root, "worker", f))
		if err != nil {
			return nil, fmt.Errorf("%w (messages_gen.mjs: mise run i18n:messages)", err)
		}
		if err := os.WriteFile(filepath.Join(dir, f), b, 0o644); err != nil {
			return nil, err
		}
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	addr := l.Addr().String()
	l.Close()
	files := map[string]string{
		"main.mjs": `import { goIntl } from "./gointl.mjs";
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
      (name = "gointl.mjs", esModule = embed "gointl.mjs"),
      (name = "intl.mjs", esModule = embed "intl.mjs"),
      (name = "messages_gen.mjs", esModule = embed "messages_gen.mjs"),
    ],
    compatibilityDate = "` + string(date[1]) + `"))],
  sockets = [(name = "http", address = "` + addr + `", http = (), service = "main")],
);
`,
	}
	for name, s := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(s), 0o644); err != nil {
			return nil, err
		}
	}
	cmd := exec.Command(workerd, "serve", filepath.Join(dir, "config.capnp"))
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	stop = func() {
		cmd.Process.Kill()
		cmd.Wait()
		os.RemoveAll(dir)
	}
	base = "http://" + addr
	for deadline := time.Now().Add(20 * time.Second); ; time.Sleep(50 * time.Millisecond) {
		if resp, err := client.Get(base); err == nil {
			resp.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			stop()
			return nil, fmt.Errorf("workerd didn't answer on %s", addr)
		}
	}
	i18n.Runtime = func(method string, args ...string) string {
		s, err := post(map[string]any{"method": method, "args": args})
		if err != nil {
			panic("i18ntest: " + method + ": " + err.Error())
		}
		return s
	}
	return stop, nil
}

// Tree formats a message tree as msggen writes it (worker/intl.mjs message) in lang with args.
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

// repoRoot is the nearest directory up from here with worker/gointl.mjs.
func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "worker", "gointl.mjs")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("no worker/gointl.mjs above the working directory")
		}
		dir = parent
	}
}
