//go:build !js

package intltest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Browser runs cases through Intl in a real browser: it serves a page that evaluates them with intl.mjs (the same
// code as the workerd Oracle) and posts the results back. Chrome shares V8 and Chromium's ICU with Cloudflare
// Workers, but its version drifts from the pinned workerd; Firefox builds its own ICU. The drift tells whether
// browser code that re-formats server text (static/relative-time.js) would change it.
type Browser struct {
	Name    string   // "chrome" or "firefox"
	Command []string // binary and flags before the URL; default FindBrowser(Name)
	Log     io.Writer
}

// FindBrowser returns the command for a headless browser: $CHROME or $FIREFOX, else the usual install locations.
func FindBrowser(name string) ([]string, error) {
	var env string
	var paths []string
	switch name {
	case "chrome":
		env = "CHROME"
		paths = []string{"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome", "google-chrome", "google-chrome-stable", "chromium", "chromium-browser"}
	case "firefox":
		env = "FIREFOX"
		paths = []string{"/Applications/Firefox.app/Contents/MacOS/firefox", "firefox"}
	default:
		return nil, fmt.Errorf("intltest: unknown browser %q (chrome or firefox)", name)
	}
	if p := os.Getenv(env); p != "" {
		paths = []string{p}
	}
	for _, p := range paths {
		if full, err := exec.LookPath(p); err == nil {
			return []string{full}, nil
		}
	}
	return nil, fmt.Errorf("intltest: %s not found (set $%s)", name, env)
}

const browserPage = `<!doctype html><meta charset="utf-8"><title>intl cases</title><script type="module">
import { evaluate } from "/intl.mjs";
let body;
try {
  body = JSON.stringify(evaluate(await (await fetch("/cases.json")).json()));
} catch (e) {
  body = JSON.stringify({ error: String(e) });
}
await fetch("/result", { method: "POST", body });
document.title = "done";
</script>`

// Run starts the browser headless on a throwaway profile, waits for the results and stops it.
func (b Browser) Run(ctx context.Context, cases []Case) (*Golden, error) {
	cmdline := b.Command
	if len(cmdline) == 0 {
		var err error
		if cmdline, err = FindBrowser(b.Name); err != nil {
			return nil, err
		}
	}
	logw := b.Log
	if logw == nil {
		logw = io.Discard
	}
	casesJSON, err := json.Marshal(cases)
	if err != nil {
		return nil, err
	}
	profile, err := os.MkdirTemp("", "intl-browser-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(profile)

	type result struct {
		body []byte
		err  error
	}
	got := make(chan result, 1)
	mux := http.NewServeMux()
	serve := func(path, contentType string, body []byte) {
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", contentType)
			w.Write(body)
		})
	}
	serve("/", "text/html; charset=utf-8", []byte(browserPage))
	serve("/intl.mjs", "text/javascript; charset=utf-8", intlJS)
	serve("/cases.json", "application/json", casesJSON)
	mux.HandleFunc("/result", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		w.WriteHeader(http.StatusNoContent)
		select {
		case got <- result{body, err}:
		default:
		}
	})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)
	defer srv.Close()
	url := "http://" + ln.Addr().String() + "/"

	args := slices.Clone(cmdline[1:])
	switch b.Name {
	case "firefox":
		args = append(args, "--headless", "--no-remote", "--profile", profile)
	default:
		args = append(args, "--headless", "--disable-gpu", "--no-first-run", "--no-default-browser-check", "--disable-extensions", "--user-data-dir="+profile)
	}
	cmd := exec.CommandContext(ctx, cmdline[0], append(args, url)...)
	cmd.Stdout, cmd.Stderr = logw, logw
	// Intl defaults follow the host; pin them like the workerd oracle does.
	cmd.Env = append(os.Environ(), "TZ=UTC", "LANG=en_US.UTF-8", "LC_ALL=en_US.UTF-8")
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	defer stop(cmd)

	var res result
	select {
	case res = <-got:
	case <-time.After(3 * time.Minute):
		return nil, fmt.Errorf("intltest: %s posted no results within 3 minutes", b.Name)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if res.err != nil {
		return nil, res.err
	}
	var out struct {
		Error   string            `json:"error"`
		Meta    map[string]string `json:"meta"`
		Results map[string]string `json:"results"`
	}
	if err := json.Unmarshal(res.body, &out); err != nil {
		return nil, err
	}
	if out.Error != "" {
		return nil, errors.New("intltest: " + b.Name + ": " + out.Error)
	}
	if len(out.Results) != len(cases) {
		return nil, fmt.Errorf("intltest: %s returned %d results for %d cases", b.Name, len(out.Results), len(cases))
	}
	return &Golden{Runtime: browserVersion(out.Meta["userAgent"]), Meta: out.Meta, Results: out.Results}, nil
}

// browserVersion names the browser from its user agent, e.g. "Chrome 152.0.7600.0", "Firefox 155.0".
func browserVersion(ua string) string {
	for _, product := range []string{"Firefox/", "Chrome/", "Version/"} {
		if i := strings.Index(ua, product); i >= 0 {
			v := ua[i+len(product):]
			if j := strings.IndexByte(v, ' '); j >= 0 {
				v = v[:j]
			}
			name := strings.TrimSuffix(product, "/")
			if name == "Version" {
				name = "Safari"
			}
			return name + " " + v
		}
	}
	return ua
}

// Drift is how a runtime's Intl output differs from a golden file.
type Drift struct {
	Want, Got string               // runtimes, e.g. "workerd 2026-09-11", "Chrome 152.0.7600.0"
	Areas     []AreaDrift          // per case area, sorted by name
	Diffs     map[string][2]string // case ID → {golden, got}
	Missing   []string             // golden cases the run lacks
}

// AreaDrift counts cases and differences in one area ("datetime", "relative", …).
type AreaDrift struct {
	Area         string
	Cases, Diffs int
}

// Compare returns got's differences from want, grouped by area (a case ID's second segment: "ja/datetime/…").
func Compare(want, got *Golden) Drift {
	d := Drift{Want: want.Runtime, Got: got.Runtime, Diffs: map[string][2]string{}}
	areas := map[string]*AreaDrift{}
	for id, w := range want.Results {
		a := areaOf(id)
		if areas[a] == nil {
			areas[a] = &AreaDrift{Area: a}
		}
		areas[a].Cases++
		g, ok := got.Results[id]
		switch {
		case !ok:
			d.Missing = append(d.Missing, id)
		case g != w:
			areas[a].Diffs++
			d.Diffs[id] = [2]string{w, g}
		}
	}
	for _, a := range areas {
		d.Areas = append(d.Areas, *a)
	}
	slices.SortFunc(d.Areas, func(x, y AreaDrift) int { return strings.Compare(x.Area, y.Area) })
	slices.Sort(d.Missing)
	return d
}

func areaOf(id string) string {
	parts := strings.SplitN(id, "/", 3)
	if len(parts) == 3 {
		return parts[1]
	}
	return parts[0]
}

// Summary is a plain-text table of the drift, with up to perArea example differences per area.
func (d Drift) Summary(perArea int) string {
	var b strings.Builder
	total, diffs := 0, 0
	for _, a := range d.Areas {
		total += a.Cases
		diffs += a.Diffs
	}
	fmt.Fprintf(&b, "%s vs %s: %d of %d cases differ\n", d.Got, d.Want, diffs, total)
	ids := make([]string, 0, len(d.Diffs))
	for id := range d.Diffs {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, a := range d.Areas {
		fmt.Fprintf(&b, "  %-12s %6d cases %6d differ\n", a.Area, a.Cases, a.Diffs)
		shown := 0
		for _, id := range ids {
			if shown == perArea || areaOf(id) != a.Area {
				continue
			}
			shown++
			fmt.Fprintf(&b, "      %s\n        golden %s\n        got    %s\n", id, strconv.Quote(d.Diffs[id][0]), strconv.Quote(d.Diffs[id][1]))
		}
	}
	if len(d.Missing) > 0 {
		fmt.Fprintf(&b, "  missing: %d cases (%s …)\n", len(d.Missing), d.Missing[0])
	}
	return b.String()
}
