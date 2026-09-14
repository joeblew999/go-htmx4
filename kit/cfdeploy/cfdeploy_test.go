package cfdeploy_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/joeblew999/go-htmx4/kit/cfdeploy"
)

// fakeCF implements the slice of the Cloudflare API a deploy uses, for one account ("acc", token "tok").
type fakeCF struct {
	t  *testing.T
	mu sync.Mutex

	scripts    map[string]string // name → applied Durable Object migration tag
	dbs        map[string]string // name → uuid
	migrations map[string]bool   // _migrations rows (all databases)
	sql        []string          // migration SQL applied, in order
	uploaded   map[string]string // hash → decoded asset content
	putMeta    map[string]any    // last script metadata
	putParts   []part            // last script module parts
	subdomain  bool
	domains    map[string]string // custom domain hostname → Worker
	domainPuts int
}

type part struct{ name, filename, contentType string }

func newFakeCF(t *testing.T) (*fakeCF, *httptest.Server) {
	f := &fakeCF{t: t, scripts: map[string]string{}, dbs: map[string]string{}, migrations: map[string]bool{}, uploaded: map[string]string{},
		domains: map[string]string{"auth.example.com": "auth-worker"}}
	srv := httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(srv.Close)
	return f, srv
}

func (f *fakeCF) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	reply := func(result any) {
		json.NewEncoder(w).Encode(map[string]any{"success": true, "errors": []any{}, "result": result})
	}
	fail := func(status int, msg string) {
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(map[string]any{"success": false, "errors": []any{map[string]any{"code": status, "message": msg}}})
	}
	auth := r.Header.Get("Authorization")
	if r.URL.Path == "/zones" {
		if auth != "Bearer tok" {
			fail(401, "bad token")
			return
		}
		var out []map[string]any
		if r.URL.Query().Get("name") == "example.com" {
			out = append(out, map[string]any{"id": "zone-example", "account": map[string]string{"id": "acc"}})
		}
		reply(out)
		return
	}
	p := strings.TrimPrefix(r.URL.Path, "/accounts/acc")
	if p == r.URL.Path {
		fail(403, "wrong account")
		return
	}
	if p == "/workers/assets/upload" {
		if auth != "Bearer session-jwt" {
			fail(401, "asset upload needs the session JWT, got "+auth)
			return
		}
	} else if auth != "Bearer tok" {
		fail(401, "bad token")
		return
	}
	switch {
	case r.Method == "GET" && p == "/workers/scripts":
		var out []map[string]string
		for name, tag := range f.scripts {
			out = append(out, map[string]string{"id": name, "migration_tag": tag})
		}
		reply(out)
	case r.Method == "GET" && p == "/d1/database":
		var out []map[string]string
		if id, ok := f.dbs[r.URL.Query().Get("name")]; ok {
			out = append(out, map[string]string{"uuid": id, "name": r.URL.Query().Get("name")})
		}
		reply(out)
	case r.Method == "POST" && p == "/d1/database":
		var body struct{ Name string }
		json.NewDecoder(r.Body).Decode(&body)
		f.dbs[body.Name] = fmt.Sprintf("00000000-0000-0000-0000-%012d", len(f.dbs)+1)
		reply(map[string]string{"uuid": f.dbs[body.Name]})
	case r.Method == "POST" && strings.HasPrefix(p, "/d1/database/") && strings.HasSuffix(p, "/query"):
		var body struct {
			SQL    string `json:"sql"`
			Params []any  `json:"params"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		rows := []map[string]any{}
		switch {
		case strings.HasPrefix(body.SQL, "CREATE TABLE IF NOT EXISTS _migrations"):
		case body.SQL == "SELECT name FROM _migrations":
			for name := range f.migrations {
				rows = append(rows, map[string]any{"name": name})
			}
		case strings.HasPrefix(body.SQL, "INSERT INTO _migrations"):
			f.migrations[fmt.Sprint(body.Params[0])] = true
		default:
			f.sql = append(f.sql, body.SQL)
		}
		reply([]map[string]any{{"results": rows}})
	case r.Method == "POST" && strings.HasSuffix(p, "/assets-upload-session"):
		var body struct {
			Manifest map[string]cfdeploy.ManifestEntry `json:"manifest"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		var missing []string
		for _, e := range body.Manifest {
			if _, ok := f.uploaded[e.Hash]; !ok {
				missing = append(missing, e.Hash)
			}
		}
		slices.Sort(missing)
		var buckets [][]string
		for i := 0; i < len(missing); i += 2 { // two files per bucket: exercise several uploads
			buckets = append(buckets, missing[i:min(i+2, len(missing))])
		}
		reply(map[string]any{"jwt": "session-jwt", "buckets": buckets})
	case r.Method == "POST" && p == "/workers/assets/upload":
		if r.URL.Query().Get("base64") != "true" {
			fail(400, "want base64=true")
			return
		}
		mr := f.multipart(r)
		for {
			pt, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			b, _ := io.ReadAll(pt)
			raw, err := base64.StdEncoding.DecodeString(string(b))
			if err != nil {
				fail(400, "part is not base64")
				return
			}
			f.uploaded[pt.FormName()] = string(raw)
		}
		reply(map[string]string{"jwt": "completion-jwt"})
	case r.Method == "PUT" && strings.HasPrefix(p, "/workers/scripts/"):
		name := strings.TrimPrefix(p, "/workers/scripts/")
		mr := f.multipart(r)
		f.putMeta, f.putParts = nil, nil
		for {
			pt, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			b, _ := io.ReadAll(pt)
			if pt.FormName() == "metadata" {
				json.Unmarshal(b, &f.putMeta)
				continue
			}
			// Part.FileName strips directories; the module name is the raw filename parameter.
			_, disp, _ := mime.ParseMediaType(pt.Header.Get("Content-Disposition"))
			f.putParts = append(f.putParts, part{pt.FormName(), disp["filename"], pt.Header.Get("Content-Type")})
		}
		tag := f.scripts[name]
		if m, ok := f.putMeta["migrations"].(map[string]any); ok {
			tag = m["new_tag"].(string)
		}
		f.scripts[name] = tag
		reply(nil)
	case r.Method == "GET" && p == "/workers/domains":
		var out []map[string]string
		for host, svc := range f.domains {
			out = append(out, map[string]string{"hostname": host, "service": svc})
		}
		reply(out)
	case r.Method == "PUT" && p == "/workers/domains":
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["zone_id"] != "zone-example" || body["environment"] != "production" || body["service"] == "" {
			fail(400, fmt.Sprintf("bad custom domain body %v", body))
			return
		}
		f.domains[body["hostname"]] = body["service"]
		f.domainPuts++
		reply(body)
	case r.Method == "POST" && strings.HasSuffix(p, "/subdomain"):
		f.subdomain = true
		reply(nil)
	case r.Method == "GET" && p == "/workers/subdomain":
		reply(map[string]string{"subdomain": "acme"})
	default:
		fail(404, r.Method+" "+p)
	}
}

func (f *fakeCF) multipart(r *http.Request) *multipart.Reader {
	_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		f.t.Fatalf("%s %s: content type: %v", r.Method, r.URL.Path, err)
	}
	return multipart.NewReader(r.Body, params["boundary"])
}

// project writes a Worker project the way the go-htmx4 app lays it out.
func project(t *testing.T) string {
	dir := t.TempDir()
	write := func(name, content string) {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("worker/index.mjs", `import go from "./build/worker.mjs"; export { Room } from "./room.mjs";`)
	write("worker/room.mjs", `export class Room {}`)
	for _, f := range cfdeploy.BuildFiles {
		write("build/"+f, "build "+f)
	}
	write("site/static/htmx.min.js", "htmx")
	write("site/assets/app.css", "body{}")
	write("site/index.txt", "hello")
	write("site/.DS_Store", "junk")
	write("migrations/0001_board.sql", "CREATE TABLE board (topic TEXT PRIMARY KEY)")
	write("migrations/0002_note.sql", "CREATE TABLE note (id INTEGER PRIMARY KEY)")
	return dir
}

func config(dir string) cfdeploy.Config {
	return cfdeploy.Config{
		Name:              "app",
		BuildDir:          filepath.Join(dir, "build"),
		AssetsDir:         filepath.Join(dir, "site"),
		CompatibilityDate: "2026-09-11",
		MainModule:        filepath.Join(dir, "worker/index.mjs"),
		Modules:           []string{filepath.Join(dir, "worker/room.mjs")},
		Vars:              map[string]string{"APP_ENV": "cloudflare"},
		D1:                map[string]string{"DB": "app"},
		D1Create:          true,
		MigrationsDir:     filepath.Join(dir, "migrations"),
		DurableObjects:    map[string]string{"ROOM": "Room"},
		RateLimits:        map[string]cfdeploy.RateLimit{"WRITES": {NamespaceID: "1001", Limit: 60, Period: 10}},
		MigrationTag:      "v1",
		NewSQLiteClasses:  []string{"Room"},
	}
}

func TestDeploy(t *testing.T) {
	f, srv := newFakeCF(t)
	dir := project(t)
	var logs []string
	c := &cfdeploy.Client{Token: "tok", AccountID: "acc", BaseURL: srv.URL,
		Logf: func(format string, args ...any) { logs = append(logs, fmt.Sprintf(format, args...)) }}
	cfg := config(dir)

	url, err := c.Deploy(context.Background(), cfg)
	if err != nil {
		t.Fatalf("first deploy: %v\nlogs: %v", err, logs)
	}
	if url != "https://app.acme.workers.dev" || !f.subdomain {
		t.Errorf("url = %q, subdomain enabled %v", url, f.subdomain)
	}
	if id := f.dbs["app"]; id == "" {
		t.Errorf("D1 database not created: %v", f.dbs)
	}
	if len(f.sql) != 2 || !strings.Contains(f.sql[0], "board") || !strings.Contains(f.sql[1], "note") {
		t.Errorf("migrations applied = %q, want 0001 then 0002", f.sql)
	}
	for _, name := range []string{"/static/htmx.min.js", "/assets/app.css", "/index.txt"} {
		content := map[string]string{"/static/htmx.min.js": "htmx", "/assets/app.css": "body{}", "/index.txt": "hello"}[name]
		hash := cfdeploy.AssetHash([]byte(content), filepath.Ext(name))
		if f.uploaded[hash] != content {
			t.Errorf("asset %s not uploaded as %q", name, content)
		}
	}
	if len(f.uploaded) != 3 {
		t.Errorf("uploaded %d assets, want 3 (dotfiles skipped)", len(f.uploaded))
	}

	wantParts := []part{
		{"index.mjs", "index.mjs", "application/javascript+module"},
		{"room.mjs", "room.mjs", "application/javascript+module"},
		{"build/worker.mjs", "build/worker.mjs", "application/javascript+module"},
		{"build/wasm_exec.js", "build/wasm_exec.js", "application/javascript+module"},
		{"build/runtime.mjs", "build/runtime.mjs", "application/javascript+module"},
		{"build/app.wasm", "build/app.wasm", "application/wasm"},
	}
	if !slices.Equal(f.putParts, wantParts) {
		t.Errorf("script parts = %v\nwant %v", f.putParts, wantParts)
	}
	meta, _ := json.Marshal(f.putMeta)
	for _, want := range []string{
		`"main_module":"index.mjs"`, `"compatibility_date":"2026-09-11"`,
		`{"name":"APP_ENV","text":"cloudflare","type":"plain_text"}`,
		`{"id":"` + f.dbs["app"] + `","name":"DB","type":"d1"}`,
		`{"class_name":"Room","name":"ROOM","type":"durable_object_namespace"}`,
		`{"name":"WRITES","namespace_id":"1001","simple":{"limit":60,"period":10},"type":"ratelimit"}`,
		`"migrations":{"new_sqlite_classes":["Room"],"new_tag":"v1"}`,
		`"assets":{"jwt":"completion-jwt"}`,
	} {
		if !strings.Contains(string(meta), want) {
			t.Errorf("metadata missing %s\n%s", want, meta)
		}
	}

	// Redeploy: script exists, migration applied, database and assets unchanged.
	sqlBefore := len(f.sql)
	if _, err := c.Deploy(context.Background(), cfg); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("redeploy without AllowExisting: err = %v, want already exists", err)
	}
	cfg.AllowExisting = true
	if _, err := c.Deploy(context.Background(), cfg); err != nil {
		t.Fatalf("redeploy: %v", err)
	}
	if len(f.sql) != sqlBefore {
		t.Errorf("redeploy re-applied migrations: %q", f.sql[sqlBefore:])
	}
	if _, ok := f.putMeta["migrations"]; ok {
		t.Errorf("redeploy resent the Durable Object migration: %v", f.putMeta["migrations"])
	}
	if a, _ := f.putMeta["assets"].(map[string]any); a["jwt"] != "session-jwt" {
		t.Errorf("unchanged assets: jwt = %v, want the session JWT", a["jwt"])
	}

	// Custom domains: attached once, idempotent on redeploy, never taken from another Worker, zone looked up by parent.
	cfg.Domains = []string{"app.example.com"}
	if _, err := c.Deploy(context.Background(), cfg); err != nil {
		t.Fatalf("deploy with a custom domain: %v", err)
	}
	if _, err := c.Deploy(context.Background(), cfg); err != nil {
		t.Fatalf("redeploy with a custom domain: %v", err)
	}
	if f.domains["app.example.com"] != "app" || f.domainPuts != 1 {
		t.Errorf("custom domain: attached to %q with %d PUTs, want app with 1", f.domains["app.example.com"], f.domainPuts)
	}
	cfg.Domains = []string{"auth.example.com"}
	if _, err := c.Deploy(context.Background(), cfg); err == nil || !strings.Contains(err.Error(), `already attached to Worker "auth-worker"`) {
		t.Errorf("taken hostname: err = %v", err)
	}
	cfg.Domains = []string{"app.elsewhere.org"}
	if _, err := c.Deploy(context.Background(), cfg); err == nil || !strings.Contains(err.Error(), "no zone for app.elsewhere.org") {
		t.Errorf("zone not in account: err = %v", err)
	}
	cfg.Domains = nil

	cfg.MigrationTag = "v2"
	if _, err := c.Deploy(context.Background(), cfg); err == nil || !strings.Contains(err.Error(), "explicit migration step") {
		t.Errorf("migration v1 → v2: err = %v, want explicit migration step", err)
	}
}

func TestDeployErrors(t *testing.T) {
	_, srv := newFakeCF(t)
	dir := project(t)
	ctx := context.Background()

	cfg := config(dir)
	cfg.D1Create = false
	c := &cfdeploy.Client{Token: "tok", AccountID: "acc", BaseURL: srv.URL}
	if _, err := c.Deploy(ctx, cfg); err == nil || !strings.Contains(err.Error(), `no D1 database named "app"`) {
		t.Errorf("missing D1: err = %v", err)
	}
	bad := &cfdeploy.Client{Token: "wrong", AccountID: "acc", BaseURL: srv.URL}
	if _, err := bad.Deploy(ctx, config(dir)); err == nil || !strings.Contains(err.Error(), "HTTP 401") {
		t.Errorf("bad token: err = %v", err)
	}
	if _, err := (&cfdeploy.Client{BaseURL: srv.URL}).Deploy(ctx, config(dir)); err == nil || !strings.Contains(err.Error(), "Token and AccountID") {
		t.Errorf("no credentials: err = %v", err)
	}
}

func TestNewPlan(t *testing.T) {
	dir := project(t)

	// Go-only layout: build/worker.mjs is the entry.
	p, err := cfdeploy.NewPlan(cfdeploy.Config{Name: "x", CompatibilityDate: "2026-09-11", BuildDir: filepath.Join(dir, "build")})
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, m := range p.Modules {
		names = append(names, m.Name)
	}
	if !slices.Equal(names, cfdeploy.BuildFiles) || len(p.Assets) != 0 {
		t.Errorf("modules = %v, assets %d; want %v and none", names, len(p.Assets), cfdeploy.BuildFiles)
	}

	cfg := config(dir)
	cfg.Modules = append(cfg.Modules, filepath.Join(dir, "worker/missing.mjs"))
	if _, err := cfdeploy.NewPlan(cfg); err == nil || !strings.Contains(err.Error(), "missing.mjs") {
		t.Errorf("missing module: err = %v", err)
	}
	cfg = config(dir)
	cfg.RateLimits = map[string]cfdeploy.RateLimit{"WRITES": {NamespaceID: "1001", Limit: 60, Period: 30}}
	if _, err := cfdeploy.NewPlan(cfg); err == nil || !strings.Contains(err.Error(), "period 10 or 60") {
		t.Errorf("rate limit period 30: err = %v", err)
	}
	if _, err := cfdeploy.NewPlan(cfdeploy.Config{CompatibilityDate: "x"}); err == nil {
		t.Errorf("no name: want an error")
	}
}

// AssetHash must match Cloudflare's Direct Upload example: sha256(base64(content) + extension)[:32].
func TestAssetHash(t *testing.T) {
	sum := sha256.Sum256([]byte(base64.StdEncoding.EncodeToString([]byte("body{}")) + "css"))
	want := hex.EncodeToString(sum[:])[:32]
	for _, ext := range []string{".css", "css"} {
		if got := cfdeploy.AssetHash([]byte("body{}"), ext); got != want {
			t.Errorf("AssetHash(ext %q) = %s, want %s", ext, got, want)
		}
	}
}

func ExampleClient_Deploy() {
	c := &cfdeploy.Client{Token: os.Getenv("CLOUDFLARE_API_TOKEN"), AccountID: os.Getenv("CLOUDFLARE_ACCOUNT_ID")}
	url, err := c.Deploy(context.Background(), cfdeploy.Config{
		Name:              "my-app",
		CompatibilityDate: "2026-09-11",
		AssetsDir:         "dist/site",
		MainModule:        "worker/index.mjs",
		Modules:           []string{"worker/room.mjs"},
		D1:                map[string]string{"DB": "my-app"},
		D1Create:          true,
		MigrationsDir:     "migrations",
		DurableObjects:    map[string]string{"ROOM": "Room"},
		MigrationTag:      "v1",
		NewSQLiteClasses:  []string{"Room"},
		AllowExisting:     true,
	})
	fmt.Println(url, err)
}
