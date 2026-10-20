package i18n

import (
	"errors"
	"slices"
	"strings"
	"time"
)

// This file resolves time zones the way V8 does (IANA ids, case-insensitive, CLDR canonical ids, UTC
// offsets) and ports the ICU 78 TimeZoneFormat, TimeZoneNames and TimeZoneGenericNames logic that
// Intl.DateTimeFormat uses for timeZoneName (tzfmt.cpp, tznames_impl.cpp, tzgnames.cpp, zonemeta.cpp).
// UTC offsets come from Go's time package: an app that formats named zones on wasm must import
// time/tzdata (or embed tzdata another way), since time.LoadLocation has no system database there.

// ErrTimeZone is returned (wrapped) for an unknown time zone.
var ErrTimeZone = errors.New("i18n: invalid time zone")

// zone is a resolved time zone.
type zone struct {
	id        string         // resolvedOptions timeZone
	canonical string         // CLDR canonical id for display names; "" for offset zones
	info      *ZoneInfo      // canonical zone info, nil for offset zones
	loc       *time.Location // nil for offset zones
	fixed     int            // offset zone: seconds east of UTC
}

// zoneState is a zone's offset at an instant, split like ICU's raw and DST offsets.
type zoneState struct {
	offset int // total seconds east of UTC
	dst    bool
}

// resolveZone resolves an Intl timeZone option value (or a Go location's name) against the data set.
func (d *Data) resolveZone(id string, loc *time.Location) (*zone, error) {
	if loc != nil && id == "" {
		id = loc.String()
		if id == "UTC" {
			return d.resolveZone("UTC", nil)
		}
		if d.zoneInfo(canonicalizeZoneAlias(id)) == nil {
			// A location that isn't a known zone id ("Local", time.FixedZone): its offsets, GMT names.
			if id == "" {
				_, off := time.Unix(0, 0).In(loc).Zone()
				id = formatOffsetID(off)
			}
			return &zone{id: id, loc: loc}, nil
		}
	}
	if off, ok := parseOffsetZone(id); ok {
		return &zone{id: formatOffsetID(off), fixed: off}, nil
	}
	name := canonicalizeZoneAlias(id)
	info := d.zoneInfo(name)
	if info == nil {
		return nil, &zoneError{id}
	}
	canon := info
	if info.Canonical != "" {
		if c := d.zoneInfo(info.Canonical); c != nil {
			canon = c
		}
	}
	z := &zone{id: canon.ID, canonical: canon.ID, info: canon, loc: loc}
	if z.id == "Etc/UTC" || z.id == "Etc/GMT" {
		z.id = "UTC"
	}
	if z.loc == nil {
		var err error
		for _, try := range []string{info.ID, canon.ID} {
			if z.loc, err = time.LoadLocation(try); err == nil {
				break
			}
		}
		if z.loc == nil {
			return nil, &zoneError{id + " (" + err.Error() + ")"}
		}
	}
	return z, nil
}

type zoneError struct{ id string }

func (e *zoneError) Error() string { return ErrTimeZone.Error() + ": " + e.id }
func (e *zoneError) Unwrap() error { return ErrTimeZone }

// canonicalizeZoneAlias applies V8's CanonicalizeTimeZoneID special cases for UTC/GMT spellings.
func canonicalizeZoneAlias(id string) string {
	up := strings.ToUpper(id)
	switch up {
	case "GMT", "UTC", "ETC/UTC", "ETC/GMT", "ETC/UCT", "GMT0", "GMT+0", "GMT-0":
		return "Etc/UTC"
	}
	return id
}

// zoneInfo finds a zone id case-insensitively.
func (d *Data) zoneInfo(id string) *ZoneInfo {
	if d == nil || id == "" {
		return nil
	}
	key := strings.ToLower(id)
	zs := d.TimeZones.Zones
	i, ok := slices.BinarySearchFunc(zs, key, func(z ZoneInfo, k string) int { return strings.Compare(strings.ToLower(z.ID), k) })
	if !ok {
		return nil
	}
	return &zs[i]
}

// parseOffsetZone parses ECMA-402 offset time zones: ±HH, ±HHMM, ±HH:MM (U+2212 allowed for minus).
func parseOffsetZone(s string) (int, bool) {
	sign := 0
	switch {
	case strings.HasPrefix(s, "+"):
		sign, s = 1, s[1:]
	case strings.HasPrefix(s, "-"):
		sign, s = -1, s[1:]
	case strings.HasPrefix(s, "−"):
		sign, s = -1, s[len("−"):]
	default:
		return 0, false
	}
	digit := func(c byte) bool { return c >= '0' && c <= '9' }
	if len(s) < 2 || !digit(s[0]) || !digit(s[1]) {
		return 0, false
	}
	h := int(s[0]-'0')*10 + int(s[1]-'0')
	if h > 23 {
		return 0, false
	}
	s = s[2:]
	m := 0
	if s != "" {
		if s[0] == ':' {
			s = s[1:]
		}
		if len(s) != 2 || s[0] < '0' || s[0] > '5' || !digit(s[1]) {
			return 0, false
		}
		m = int(s[0]-'0')*10 + int(s[1]-'0')
	}
	return sign * (h*3600 + m*60), true
}

func formatOffsetID(off int) string {
	sign := byte('+')
	if off < 0 {
		sign, off = '-', -off
	}
	h, m := off/3600, off%3600/60
	return string([]byte{sign, byte('0' + h/10), byte('0' + h%10), ':', byte('0' + m/10), byte('0' + m%10)})
}

// state returns the zone's offset and DST flag at t. DST follows ICU's (rearguard tzdata) convention:
// when the neighbouring period differs in both offset and DST flag, the larger offset is daylight
// time, so zones with negative DST in vanguard data (Europe/Dublin winter, Morocco's Ramadan) get the
// same answer from Go's embedded tzdata as from ICU.
func (z *zone) state(t time.Time) zoneState {
	if z.loc == nil {
		return zoneState{offset: z.fixed}
	}
	lt := t.In(z.loc)
	_, off := lt.Zone()
	dst := lt.IsDST()
	// Only seasonal alternation counts: both periods bounded and shorter than a year (not Africa/Windhoek
	// since 2017, nor Europe/Minsk's decades of Moscow time after wartime CEST).
	const year = 366 * 24 * time.Hour
	start, end := lt.ZoneBounds()
	if start.IsZero() || end.IsZero() || end.Sub(start) > year {
		return zoneState{offset: off, dst: dst}
	}
	for _, n := range [2]time.Time{start.Add(-time.Second).In(z.loc), end.In(z.loc)} {
		_, noff := n.Zone()
		ns, ne := n.ZoneBounds()
		if noff == off || n.IsDST() == dst || ns.IsZero() || ne.IsZero() || ne.Sub(ns) > year {
			continue
		}
		if dst && noff > off || !dst && noff < off {
			return zoneState{offset: off, dst: !dst} // negative DST in vanguard tzdata
		}
		break
	}
	return zoneState{offset: off, dst: dst}
}

// usesDSTAround reports whether the zone has daylight time within ICU's 184-day check range of t.
func (z *zone) usesDSTAround(t time.Time) bool {
	if z.loc == nil {
		return false
	}
	const rng = 184 * 24 * time.Hour
	start, end := t.In(z.loc).ZoneBounds()
	if !start.IsZero() && t.Sub(start) < rng {
		if z.state(start.Add(-time.Second)).dst {
			return true
		}
	}
	if !end.IsZero() && end.Sub(t) < rng {
		if z.state(end).dst {
			return true
		}
	}
	return false
}

// metaZone returns the metazone a canonical zone uses at t ("" = none).
func (z *zone) metaZone(t time.Time) string {
	if z.info == nil {
		return ""
	}
	sec := t.Unix()
	for _, s := range z.info.MetaZones {
		if s.From <= sec && sec < s.To {
			return s.MetaZone
		}
	}
	return ""
}

// zoneFormatter is ICU's TimeZoneFormat for one locale.
type zoneFormatter struct {
	d            *Data
	zs           *ZoneStrings
	digits       [10]string
	targetRegion string
	gmtPrefix    string
	gmtSuffix    string
	offsetItems  [6][]gmtItem // positive H, HM, HMS, negative H, HM, HMS
}

type gmtItem struct {
	kind byte // 0 text, 'H', 'm', 's'
	text string
}

func (l *Locale) newZoneFormatter(sys *NumberSystem) *zoneFormatter {
	f := &zoneFormatter{d: l.set, zs: &l.Data.DateTime.Zone, targetRegion: l.region()}
	i := 0
	for _, r := range sys.Digits {
		if i < 10 {
			f.digits[i] = string(r)
		}
		i++
	}
	if i != 10 {
		for k := 0; k < 10; k++ {
			f.digits[k] = string(rune('0' + k))
		}
	}
	gmt := cmpOr(f.zs.GMTFormat, "GMT{0}")
	if k := strings.Index(gmt, "{0}"); k >= 0 {
		f.gmtPrefix, f.gmtSuffix = unquotePattern(gmt[:k]), unquotePattern(gmt[k+3:])
	} else {
		f.gmtPrefix = "GMT"
	}
	pos, neg := "+H:mm", "-H:mm"
	defaults := true
	if hf := f.zs.HourFormat; hf != "" {
		if k := strings.IndexByte(hf, ';'); k >= 0 {
			p, n := hf[:k], hf[k+1:]
			ph, pok := truncateOffsetPattern(p)
			nh, nok := truncateOffsetPattern(n)
			ps, pok2 := expandOffsetPattern(p)
			ns, nok2 := expandOffsetPattern(n)
			if pok && nok && pok2 && nok2 {
				f.offsetItems = [6][]gmtItem{parseOffsetPattern(ph), parseOffsetPattern(p), parseOffsetPattern(ps),
					parseOffsetPattern(nh), parseOffsetPattern(n), parseOffsetPattern(ns)}
				defaults = false
			}
		}
	}
	if defaults {
		f.offsetItems = [6][]gmtItem{parseOffsetPattern("+H"), parseOffsetPattern(pos), parseOffsetPattern("+H:mm:ss"),
			parseOffsetPattern("-H"), parseOffsetPattern(neg), parseOffsetPattern("-H:mm:ss")}
	}
	return f
}

// unquotePattern is TimeZoneFormat::unquote.
func unquotePattern(p string) string {
	if strings.IndexByte(p, '\'') < 0 {
		return p
	}
	var b strings.Builder
	prevQuote := false
	for i := 0; i < len(p); i++ {
		c := p[i]
		if c == '\'' {
			if prevQuote {
				b.WriteByte(c)
				prevQuote = false
			} else {
				prevQuote = true
			}
			continue
		}
		prevQuote = false
		b.WriteByte(c)
	}
	return b.String()
}

func expandOffsetPattern(hm string) (string, bool) {
	k := strings.Index(hm, "mm")
	if k < 0 {
		return "", false
	}
	sep := ""
	if h := strings.LastIndexByte(hm[:k], 'H'); h >= 0 {
		sep = hm[h+1 : k]
	}
	return hm[:k+2] + sep + "ss" + hm[k+2:], true
}

func truncateOffsetPattern(hm string) (string, bool) {
	k := strings.Index(hm, "mm")
	if k < 0 {
		return "", false
	}
	if h := strings.LastIndex(hm[:k], "HH"); h >= 0 {
		return hm[:h+2], true
	}
	if h := strings.LastIndexByte(hm[:k], 'H'); h >= 0 {
		return hm[:h+1], true
	}
	return "", false
}

// parseOffsetPattern is TimeZoneFormat::parseOffsetPattern (field widths are not kept: ICU formats
// hours with 1 or 2 digits by style and minutes and seconds with 2).
func parseOffsetPattern(p string) []gmtItem {
	var items []gmtItem
	var text strings.Builder
	inQuote, prevQuote := false, false
	var cur byte
	flushText := func() {
		if text.Len() > 0 {
			items = append(items, gmtItem{text: text.String()})
			text.Reset()
		}
	}
	for i := 0; i < len(p); i++ {
		c := p[i]
		if c == '\'' {
			if prevQuote {
				text.WriteByte('\'')
				prevQuote = false
			} else {
				prevQuote = true
				if cur != 0 {
					items = append(items, gmtItem{kind: cur})
					cur = 0
				}
			}
			inQuote = !inQuote
			continue
		}
		prevQuote = false
		if inQuote {
			text.WriteByte(c)
			continue
		}
		if c == 'H' || c == 'm' || c == 's' {
			if c != cur {
				if cur == 0 {
					flushText()
				} else {
					items = append(items, gmtItem{kind: cur})
				}
				cur = c
			}
			continue
		}
		if cur != 0 {
			items = append(items, gmtItem{kind: cur})
			cur = 0
		}
		text.WriteByte(c)
	}
	if cur != 0 {
		items = append(items, gmtItem{kind: cur})
	} else {
		flushText()
	}
	return items
}

func (f *zoneFormatter) appendDigits(b *strings.Builder, n, minDigits int) {
	nd := 1
	if n >= 10 {
		nd = 2
	}
	for i := 0; i < minDigits-nd; i++ {
		b.WriteString(f.digits[0])
	}
	if nd == 2 {
		b.WriteString(f.digits[n/10])
	}
	b.WriteString(f.digits[n%10])
}

// localizedGMT is formatOffsetLocalizedGMT.
func (f *zoneFormatter) localizedGMT(offset int, short bool) string {
	positive := offset >= 0
	if !positive {
		offset = -offset
	}
	h, m, s := offset/3600, offset%3600/60, offset%60
	idx := 1
	switch {
	case s != 0:
		idx = 2
	case m != 0 || !short:
		idx = 1
	default:
		idx = 0
	}
	if !positive {
		idx += 3
	}
	var b strings.Builder
	b.WriteString(f.gmtPrefix)
	for _, it := range f.offsetItems[idx] {
		switch it.kind {
		case 0:
			b.WriteString(it.text)
		case 'H':
			if short {
				f.appendDigits(&b, h, 1)
			} else {
				f.appendDigits(&b, h, 2)
			}
		case 'm':
			f.appendDigits(&b, m, 2)
		case 's':
			f.appendDigits(&b, s, 2)
		}
	}
	b.WriteString(f.gmtSuffix)
	return b.String()
}

// iso8601 is formatOffsetISO8601.
func iso8601(offset int, basic, utcIndicator, short, ignoreSeconds bool) string {
	abs := offset
	if abs < 0 {
		abs = -abs
	}
	if utcIndicator && (abs < 1 || ignoreSeconds && abs < 60) {
		return "Z"
	}
	minF, maxF := 1, 2
	if short {
		minF = 0
	}
	if ignoreSeconds {
		maxF = 1
	}
	fields := [3]int{abs / 3600, abs % 3600 / 60, abs % 60}
	last := maxF
	for last > minF && fields[last] == 0 {
		last--
	}
	sign := byte('+')
	if offset < 0 {
		for i := 0; i <= last; i++ {
			if fields[i] != 0 {
				sign = '-'
				break
			}
		}
	}
	b := []byte{sign}
	for i := 0; i <= last; i++ {
		if !basic && i != 0 {
			b = append(b, ':')
		}
		b = append(b, byte('0'+fields[i]/10), byte('0'+fields[i]%10))
	}
	return string(b)
}

// zone name types (UTZNM_*).
const (
	nameLongGeneric = iota
	nameLongStandard
	nameLongDaylight
	nameShortGeneric
	nameShortStandard
	nameShortDaylight
)

// nameCity is the exemplar city's index in ZoneNames.Names.
const nameCity = 6

// name returns field typ of the packed names ("" when absent).
func (n *ZoneNames) name(typ int) string {
	return packedField(n.Names, typ)
}

// packedField returns the i-th U+001F-separated field of s.
func packedField(s string, i int) string {
	for ; i > 0; i-- {
		k := strings.IndexByte(s, 0x1f)
		if k < 0 {
			return ""
		}
		s = s[k+1:]
	}
	if k := strings.IndexByte(s, 0x1f); k >= 0 {
		return s[:k]
	}
	return s
}

func findZoneNames(list []ZoneNames, id string) *ZoneNames {
	i, ok := slices.BinarySearchFunc(list, id, func(n ZoneNames, s string) int { return strings.Compare(n.ID, s) })
	if !ok {
		return nil
	}
	return &list[i]
}

func (f *zoneFormatter) tzName(tzID string, typ int) string {
	if n := findZoneNames(f.zs.Zones, tzID); n != nil {
		return n.name(typ)
	}
	return ""
}

func (f *zoneFormatter) mzName(mzID string, typ int) string {
	if mzID == "" {
		return ""
	}
	if n := findZoneNames(f.zs.MetaZones, mzID); n != nil {
		return n.name(typ)
	}
	return ""
}

// displayName is TimeZoneNames::getDisplayName: the zone's own name, else its metazone's at t.
func (f *zoneFormatter) displayName(z *zone, typ int, t time.Time) string {
	if s := f.tzName(z.canonical, typ); s != "" {
		return s
	}
	return f.mzName(z.metaZone(t), typ)
}

// exemplarCity is TimeZoneNamesImpl::getExemplarLocationName with the default derived from the id.
func (f *zoneFormatter) exemplarCity(tzID string) string {
	if n := findZoneNames(f.zs.Zones, tzID); n != nil {
		if c := n.name(nameCity); c != "" {
			return c
		}
	}
	if tzID == "" || strings.HasPrefix(tzID, "Etc/") || strings.HasPrefix(tzID, "SystemV/") || strings.Index(tzID, "Riyadh8") > 0 {
		return ""
	}
	sep := strings.LastIndexByte(tzID, '/')
	if sep > 0 && sep+1 < len(tzID) {
		return strings.ReplaceAll(tzID[sep+1:], "_", " ")
	}
	return ""
}

// regionName is LocaleDisplayNames::regionDisplayName over the packed region names (the code when absent).
func (f *zoneFormatter) regionName(code string) string {
	s := f.zs.RegionNames
	for s != "" {
		entry := s
		if k := strings.IndexByte(s, 0x1f); k >= 0 {
			entry, s = s[:k], s[k+1:]
		} else {
			s = ""
		}
		if c, name, ok := strings.Cut(entry, "\x1e"); ok && c == code {
			return name
		}
	}
	return code
}

// referenceZone is ZoneMeta::getZoneIdByMetazone.
func (f *zoneFormatter) referenceZone(mzID, region string) string {
	if f.d == nil {
		return ""
	}
	ms := f.d.TimeZones.MetaZones
	i, ok := slices.BinarySearchFunc(ms, mzID, func(m MetaZoneInfo, s string) int { return strings.Compare(m.ID, s) })
	if !ok {
		return ""
	}
	world := ""
	for _, rz := range ms[i].Zones {
		if rz.Region == region && (len(region) == 2 || len(region) == 3) {
			return rz.Zone
		}
		if rz.Region == "001" {
			world = rz.Zone
		}
	}
	return world
}

// genericLocation is TZGNCore::getGenericLocationName.
func (f *zoneFormatter) genericLocation(z *zone) string {
	if z.info == nil || z.info.Region == "" {
		return ""
	}
	loc := ""
	if z.info.Primary {
		loc = f.regionName(z.info.Region)
	} else {
		loc = f.exemplarCity(z.canonical)
	}
	return simpleFormat(cmpOr(f.zs.RegionFormat, "{0}"), loc)
}

// partialLocation is TZGNCore::getPartialLocationName.
func (f *zoneFormatter) partialLocation(z *zone, mzID, mzName string) string {
	location := ""
	if z.info != nil && z.info.Region != "" {
		if f.referenceZone(mzID, z.info.Region) == z.canonical {
			location = f.regionName(z.info.Region)
		} else {
			location = f.exemplarCity(z.canonical)
		}
	} else {
		location = f.exemplarCity(z.canonical)
		if location == "" {
			location = z.canonical
		}
	}
	return simpleFormat(cmpOr(f.zs.FallbackFormat, "{1} ({0})"), location, mzName)
}

// genericNonLocation is TZGNCore::formatGenericNonLocationName.
func (f *zoneFormatter) genericNonLocation(z *zone, long bool, t time.Time) string {
	if z.canonical == "" {
		return ""
	}
	typ := nameShortGeneric
	if long {
		typ = nameLongGeneric
	}
	if s := f.tzName(z.canonical, typ); s != "" {
		return s
	}
	mz := z.metaZone(t)
	if mz == "" {
		return ""
	}
	st := z.state(t)
	name := ""
	if !st.dst && !z.usesDSTAround(t) {
		stdType := nameShortStandard
		if long {
			stdType = nameLongStandard
		}
		if std := f.displayName(z, stdType, t); std != "" {
			name = std
			if strings.EqualFold(std, f.mzName(mz, typ)) {
				name = ""
			}
		}
	}
	if name == "" {
		mzName := f.mzName(mz, typ)
		if mzName == "" {
			return ""
		}
		golden := f.referenceZone(mz, f.targetRegion)
		if golden != "" && golden != z.canonical {
			gz := &zone{canonical: golden}
			gz.info = f.d.zoneInfo(golden)
			if loc, err := time.LoadLocation(golden); err == nil {
				gz.loc = loc
				local := t.Add(time.Duration(st.offset) * time.Second)
				gs := gz.state(local)
				gs = gz.state(local.Add(-time.Duration(gs.offset) * time.Second))
				if gs.offset != st.offset || gs.dst != st.dst {
					return f.partialLocation(z, mz, mzName)
				}
			}
		}
		name = mzName
	}
	return name
}

// Zone name styles of the pattern letters.
const (
	tzSpecificShort = iota
	tzSpecificLong
	tzGenericShort
	tzGenericLong
	tzGenericLocation
	tzGMTShort
	tzGMTLong
	tzExemplar
	tzID
	tzShortID
)

// format is TimeZoneFormat::format for the display styles (with the localized GMT fallback).
func (f *zoneFormatter) format(style int, z *zone, t time.Time) string {
	name := ""
	switch style {
	case tzGenericLocation:
		name = f.genericLocation(z)
	case tzGenericLong, tzGenericShort:
		name = f.genericNonLocation(z, style == tzGenericLong, t)
		if name == "" {
			name = f.genericLocation(z)
		}
	case tzSpecificLong, tzSpecificShort:
		if z.canonical != "" {
			st := z.state(t)
			typ := nameShortStandard
			switch {
			case style == tzSpecificLong && st.dst:
				typ = nameLongDaylight
			case style == tzSpecificLong:
				typ = nameLongStandard
			case st.dst:
				typ = nameShortDaylight
			}
			name = f.displayName(z, typ, t)
		}
	case tzID:
		if z.canonical != "" {
			return z.canonical
		}
		return "GMT" + strings.ReplaceAll(z.id, ":", "")
	case tzShortID:
		return "unk"
	case tzExemplar:
		if c := f.exemplarCity(z.canonical); c != "" {
			return c
		}
		if c := f.exemplarCity("Etc/Unknown"); c != "" {
			return c
		}
		return "Unknown"
	}
	if name != "" {
		return name
	}
	off := z.state(t).offset
	switch style {
	case tzGenericLocation, tzGenericLong, tzSpecificLong, tzGMTLong:
		return f.localizedGMT(off, false)
	}
	return f.localizedGMT(off, true)
}
