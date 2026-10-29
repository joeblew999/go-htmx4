package fresh

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestFreshWorker is the proof of Step 3 (#90): a brand-new gsx app runs on Cloudflare Workers with charter alone,
// set up as charter says: a git repo with charter's library, then charter init (charter.toml, the mise tasks, Workers
// Builds' launcher, the skill, the CI workflow, TinyGo's pins and charter's files for the repo, which charter check
// then passes), its bindings in charter.toml, one main package (testdata/worker). Its own go test runs on charter's
// stand-ins (D1 as SQLite, the rate limit counted); its Worker build runs on local workerd (charter with), where a
// probe checks the page, D1 and the rate limit.
//
// charter is the command mise.toml pins and the library at the version go.mod requires (the same: charter check holds
// them to it); FRESH_CHARTER=<a charter checkout> takes both from the checkout instead. FRESH_DEPLOY=1 also deploys it to Cloudflare, the Worker charter-fresh on workers.dev with its own D1, and
// probes that; it needs Cloudflare credentials (from the environment, else fnox).
func TestFreshWorker(t *testing.T) {
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	app := t.TempDir()
	env := append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=mod")
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir, cmd.Env = app, env
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

	// A new module with gsx as a tool and charter's library, at the versions this repo builds with, in a git repo of
	// its own (charter init writes charter's files at the repo's top).
	gomod := read(t, filepath.Join(repo, "go.mod"))
	version := func(module string) string {
		return regexp.MustCompile(regexp.QuoteMeta(module) + ` (v\S+)`).FindStringSubmatch(gomod)[1]
	}
	write("go.mod", "module example.com/charter-fresh\n\ngo "+regexp.MustCompile(`(?m)^go (\S+)`).FindStringSubmatch(gomod)[1]+
		"\n\ntoolchain "+regexp.MustCompile(`(?m)^toolchain (\S+)`).FindStringSubmatch(gomod)[1]+"\n")
	run("git", "init", "-q")
	module := "github.com/joeblew999/charter"
	run("go", "mod", "edit", "-require=github.com/gsxhq/gsx@"+version("github.com/gsxhq/gsx"), "-tool=github.com/gsxhq/gsx/cmd/gsx",
		"-require="+module+"@"+version(module))
	charter, err := exec.LookPath("charter")
	if dir := os.Getenv("FRESH_CHARTER"); dir != "" {
		abs, err := filepath.Abs(dir)
		if err != nil {
			t.Fatal(err)
		}
		run("go", "mod", "edit", "-replace="+module+"="+abs)
		charter = filepath.Join(t.TempDir(), "charter")
		build := exec.Command("go", "build", "-o", charter, "./cmd/charter")
		build.Dir = abs
		if out, err := build.CombinedOutput(); err != nil {
			t.Fatalf("charter from %s: %v\n%s", abs, err, out)
		}
	} else if err != nil {
		t.Fatal("no charter on PATH: mise install, then run this through mise (mise run fresh)")
	}
	for _, f := range []string{"main.go", "page.gsx", "main_test.go", "probe/probe_test.go", "migrations/0001_visits.sql"} {
		write(f, read(t, filepath.Join("testdata", "worker", f)))
	}
	run("go", "mod", "tidy")

	// charter init, then the app's bindings: a D1 database with its migrations, and a rate limit.
	run(charter, "init")
	write("charter.toml", read(t, filepath.Join(app, "charter.toml"))+read(t, filepath.Join("testdata", "worker", "bindings.toml")))
	if !strings.Contains(read(t, filepath.Join(app, ".gitignore")), "/build/") {
		t.Error("charter init didn't ignore build/")
	}
	for _, f := range []string{".mise/tasks/charter.toml", ".mise/tasks/charter-builds", ".claude/skills/charter/SKILL.md", ".github/workflows/charter.yml",
		".config/mise/conf.d/charter.toml", "docs/charter/README.md"} {
		if _, err := os.Stat(filepath.Join(app, f)); err != nil {
			t.Errorf("charter init didn't write %s: %v", f, err)
		}
	}
	// Everything init wrote, and the bindings added to charter.toml, as charter checks them.
	run(charter, "check")
	run("go", "tool", "gsx", "generate")
	run("go", "test", "-count=1", ".")

	// The Worker on local workerd. The standard compiler (seconds); TinyGo, which ships, when it deploys.
	compiler := "go"
	if os.Getenv("FRESH_DEPLOY") == "1" {
		compiler = "tinygo"
	}
	run(charter, "build", "-compiler="+compiler)
	run(charter, "with", "-strict", "env", "PROBE_LIMIT=1", "go", "test", "-count=1", "./probe")

	if os.Getenv("FRESH_DEPLOY") == "1" {
		deploy := []string{charter, "deploy", "production"}
		if os.Getenv("CLOUDFLARE_API_TOKEN") == "" {
			deploy = append([]string{"fnox", "exec", "--"}, deploy...)
		}
		url := strings.TrimSpace(lastLine(run(deploy...)))
		t.Logf("deployed: %s", url)
		probe := exec.Command("go", "test", "-count=1", "./probe")
		probe.Dir, probe.Env = app, append(env, "BASE="+url)
		if out, err := probe.CombinedOutput(); err != nil {
			t.Errorf("the deployed Worker: %v\n%s", err, out)
		}
	}
}

// lastLine is s's last non-empty line.
func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return lines[len(lines)-1]
}
