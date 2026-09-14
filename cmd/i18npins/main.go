// Command i18npins checks and moves kit/i18n's reference data pins (kit/i18n/README.md, "Moving a pin").
//
//	go run ./cmd/i18npins                               # pins consistent with the pinned workerd release?
//	go run ./cmd/i18npins -latest [-strict]             # what the newest workerd release uses (-strict: fail if its ICU moved)
//	go run ./cmd/i18npins -workerd 1.20261001.0 -write  # move the workerd and Chromium ICU pins
//
// -write edits mise.toml (workerd) and kit/i18n/cldrgen/sources.go (Workerd, ChromiumICU); then follow the README:
// mise install, mise run i18n:golden, mise run i18n:generate, go test ./kit/i18n, mise run i18n:verify.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/joeblew999/go-htmx4/kit/i18n/cldrgen"
)

func main() {
	release := flag.String("workerd", cldrgen.Workerd, "workerd release to inspect (without the v)")
	latest := flag.Bool("latest", false, "inspect the newest workerd release on GitHub")
	write := flag.Bool("write", false, "move the pins to -workerd / -latest")
	strict := flag.Bool("strict", false, "exit 1 when the inspected release's Chromium ICU differs from the pin (a scheduled \"time to move the pins\" check)")
	flag.Parse()

	if *latest {
		r, err := latestWorkerd()
		if err != nil {
			fail(err)
		}
		*release = r
	}
	w, err := cldrgen.LookupWorkerdICU(*release)
	if err != nil {
		fail(err)
	}
	fmt.Printf("workerd %s: Chromium ICU %s (ICU %s, CLDR %s)\n", w.Release, w.ChromiumICU, w.ICUVersion, w.CLDRVersion)
	fmt.Printf("pinned:  workerd %s, Chromium ICU %s, cldr-json %s\n", cldrgen.Workerd, cldrgen.ChromiumICU, cldrgen.DefaultTag)

	if *write {
		if err := edit("mise.toml", `(?m)^workerd = "[^"]+"`, `workerd = "`+w.Release+`"`); err != nil {
			fail(err)
		}
		if err := edit("kit/i18n/cldrgen/sources.go", `Workerd = "[^"]+"`, `Workerd = "`+w.Release+`"`); err != nil {
			fail(err)
		}
		if err := edit("kit/i18n/cldrgen/sources.go", `ChromiumICU = "[0-9a-f]{40}"`, `ChromiumICU = "`+w.ChromiumICU+`"`); err != nil {
			fail(err)
		}
		fmt.Println("✓ pins moved. Next: mise install; mise run i18n:golden; mise run i18n:generate; go test ./kit/i18n; mise run i18n:verify")
		if major, _, _ := strings.Cut(cldrgen.DefaultTag, "."); major != w.CLDRVersion {
			fmt.Printf("! this ICU has CLDR %s: move DefaultTag/DefaultCLDRTag too (README, \"CLDR\")\n", w.CLDRVersion)
		}
		return
	}
	if *release != cldrgen.Workerd {
		if w.ChromiumICU != cldrgen.ChromiumICU {
			fmt.Printf("! workerd %s moved to Chromium ICU %s: move the pins (-workerd %s -write)\n", w.Release, w.ChromiumICU, w.Release)
			if *strict {
				os.Exit(1)
			}
		} else {
			fmt.Printf("✓ workerd %s uses the pinned Chromium ICU\n", w.Release)
		}
		return
	}
	problems, err := cldrgen.CheckPins()
	if err != nil {
		fail(err)
	}
	for _, p := range problems {
		fmt.Println("✗", p)
	}
	if len(problems) > 0 {
		os.Exit(1)
	}
	fmt.Println("✓ pins match the pinned workerd release")
}

func latestWorkerd() (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, _ := http.NewRequest("GET", "https://api.github.com/repos/cloudflare/workerd/releases/latest", nil)
	if tok := os.Getenv("GITHUB_TOKEN"); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	var rel struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(res.Body).Decode(&rel); err != nil {
		return "", err
	}
	if rel.TagName == "" {
		return "", fmt.Errorf("GitHub returned no latest workerd release (%s)", res.Status)
	}
	return strings.TrimPrefix(rel.TagName, "v"), nil
}

func edit(path, pattern, replacement string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	re := regexp.MustCompile(pattern)
	if n := len(re.FindAllIndex(b, -1)); n != 1 {
		return fmt.Errorf("%s: %d matches for %s, want 1", path, n, pattern)
	}
	return os.WriteFile(path, re.ReplaceAll(b, []byte(replacement)), 0o644)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "i18npins:", err)
	os.Exit(1)
}
