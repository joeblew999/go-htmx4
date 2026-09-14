package cldrgen

import (
	"fmt"
	"os"
	"path/filepath"
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
