package i18n

import "strings"

// This file ports ICU's DateTimePatternGenerator (dtptngen.cpp, ICU 78): skeleton canonicalization,
// the distance between skeletons, best-pattern selection with field-type and field-length adjustment,
// appended fields and date + time composition. It works on generated pattern maps
// (DateTimeData.Skeletons), which [BuildSkeletonPatterns] builds at generate time.

// Pattern generator fields (UDATPG_*_FIELD).
const (
	pgEra = iota
	pgYear
	pgQuarter
	pgMonth
	pgWeekOfYear
	pgWeekOfMonth
	pgWeekday
	pgDayOfYear
	pgDayOfWeekInMonth
	pgDay
	pgDayPeriod
	pgHour
	pgMinute
	pgSecond
	pgFractionalSecond
	pgZone
	pgFieldCount
)

const (
	dtNarrow  = -0x101
	dtShorter = -0x102
	dtShort   = -0x103
	dtLong    = -0x104
	dtNumeric = 0x100
	dtDelta   = 0x10

	extraField   = 0x10000
	missingField = 0x1000

	pgMatchHourFieldLength = 1 << pgHour
	pgMatchNoOptions       = 0

	pgFlagFixFractionalSeconds = 1
	pgFlagSkeletonUsesCapJ     = 2

	pgDateMask = (1 << pgDayPeriod) - 1
	pgTimeMask = (1 << pgFieldCount) - 1 - pgDateMask

	pgFractionalMask         = 1 << pgFractionalSecond
	pgSecondAndFractionalMsk = 1<<pgSecond | 1<<pgFractionalSecond
)

type dtTypeElem struct {
	ch     byte
	field  int8
	typ    int16
	minLen int8
}

// dtTypes is ICU's dtTypes table (without the unused weight column).
var dtTypes = [...]dtTypeElem{
	{'G', pgEra, dtShort, 1}, {'G', pgEra, dtLong, 4}, {'G', pgEra, dtNarrow, 5},
	{'y', pgYear, dtNumeric, 1}, {'Y', pgYear, dtNumeric + dtDelta, 1}, {'u', pgYear, dtNumeric + 2*dtDelta, 1},
	{'r', pgYear, dtNumeric + 3*dtDelta, 1}, {'U', pgYear, dtShort, 1}, {'U', pgYear, dtLong, 4}, {'U', pgYear, dtNarrow, 5},
	{'Q', pgQuarter, dtNumeric, 1}, {'Q', pgQuarter, dtShort, 3}, {'Q', pgQuarter, dtLong, 4}, {'Q', pgQuarter, dtNarrow, 5},
	{'q', pgQuarter, dtNumeric + dtDelta, 1}, {'q', pgQuarter, dtShort - dtDelta, 3}, {'q', pgQuarter, dtLong - dtDelta, 4}, {'q', pgQuarter, dtNarrow - dtDelta, 5},
	{'M', pgMonth, dtNumeric, 1}, {'M', pgMonth, dtShort, 3}, {'M', pgMonth, dtLong, 4}, {'M', pgMonth, dtNarrow, 5},
	{'L', pgMonth, dtNumeric + dtDelta, 1}, {'L', pgMonth, dtShort - dtDelta, 3}, {'L', pgMonth, dtLong - dtDelta, 4}, {'L', pgMonth, dtNarrow - dtDelta, 5},
	{'l', pgMonth, dtNumeric + dtDelta, 1},
	{'w', pgWeekOfYear, dtNumeric, 1},
	{'W', pgWeekOfMonth, dtNumeric, 1},
	{'E', pgWeekday, dtShort, 1}, {'E', pgWeekday, dtLong, 4}, {'E', pgWeekday, dtNarrow, 5}, {'E', pgWeekday, dtShorter, 6},
	{'c', pgWeekday, dtNumeric + 2*dtDelta, 1}, {'c', pgWeekday, dtShort - 2*dtDelta, 3}, {'c', pgWeekday, dtLong - 2*dtDelta, 4},
	{'c', pgWeekday, dtNarrow - 2*dtDelta, 5}, {'c', pgWeekday, dtShorter - 2*dtDelta, 6},
	{'e', pgWeekday, dtNumeric + dtDelta, 1}, {'e', pgWeekday, dtShort - dtDelta, 3}, {'e', pgWeekday, dtLong - dtDelta, 4},
	{'e', pgWeekday, dtNarrow - dtDelta, 5}, {'e', pgWeekday, dtShorter - dtDelta, 6},
	{'d', pgDay, dtNumeric, 1}, {'g', pgDay, dtNumeric + dtDelta, 1},
	{'D', pgDayOfYear, dtNumeric, 1},
	{'F', pgDayOfWeekInMonth, dtNumeric, 1},
	{'a', pgDayPeriod, dtShort, 1}, {'a', pgDayPeriod, dtLong, 4}, {'a', pgDayPeriod, dtNarrow, 5},
	{'b', pgDayPeriod, dtShort - dtDelta, 1}, {'b', pgDayPeriod, dtLong - dtDelta, 4}, {'b', pgDayPeriod, dtNarrow - dtDelta, 5},
	{'B', pgDayPeriod, dtShort - 3*dtDelta, 1}, {'B', pgDayPeriod, dtLong - 3*dtDelta, 4}, {'B', pgDayPeriod, dtNarrow - 3*dtDelta, 5},
	{'H', pgHour, dtNumeric + 10*dtDelta, 1}, {'k', pgHour, dtNumeric + 11*dtDelta, 1},
	{'h', pgHour, dtNumeric, 1}, {'K', pgHour, dtNumeric + dtDelta, 1},
	{'J', pgHour, dtNumeric + 5*dtDelta, 1}, {'j', pgHour, dtNumeric + 6*dtDelta, 1}, {'C', pgHour, dtNumeric + 7*dtDelta, 1},
	{'m', pgMinute, dtNumeric, 1},
	{'s', pgSecond, dtNumeric, 1}, {'A', pgSecond, dtNumeric + dtDelta, 1},
	{'S', pgFractionalSecond, dtNumeric, 1},
	{'v', pgZone, dtShort - 2*dtDelta, 1}, {'v', pgZone, dtLong - 2*dtDelta, 4},
	{'z', pgZone, dtShort, 1}, {'z', pgZone, dtLong, 4},
	{'Z', pgZone, dtNarrow - dtDelta, 1}, {'Z', pgZone, dtLong - dtDelta, 4}, {'Z', pgZone, dtShort - dtDelta, 5},
	{'O', pgZone, dtShort - dtDelta, 1}, {'O', pgZone, dtLong - dtDelta, 4},
	{'V', pgZone, dtShort - dtDelta, 1}, {'V', pgZone, dtLong - dtDelta, 2}, {'V', pgZone, dtLong - 1 - dtDelta, 3}, {'V', pgZone, dtLong - 2 - dtDelta, 4},
	{'X', pgZone, dtNarrow - dtDelta, 1}, {'X', pgZone, dtShort - dtDelta, 2}, {'X', pgZone, dtLong - dtDelta, 4},
	{'x', pgZone, dtNarrow - dtDelta, 1}, {'x', pgZone, dtShort - dtDelta, 2}, {'x', pgZone, dtLong - dtDelta, 4},
}

// canonicalItems are the single-letter patterns ICU adds first (GyQMwWEDFdaHmsSv).
const canonicalItems = "GyQMwWEDFdaHmsSv"

func isASCIILetter(c byte) bool { return c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' }

// canonicalIndex is FormatParser::getCanonicalIndex (strict): the dtTypes row for a run of one letter.
func canonicalIndex(field string) int {
	n := len(field)
	if n == 0 || !isASCIILetter(field[0]) {
		return -1
	}
	ch := field[0]
	for i := 1; i < n; i++ {
		if field[i] != ch {
			return -1
		}
	}
	for i := 0; i < len(dtTypes); i++ {
		if dtTypes[i].ch != ch {
			continue
		}
		if i+1 >= len(dtTypes) || dtTypes[i+1].ch != ch {
			return i
		}
		if int(dtTypes[i+1].minLen) <= n {
			continue
		}
		return i
	}
	return -1
}

// patternTokens is FormatParser::set: runs of one ASCII letter, and every other character alone (at
// most 50 tokens, as in ICU).
func patternTokens(pattern string) []string {
	var items []string
	for i := 0; i < len(pattern) && len(items) < 50; {
		c := pattern[i]
		if isASCIILetter(c) {
			j := i + 1
			for j < len(pattern) && pattern[j] == c {
				j++
			}
			items = append(items, pattern[i:j])
			i = j
			continue
		}
		size := 1
		if c >= 0x80 {
			for size < 4 && i+size < len(pattern) && pattern[i+size]&0xC0 == 0x80 {
				size++
			}
		}
		items = append(items, pattern[i:i+size])
		i += size
	}
	return items
}

// quoteLiteral is FormatParser::getQuoteLiteral: the quoted literal starting at items[i] (quotes
// included) and the index of its last token.
func quoteLiteral(items []string, i int) (string, int) {
	var b strings.Builder
	if items[i][0] == '\'' {
		b.WriteString(items[i])
		i++
	}
	for i < len(items) {
		if items[i][0] == '\'' {
			if i+1 < len(items) && items[i+1][0] == '\'' {
				b.WriteString(items[i])
				b.WriteString(items[i+1])
				i += 2
				continue
			}
			b.WriteString(items[i])
			break
		}
		b.WriteString(items[i])
		i++
	}
	return b.String(), i
}

// ptnSkeleton is ICU's PtnSkeleton.
type ptnSkeleton struct {
	typ                   [pgFieldCount]int16
	origChar, baseChar    [pgFieldCount]byte
	origLen, baseLen      [pgFieldCount]int8
	addedDefaultDayPeriod bool
}

// parseSkeleton is DateTimeMatcher::set.
func parseSkeleton(pattern string) ptnSkeleton {
	var s ptnSkeleton
	items := patternTokens(pattern)
	for i := 0; i < len(items); i++ {
		v := items[i]
		if v[0] == '\'' {
			_, i = quoteLiteral(items, i)
			continue
		}
		ci := canonicalIndex(v)
		if ci < 0 {
			continue
		}
		row := &dtTypes[ci]
		f := row.field
		s.origChar[f], s.origLen[f] = v[0], int8(len(v))
		s.baseChar[f], s.baseLen[f] = row.ch, row.minLen
		sub := row.typ
		if row.typ > 0 {
			sub += int16(len(v))
		}
		s.typ[f] = sub
	}
	if s.origLen[pgMinute] != 0 && s.origLen[pgFractionalSecond] != 0 && s.origLen[pgSecond] == 0 {
		s.origChar[pgSecond], s.origLen[pgSecond] = 's', 1
		s.baseChar[pgSecond], s.baseLen[pgSecond] = 's', 1
		s.typ[pgSecond] = dtNumeric + 1
	}
	if s.origLen[pgHour] != 0 {
		if hc := s.origChar[pgHour]; hc == 'h' || hc == 'K' {
			if s.origLen[pgDayPeriod] == 0 {
				s.origChar[pgDayPeriod], s.origLen[pgDayPeriod] = 'a', 1
				s.baseChar[pgDayPeriod], s.baseLen[pgDayPeriod] = 'a', 1
				s.typ[pgDayPeriod] = dtShort
				s.addedDefaultDayPeriod = true
			}
		} else {
			s.origChar[pgDayPeriod], s.origLen[pgDayPeriod] = 0, 0
			s.baseChar[pgDayPeriod], s.baseLen[pgDayPeriod] = 0, 0
			s.typ[pgDayPeriod] = 0
		}
	}
	return s
}

func (s *ptnSkeleton) original() string    { return appendFields(s.origChar[:], s.origLen[:]) }
func (s *ptnSkeleton) basePattern() string { return appendFields(s.baseChar[:], s.baseLen[:]) }

// skeleton is PtnSkeleton::getSkeleton: the original fields without a default day period ICU added.
func (s *ptnSkeleton) skeleton() string {
	r := s.original()
	if s.addedDefaultDayPeriod {
		if i := strings.IndexByte(r, 'a'); i >= 0 {
			r = r[:i] + r[i+1:]
		}
	}
	return r
}

func appendFields(chars []byte, lens []int8) string {
	var b strings.Builder
	for f := range chars {
		for k := int8(0); k < lens[f]; k++ {
			b.WriteByte(chars[f])
		}
	}
	return b.String()
}

func (s *ptnSkeleton) firstChar() byte {
	for f := 0; f < pgFieldCount; f++ {
		if s.baseLen[f] != 0 {
			return s.baseChar[f]
		}
	}
	return 0
}

func (s *ptnSkeleton) fieldMask() int {
	m := 0
	for f := 0; f < pgFieldCount; f++ {
		if s.typ[f] != 0 {
			m |= 1 << f
		}
	}
	return m
}

func (s *ptnSkeleton) sameOriginal(o *ptnSkeleton) bool {
	return s.origChar == o.origChar && s.origLen == o.origLen
}

// distance is DateTimeMatcher::getDistance.
func (s *ptnSkeleton) distance(other *ptnSkeleton, includeMask int) (dist, missing, extra int) {
	for i := 0; i < pgFieldCount; i++ {
		my := int(s.typ[i])
		if includeMask&(1<<i) == 0 {
			my = 0
		}
		ot := int(other.typ[i])
		if my == ot {
			continue
		}
		switch {
		case my == 0:
			dist += extraField
			extra |= 1 << i
		case ot == 0:
			dist += missingField
			missing |= 1 << i
		default:
			d := my - ot
			if d < 0 {
				d = -d
			}
			dist += d
		}
	}
	return
}

// BuildSkeletonPatterns builds a locale's pattern generator map the way ICU's DateTimePatternGenerator
// constructor does: the canonical single-field items, then the standard patterns (derived skeletons,
// never overriding), then availableFormats entries as (skeleton, pattern) pairs in ICU's load order
// (the locale's own keys sorted, then each parent's). The result is in ICU's iteration order. It is used
// by the CLDR generator (kit/i18n/cldrgen).
func BuildSkeletonPatterns(std []string, available [][2]string) []SkeletonPattern {
	type elem struct {
		base      string
		skel      ptnSkeleton
		pattern   string
		specified bool
		std       bool
	}
	var boot [52][]*elem
	bootIndex := func(c byte) int {
		switch {
		case c >= 'A' && c <= 'Z':
			return int(c - 'A')
		case c >= 'a' && c <= 'z':
			return 26 + int(c-'a')
		}
		return -1
	}
	add := func(pattern string, skeletonToUse *string, override, fromStd bool) {
		var skel ptnSkeleton
		if skeletonToUse == nil {
			skel = parseSkeleton(pattern)
		} else {
			skel = parseSkeleton(*skeletonToUse)
		}
		base := skel.basePattern()
		bi := bootIndex(base[0])
		if bi < 0 {
			return
		}
		for _, e := range boot[bi] {
			if e.base == base {
				if !e.specified || skeletonToUse != nil && !override {
					if !override {
						return
					}
				}
				break
			}
		}
		for _, e := range boot[bootIndex(skel.firstChar())] {
			if e.skel.sameOriginal(&skel) {
				if !override || skeletonToUse != nil && e.specified {
					return
				}
				break
			}
		}
		for _, e := range boot[bi] {
			if e.base == base && e.skel.typ == skel.typ {
				e.pattern = pattern
				e.specified = skeletonToUse != nil
				e.std = e.std && fromStd
				return
			}
		}
		boot[bi] = append(boot[bi], &elem{base: base, skel: skel, pattern: pattern, specified: skeletonToUse != nil, std: fromStd})
	}
	for i := 0; i < len(canonicalItems); i++ {
		add(canonicalItems[i:i+1], nil, false, false)
	}
	for _, p := range std {
		add(p, nil, false, true)
	}
	seen := map[string]bool{}
	for _, kv := range available {
		if seen[kv[0]] {
			continue
		}
		seen[kv[0]] = true
		key := kv[0]
		add(kv[1], &key, true, false)
	}
	var out []SkeletonPattern
	for _, list := range boot {
		for _, e := range list {
			out = append(out, SkeletonPattern{Skeleton: e.skel.original(), Pattern: e.pattern, Specified: e.specified, Std: e.std})
		}
	}
	return out
}

// allowed hour formats (ICU AllowedHourFormat).
const (
	hourFmtUnknown = iota
	hourFmth
	hourFmtH
	hourFmtK
	hourFmtk
	hourFmthb
	hourFmthB
	hourFmtKb
	hourFmtKB
	hourFmtHb
	hourFmtHB
)

func hourFormatFromString(s string) int {
	switch s {
	case "h":
		return hourFmth
	case "H":
		return hourFmtH
	case "K":
		return hourFmtK
	case "k":
		return hourFmtk
	case "hb":
		return hourFmthb
	case "hB":
		return hourFmthB
	case "Kb":
		return hourFmtKb
	case "KB":
		return hourFmtKB
	case "Hb":
		return hourFmtHb
	case "HB":
		return hourFmtHB
	}
	return hourFmtUnknown
}

// patternGenerator is a DateTimePatternGenerator instance for one locale (with its -u-hc and -u-nu).
type patternGenerator struct {
	dt              *DateTimeData
	entries         []SkeletonPattern
	skels           []ptnSkeleton
	defaultHourChar byte
	allowedHour     int // first allowed hour format
	decimal         string

	matcher ptnSkeleton
	missing int
	extra   int
	spec    *ptnSkeleton
	specIdx int
}

// newPatternGenerator creates the generator for a locale. hourCycle is the -u-hc keyword ("" = none);
// noStd leaves out the standard patterns (ICU createInstanceNoStdPat).
func (l *Locale) newPatternGenerator(hourCycle, numbering string, noStd bool) *patternGenerator {
	g := &patternGenerator{dt: &l.Data.DateTime}
	for i := range g.dt.Skeletons {
		if noStd && g.dt.Skeletons[i].Std {
			continue
		}
		g.entries = append(g.entries, g.dt.Skeletons[i])
	}
	g.skels = make([]ptnSkeleton, len(g.entries))
	for i := range g.entries {
		g.skels[i] = parseSkeleton(g.entries[i].Skeleton)
	}
	sys, _ := l.numberSystem(numbering)
	g.decimal = sys.Symbols.Decimal

	pref, allowed := l.hourData()
	list0 := hourFormatFromString(pref)
	fields := strings.Fields(allowed)
	g.allowedHour = hourFmtH
	if len(fields) > 0 {
		g.allowedHour = hourFormatFromString(fields[0])
		if list0 == hourFmtUnknown {
			list0 = g.allowedHour
		}
	} else if list0 == hourFmtUnknown {
		list0 = hourFmtH
		g.allowedHour = hourFmtH
	} else {
		g.allowedHour = list0
	}
	switch hourCycle {
	case "h24":
		g.defaultHourChar = 'k'
	case "h23":
		g.defaultHourChar = 'H'
	case "h12":
		g.defaultHourChar = 'h'
	case "h11":
		g.defaultHourChar = 'K'
	}
	if g.defaultHourChar == 0 {
		switch list0 {
		case hourFmth:
			g.defaultHourChar = 'h'
		case hourFmtK:
			g.defaultHourChar = 'K'
		case hourFmtk:
			g.defaultHourChar = 'k'
		default:
			g.defaultHourChar = 'H'
		}
	}
	return g
}

// hourData returns the locale's timeData preferred and allowed hour formats (by the maximized region,
// falling back to "001").
func (l *Locale) hourData() (preferred, allowed string) {
	region := l.region()
	if l.set != nil {
		for _, key := range []string{region, "001"} {
			for _, rh := range l.set.RegionHourCycle {
				if rh.Region == key {
					return rh.Preferred, rh.Allowed
				}
			}
		}
	}
	return "H", "H"
}

// region is the locale's region for supplemental data (hour cycles, week data, zone names): -u-rg, the
// requested tag's region (en-GB served by en still gets GB's 24-hour clock), the served locale's region,
// or its likely region.
func (l *Locale) region() string {
	if rg := l.Tag.Keyword("rg"); len(rg) >= 3 {
		return strings.ToUpper(rg[:2])
	}
	if r := l.Tag.Region; len(r) == 2 || len(r) == 3 {
		return r
	}
	t := MustParseTag(l.Data.ID)
	if t.Region != "" {
		return t.Region
	}
	if l.set != nil {
		return l.set.Maximize(t).Region
	}
	return MustParseTag(l.Data.Maximal).Region
}

// defaultHourCycle is getDefaultHourCycle.
func (g *patternGenerator) defaultHourCycle() HourCycle {
	switch g.defaultHourChar {
	case 'K':
		return H11
	case 'h':
		return H12
	case 'k':
		return H24
	}
	return H23
}

// mapSkeletonMetacharacters replaces j, J and C.
func (g *patternGenerator) mapSkeletonMetacharacters(form string, flags *int) string {
	var b strings.Builder
	inQuoted := false
	for i := 0; i < len(form); i++ {
		c := form[i]
		if c == '\'' {
			inQuoted = !inQuoted
			continue
		}
		if inQuoted {
			continue
		}
		switch c {
		case 'j', 'C':
			extra := 0
			for i+1 < len(form) && form[i+1] == c {
				extra++
				i++
			}
			hourLen := 1 + extra&1
			dpLen := 1
			if extra >= 2 {
				dpLen = 3 + extra>>1
			}
			hourChar, dpChar := byte('h'), byte('a')
			if c == 'j' {
				hourChar = g.defaultHourChar
			} else {
				best := g.allowedHour
				switch best {
				case hourFmtH, hourFmtHB, hourFmtHb:
					hourChar = 'H'
				case hourFmtK, hourFmtKB, hourFmtKb:
					hourChar = 'K'
				case hourFmtk:
					hourChar = 'k'
				}
				switch best {
				case hourFmtHB, hourFmthB, hourFmtKB:
					dpChar = 'B'
				case hourFmtHb, hourFmthb, hourFmtKb:
					dpChar = 'b'
				}
			}
			if hourChar == 'H' || hourChar == 'k' {
				dpLen = 0
			}
			for ; dpLen > 0; dpLen-- {
				b.WriteByte(dpChar)
			}
			for ; hourLen > 0; hourLen-- {
				b.WriteByte(hourChar)
			}
		case 'J':
			b.WriteByte('H')
			*flags |= pgFlagSkeletonUsesCapJ
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// bestPattern is getBestPattern(skeleton, options).
func (g *patternGenerator) bestPattern(skeleton string, options int) string {
	flags := 0
	mapped := g.mapSkeletonMetacharacters(skeleton, &flags)
	g.matcher = parseSkeleton(mapped)
	best := g.bestRaw(-1)
	if best < 0 {
		return ""
	}
	if g.missing == 0 && g.extra == 0 {
		return g.adjustFieldTypes(g.entries[best].Pattern, g.spec, flags, options)
	}
	needed := g.matcher.fieldMask()
	datePattern := g.bestAppending(needed&pgDateMask, flags, options)
	timePattern := g.bestAppending(needed&pgTimeMask, flags, options)
	if datePattern == "" {
		return timePattern
	}
	if timePattern == "" {
		return datePattern
	}
	style := 3
	switch g.matcher.baseLen[pgMonth] {
	case 4:
		style = 1
		if g.matcher.baseLen[pgWeekday] > 0 {
			style = 0
		}
	case 3:
		style = 2
	}
	return simpleFormat(g.dt.AtTimeFormats[style], timePattern, datePattern)
}

// bestRaw is getBestRaw: the index of the closest entry; it sets missing, extra and spec.
func (g *patternGenerator) bestRaw(includeMask int) int {
	bestDistance := 1<<31 - 1
	bestMissing := -1
	best := -1
	for i := range g.skels {
		d, missing, extra := g.matcher.distance(&g.skels[i], includeMask)
		if d < bestDistance || d == bestDistance && bestMissing < missing {
			bestDistance, bestMissing, best = d, missing, i
			g.missing, g.extra = missing, extra
			if d == 0 {
				break
			}
		}
	}
	if best >= 0 {
		g.spec = nil
		if g.entries[best].Specified {
			g.spec = &g.skels[best]
		}
	}
	return best
}

// adjustFieldTypes is DateTimePatternGenerator::adjustFieldTypes.
func (g *patternGenerator) adjustFieldTypes(pattern string, spec *ptnSkeleton, flags, options int) string {
	var b strings.Builder
	items := patternTokens(pattern)
	for i := 0; i < len(items); i++ {
		field := items[i]
		if field[0] == '\'' {
			var lit string
			lit, i = quoteLiteral(items, i)
			b.WriteString(lit)
			continue
		}
		ci := canonicalIndex(field)
		if ci < 0 {
			b.WriteString(field)
			continue
		}
		row := &dtTypes[ci]
		tv := int(row.field)
		m := &g.matcher
		if flags&pgFlagFixFractionalSeconds != 0 && tv == pgSecond {
			b.WriteString(field)
			b.WriteString(g.decimal)
			for k := int8(0); k < m.origLen[pgFractionalSecond]; k++ {
				b.WriteByte(m.origChar[pgFractionalSecond])
			}
			continue
		}
		if m.typ[tv] == 0 {
			b.WriteString(field)
			continue
		}
		reqChar := m.origChar[tv]
		reqLen := int(m.origLen[tv])
		if reqChar == 'E' && reqLen < 3 {
			reqLen = 3
		}
		adjLen := reqLen
		if tv == pgHour && options&pgMatchHourFieldLength == 0 || tv == pgMinute || tv == pgSecond {
			adjLen = len(field)
		} else if spec != nil && reqChar != 'c' && reqChar != 'e' {
			skelLen := int(spec.origLen[tv])
			patNumeric := row.typ > 0
			reqNumeric := m.typ[tv] > 0
			if skelLen == reqLen || patNumeric != reqNumeric {
				adjLen = len(field)
			}
		}
		c := field[0]
		if tv != pgHour && tv != pgMonth && tv != pgWeekday && (tv != pgYear || reqChar == 'Y') {
			c = reqChar
		}
		if c == 'E' && adjLen < 3 {
			c = 'e'
		}
		if tv == pgHour && g.defaultHourChar != 0 {
			dh := g.defaultHourChar
			switch {
			case flags&pgFlagSkeletonUsesCapJ != 0 || reqChar == dh:
				c = dh
			case reqChar == 'h' && dh == 'K':
				c = 'K'
			case reqChar == 'H' && dh == 'k':
				c = 'k'
			case reqChar == 'k' && dh == 'H':
				c = 'H'
			case reqChar == 'K' && dh == 'h':
				c = 'h'
			}
		}
		for k := 0; k < adjLen; k++ {
			b.WriteByte(c)
		}
	}
	return b.String()
}

// bestAppending is getBestAppending.
func (g *patternGenerator) bestAppending(missingFields, flags, options int) string {
	if missingFields == 0 {
		return ""
	}
	best := g.bestRaw(missingFields)
	if best < 0 {
		return ""
	}
	spec := g.spec
	result := g.adjustFieldTypes(g.entries[best].Pattern, spec, flags, options)
	if g.missing == 0 {
		return result
	}
	last := 0
	for g.missing != 0 {
		if last == g.missing {
			break
		}
		if g.missing&pgSecondAndFractionalMsk == pgFractionalMask && missingFields&pgSecondAndFractionalMsk == pgSecondAndFractionalMsk {
			result = g.adjustFieldTypes(result, spec, flags|pgFlagFixFractionalSeconds, options)
			g.missing &^= pgFractionalMask
			continue
		}
		starting := g.missing
		best = g.bestRaw(g.missing)
		spec = g.spec
		temp := g.adjustFieldTypes(g.entries[best].Pattern, spec, flags, options)
		found := starting &^ g.missing
		top := topBitNumber(found)
		item := g.dt.AppendItems[top]
		if item == "" {
			item = "{0} ├{2}: {1}┤"
		}
		name := g.dt.FieldNames[top]
		if name == "" {
			name = defaultFieldName(top)
		}
		result = simpleFormat(item, result, temp, "'"+name+"'")
		last = g.missing
	}
	return result
}

func topBitNumber(mask int) int {
	if mask == 0 {
		return 0
	}
	i := 0
	for mask != 0 {
		mask >>= 1
		i++
	}
	if i-1 > pgZone {
		return pgZone
	}
	return i - 1
}

func defaultFieldName(f int) string {
	if f < 10 {
		return "F" + string(rune('0'+f))
	}
	return "F1" + string(rune('0'+f-10))
}

// simpleFormat is ICU SimpleFormatter::format with apostrophe mode DOUBLE_OPTIONAL: {n} is args[n],
// "”" is one apostrophe, and an apostrophe before a brace quotes literal text up to the next apostrophe.
func simpleFormat(pattern string, args ...string) string {
	var b strings.Builder
	inQuote := false
	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		if c == '\'' {
			if i+1 < len(pattern) && pattern[i+1] == '\'' {
				b.WriteByte('\'')
				i++
				continue
			}
			if inQuote {
				inQuote = false
				continue
			}
			if i+1 < len(pattern) && (pattern[i+1] == '{' || pattern[i+1] == '}') {
				inQuote = true
				continue
			}
			b.WriteByte(c)
			continue
		}
		if !inQuote && c == '{' {
			j := i + 1
			n := 0
			for j < len(pattern) && pattern[j] >= '0' && pattern[j] <= '9' {
				n = n*10 + int(pattern[j]-'0')
				j++
			}
			if j > i+1 && j < len(pattern) && pattern[j] == '}' {
				if n < len(args) {
					b.WriteString(args[n])
				}
				i = j
				continue
			}
		}
		b.WriteByte(c)
	}
	return b.String()
}

// patternSkeleton is DateTimePatternGenerator::staticGetSkeleton.
func patternSkeleton(pattern string) string {
	s := parseSkeleton(pattern)
	return s.skeleton()
}
