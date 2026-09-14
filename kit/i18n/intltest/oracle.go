//go:build !js

package intltest

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

//go:embed oracle.mjs
var oracleJS []byte

//go:embed intl.mjs
var intlJS []byte

// Golden is what the oracle recorded for a set of cases.
type Golden struct {
	Runtime string            `json:"runtime"` // workerd --version, e.g. "workerd 2026-09-11" (release 1.20260911.x)
	Meta    map[string]string `json:"meta"`
	Results map[string]string `json:"results"` // case ID → output
}

// Oracle runs cases through Intl in a throwaway workerd.
type Oracle struct {
	Workerd string // workerd binary (default "workerd" on PATH)
	Port    int    // loopback port (default 8946)
	Log     io.Writer
}

const oracleConfig = `using Workerd = import "/workerd/workerd.capnp";
const config :Workerd.Config = (
  services = [ (name = "oracle", worker = .oracle) ],
  sockets = [ (name = "http", address = "127.0.0.1:%d", http = (), service = "oracle") ],
);
const oracle :Workerd.Worker = (
  modules = [
    (name = "oracle.mjs", esModule = embed "oracle.mjs"),
    (name = "intl.mjs", esModule = embed "intl.mjs"),
    (name = "cases.json", json = embed "cases.json"),
  ],
  compatibilityDate = "2026-09-11",
);
`

// Run evaluates cases and returns the recorded outputs.
func (o Oracle) Run(ctx context.Context, cases []Case) (*Golden, error) {
	bin := o.Workerd
	if bin == "" {
		bin = "workerd"
	}
	port := o.Port
	if port == 0 {
		port = 8946
	}
	logw := o.Log
	if logw == nil {
		logw = io.Discard
	}
	dir, err := os.MkdirTemp("", "intl-oracle-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	casesJSON, err := json.Marshal(cases)
	if err != nil {
		return nil, err
	}
	files := map[string][]byte{
		"oracle.mjs":   oracleJS,
		"intl.mjs":     intlJS,
		"cases.json":   casesJSON,
		"config.capnp": []byte(fmt.Sprintf(oracleConfig, port)),
	}
	for name, b := range files {
		if err := os.WriteFile(filepath.Join(dir, name), b, 0o644); err != nil {
			return nil, err
		}
	}
	version, _ := exec.CommandContext(ctx, bin, "--version").Output()

	base := "http://127.0.0.1:" + strconv.Itoa(port) + "/"
	if res, err := http.Get(base); err == nil {
		res.Body.Close()
		return nil, fmt.Errorf("port %d is already in use", port)
	}
	cmd := exec.CommandContext(ctx, bin, "serve", filepath.Join(dir, "config.capnp"))
	var stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = logw, &stderr
	// Intl defaults follow the host; pin them so no output depends on this machine.
	cmd.Env = append(os.Environ(), "TZ=UTC", "LANG=en_US.UTF-8", "LC_ALL=en_US.UTF-8")
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	defer stop(cmd)

	client := &http.Client{Timeout: 120 * time.Second}
	var body []byte
	deadline := time.Now().Add(30 * time.Second)
	for {
		res, err := client.Get(base)
		if err == nil {
			body, err = io.ReadAll(res.Body)
			res.Body.Close()
			if err != nil {
				return nil, err
			}
			if res.StatusCode != http.StatusOK {
				return nil, fmt.Errorf("oracle: HTTP %d: %s", res.StatusCode, body)
			}
			break
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("workerd did not start: %v\n%s", err, stderr.String())
		}
		time.Sleep(200 * time.Millisecond)
	}
	var out struct {
		Meta    map[string]string `json:"meta"`
		Results map[string]string `json:"results"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	if len(out.Results) != len(cases) {
		return nil, fmt.Errorf("oracle returned %d results for %d cases", len(out.Results), len(cases))
	}
	return &Golden{Runtime: strings.TrimSpace(string(version)), Meta: out.Meta, Results: out.Results}, nil
}

// stop ends workerd: SIGTERM, then SIGKILL after 3 s (workerd can ignore SIGTERM).
func stop(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	cmd.Process.Signal(syscall.SIGTERM)
	done := make(chan struct{})
	go func() { cmd.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		cmd.Process.Kill()
		<-done
	}
}

// Load reads a golden file.
func Load(path string) (*Golden, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var g Golden
	if err := json.Unmarshal(b, &g); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if g.Results == nil {
		return nil, errors.New(path + ": no results")
	}
	return &g, nil
}

// Save writes a golden file (JSON, keys sorted, one result per line).
func (g *Golden) Save(path string) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", " ")
	if err := enc.Encode(g); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}
