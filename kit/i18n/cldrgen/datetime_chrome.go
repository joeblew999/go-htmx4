package cldrgen

import (
	"fmt"
	"regexp"
	"strings"
)

// chromeTrims returns the zones whose zoneStrings Chromium's ICU data filter removes entirely, and those
// whose exemplar city it removes: the "-/zoneStrings/<zone>[/ec]" rules of the zone_tree category in
// chromium/deps/icu filters/common.json at Config.ChromiumICU (zone ids use ICU's ":" separator there).
// Chrome and Cloudflare Workers ship that data, so ICU there derives exemplar cities from the zone id
// (America/St_Johns → "St Johns"); cldrgen does the same to match their Intl output.
func (g *gen) chromeTrims() (all, city map[string]bool, err error) {
	b, err := g.csrc.raw("filters/common.json")
	if err != nil {
		return nil, nil, fmt.Errorf("chromium icu filter: %w", err)
	}
	all, city = map[string]bool{}, map[string]bool{}
	for _, m := range zoneRule.FindAllStringSubmatch(string(b), -1) {
		id := strings.ReplaceAll(m[1], ":", "/")
		if base, ok := strings.CutSuffix(id, "/ec"); ok {
			city[base] = true
		} else {
			all[id] = true
		}
	}
	if len(all) == 0 {
		return nil, nil, fmt.Errorf("chromium icu filter at %s has no zoneStrings rules", g.cfg.ChromiumICU)
	}
	return all, city, nil
}

var zoneRule = regexp.MustCompile(`"-/zoneStrings/([^"]+)"`)
