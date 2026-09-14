// Command deploy uploads demos/workers to Cloudflare Workers with the REST API: no wrangler,
// no Node. It follows https://developers.cloudflare.com/workers/static-assets/direct-upload/:
//
//  0. D1: find (or create) each database by name and apply migrations/*.sql not yet recorded
//     in its _migrations table (D1 REST API)
//  1. hash every file in public/ into a manifest → POST …/assets-upload-session
//  2. upload the buckets Cloudflare asks for (base64 multipart, upload JWT) → completion JWT
//  3. PUT the script: index.mjs + JS modules + build/ (TinyGo) + metadata (compatibility date,
//     bindings, Durable Object migration if not yet applied, assets JWT)
//  4. enable https://<name>.<account subdomain>.workers.dev
//
// Credentials come from CLOUDFLARE_API_TOKEN and CLOUDFLARE_ACCOUNT_ID; `mise run
// demo:workers:deploy` injects them with `fnox exec`. This is local-only tooling (standard Go);
// the Worker itself is the TinyGo build in build/.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
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
	"time"
)

const apiBase = "https://api.cloudflare.com/client/v4"

// buildFiles are what workers-assets-gen + tinygo write to build/.
var buildFiles = []string{"worker.mjs", "wasm_exec.js", "runtime.mjs", "app.wasm"}

// module is one part of the script upload; name is the module name imports resolve against.
type module struct{ name, path string }

func (m module) contentType() string {
	if strings.HasSuffix(m.name, ".wasm") {
		return "application/wasm"
	}
	return "application/javascript+module"
}

var uuidRe = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

type list []string

func (l *list) String() string     { return strings.Join(*l, ",") }
func (l *list) Set(s string) error { *l = append(*l, s); return nil }

type vars map[string]string

func (v vars) String() string { return fmt.Sprint(map[string]string(v)) }
func (v vars) Set(s string) error {
	k, val, ok := strings.Cut(s, "=")
	if !ok || k == "" {
		return fmt.Errorf("want KEY=VALUE, got %q", s)
	}
	v[k] = val
	return nil
}

func main() {
	log.SetFlags(0)
	var (
		name          = flag.String("name", "", "Worker script name (required)")
		buildDir      = flag.String("build", "build", "directory with worker.mjs, wasm_exec.js, runtime.mjs, app.wasm")
		assetsDir     = flag.String("assets", "public", `static assets directory ("" for none)`)
		compatDate    = flag.String("compat", "2026-09-11", "compatibility_date")
		allowExisting = flag.Bool("allow-existing", false, "update the Worker if a script with this name already exists")
		dryRun        = flag.Bool("dry-run", false, "print what would be uploaded; make no API calls")
		mainModule    = flag.String("main", "index.mjs", `entry module that imports ./build/worker.mjs ("" = build/worker.mjs is the entry)`)
		d1Create      = flag.Bool("d1-create", false, "create D1 databases named in -d1 that don't exist yet")
		migrations    = flag.String("migrations", "", "apply these *.sql files (in name order) to every -d1 database")
		migrationTag  = flag.String("migration-tag", "", "Durable Object migration tag to reach (sent only if not yet applied)")
		textVars      = vars{}
		d1Bindings    = vars{}
		doBindings    = vars{}
		jsModules     list
		sqliteClasses list
	)
	flag.Var(textVars, "var", "plain_text binding KEY=VALUE (repeatable)")
	flag.Var(d1Bindings, "d1", "D1 binding NAME=DATABASE_NAME or NAME=DATABASE_ID (repeatable)")
	flag.Var(doBindings, "do", "Durable Object binding NAME=CLASS (repeatable)")
	flag.Var(&jsModules, "module", "extra ES module next to -main, e.g. room.mjs (repeatable)")
	flag.Var(&sqliteClasses, "new-sqlite-class", "Durable Object class created by -migration-tag (repeatable)")
	flag.Parse()
	if *name == "" {
		log.Fatal("deploy: -name is required")
	}

	manifest, files := map[string]manifestEntry{}, map[string]string{}
	if *assetsDir != "" {
		var err error
		if manifest, files, err = buildManifest(*assetsDir); err != nil {
			log.Fatalf("deploy: %v", err)
		}
	}
	var modules []module
	if *mainModule == "" {
		for _, f := range buildFiles {
			modules = append(modules, module{f, filepath.Join(*buildDir, f)})
		}
	} else {
		modules = append(modules, module{*mainModule, *mainModule})
		for _, m := range jsModules {
			modules = append(modules, module{m, m})
		}
		for _, f := range buildFiles {
			modules = append(modules, module{"build/" + f, filepath.Join(*buildDir, f)})
		}
	}
	for _, m := range modules {
		if _, err := os.Stat(m.path); err != nil {
			log.Fatalf("deploy: %v (run mise run demo:workers:build)", err)
		}
	}
	log.Printf("worker %q: %d modules (main %s), %d assets from %q, vars %v, d1 %v, do %v, migration %q %v",
		*name, len(modules), modules[0].name, len(manifest), *assetsDir, textVars, d1Bindings, doBindings, *migrationTag, sqliteClasses)
	if *dryRun {
		for _, m := range modules {
			log.Printf("  module %-22s ← %s", m.name, m.path)
		}
		for p, e := range manifest {
			log.Printf("  asset  %-22s %s %dB", p, e.Hash, e.Size)
		}
		return
	}

	c := &client{
		token:   os.Getenv("CLOUDFLARE_API_TOKEN"),
		account: os.Getenv("CLOUDFLARE_ACCOUNT_ID"),
		http:    &http.Client{Timeout: 2 * time.Minute},
	}
	if c.token == "" || c.account == "" {
		log.Fatal("deploy: CLOUDFLARE_API_TOKEN and CLOUDFLARE_ACCOUNT_ID must be set (use fnox exec)")
	}

	exists, appliedTag, err := c.script(*name)
	if err != nil {
		log.Fatalf("deploy: list scripts: %v", err)
	}
	if exists && !*allowExisting {
		log.Fatalf("deploy: a Worker named %q already exists in this account; pass -allow-existing to update it", *name)
	}

	d1IDs := vars{}
	for binding, ref := range d1Bindings {
		id, err := c.resolveD1(ref, *d1Create)
		if err != nil {
			log.Fatalf("deploy: d1 %s=%s: %v", binding, ref, err)
		}
		d1IDs[binding] = id
		if *migrations != "" {
			if err := c.migrateD1(id, *migrations); err != nil {
				log.Fatalf("deploy: d1 %s migrations: %v", ref, err)
			}
		}
	}

	var doMigration map[string]any
	switch {
	case *migrationTag == "" || appliedTag == *migrationTag:
		if *migrationTag != "" {
			log.Printf("✓ durable object migration %q already applied", appliedTag)
		}
	case appliedTag == "":
		doMigration = map[string]any{"new_tag": *migrationTag, "new_sqlite_classes": sqliteClasses}
	default:
		log.Fatalf("deploy: script is at migration %q; moving to %q needs an explicit migration step", appliedTag, *migrationTag)
	}

	jwt := ""
	if len(manifest) > 0 {
		if jwt, err = c.uploadAssets(*name, manifest, files); err != nil {
			log.Fatalf("deploy: assets: %v", err)
		}
	}
	if err := c.putScript(*name, modules, *compatDate, textVars, d1IDs, doBindings, doMigration, jwt); err != nil {
		log.Fatalf("deploy: script: %v", err)
	}
	log.Printf("✓ script uploaded (%d modules%s)", len(modules), map[bool]string{true: ", durable object migration " + *migrationTag, false: ""}[doMigration != nil])

	if err := c.do("POST", "/accounts/"+c.account+"/workers/scripts/"+*name+"/subdomain", "",
		jsonBody(map[string]bool{"enabled": true, "previews_enabled": false}), nil); err != nil {
		log.Fatalf("deploy: enable workers.dev: %v", err)
	}
	var sub struct {
		Subdomain string `json:"subdomain"`
	}
	if err := c.do("GET", "/accounts/"+c.account+"/workers/subdomain", "", nil, &sub); err != nil {
		log.Fatalf("deploy: account workers.dev subdomain: %v", err)
	}
	fmt.Printf("https://%s.%s.workers.dev\n", *name, sub.Subdomain)
}

type manifestEntry struct {
	Hash string `json:"hash"`
	Size int    `json:"size"`
}

// buildManifest hashes each file the way the Direct Upload example does:
// sha256(base64(content) + extension), first 32 hex characters.
func buildManifest(dir string) (map[string]manifestEntry, map[string]string, error) {
	manifest := map[string]manifestEntry{}
	files := map[string]string{} // hash → local path
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
		sum := sha256.Sum256([]byte(base64.StdEncoding.EncodeToString(b) + strings.TrimPrefix(filepath.Ext(p), ".")))
		hash := hex.EncodeToString(sum[:])[:32]
		manifest["/"+filepath.ToSlash(rel)] = manifestEntry{Hash: hash, Size: len(b)}
		files[hash] = p
		return nil
	})
	if err == nil && len(manifest) == 0 {
		err = fmt.Errorf("no files in %s", dir)
	}
	return manifest, files, err
}

type client struct {
	token, account string
	http           *http.Client
}

type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// do calls the Cloudflare API and decodes the envelope's result into out. bearer overrides
// the API token (asset uploads authenticate with the upload-session JWT).
func (c *client) do(method, apiPath, bearer string, body *requestBody, out any) error {
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body.data)
	}
	req, err := http.NewRequest(method, apiBase+apiPath, r)
	if err != nil {
		return err
	}
	if bearer == "" {
		bearer = c.token
	}
	req.Header.Set("Authorization", "Bearer "+bearer)
	if body != nil {
		req.Header.Set("Content-Type", body.contentType)
	}
	resp, err := c.http.Do(req)
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
func (c *client) script(name string) (exists bool, migrationTag string, err error) {
	var scripts []struct {
		ID           string `json:"id"`
		MigrationTag string `json:"migration_tag"`
	}
	if err := c.do("GET", "/accounts/"+c.account+"/workers/scripts", "", nil, &scripts); err != nil {
		return false, "", err
	}
	for _, s := range scripts {
		if s.ID == name {
			return true, s.MigrationTag, nil
		}
	}
	return false, "", nil
}

// resolveD1 turns a database name (or id) into its id, creating it if asked.
func (c *client) resolveD1(ref string, create bool) (string, error) {
	if uuidRe.MatchString(ref) {
		return ref, nil
	}
	var dbs []struct {
		UUID string `json:"uuid"`
		Name string `json:"name"`
	}
	if err := c.do("GET", "/accounts/"+c.account+"/d1/database?name="+url.QueryEscape(ref), "", nil, &dbs); err != nil {
		return "", err
	}
	for _, db := range dbs {
		if db.Name == ref {
			log.Printf("✓ d1 %s (%s)", ref, db.UUID)
			return db.UUID, nil
		}
	}
	if !create {
		return "", fmt.Errorf("no D1 database named %q (pass -d1-create)", ref)
	}
	var created struct {
		UUID string `json:"uuid"`
	}
	if err := c.do("POST", "/accounts/"+c.account+"/d1/database", "", jsonBody(map[string]string{"name": ref}), &created); err != nil {
		return "", err
	}
	log.Printf("✓ d1 %s created (%s)", ref, created.UUID)
	return created.UUID, nil
}

// migrateD1 applies dir/*.sql in name order, recording each in _migrations so it runs once.
func (c *client) migrateD1(id, dir string) error {
	if _, err := c.d1Query(id, `CREATE TABLE IF NOT EXISTS _migrations (name TEXT PRIMARY KEY, applied_at TEXT NOT NULL DEFAULT (datetime('now')))`); err != nil {
		return err
	}
	rows, err := c.d1Query(id, `SELECT name FROM _migrations`)
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
		if _, err := c.d1Query(id, string(body)); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if _, err := c.d1Query(id, `INSERT INTO _migrations (name) VALUES (?)`, name); err != nil {
			return err
		}
		log.Printf("✓ d1 migration %s applied", name)
	}
	return nil
}

// d1Query runs SQL through the D1 REST API and returns the last statement's rows.
func (c *client) d1Query(id, sql string, params ...any) ([]map[string]any, error) {
	req := map[string]any{"sql": sql}
	if len(params) > 0 {
		req["params"] = params
	}
	var results []struct {
		Results []map[string]any `json:"results"`
	}
	if err := c.do("POST", "/accounts/"+c.account+"/d1/database/"+id+"/query", "", jsonBody(req), &results); err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, nil
	}
	return results[len(results)-1].Results, nil
}

// uploadAssets runs steps 1–2 and returns the completion JWT.
func (c *client) uploadAssets(name string, manifest map[string]manifestEntry, files map[string]string) (string, error) {
	var session struct {
		JWT     string     `json:"jwt"`
		Buckets [][]string `json:"buckets"`
	}
	if err := c.do("POST", "/accounts/"+c.account+"/workers/scripts/"+name+"/assets-upload-session", "",
		jsonBody(map[string]any{"manifest": manifest}), &session); err != nil {
		return "", err
	}
	if session.JWT == "" {
		return "", errors.New("upload session returned no JWT")
	}
	if len(session.Buckets) == 0 {
		log.Printf("✓ assets unchanged (nothing to upload)")
		return session.JWT, nil
	}
	completion := ""
	for i, bucket := range session.Buckets {
		body, err := multipartBody(func(mw *multipart.Writer) error {
			for _, hash := range bucket {
				p, ok := files[hash]
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
		if err := c.do("POST", "/accounts/"+c.account+"/workers/assets/upload?base64=true", session.JWT, body, &res); err != nil {
			return "", fmt.Errorf("bucket %d/%d: %w", i+1, len(session.Buckets), err)
		}
		if res.JWT != "" {
			completion = res.JWT
		}
		log.Printf("✓ assets bucket %d/%d (%d files)", i+1, len(session.Buckets), len(bucket))
	}
	if completion == "" {
		return "", errors.New("uploads finished without a completion JWT")
	}
	return completion, nil
}

// putScript runs step 3: the multipart script upload with its metadata part.
func (c *client) putScript(name string, modules []module, compatDate string, textVars, d1IDs, doBindings vars, doMigration map[string]any, assetsJWT string) error {
	bindings := []map[string]string{}
	for k, v := range textVars {
		bindings = append(bindings, map[string]string{"type": "plain_text", "name": k, "text": v})
	}
	for k, id := range d1IDs {
		bindings = append(bindings, map[string]string{"type": "d1", "name": k, "id": id})
	}
	for k, class := range doBindings {
		bindings = append(bindings, map[string]string{"type": "durable_object_namespace", "name": k, "class_name": class})
	}
	meta := map[string]any{
		"main_module":        modules[0].name,
		"compatibility_date": compatDate,
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
			b, err := os.ReadFile(m.path)
			if err != nil {
				return err
			}
			if err := writePart(mw, m.name, m.name, m.contentType(), b); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return c.do("PUT", "/accounts/"+c.account+"/workers/scripts/"+name, "", body, nil)
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
