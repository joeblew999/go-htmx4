// Package workerd is intl's Intl outside a Cloudflare Worker: workerd, the Workers runtime, serving intl's
// js/intl-worker.mjs with the app's catalogs, set as intl.Runtime. On a Worker, Intl is the Worker's own; anywhere
// else (an ordinary Go server, go test, go tool intl) it is this one, so every place formats with the same engine.
//
// An app on an ordinary Go server starts it before serving, with the catalogs go tool intl gen embedded in its
// messages package:
//
//	stop, err := workerd.Start(locales.Messages)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer stop()
//
// workerd must be on PATH: mise installs it (go tool intl init pins Version), or put its release binary in the
// server's image. Each Intl call is a request to it on 127.0.0.1, about 0.1 ms. It ends with the process that started
// it, however that process ends (launch).
package workerd

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
	"strings"
	"time"

	"github.com/joeblew999/go-htmx4/intl"
	intljs "github.com/joeblew999/go-htmx4/intl/js"
)

// CompatibilityDate is the Workers compatibility date the Intl Worker runs on.
const CompatibilityDate = "2026-09-11"

// Version is the workerd release intl is tested with: go tool intl init pins it in the app's mise.toml, and intl's
// mise tasks pin it for themselves.
const Version = "1.20261019.1"

// noWorkerd says how to get workerd.
const noWorkerd = "workerd, the Workers runtime whose Intl intl uses outside a Worker, is not installed: add it to mise.toml's [tools] (workerd = \"" + Version + "\"; go tool intl init does) and run through mise, or put its release binary on the server's PATH"

var (
	base   string
	client = &http.Client{Transport: &http.Transport{MaxIdleConnsPerHost: 32}, Timeout: 10 * time.Second}
)

// Start starts workerd serving intl's JS with catalogs (an app's intl-messages.mjs: its messages package's
// Messages; "export default {};" for none) and sets intl.Runtime to it. stop ends it.
func Start(catalogs []byte) (stop func(), err error) {
	workerd, err := exec.LookPath("workerd")
	if err != nil {
		return nil, errors.New(noWorkerd)
	}
	dir, err := os.MkdirTemp("", "intl-workerd-")
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
	c, err := launch(workerd, filepath.Join(dir, "config.capnp"), io.MultiWriter(os.Stderr, &errs))
	if err != nil {
		return nil, err
	}
	// Who owns this workerd: launch ends it when this process ends, and a leftover (a system with no shell and no
	// parent-death signal) goes at the next Start's reapOrphans.
	os.WriteFile(filepath.Join(dir, "owner"), []byte(fmt.Sprintf("%d %d", os.Getpid(), c.pid())), 0o644)
	exited := c.exited
	stop = func() {
		c.stop()
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
			fmt.Fprintln(os.Stderr, "intl/workerd:", method, err)
			return ""
		}
		return s
	}
	return stop, nil
}

// Tree formats a message tree as go tool intl gen writes it (js/intl.mjs message) in lang with args: for tests of
// the catalogs' compiler.
func Tree(lang string, tree json.RawMessage, args ...string) (string, error) {
	return post(map[string]any{"method": "tree", "args": append([]string{lang}, args...), "tree": tree})
}

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

// reapOrphans stops the workerds of earlier processes that died without stopping them.
func reapOrphans() {
	dirs, _ := filepath.Glob(filepath.Join(os.TempDir(), "intl-workerd-*"))
	for _, d := range dirs {
		b, err := os.ReadFile(filepath.Join(d, "owner"))
		if err != nil {
			continue
		}
		var owner, worker int
		if _, err := fmt.Sscan(string(b), &owner, &worker); err != nil || alive(owner) {
			continue
		}
		if alive(worker) {
			killGroup(worker)
		}
		os.RemoveAll(d)
	}
}
