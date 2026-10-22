//go:build !js

package app

import (
	"os"
	"regexp"
	"slices"
	"testing"

	"github.com/joeblew999/charter/go/lib/project"
)

// TestCharterToml: charter.toml is where the Worker's facts live (deploys and local workerd read it). The copies other
// files still keep, and the bindings the code names, agree with it.
func TestCharterToml(t *testing.T) {
	p, err := project.Load("..")
	if err != nil {
		t.Fatal(err)
	}
	read := func(path string) string {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	value := func(path, pattern string) string {
		m := regexp.MustCompile(pattern).FindStringSubmatch(read(path))
		if m == nil {
			t.Fatalf("%s: no %s", path, pattern)
		}
		return m[1]
	}
	for _, c := range []struct{ what, got, want string }{
		{"mise.toml's APP_NAME", value("../mise.toml", `(?m)^APP_NAME = "(.*)"`), p.Name},
		{"mise.toml's APP_DOMAIN", value("../mise.toml", `(?m)^APP_DOMAIN = "(.*)"`), p.Production.Domain},
		{"mise.toml's APP_REPO", value("../mise.toml", `(?m)^APP_REPO = "(.*)"`), p.Repo},
		{"mise.toml's CF_SUBDOMAIN", value("../mise.toml", `(?m)^CF_SUBDOMAIN = "(.*)"`), p.Subdomain},
		{"production's APP_DOMAIN var", p.Production.Vars["APP_DOMAIN"], p.Production.Domain},
		{"mise run dev's APP_DEV_PORT", value("../tasks/app.toml", `APP_DEV_PORT = "(\d+)"`), p.LocalPort()},
	} {
		if c.got != c.want {
			t.Errorf("%s is %q, charter.toml says %q", c.what, c.got, c.want)
		}
	}
	// The bindings board.go, store.go and platform.go open, and the Worker's entry routes /live/ to.
	if _, ok := p.RateLimit("WRITES"); !ok {
		t.Error("charter.toml has no WRITES rate limit (board.go's)")
	}
	if !slices.ContainsFunc(p.D1, func(d project.D1) bool { return d.Binding == "DB" }) {
		t.Error("charter.toml has no DB database (store.go's)")
	}
	if !slices.ContainsFunc(p.DurableObjects, func(d project.DurableObject) bool { return d.Binding == "ROOM" && d.Class == "Room" }) {
		t.Error("charter.toml has no ROOM Durable Object of class Room (platform.go's, worker/index.mjs's)")
	}
}
