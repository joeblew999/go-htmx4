package main

import (
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// message is a translated message with arguments: a typed intl.Msg, not a component call.
var message = regexp.MustCompile(`\{\s*locales\.[A-Z][A-Za-z0-9]*\(`)

// TestGsxSmells runs gsx's own smell finder (its gsx skill's scripts/find-smells.sh) over the views: every hit is
// gsx written the non-idiomatic way. The one shape it can't tell apart is allowed: a message with arguments alone on
// a line ({ locales.PreferencesTimeZoneAuto(zone) }), which gsx renders like gsx's own Translation keys pattern.
func TestGsxSmells(t *testing.T) {
	dir, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/gsxhq/gsx").Output()
	if err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(strings.TrimSpace(string(dir)), "skills", "gsx", "scripts", "find-smells.sh")
	out, err := exec.Command("bash", script, "views").Output()
	if err != nil {
		t.Fatal(err)
	}
	var section string
	for _, line := range strings.Split(string(out), "\n") {
		switch {
		case line == "":
		case strings.HasPrefix(line, "== "):
			section = strings.TrimPrefix(line, "== ")
		case strings.HasPrefix(section, "component called as a Go function") && message.MatchString(line):
		default:
			t.Errorf("%s: %s", section, line)
		}
	}
}
