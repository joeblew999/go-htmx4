package cldrgen

import (
	"encoding/xml"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/joeblew999/go-htmx4/kit/i18n"
)

// loadDateTimeSupplemental fills Data.TimeZones from cldr-bcp47 timezone.json (ids, aliases, canonical
// ids), supplemental metaZones.json (metazone history and golden zones) and primaryZones.json.
func (g *gen) loadDateTimeSupplemental(d *i18n.Data) error {
	bcp, err := g.src.read("cldr-bcp47/bcp47/timezone.json")
	if err != nil {
		return err
	}
	types := mapAt(bcp, "keyword", "u", "tz")
	type tzType struct {
		aliases   []string
		preferred string
		region    string
	}
	byShort := map[string]*tzType{}
	for short, v := range types {
		m, ok := v.(obj)
		if !ok {
			continue
		}
		t := &tzType{aliases: strings.Fields(str(m, "_alias")), preferred: str(m, "_preferred")}
		if len(t.aliases) == 0 {
			continue
		}
		if len(short) >= 5 && !strings.HasPrefix(short, "utc") {
			t.region = strings.ToUpper(short[:2])
		}
		if r, ok := zoneRegionExceptions[t.aliases[0]]; ok {
			t.region = r
		}
		byShort[short] = t
	}
	canonicalOf := func(short string) *tzType {
		t := byShort[short]
		for i := 0; i < 5 && t != nil && t.preferred != ""; i++ {
			if p := byShort[t.preferred]; p != nil {
				t = p
			} else {
				break
			}
		}
		return t
	}

	mzData, err := g.src.read("cldr-core/supplemental/metaZones.json")
	if err != nil {
		return err
	}
	spans := map[string][]i18n.MetaZoneSpan{}
	var walk func(prefix string, node any) error
	walk = func(prefix string, node any) error {
		switch n := node.(type) {
		case obj:
			for _, k := range sortedKeys(n) {
				p := k
				if prefix != "" {
					p = prefix + "/" + k
				}
				if err := walk(p, n[k]); err != nil {
					return err
				}
			}
		case []any:
			for _, e := range n {
				u := mapAt(e, "usesMetazone")
				// ICU (metaZones.txt, ZoneMeta) bounds open spans at 1970-01-01 00:00 and 9999-12-31 23:59.
				span := i18n.MetaZoneSpan{MetaZone: str(u, "_mzone"), From: 0, To: 253402300740}
				if s := str(u, "_from"); s != "" {
					t, err := time.Parse("2006-01-02 15:04", s)
					if err != nil {
						return fmt.Errorf("metaZones %s: %w", prefix, err)
					}
					span.From = t.Unix()
				}
				if s := str(u, "_to"); s != "" {
					t, err := time.Parse("2006-01-02 15:04", s)
					if err != nil {
						return fmt.Errorf("metaZones %s: %w", prefix, err)
					}
					span.To = t.Unix()
				}
				spans[prefix] = append(spans[prefix], span)
			}
		}
		return nil
	}
	if err := walk("", mapAt(mzData, "supplemental", "metaZones", "metazoneInfo", "timezone")); err != nil {
		return err
	}
	golden := map[string][]i18n.RegionZone{}
	mapList, _ := get(mzData, "supplemental", "metaZones", "metazones").([]any)
	for _, e := range mapList {
		m := mapAt(e, "mapZone")
		golden[str(m, "_other")] = append(golden[str(m, "_other")], i18n.RegionZone{Region: str(m, "_territory"), Zone: str(m, "_type")})
	}
	for _, id := range sortedKeys(golden) {
		zs := golden[id]
		slices.SortFunc(zs, func(a, b i18n.RegionZone) int { return strings.Compare(a.Region, b.Region) })
		d.TimeZones.MetaZones = append(d.TimeZones.MetaZones, i18n.MetaZoneInfo{ID: id, Zones: zs})
	}

	pzData, err := g.src.read("cldr-core/supplemental/primaryZones.json")
	if err != nil {
		return err
	}
	primary := map[string]string{}
	for region, v := range mapAt(pzData, "supplemental", "primaryZones") {
		primary[region], _ = v.(string)
	}

	// zones per region among canonical (non-deprecated) zones
	perRegion := map[string]int{}
	for _, t := range byShort {
		if t.preferred == "" && t.region != "" && t.aliases[0] != "Etc/Unknown" {
			perRegion[t.region]++
		}
	}
	seen := map[string]bool{}
	for _, short := range sortedKeys(byShort) {
		t := byShort[short]
		canon := canonicalOf(short)
		cid := canon.aliases[0]
		for _, alias := range t.aliases {
			if alias == "Etc/Unknown" || seen[strings.ToLower(alias)] {
				continue
			}
			seen[strings.ToLower(alias)] = true
			zi := i18n.ZoneInfo{ID: alias}
			if alias != cid {
				zi.Canonical = cid
			} else {
				zi.Region = canon.region
				zi.Primary = canon.region != "" && (perRegion[canon.region] == 1 || primary[canon.region] == cid)
				zi.MetaZones = spans[cid]
			}
			d.TimeZones.Zones = append(d.TimeZones.Zones, zi)
		}
	}
	slices.SortFunc(d.TimeZones.Zones, func(a, b i18n.ZoneInfo) int {
		return strings.Compare(strings.ToLower(a.ID), strings.ToLower(b.ID))
	})
	return nil
}

// zoneRegionExceptions are canonical zones whose IANA zone.tab region differs from the country code in
// their BCP 47 short id (which ICU's zoneinfo64 Regions follow).
var zoneRegionExceptions = map[string]string{
	"America/Curacao":       "CW",
	"America/Marigot":       "MF",
	"America/St_Barthelemy": "BL",
	"Asia/Gaza":             "PS",
	"Asia/Hebron":           "PS",
	"Asia/Jerusalem":        "IL",
	"Europe/Mariehamn":      "AX",
	"Pacific/Johnston":      "UM",
}

// buildDateTime fills LocaleData.DateTime from cldr-dates-full (ca-gregorian, dateFields,
// timeZoneNames), supplemental dayPeriods, localenames territories and, for the pattern generator's
// availableFormats load order, CLDR XML.
func (g *gen) buildDateTime(ld *i18n.LocaleData) error {
	dt := &ld.DateTime
	cal, err := g.localeFile("cldr-dates-full", "ca-gregorian.json", ld.DataID)
	if err != nil {
		return err
	}
	greg := mapAt(cal, "dates", "calendars", "gregorian")
	if greg == nil {
		return errors.New("no gregorian calendar")
	}
	contexts := []string{"format", "stand-alone"}
	for ci, ctx := range contexts {
		for wi, w := range []string{"abbreviated", "wide", "narrow"} {
			for m := 1; m <= 12; m++ {
				dt.Months[ci][wi][m-1] = str(greg, "months", ctx, w, strconv.Itoa(m))
			}
		}
		for wi, w := range []string{"abbreviated", "wide", "narrow", "short"} {
			for di, day := range []string{"sun", "mon", "tue", "wed", "thu", "fri", "sat"} {
				dt.Days[ci][wi][di] = str(greg, "days", ctx, w, day)
			}
		}
	}
	for wi, w := range []string{"eraAbbr", "eraNames", "eraNarrow"} {
		dt.Eras[wi][0] = str(greg, "eras", w, "0")
		dt.Eras[wi][1] = str(greg, "eras", w, "1")
	}
	for wi, w := range []string{"abbreviated", "wide", "narrow"} {
		dt.AmPm[wi][0] = str(greg, "dayPeriods", "format", w, "am")
		dt.AmPm[wi][1] = str(greg, "dayPeriods", "format", w, "pm")
		for pi, p := range dayPeriodKeys {
			dt.DayPeriods[wi][pi] = str(greg, "dayPeriods", "format", w, p)
		}
	}
	for pi := range dayPeriodKeys {
		for wi := 1; wi < 3; wi++ {
			if dt.DayPeriods[wi][pi] == "" {
				dt.DayPeriods[wi][pi] = dt.DayPeriods[0][pi]
			}
		}
	}
	lengths := []string{"full", "long", "medium", "short"}
	for i, l := range lengths {
		dt.DateFormats[i] = str(greg, "dateFormats", l)
		dt.TimeFormats[i] = str(greg, "timeFormats", l)
		dt.DateTimeFormats[i] = str(greg, "dateTimeFormats", l)
		dt.AtTimeFormats[i] = str(greg, "dateTimeFormats-atTime", "standard", l)
	}

	// pattern generator
	avail := mapAt(greg, "dateTimeFormats", "availableFormats")
	order, err := g.availableFormatsOrder(ld.DataID)
	if err != nil {
		return err
	}
	var pairs [][2]string
	for _, key := range order {
		v := str(avail, key)
		pairs = append(pairs, [2]string{key, v})
	}
	std := append(slices.Clone(dt.TimeFormats[:]), dt.DateFormats[:]...)
	dt.Skeletons = i18n.BuildSkeletonPatterns(std, pairs)
	for i, name := range appendItemKeys {
		if name != "" {
			dt.AppendItems[i] = str(greg, "dateTimeFormats", "appendItems", name)
		}
	}
	fields, err := g.localeFile("cldr-dates-full", "dateFields.json", ld.DataID)
	if err != nil {
		return err
	}
	for i, name := range fieldNameKeys {
		if name != "" {
			dt.FieldNames[i] = str(fields, "dates", "fields", name, "displayName")
		}
	}

	// intervals
	itv := mapAt(greg, "dateTimeFormats", "intervalFormats")
	dt.IntervalFallback = str(itv, "intervalFormatFallback")
	for _, skel := range sortedKeys(itv) {
		m, ok := itv[skel].(obj)
		if !ok || strings.Contains(skel, "-alt-") {
			continue
		}
		var patterns [7]string
		for _, letter := range sortedKeys(m) {
			idx := intervalLetterIndex(letter)
			if idx < 0 || patterns[idx] != "" {
				continue
			}
			patterns[idx], _ = m[letter].(string)
		}
		list := patterns[:]
		for len(list) > 0 && list[len(list)-1] == "" {
			list = list[:len(list)-1]
		}
		dt.Intervals = append(dt.Intervals, i18n.IntervalFormats{Skeleton: skel, PatternList: strings.Join(list, "\x1f")})
	}
	slices.SortFunc(dt.Intervals, func(a, b i18n.IntervalFormats) int { return strings.Compare(a.Skeleton, b.Skeleton) })

	if err := g.dayPeriodRules(ld); err != nil {
		return err
	}
	return g.buildZoneStrings(ld)
}

var dayPeriodKeys = []string{"midnight", "noon", "morning1", "afternoon1", "evening1", "night1", "morning2", "afternoon2", "evening2", "night2"}

// appendItemKeys are ICU's CLDR_FIELD_APPEND names by pattern generator field.
var appendItemKeys = [16]string{"Era", "Year", "Quarter", "Month", "Week", "", "Day-Of-Week", "", "", "Day", "", "Hour", "Minute", "Second", "", "Timezone"}

// fieldNameKeys are ICU's CLDR_FIELD_NAME keys by pattern generator field.
var fieldNameKeys = [16]string{"era", "year", "quarter", "month", "week", "weekOfMonth", "weekday", "dayOfYear", "weekdayOfMonth", "day", "dayperiod", "hour", "minute", "second", "", "zone"}

// intervalLetterIndex is DateIntervalInfo's validateAndProcessPatternLetter mapped to interval indexes.
func intervalLetterIndex(letter string) int {
	switch letter {
	case "G":
		return 0
	case "y":
		return 1
	case "M":
		return 2
	case "d":
		return 3
	case "a", "B":
		return 4
	case "h", "H":
		return 5
	case "m":
		return 6
	}
	return -1
}

// availableFormatsOrder returns the availableFormats keys in ICU's load order: each level of the parent
// chain (child first, root last) contributes its own keys sorted, skipping alt variants, drafts ICU
// excludes and inheritance markers; count variants are one key.
func (g *gen) availableFormatsOrder(dataID string) ([]string, error) {
	chain := append(g.chain(dataID), "root")
	var order []string
	seen := map[string]bool{}
	for _, id := range chain {
		file := "common/main/" + strings.ReplaceAll(id, "-", "_") + ".xml"
		b, err := g.xsrc.raw(file)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		var doc struct {
			Calendars []struct {
				Type  string `xml:"type,attr"`
				Items []struct {
					ID    string `xml:"id,attr"`
					Alt   string `xml:"alt,attr"`
					Draft string `xml:"draft,attr"`
					Value string `xml:",chardata"`
				} `xml:"dateTimeFormats>availableFormats>dateFormatItem"`
			} `xml:"dates>calendars>calendar"`
		}
		if err := xml.Unmarshal(b, &doc); err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		var keys []string
		for _, c := range doc.Calendars {
			if c.Type != "gregorian" {
				continue
			}
			for _, it := range c.Items {
				if it.Alt != "" || it.Draft == "unconfirmed" || it.Draft == "provisional" || it.Value == inheritMarker {
					continue
				}
				if !seen[it.ID] {
					seen[it.ID] = true
					keys = append(keys, it.ID)
				}
			}
		}
		slices.Sort(keys)
		order = append(order, keys...)
	}
	return order, nil
}

// dayPeriodRules resolves supplemental dayPeriodRuleSet the way ICU's DayPeriodRules does: by locale
// id, truncating subtags.
func (g *gen) dayPeriodRules(ld *i18n.LocaleData) error {
	dp, err := g.src.read("cldr-core/supplemental/dayPeriods.json")
	if err != nil {
		return err
	}
	sets := mapAt(dp, "supplemental", "dayPeriodRuleSet")
	r := &ld.DateTime.DayPeriodRules
	for i := range r.Hours {
		r.Hours[i] = -1
	}
	id := ld.ID
	var rules obj
	for id != "" {
		if rules = mapAt(sets, id); rules != nil {
			break
		}
		k := strings.LastIndexByte(id, '-')
		if k < 0 {
			break
		}
		id = id[:k]
	}
	if rules == nil {
		return nil
	}
	index := map[string]int8{"am": 10, "pm": 11}
	for i, k := range dayPeriodKeys {
		index[k] = int8(i)
	}
	hour := func(s string) int {
		h, _ := strconv.Atoi(strings.SplitN(s, ":", 2)[0])
		return h
	}
	for _, name := range sortedKeys(rules) {
		p, ok := index[name]
		if !ok {
			continue
		}
		rule := mapAt(rules, name)
		if at := str(rule, "_at"); at != "" {
			switch name {
			case "midnight":
				r.Midnight = true
			case "noon":
				r.Noon = true
			}
			continue
		}
		from, before := hour(str(rule, "_from")), hour(str(rule, "_before"))
		for i := from; i != before; i++ {
			if i == 24 {
				i = 0
				if i == before {
					break
				}
			}
			r.Hours[i] = p
		}
	}
	return nil
}

// buildZoneStrings fills the locale's time zone names. Zone-specific strings follow Chromium's ICU data
// filter (chromeTrimmedZones), which Chrome and Cloudflare Workers ship: Intl output then matches theirs.
func (g *gen) buildZoneStrings(ld *i18n.LocaleData) error {
	// cldr-json locale files are already resolved and honour CLDR's no-inheritance marker (en-001 and
	// pt-PT drop the short American and Brasília metazone names); merging parents again would restore them.
	rel := "cldr-dates-full/main/" + ld.DataID + "/timeZoneNames.json"
	tznFile, err := g.src.read(rel)
	if err != nil {
		return err
	}
	names := mapAt(tznFile, "main", ld.DataID, "dates", "timeZoneNames")
	if names == nil {
		return fmt.Errorf("%s: no timeZoneNames", rel)
	}
	zs := &ld.DateTime.Zone
	zs.HourFormat = str(names, "hourFormat")
	zs.GMTFormat = str(names, "gmtFormat")
	zs.GMTZeroFormat = str(names, "gmtZeroFormat")
	zs.RegionFormat = str(names, "regionFormat")
	zs.FallbackFormat = str(names, "fallbackFormat")
	zoneNames := func(id string, m obj, city bool) i18n.ZoneNames {
		fields := []string{str(m, "long", "generic"), str(m, "long", "standard"), str(m, "long", "daylight"),
			str(m, "short", "generic"), str(m, "short", "standard"), str(m, "short", "daylight"), ""}
		if city {
			fields[6] = str(m, "exemplarCity")
		}
		for len(fields) > 0 && fields[len(fields)-1] == "" {
			fields = fields[:len(fields)-1]
		}
		return i18n.ZoneNames{ID: id, Names: strings.Join(fields, "\x1f")}
	}
	mz := mapAt(names, "metazone")
	for _, id := range sortedKeys(mz) {
		if n := zoneNames(id, mapAt(mz, id), false); n.Names != "" {
			zs.MetaZones = append(zs.MetaZones, n)
		}
	}
	trimmed, trimmedCity, err := g.chromeTrims()
	if err != nil {
		return err
	}
	var walk func(prefix string, node obj)
	walk = func(prefix string, node obj) {
		for _, k := range sortedKeys(node) {
			child, ok := node[k].(obj)
			if !ok {
				continue
			}
			id := k
			if prefix != "" {
				id = prefix + "/" + k
			}
			if str(child, "_type") != "zone" {
				walk(id, child)
				continue
			}
			if trimmed[id] {
				continue
			}
			if n := zoneNames(id, child, !trimmedCity[id]); n.Names != "" {
				zs.Zones = append(zs.Zones, n)
			}
		}
	}
	walk("", mapAt(names, "zone"))
	slices.SortFunc(zs.Zones, func(a, b i18n.ZoneNames) int { return strings.Compare(a.ID, b.ID) })

	terr, err := g.localeFile("cldr-localenames-full", "territories.json", ld.DataID)
	if err != nil {
		return err
	}
	T := mapAt(terr, "localeDisplayNames", "territories")
	regions := map[string]bool{}
	for _, z := range g.data.TimeZones.Zones {
		if z.Region != "" {
			regions[z.Region] = true
		}
	}
	var entries []string
	for _, code := range sortedKeys(regions) {
		v := str(T, code)
		// Chromium's ICU region data (chromium/deps/icu source/data/region/*.txt) names these regions
		// with CLDR's alternates; zone location names in Chrome and Workers show those.
		if alt := chromeRegionAlt[code]; alt != "" && str(T, code+alt) != "" {
			v = str(T, code+alt)
		}
		if v != "" {
			entries = append(entries, code+"\x1e"+v)
		}
	}
	zs.RegionNames = strings.Join(entries, "\x1f")
	return nil
}

// chromeRegionAlt are the CLDR territory alternates Chromium's ICU data uses as the region name.
var chromeRegionAlt = map[string]string{"HK": "-alt-short", "MO": "-alt-short", "PS": "-alt-short", "FK": "-alt-variant"}
