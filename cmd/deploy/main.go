// Command deploy uploads the app to Cloudflare Workers with the REST API (no wrangler, no Node): flags
// for kit/cfdeploy, which documents the steps.
//
// Credentials come from CLOUDFLARE_API_TOKEN and CLOUDFLARE_ACCOUNT_ID; `mise run deploy` injects them
// with `fnox exec`. It prints the Worker's workers.dev URL on stdout and progress on stderr.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joeblew999/go-htmx4/kit/cfdeploy"
)

type list []string

func (l *list) String() string     { return strings.Join(*l, ",") }
func (l *list) Set(s string) error { *l = append(*l, s); return nil }

// rateLimits parses NAME=NAMESPACE_ID:LIMIT/PERIOD, e.g. WRITES=1001:60/10.
type rateLimits map[string]cfdeploy.RateLimit

func (r rateLimits) String() string { return fmt.Sprint(map[string]cfdeploy.RateLimit(r)) }
func (r rateLimits) Set(s string) error {
	var rl cfdeploy.RateLimit
	name, spec, ok := strings.Cut(s, "=")
	ns, window, ok2 := strings.Cut(spec, ":")
	if !ok || !ok2 || name == "" {
		return fmt.Errorf("want NAME=NAMESPACE_ID:LIMIT/PERIOD, got %q", s)
	}
	if _, err := fmt.Sscanf(window, "%d/%d", &rl.Limit, &rl.Period); err != nil {
		return fmt.Errorf("want NAME=NAMESPACE_ID:LIMIT/PERIOD, got %q", s)
	}
	rl.NamespaceID = ns
	r[name] = rl
	return nil
}

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
	cfg := cfdeploy.Config{Vars: vars{}, D1: vars{}, DurableObjects: vars{}, RateLimits: rateLimits{}}
	flag.StringVar(&cfg.Name, "name", "", "Worker script name (required)")
	flag.StringVar(&cfg.BuildDir, "build", "build", "directory with worker.mjs, wasm_exec.js, runtime.mjs, app.wasm")
	flag.StringVar(&cfg.AssetsDir, "assets", "", `static assets directory ("" for none)`)
	flag.StringVar(&cfg.CompatibilityDate, "compat", "2026-09-11", "compatibility_date")
	flag.BoolVar(&cfg.AllowExisting, "allow-existing", false, "update the Worker if a script with this name already exists")
	flag.StringVar(&cfg.MainModule, "main", "", `entry module that imports ./build/worker.mjs, e.g. worker/index.mjs ("" = build/worker.mjs is the entry)`)
	flag.BoolVar(&cfg.D1Create, "d1-create", false, "create D1 databases named in -d1 that don't exist yet")
	flag.StringVar(&cfg.MigrationsDir, "migrations", "", "apply these *.sql files (in name order) to every -d1 database")
	flag.StringVar(&cfg.MigrationTag, "migration-tag", "", "Durable Object migration tag to reach (sent only if not yet applied)")
	flag.Var(vars(cfg.Vars), "var", "plain_text binding KEY=VALUE (repeatable)")
	flag.Var(vars(cfg.D1), "d1", "D1 binding NAME=DATABASE_NAME or NAME=DATABASE_ID (repeatable)")
	flag.Var(vars(cfg.DurableObjects), "do", "Durable Object binding NAME=CLASS (repeatable)")
	flag.Var(rateLimits(cfg.RateLimits), "ratelimit", "rate limiting binding NAME=NAMESPACE_ID:LIMIT/PERIOD, e.g. WRITES=1001:60/10 (repeatable)")
	flag.Var((*list)(&cfg.Modules), "module", "extra ES module next to -main, e.g. worker/room.mjs (repeatable)")
	flag.Var((*list)(&cfg.NewSQLiteClasses), "new-sqlite-class", "Durable Object class created by -migration-tag (repeatable)")
	dryRun := flag.Bool("dry-run", false, "print what would be uploaded; make no API calls")
	flag.Parse()

	plan, err := cfdeploy.NewPlan(cfg)
	if err != nil {
		log.Fatalf("deploy: %v (run mise run build)", err)
	}
	log.Printf("worker %q: %d modules (main %s), %d assets from %q, vars %v, d1 %v, do %v, ratelimit %v, migration %q %v",
		cfg.Name, len(plan.Modules), plan.Modules[0].Name, len(plan.Assets), cfg.AssetsDir, cfg.Vars, cfg.D1, cfg.DurableObjects, cfg.RateLimits, cfg.MigrationTag, cfg.NewSQLiteClasses)
	if *dryRun {
		for _, m := range plan.Modules {
			log.Printf("  module %-22s ← %s", m.Name, m.Path)
		}
		for p, e := range plan.Assets {
			log.Printf("  asset  %-22s %s %dB", p, e.Hash, e.Size)
		}
		return
	}

	c := &cfdeploy.Client{
		Token:     os.Getenv("CLOUDFLARE_API_TOKEN"),
		AccountID: os.Getenv("CLOUDFLARE_ACCOUNT_ID"),
		HTTP:      &http.Client{Timeout: 2 * time.Minute},
		Logf:      log.Printf,
	}
	if c.Token == "" || c.AccountID == "" {
		log.Fatal("deploy: CLOUDFLARE_API_TOKEN and CLOUDFLARE_ACCOUNT_ID must be set (use fnox exec)")
	}
	url, err := c.Deploy(context.Background(), cfg)
	if err != nil {
		log.Fatalf("deploy: %v", err)
	}
	fmt.Println(url)
}
