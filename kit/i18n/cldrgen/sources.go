package cldrgen

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Reference data pins. Every upstream input of kit/i18n's generated tables (kit/i18n/cldr) and of the
// conformance goldens (kit/i18n/testdata/golden) is pinned here or in mise.toml; kit/i18n/README.md
// ("Reference data") explains how to move them. `mise run i18n:verify` proves the committed data comes from
// exactly these versions.
const (
	// DefaultTag is the cldr-json release the tables are generated from
	// (https://github.com/unicode-org/cldr-json/tree/48.2.1). It matches the CLDR major of ChromiumICU.
	DefaultTag = "48.2.1"

	// DefaultCLDRTag is the unicode-org/cldr release matching DefaultTag, for the few things cldr-json's
	// resolved JSON leaves out: root's explicit arab/arabext number symbols, draft status, and ICU's
	// availableFormats order.
	DefaultCLDRTag = "release-48-2"

	// Workerd is the workerd release whose Intl the goldens record (`mise run i18n:golden`). mise.toml's
	// [tools] workerd must be the same version (TestReferencePins).
	Workerd = "1.20260911.1"

	// ChromiumICU is the chromium/deps/icu commit that workerd release is built with (ICU 78.2): workerd's
	// build/deps/gen/deps.MODULE.bazel, repository com_googlesource_chromium_icu. Chrome and Workers ship its
	// filtered ICU data; the generator reads its data filter (filters/common.json) so Intl output matches
	// theirs, and the `known` conformance entries cite its locale files as evidence.
	ChromiumICU = "8cc91d9b6ab9991802fd208ee03a69714fd0251c"
)

const (
	workerdBase  = "https://raw.githubusercontent.com/cloudflare/workerd/"
	cldrJSONBase = "https://raw.githubusercontent.com/unicode-org/cldr-json/"
	cldrXMLBase  = "https://raw.githubusercontent.com/unicode-org/cldr/"
	chromiumBase = "https://chromium.googlesource.com/chromium/deps/icu/+/"
)

// CacheRoot is where upstream files are cached, one directory per source and version:
// <user cache dir>/go-htmx4 (macOS ~/Library/Caches/go-htmx4, Linux ~/.cache/go-htmx4). The environment
// variable I18N_CACHE overrides it (e.g. a CI cache directory).
func CacheRoot() (string, error) {
	if dir := os.Getenv("I18N_CACHE"); dir != "" {
		return dir, nil
	}
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "go-htmx4"), nil
}

func chromiumSource(commit, cache string) *source {
	return &source{base: chromiumBase + commit, cache: cache, query: "?format=TEXT", base64: true}
}

// Reference returns a pinned upstream file, fetched once into CacheRoot. kind is "cldr-json" (a path under
// cldr-json's cldr-json/ directory at DefaultTag, e.g. "cldr-dates-full/main/es/ca-gregorian.json"),
// "cldr" (unicode-org/cldr at DefaultCLDRTag, e.g. "common/main/de.xml") or "chromium-icu"
// (chromium/deps/icu at ChromiumICU, e.g. "source/data/locales/es.txt"). Verification tests use it to check
// evidence against the same bytes the generator reads.
func Reference(kind, path string) ([]byte, error) {
	root, err := CacheRoot()
	if err != nil {
		return nil, err
	}
	var s *source
	switch kind {
	case "cldr-json":
		s = &source{base: cldrJSONBase + DefaultTag + "/cldr-json", cache: filepath.Join(root, "cldr-json", DefaultTag)}
	case "cldr":
		s = &source{base: cldrXMLBase + DefaultCLDRTag, cache: filepath.Join(root, "cldr-json", "cldr-"+DefaultCLDRTag)}
	case "chromium-icu":
		s = chromiumSource(ChromiumICU, filepath.Join(root, "chromium-icu", ChromiumICU))
	default:
		return nil, fmt.Errorf("cldrgen: unknown reference source %q", kind)
	}
	return s.raw(path)
}

// WorkerdICU is the ICU a workerd release is built with.
type WorkerdICU struct {
	Release     string // e.g. "1.20260911.1"
	ChromiumICU string // chromium/deps/icu commit
	ICUVersion  string // e.g. "78.2.0.0"
	CLDRVersion string // CLDR major in that ICU's data, e.g. "48"
}

var (
	chromiumICURepo = regexp.MustCompile(`(?s)name = "com_googlesource_chromium_icu",.*?commit = "([0-9a-f]{40})"`)
	icuVersion      = regexp.MustCompile(`ICUVersion\{"([^"]+)"\}`)
	icuCLDRVersion  = regexp.MustCompile(`CLDRVersion\{"([^"]+)"\}`)
)

// LookupWorkerdICU reads which Chromium ICU commit a workerd release builds with (repository
// com_googlesource_chromium_icu in the release tag's build/deps/gen/deps.MODULE.bazel) and that commit's ICU and
// CLDR versions (source/data/misc/icuver.txt). Files are cached in CacheRoot: tags and commits don't change.
func LookupWorkerdICU(release string) (WorkerdICU, error) {
	root, err := CacheRoot()
	if err != nil {
		return WorkerdICU{}, err
	}
	w := WorkerdICU{Release: release}
	deps := &source{base: workerdBase + "v" + release, cache: filepath.Join(root, "workerd", release)}
	b, err := deps.raw("build/deps/gen/deps.MODULE.bazel")
	if err != nil {
		return w, fmt.Errorf("workerd %s deps: %w", release, err)
	}
	m := chromiumICURepo.FindSubmatch(b)
	if m == nil {
		return w, fmt.Errorf("workerd %s: no com_googlesource_chromium_icu commit in build/deps/gen/deps.MODULE.bazel", release)
	}
	w.ChromiumICU = string(m[1])
	icu := chromiumSource(w.ChromiumICU, filepath.Join(root, "chromium-icu", w.ChromiumICU))
	ver, err := icu.raw("source/data/misc/icuver.txt")
	if err != nil {
		return w, fmt.Errorf("chromium icu %s icuver.txt: %w", w.ChromiumICU, err)
	}
	if m := icuVersion.FindSubmatch(ver); m != nil {
		w.ICUVersion = string(m[1])
	}
	if m := icuCLDRVersion.FindSubmatch(ver); m != nil {
		w.CLDRVersion = string(m[1])
	}
	return w, nil
}

// CheckPins compares the pins with what the pinned workerd release really uses; each problem is one line.
func CheckPins() ([]string, error) {
	w, err := LookupWorkerdICU(Workerd)
	if err != nil {
		return nil, err
	}
	var problems []string
	if w.ChromiumICU != ChromiumICU {
		problems = append(problems, fmt.Sprintf("ChromiumICU is %s, but workerd %s builds with %s (ICU %s)", ChromiumICU, Workerd, w.ChromiumICU, w.ICUVersion))
	}
	if major, _, _ := strings.Cut(DefaultTag, "."); w.CLDRVersion != "" && major != w.CLDRVersion {
		problems = append(problems, fmt.Sprintf("DefaultTag is cldr-json %s, but workerd %s's ICU %s has CLDR %s", DefaultTag, Workerd, w.ICUVersion, w.CLDRVersion))
	}
	return problems, nil
}
