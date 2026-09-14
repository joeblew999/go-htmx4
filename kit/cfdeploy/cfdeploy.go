// Package cfdeploy deploys a Go Worker (TinyGo build from workers-go) to Cloudflare with the REST API:
// no wrangler, no Node. It follows https://developers.cloudflare.com/workers/static-assets/direct-upload/:
//
//  0. D1: find (or create) each database by name and apply migrations/*.sql not yet recorded in its
//     _migrations table (D1 REST API)
//  1. hash every file in the assets directory into a manifest → POST …/assets-upload-session
//  2. upload the buckets Cloudflare asks for (base64 multipart, upload JWT) → completion JWT
//  3. PUT the script: entry module + extra JS modules + build/ (TinyGo) + metadata (compatibility
//     date, bindings, Durable Object migration if not yet applied, assets JWT)
//  4. enable https://<name>.<account subdomain>.workers.dev
//
// It is local tooling (standard Go); the Worker itself is the TinyGo build. Credentials are an API token
// and account id, e.g. from CLOUDFLARE_API_TOKEN and CLOUDFLARE_ACCOUNT_ID injected by fnox.
package cfdeploy

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// DefaultBaseURL is the Cloudflare API.
const DefaultBaseURL = "https://api.cloudflare.com/client/v4"

// BuildFiles are what workers-assets-gen + tinygo write to the build directory.
var BuildFiles = []string{"worker.mjs", "wasm_exec.js", "runtime.mjs", "app.wasm"}

// Config describes one Worker deploy.
type Config struct {
	Name              string // Worker script name (required)
	BuildDir          string // directory with BuildFiles (default "build")
	AssetsDir         string // Static Assets directory ("" for none)
	CompatibilityDate string // compatibility_date (required)
	AllowExisting     bool   // update the Worker if a script with this name already exists

	// MainModule is the entry module, which imports ./build/worker.mjs (e.g. "worker/index.mjs"); "" makes
	// build/worker.mjs the entry. JS modules are named by file name ("worker/index.mjs" → "index.mjs"), so
	// their relative imports (./room.mjs, ./build/worker.mjs) resolve as they do locally.
	MainModule string
	Modules    []string // extra ES modules next to MainModule, e.g. "worker/room.mjs"

	Vars          map[string]string // plain_text bindings
	D1            map[string]string // binding → database name or id
	D1Create      bool              // create D1 databases in D1 that don't exist yet
	MigrationsDir string            // apply these *.sql files (in name order) to every D1 database

	DurableObjects   map[string]string // binding → class name
	MigrationTag     string            // Durable Object migration tag to reach (sent only if not yet applied)
	NewSQLiteClasses []string          // classes created by MigrationTag
}

// Module is one part of the script upload; Name is what imports resolve against.
type Module struct{ Name, Path string }

// ContentType is the upload content type for the module.
func (m Module) ContentType() string {
	if strings.HasSuffix(m.Name, ".wasm") {
		return "application/wasm"
	}
	return "application/javascript+module"
}

// ManifestEntry is one Static Assets file in the upload manifest.
type ManifestEntry struct {
	Hash string `json:"hash"`
	Size int    `json:"size"`
}

// Plan is what a deploy uploads, computed locally without API calls (a dry run).
type Plan struct {
	Modules []Module                 // entry module first
	Assets  map[string]ManifestEntry // URL path → entry
	files   map[string]string        // hash → local path
}

// NewPlan validates cfg, checks every module file exists and hashes the assets.
func NewPlan(cfg Config) (*Plan, error) {
	if cfg.Name == "" {
		return nil, errors.New("cfdeploy: Name is required")
	}
	if cfg.CompatibilityDate == "" {
		return nil, errors.New("cfdeploy: CompatibilityDate is required")
	}
	buildDir := cfg.BuildDir
	if buildDir == "" {
		buildDir = "build"
	}
	p := &Plan{Assets: map[string]ManifestEntry{}, files: map[string]string{}}
	if cfg.AssetsDir != "" {
		var err error
		if p.Assets, p.files, err = BuildManifest(cfg.AssetsDir); err != nil {
			return nil, fmt.Errorf("cfdeploy: assets: %w", err)
		}
	}
	if cfg.MainModule == "" {
		for _, f := range BuildFiles {
			p.Modules = append(p.Modules, Module{f, filepath.Join(buildDir, f)})
		}
	} else {
		p.Modules = append(p.Modules, Module{filepath.Base(cfg.MainModule), cfg.MainModule})
		for _, m := range cfg.Modules {
			p.Modules = append(p.Modules, Module{filepath.Base(m), m})
		}
		for _, f := range BuildFiles {
			p.Modules = append(p.Modules, Module{"build/" + f, filepath.Join(buildDir, f)})
		}
	}
	for _, m := range p.Modules {
		if _, err := os.Stat(m.Path); err != nil {
			return nil, fmt.Errorf("cfdeploy: module %s: %w", m.Name, err)
		}
	}
	return p, nil
}

// BuildManifest hashes each file under dir (skipping dotfiles) the way the Direct Upload example does:
// sha256(base64(content) + extension), first 32 hex characters. It returns the manifest keyed by URL path
// and a map from hash to local path.
func BuildManifest(dir string) (map[string]ManifestEntry, map[string]string, error) {
	manifest := map[string]ManifestEntry{}
	files := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || strings.HasPrefix(d.Name(), ".") {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		hash := AssetHash(b, filepath.Ext(p))
		manifest["/"+filepath.ToSlash(rel)] = ManifestEntry{Hash: hash, Size: len(b)}
		files[hash] = p
		return nil
	})
	if err == nil && len(manifest) == 0 {
		err = fmt.Errorf("no files in %s", dir)
	}
	return manifest, files, err
}

// AssetHash is the Direct Upload asset hash of content with file extension ext (".js" or "js").
func AssetHash(content []byte, ext string) string {
	sum := sha256.Sum256([]byte(base64.StdEncoding.EncodeToString(content) + strings.TrimPrefix(ext, ".")))
	return hex.EncodeToString(sum[:])[:32]
}

// Client talks to the Cloudflare API for one account.
type Client struct {
	Token     string
	AccountID string
	BaseURL   string                           // default DefaultBaseURL
	HTTP      *http.Client                     // default http.DefaultClient
	Logf      func(format string, args ...any) // progress lines; nil discards them
}

// Deploy runs the whole deploy for cfg and returns the Worker's workers.dev URL.
func (c *Client) Deploy(ctx context.Context, cfg Config) (string, error) {
	plan, err := NewPlan(cfg)
	if err != nil {
		return "", err
	}
	if c.Token == "" || c.AccountID == "" {
		return "", errors.New("cfdeploy: Token and AccountID are required")
	}

	exists, appliedTag, err := c.script(ctx, cfg.Name)
	if err != nil {
		return "", fmt.Errorf("cfdeploy: list scripts: %w", err)
	}
	if exists && !cfg.AllowExisting {
		return "", fmt.Errorf("cfdeploy: a Worker named %q already exists in this account; set AllowExisting to update it", cfg.Name)
	}

	d1IDs := map[string]string{}
	for _, binding := range slices.Sorted(maps.Keys(cfg.D1)) {
		ref := cfg.D1[binding]
		id, err := c.resolveD1(ctx, ref, cfg.D1Create)
		if err != nil {
			return "", fmt.Errorf("cfdeploy: d1 %s=%s: %w", binding, ref, err)
		}
		d1IDs[binding] = id
		if cfg.MigrationsDir != "" {
			if err := c.migrateD1(ctx, id, cfg.MigrationsDir); err != nil {
				return "", fmt.Errorf("cfdeploy: d1 %s migrations: %w", ref, err)
			}
		}
	}

	var doMigration map[string]any
	switch {
	case cfg.MigrationTag == "" || appliedTag == cfg.MigrationTag:
		if cfg.MigrationTag != "" {
			c.logf("✓ durable object migration %q already applied", appliedTag)
		}
	case appliedTag == "":
		doMigration = map[string]any{"new_tag": cfg.MigrationTag, "new_sqlite_classes": cfg.NewSQLiteClasses}
	default:
		return "", fmt.Errorf("cfdeploy: script is at migration %q; moving to %q needs an explicit migration step", appliedTag, cfg.MigrationTag)
	}

	jwt := ""
	if len(plan.Assets) > 0 {
		if jwt, err = c.uploadAssets(ctx, cfg.Name, plan); err != nil {
			return "", fmt.Errorf("cfdeploy: assets: %w", err)
		}
	}
	if err := c.putScript(ctx, cfg, plan.Modules, d1IDs, doMigration, jwt); err != nil {
		return "", fmt.Errorf("cfdeploy: script: %w", err)
	}
	if doMigration != nil {
		c.logf("✓ script uploaded (%d modules, durable object migration %s)", len(plan.Modules), cfg.MigrationTag)
	} else {
		c.logf("✓ script uploaded (%d modules)", len(plan.Modules))
	}

	if err := c.do(ctx, "POST", c.accountPath("/workers/scripts/"+cfg.Name+"/subdomain"), "",
		jsonBody(map[string]bool{"enabled": true, "previews_enabled": false}), nil); err != nil {
		return "", fmt.Errorf("cfdeploy: enable workers.dev: %w", err)
	}
	var sub struct {
		Subdomain string `json:"subdomain"`
	}
	if err := c.do(ctx, "GET", c.accountPath("/workers/subdomain"), "", nil, &sub); err != nil {
		return "", fmt.Errorf("cfdeploy: account workers.dev subdomain: %w", err)
	}
	return fmt.Sprintf("https://%s.%s.workers.dev", cfg.Name, sub.Subdomain), nil
}

func (c *Client) logf(format string, args ...any) {
	if c.Logf != nil {
		c.Logf(format, args...)
	}
}

func (c *Client) accountPath(p string) string { return "/accounts/" + c.AccountID + p }

type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// do calls the Cloudflare API and decodes the envelope's result into out. bearer overrides the API
// token (asset uploads authenticate with the upload-session JWT).
func (c *Client) do(ctx context.Context, method, apiPath, bearer string, body *requestBody, out any) error {
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body.data)
	}
	base := c.BaseURL
	if base == "" {
		base = DefaultBaseURL
	}
	req, err := http.NewRequestWithContext(ctx, method, base+apiPath, r)
	if err != nil {
		return err
	}
	if bearer == "" {
		bearer = c.Token
	}
	req.Header.Set("Authorization", "Bearer "+bearer)
	if body != nil {
		req.Header.Set("Content-Type", body.contentType)
	}
	hc := c.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var env struct {
		Success bool            `json:"success"`
		Errors  []apiError      `json:"errors"`
		Result  json.RawMessage `json:"result"`
	}
	raw, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(raw, &env); err != nil {
		return fmt.Errorf("%s %s: HTTP %d, non-JSON response", method, apiPath, resp.StatusCode)
	}
	if !env.Success {
		return fmt.Errorf("%s %s: HTTP %d: %v", method, apiPath, resp.StatusCode, env.Errors)
	}
	if out != nil && len(env.Result) > 0 && string(env.Result) != "null" {
		return json.Unmarshal(env.Result, out)
	}
	return nil
}

// script reports whether the Worker exists and its most recent Durable Object migration tag.
func (c *Client) script(ctx context.Context, name string) (exists bool, migrationTag string, err error) {
	var scripts []struct {
		ID           string `json:"id"`
		MigrationTag string `json:"migration_tag"`
	}
	if err := c.do(ctx, "GET", c.accountPath("/workers/scripts"), "", nil, &scripts); err != nil {
		return false, "", err
	}
	for _, s := range scripts {
		if s.ID == name {
			return true, s.MigrationTag, nil
		}
	}
	return false, "", nil
}

var uuidRe = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// resolveD1 turns a database name (or id) into its id, creating it if asked.
func (c *Client) resolveD1(ctx context.Context, ref string, create bool) (string, error) {
	if uuidRe.MatchString(ref) {
		return ref, nil
	}
	var dbs []struct {
		UUID string `json:"uuid"`
		Name string `json:"name"`
	}
	if err := c.do(ctx, "GET", c.accountPath("/d1/database?name="+url.QueryEscape(ref)), "", nil, &dbs); err != nil {
		return "", err
	}
	for _, db := range dbs {
		if db.Name == ref {
			c.logf("✓ d1 %s (%s)", ref, db.UUID)
			return db.UUID, nil
		}
	}
	if !create {
		return "", fmt.Errorf("no D1 database named %q (set D1Create)", ref)
	}
	var created struct {
		UUID string `json:"uuid"`
	}
	if err := c.do(ctx, "POST", c.accountPath("/d1/database"), "", jsonBody(map[string]string{"name": ref}), &created); err != nil {
		return "", err
	}
	c.logf("✓ d1 %s created (%s)", ref, created.UUID)
	return created.UUID, nil
}

// migrateD1 applies dir/*.sql in name order, recording each in _migrations so it runs once.
func (c *Client) migrateD1(ctx context.Context, id, dir string) error {
	if _, err := c.d1Query(ctx, id, `CREATE TABLE IF NOT EXISTS _migrations (name TEXT PRIMARY KEY, applied_at TEXT NOT NULL DEFAULT (datetime('now')))`); err != nil {
		return err
	}
	rows, err := c.d1Query(ctx, id, `SELECT name FROM _migrations`)
	if err != nil {
		return err
	}
	applied := map[string]bool{}
	for _, r := range rows {
		applied[fmt.Sprint(r["name"])] = true
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return err
	}
	slices.Sort(files)
	for _, f := range files {
		name := strings.TrimSuffix(filepath.Base(f), ".sql")
		if applied[name] {
			continue
		}
		body, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		if _, err := c.d1Query(ctx, id, string(body)); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if _, err := c.d1Query(ctx, id, `INSERT INTO _migrations (name) VALUES (?)`, name); err != nil {
			return err
		}
		c.logf("✓ d1 migration %s applied", name)
	}
	return nil
}

// d1Query runs SQL through the D1 REST API and returns the last statement's rows.
func (c *Client) d1Query(ctx context.Context, id, sql string, params ...any) ([]map[string]any, error) {
	req := map[string]any{"sql": sql}
	if len(params) > 0 {
		req["params"] = params
	}
	var results []struct {
		Results []map[string]any `json:"results"`
	}
	if err := c.do(ctx, "POST", c.accountPath("/d1/database/"+id+"/query"), "", jsonBody(req), &results); err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, nil
	}
	return results[len(results)-1].Results, nil
}

// uploadAssets runs steps 1–2 and returns the completion JWT (or the session JWT when nothing changed).
func (c *Client) uploadAssets(ctx context.Context, name string, plan *Plan) (string, error) {
	var session struct {
		JWT     string     `json:"jwt"`
		Buckets [][]string `json:"buckets"`
	}
	if err := c.do(ctx, "POST", c.accountPath("/workers/scripts/"+name+"/assets-upload-session"), "",
		jsonBody(map[string]any{"manifest": plan.Assets}), &session); err != nil {
		return "", err
	}
	if session.JWT == "" {
		return "", errors.New("upload session returned no JWT")
	}
	if len(session.Buckets) == 0 {
		c.logf("✓ assets unchanged (nothing to upload)")
		return session.JWT, nil
	}
	completion := ""
	for i, bucket := range session.Buckets {
		body, err := multipartBody(func(mw *multipart.Writer) error {
			for _, hash := range bucket {
				p, ok := plan.files[hash]
				if !ok {
					return fmt.Errorf("server asked for unknown hash %s", hash)
				}
				b, err := os.ReadFile(p)
				if err != nil {
					return err
				}
				if err := writePart(mw, hash, hash, contentType(p), []byte(base64.StdEncoding.EncodeToString(b))); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return "", err
		}
		var res struct {
			JWT string `json:"jwt"`
		}
		if err := c.do(ctx, "POST", c.accountPath("/workers/assets/upload?base64=true"), session.JWT, body, &res); err != nil {
			return "", fmt.Errorf("bucket %d/%d: %w", i+1, len(session.Buckets), err)
		}
		if res.JWT != "" {
			completion = res.JWT
		}
		c.logf("✓ assets bucket %d/%d (%d files)", i+1, len(session.Buckets), len(bucket))
	}
	if completion == "" {
		return "", errors.New("uploads finished without a completion JWT")
	}
	return completion, nil
}

// putScript runs step 3: the multipart script upload with its metadata part.
func (c *Client) putScript(ctx context.Context, cfg Config, modules []Module, d1IDs map[string]string, doMigration map[string]any, assetsJWT string) error {
	bindings := []map[string]string{}
	for _, k := range slices.Sorted(maps.Keys(cfg.Vars)) {
		bindings = append(bindings, map[string]string{"type": "plain_text", "name": k, "text": cfg.Vars[k]})
	}
	for _, k := range slices.Sorted(maps.Keys(d1IDs)) {
		bindings = append(bindings, map[string]string{"type": "d1", "name": k, "id": d1IDs[k]})
	}
	for _, k := range slices.Sorted(maps.Keys(cfg.DurableObjects)) {
		bindings = append(bindings, map[string]string{"type": "durable_object_namespace", "name": k, "class_name": cfg.DurableObjects[k]})
	}
	meta := map[string]any{
		"main_module":        modules[0].Name,
		"compatibility_date": cfg.CompatibilityDate,
		"bindings":           bindings,
	}
	if doMigration != nil {
		meta["migrations"] = doMigration
	}
	if assetsJWT != "" {
		meta["assets"] = map[string]string{"jwt": assetsJWT}
	}
	metadata, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	body, err := multipartBody(func(mw *multipart.Writer) error {
		if err := writePart(mw, "metadata", "", "application/json", metadata); err != nil {
			return err
		}
		for _, m := range modules {
			b, err := os.ReadFile(m.Path)
			if err != nil {
				return err
			}
			if err := writePart(mw, m.Name, m.Name, m.ContentType(), b); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return c.do(ctx, "PUT", c.accountPath("/workers/scripts/"+cfg.Name), "", body, nil)
}

type requestBody struct {
	contentType string
	data        []byte
}

func jsonBody(v any) *requestBody {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return &requestBody{contentType: "application/json", data: b}
}

func multipartBody(fill func(*multipart.Writer) error) (*requestBody, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if err := fill(mw); err != nil {
		return nil, err
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}
	return &requestBody{contentType: mw.FormDataContentType(), data: buf.Bytes()}, nil
}

func writePart(mw *multipart.Writer, field, filename, contentType string, data []byte) error {
	h := textproto.MIMEHeader{}
	disposition := fmt.Sprintf(`form-data; name=%q`, field)
	if filename != "" {
		disposition += fmt.Sprintf(`; filename=%q`, filename)
	}
	h.Set("Content-Disposition", disposition)
	h.Set("Content-Type", contentType)
	w, err := mw.CreatePart(h)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func contentType(p string) string {
	if t := mime.TypeByExtension(path.Ext(p)); t != "" {
		return t
	}
	return "application/octet-stream"
}
