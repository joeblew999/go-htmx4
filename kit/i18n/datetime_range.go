package i18n

import (
	"slices"
	"strings"
	"time"
)

// This file ports ICU's DateIntervalFormat and DateIntervalInfo (dtitvfmt.cpp, dtitvinf.cpp, ICU 78) as
// V8's Intl.DateTimeFormat.prototype.formatRange uses them.

// interval pattern indexes (DateIntervalInfo::IntervalPatternIndex).
const (
	ipiEra = iota
	ipiYear
	ipiMonth
	ipiDate
	ipiAmPm
	ipiHour
	ipiMinute
	ipiSecond
	ipiMillisecond
	ipiCount
)

type intervalPatternInfo struct {
	first, second string
	laterFirst    bool
}

type intervalFormat struct {
	f            *DateTimeFormat
	gen          *patternGenerator
	skeleton     string
	pattern      string // the single-date pattern (fDateFormat)
	datePattern  string
	timePattern  string
	hasDateTime  bool
	dateTimeGlue string
	fallback     string
	laterFirst   bool
	patterns     [ipiCount]intervalPatternInfo
}

// FormatRange formats the range from start to end (Intl.DateTimeFormat.prototype.formatRange). When
// start and end are equal at the fields shown, it formats start alone.
func (f *DateTimeFormat) FormatRange(start, end time.Time) string {
	f.intervalOnce.Do(func() { f.interval = f.newIntervalFormat() })
	var b strings.Builder
	if !f.interval.format(&b, start, end) {
		return f.Format(start)
	}
	return replaceUnicodeSpaces(b.String())
}

func (f *DateTimeFormat) newIntervalFormat() *intervalFormat {
	hc := f.hourCycle.String()
	if hc == "" {
		hc = f.icuHC
	}
	it := &intervalFormat{f: f, gen: f.loc.newPatternGenerator(hc, f.nu, false)}
	it.skeleton = patternSkeleton(f.pattern)
	it.pattern = it.gen.bestPattern(it.skeleton, pgMatchNoOptions)
	dt := &f.loc.Data.DateTime
	it.fallback = cmpOr(dt.IntervalFallback, "{0} – {1}")
	if a, b := strings.Index(it.fallback, "{0}"), strings.Index(it.fallback, "{1}"); a > b {
		it.laterFirst = true
	}
	it.initializePattern()
	return it
}

func (it *intervalFormat) bestPattern(skeleton string) string {
	return it.gen.bestPattern(skeleton, pgMatchNoOptions)
}

func (it *intervalFormat) info(skeleton string) *IntervalFormats {
	list := it.f.loc.Data.DateTime.Intervals
	i, ok := slices.BinarySearchFunc(list, skeleton, func(x IntervalFormats, s string) int { return strings.Compare(x.Skeleton, s) })
	if !ok {
		return nil
	}
	return &list[i]
}

// intervalPattern is DateIntervalInfo::getIntervalPattern for the interval index of a field.
func (it *intervalFormat) intervalPattern(skeleton string, idx int) string {
	if x := it.info(skeleton); x != nil {
		return packedField(x.PatternList, idx)
	}
	return ""
}

func skeletonFieldWidths(s string) [58]int {
	var w [58]int
	for i := 0; i < len(s); i++ {
		if c := s[i]; c >= 'A' && c <= 'z' {
			w[c-'A']++
		}
	}
	return w
}

// bestSkeleton is DateIntervalInfo::getBestSkeleton (candidates in sorted order).
func (it *intervalFormat) bestSkeleton(skeleton string) (string, int) {
	input := skeleton
	replaced := false
	if strings.ContainsAny(skeleton, "zkKab") {
		input = strings.NewReplacer("z", "v", "k", "H", "K", "h", "a", "", "b", "").Replace(skeleton)
		replaced = true
	}
	in := skeletonFieldWidths(input)
	best := ""
	bestDistance := 56632 // ICU MAX_POSITIVE_INT
	diffInfo := 0
	found := false
	for _, x := range it.f.loc.Data.DateTime.Intervals {
		w := skeletonFieldWidths(x.Skeleton)
		distance := 0
		fieldDiff := 1
		for i := range in {
			a, b := in[i], w[i]
			if a == b {
				continue
			}
			switch {
			case a == 0 || b == 0:
				fieldDiff = -1
				distance += 0x1000
			case byte(i+'A') == 'M' && (a <= 2 && b > 2 || a > 2 && b <= 2):
				distance += 0x100
			case a > b:
				distance += a - b
			default:
				distance += b - a
			}
		}
		if distance < bestDistance {
			best, bestDistance, diffInfo, found = x.Skeleton, distance, fieldDiff, true
		}
		if distance == 0 {
			diffInfo = 0
			break
		}
	}
	if !found {
		return "", 0
	}
	if replaced && diffInfo != -1 {
		diffInfo = 2
	}
	return best, diffInfo
}

// normalizeHourMetacharacters is DateIntervalFormat::normalizeHourMetacharacters.
func (it *intervalFormat) normalizeHourMetacharacters(skeleton string) string {
	var hourMeta, dpChar byte
	hourStart, hourLen, dpStart, dpLen := 0, 0, 0, 0
	for i := 0; i < len(skeleton); i++ {
		c := skeleton[i]
		switch c {
		case 'j', 'J', 'C', 'h', 'H', 'k', 'K':
			if hourMeta == 0 {
				hourMeta, hourStart = c, i
			}
			hourLen++
		case 'a', 'b', 'B':
			if dpChar == 0 {
				dpChar, dpStart = c, i
			}
			dpLen++
		default:
			if hourMeta != 0 && dpChar != 0 {
				i = len(skeleton)
			}
		}
	}
	if hourMeta == 0 {
		return skeleton
	}
	hourChar := byte('H')
	conv := it.bestPattern(string([]byte{hourMeta}))
	for {
		q := strings.IndexByte(conv, '\'')
		if q < 0 {
			break
		}
		q2 := strings.IndexByte(conv[q+1:], '\'')
		if q2 < 0 {
			q2 = q
		} else {
			q2 += q + 1
		}
		conv = conv[:q] + conv[q2+1:]
	}
	switch {
	case strings.IndexByte(conv, 'h') >= 0:
		hourChar = 'h'
	case strings.IndexByte(conv, 'K') >= 0:
		hourChar = 'K'
	case strings.IndexByte(conv, 'k') >= 0:
		hourChar = 'k'
	}
	switch {
	case strings.IndexByte(conv, 'b') >= 0:
		dpChar = 'b'
	case strings.IndexByte(conv, 'B') >= 0:
		dpChar = 'B'
	case dpChar == 0:
		dpChar = 'a'
	}
	hourAndDP := string([]byte{hourChar})
	if hourChar != 'H' && hourChar != 'k' {
		n := 1
		switch {
		case dpLen >= 5 || hourLen >= 5:
			n = 5
		case dpLen >= 3 || hourLen >= 3:
			n = 3
		}
		hourAndDP += strings.Repeat(string([]byte{dpChar}), n)
	}
	result := skeleton[:hourStart] + hourAndDP + skeleton[hourStart+hourLen:]
	if dpStart > hourStart {
		dpStart += len(hourAndDP) - hourLen
	}
	if dpLen > 0 {
		result = result[:dpStart] + result[dpStart+dpLen:]
	}
	return result
}

// dateTimeSkeleton is DateIntervalFormat::getDateTimeSkeleton.
func dateTimeSkeleton(skeleton string) (date, normDate, tm, normTime string) {
	var d, nd, t, nt strings.Builder
	eCount, dCount, mCount, yCount, minCount, vCount, zCount := 0, 0, 0, 0, 0, 0, 0
	var hourChar byte
	for i := 0; i < len(skeleton); i++ {
		ch := skeleton[i]
		switch ch {
		case 'E':
			d.WriteByte(ch)
			eCount++
		case 'd':
			d.WriteByte(ch)
			dCount++
		case 'M':
			d.WriteByte(ch)
			mCount++
		case 'y':
			d.WriteByte(ch)
			yCount++
		case 'G', 'Y', 'u', 'Q', 'q', 'L', 'l', 'W', 'w', 'D', 'F', 'g', 'e', 'c', 'U', 'r':
			nd.WriteByte(ch)
			d.WriteByte(ch)
		case 'h', 'H', 'k', 'K':
			t.WriteByte(ch)
			if hourChar == 0 {
				hourChar = ch
			}
		case 'm':
			t.WriteByte(ch)
			minCount++
		case 'z':
			zCount++
			t.WriteByte(ch)
		case 'v':
			vCount++
			t.WriteByte(ch)
		case 'a', 'V', 'Z', 'j', 's', 'S', 'A', 'b', 'B':
			t.WriteByte(ch)
			nt.WriteByte(ch)
		}
	}
	nd.WriteString(strings.Repeat("y", yCount))
	if mCount != 0 {
		if mCount < 3 {
			nd.WriteByte('M')
		} else {
			nd.WriteString(strings.Repeat("M", min(mCount, 5)))
		}
	}
	if eCount != 0 {
		if eCount <= 3 {
			nd.WriteByte('E')
		} else {
			nd.WriteString(strings.Repeat("E", min(eCount, 5)))
		}
	}
	if dCount != 0 {
		nd.WriteByte('d')
	}
	if hourChar != 0 {
		nt.WriteByte(hourChar)
	}
	if minCount != 0 {
		nt.WriteByte('m')
	}
	if zCount != 0 {
		nt.WriteByte('z')
	}
	if vCount != 0 {
		nt.WriteByte('v')
	}
	return d.String(), nd.String(), t.String(), nt.String()
}

func (it *intervalFormat) initializePattern() {
	for i := range it.patterns {
		it.patterns[i].laterFirst = it.laterFirst
	}
	converted := it.normalizeHourMetacharacters(it.skeleton)
	dateSkel, normDate, timeSkel, normTime := dateTimeSkeleton(converted)
	if timeSkel != "" && dateSkel != "" {
		if g := it.f.loc.Data.DateTime.DateTimeFormats[2]; len(g) >= 3 {
			it.dateTimeGlue, it.hasDateTime = g, true
		}
	}
	found := it.setSeparateDateTimePattern(normDate, normTime)
	prefixYMD := func() {
		ts := "yMd" + timeSkel
		p := it.bestPattern(ts)
		it.setPatternInfo(ipiDate, nil, &p, it.laterFirst)
		it.setPatternInfo(ipiMonth, nil, &p, it.laterFirst)
		it.setPatternInfo(ipiYear, nil, &p, it.laterFirst)
		ts = "G" + ts
		p = it.bestPattern(ts)
		it.setPatternInfo(ipiEra, nil, &p, it.laterFirst)
	}
	if !found {
		if timeSkel != "" && dateSkel == "" {
			prefixYMD()
		}
		return
	}
	switch {
	case timeSkel == "":
	case dateSkel == "":
		prefixYMD()
	default:
		skel := it.skeleton
		for _, fb := range []struct {
			letter byte
			idx    int
		}{{'d', ipiDate}, {'M', ipiMonth}, {'y', ipiYear}, {'G', ipiEra}} {
			if strings.IndexByte(dateSkel, fb.letter) < 0 {
				skel = string([]byte{fb.letter}) + skel
				p := it.bestPattern(skel)
				it.setPatternInfo(fb.idx, nil, &p, it.laterFirst)
			}
		}
		if !it.hasDateTime {
			return
		}
		datePattern := it.bestPattern(dateSkel)
		for _, idx := range []int{ipiAmPm, ipiHour, ipiMinute} {
			info := it.patterns[idx]
			if info.first != "" {
				combined := simpleFormat(it.dateTimeGlue, info.first+info.second, datePattern)
				it.setIntervalPatternOrder(idx, combined, info.laterFirst)
			}
		}
	}
}

func (it *intervalFormat) setPatternInfo(idx int, first, second *string, laterFirst bool) {
	p := &it.patterns[idx]
	if first != nil {
		p.first = *first
	}
	if second != nil {
		p.second = *second
	}
	p.laterFirst = laterFirst
}

func (it *intervalFormat) setIntervalPatternOrder(idx int, pattern string, laterFirst bool) {
	order := laterFirst
	switch {
	case strings.HasPrefix(pattern, "latestFirst:"):
		order, pattern = true, pattern[len("latestFirst:"):]
	case strings.HasPrefix(pattern, "earliestFirst:"):
		order, pattern = false, pattern[len("earliestFirst:"):]
	}
	split := splitIntervalPattern(pattern)
	first, second := pattern[:split], pattern[split:]
	it.setPatternInfo(idx, &first, &second, order)
}

var intervalFieldLetters = [...]byte{ipiEra: 'G', ipiYear: 'y', ipiMonth: 'M', ipiDate: 'd', ipiAmPm: 'a', ipiHour: 'h', ipiMinute: 'm'}

// intervalFieldLevel is fgCalendarFieldToLevel for the interval fields.
var intervalFieldLevel = [...]int{ipiEra: 0, ipiYear: 10, ipiMonth: 20, ipiDate: 30, ipiAmPm: 40, ipiHour: 50, ipiMinute: 60, ipiSecond: 70, ipiMillisecond: 80}

func (it *intervalFormat) setSeparateDateTimePattern(dateSkel, timeSkel string) bool {
	skeleton := dateSkel
	if timeSkel != "" {
		skeleton = timeSkel
	}
	best, diff := it.bestSkeleton(skeleton)
	if best == "" {
		return false
	}
	if dateSkel != "" {
		it.datePattern = it.bestPattern(dateSkel)
	}
	if timeSkel != "" {
		it.timePattern = it.bestPattern(timeSkel)
	}
	if diff == -1 {
		return false
	}
	if timeSkel == "" {
		ext, extBest := "", ""
		it.setIntervalPattern(ipiDate, skeleton, best, diff, &ext, &extBest)
		if it.setIntervalPattern(ipiMonth, skeleton, best, diff, &ext, &extBest) {
			best, skeleton = extBest, ext
		}
		it.setIntervalPattern(ipiYear, skeleton, best, diff, &ext, &extBest)
		it.setIntervalPattern(ipiEra, skeleton, best, diff, &ext, &extBest)
	} else {
		it.setIntervalPattern(ipiMinute, skeleton, best, diff, nil, nil)
		it.setIntervalPattern(ipiHour, skeleton, best, diff, nil, nil)
		it.setIntervalPattern(ipiAmPm, skeleton, best, diff, nil, nil)
	}
	return true
}

func (it *intervalFormat) setIntervalPattern(idx int, skeleton, best string, diff int, ext, extBest *string) bool {
	pattern := it.intervalPattern(best, idx)
	if pattern == "" {
		if patternFieldIgnored(best, intervalFieldLevel[idx]) {
			return false
		}
		if idx == ipiAmPm {
			pattern = it.intervalPattern(best, ipiHour)
			if pattern != "" {
				adj := adjustIntervalFieldWidth(skeleton, best, pattern, diff, strings.IndexByte(it.skeleton, 'J') >= 0)
				it.setIntervalPatternOrder(idx, adj, it.laterFirst)
			}
			return false
		}
		if ext != nil {
			letter := string([]byte{intervalFieldLetters[idx]})
			*ext = letter + skeleton
			*extBest = letter + best
			pattern = it.intervalPattern(*extBest, idx)
			if pattern == "" && diff == 0 {
				tmp, d2 := it.bestSkeleton(*extBest)
				diff = d2
				if tmp != "" && d2 != -1 {
					pattern = it.intervalPattern(tmp, idx)
					best = tmp
				}
			}
		}
	}
	if pattern != "" {
		suppress := strings.IndexByte(it.skeleton, 'J') >= 0
		if diff != 0 || suppress {
			it.setIntervalPatternOrder(idx, adjustIntervalFieldWidth(skeleton, best, pattern, diff, suppress), it.laterFirst)
		} else {
			it.setIntervalPatternOrder(idx, pattern, it.laterFirst)
		}
		if ext != nil && *ext != "" {
			return true
		}
	}
	return false
}

// patternFieldIgnored is SimpleDateFormat::isFieldUnitIgnored(pattern, field) by field level.
func patternFieldIgnored(pattern string, fieldLevel int) bool {
	inQuote := false
	var prev byte
	count := 0
	for i := 0; i < len(pattern); i++ {
		ch := pattern[i]
		if ch != prev && count > 0 {
			if fieldLevel <= letterLevel(prev) {
				return false
			}
			count = 0
		}
		if ch == '\'' {
			if i+1 < len(pattern) && pattern[i+1] == '\'' {
				i++
			} else {
				inQuote = !inQuote
			}
		} else if !inQuote && isASCIILetter(ch) {
			prev = ch
			count++
		}
	}
	if count > 0 && fieldLevel <= letterLevel(prev) {
		return false
	}
	return true
}

// letterLevel is SimpleDateFormat::getLevelFromChar.
func letterLevel(c byte) int {
	switch c {
	case 'A', 'a':
		return 40
	case 'D', 'L', 'M', 'Q', 'q', 'w':
		return 20
	case 'E', 'F', 'W', 'c', 'd', 'e':
		return 30
	case 'G', 'O', 'V', 'X', 'Z', 'g', 'l', 'v', 'x', 'z':
		return 0
	case 'H', 'K', 'h', 'k':
		return 50
	case 'S':
		return 80
	case 'U', 'Y', 'r', 'u', 'y':
		return 10
	case 'm':
		return 60
	case 's':
		return 70
	}
	return -1
}

// splitIntervalPattern is DateIntervalFormat::splitPatternInto2Part.
func splitIntervalPattern(p string) int {
	var repeated [58]bool
	inQuote := false
	var prev byte
	count := 0
	i := 0
	foundRepetition := false
	for ; i < len(p); i++ {
		ch := p[i]
		if ch != prev && count > 0 {
			if !repeated[prev-'A'] {
				repeated[prev-'A'] = true
			} else {
				foundRepetition = true
				break
			}
			count = 0
		}
		if ch == '\'' {
			if i+1 < len(p) && p[i+1] == '\'' {
				i++
			} else {
				inQuote = !inQuote
			}
		} else if !inQuote && isASCIILetter(ch) {
			prev = ch
			count++
		}
	}
	if count > 0 && !foundRepetition && !repeated[prev-'A'] {
		count = 0
	}
	return i - count
}

// adjustIntervalFieldWidth is DateIntervalFormat::adjustFieldWidth.
func adjustIntervalFieldWidth(input, best, pattern string, diff int, suppressDayPeriod bool) string {
	adjusted := pattern
	in := skeletonFieldWidths(input)
	bw := skeletonFieldWidths(best)
	if suppressDayPeriod {
		for _, s := range []string{" a", " a", "a ", "a ", "a"} {
			adjusted = findReplaceInPattern(adjusted, s, "")
		}
		adjusted = strings.TrimSpace(findReplaceInPattern(adjusted, "  ", " "))
	}
	if diff == 2 {
		if strings.IndexByte(input, 'z') >= 0 {
			adjusted = findReplaceInPattern(adjusted, "v", "z")
		}
		if strings.IndexByte(input, 'K') >= 0 {
			adjusted = findReplaceInPattern(adjusted, "h", "K")
		}
		if strings.IndexByte(input, 'k') >= 0 {
			adjusted = findReplaceInPattern(adjusted, "H", "k")
		}
		if strings.IndexByte(input, 'b') >= 0 {
			adjusted = findReplaceInPattern(adjusted, "a", "b")
		}
	}
	if strings.IndexByte(adjusted, 'a') >= 0 && bw['a'-'A'] == 0 {
		bw['a'-'A'] = 1
	}
	if strings.IndexByte(adjusted, 'b') >= 0 && bw['b'-'A'] == 0 {
		bw['b'-'A'] = 1
	}
	var b strings.Builder
	inQuote := false
	var prev byte
	count := 0
	extend := func() {
		sc := prev
		if sc == 'L' {
			sc = 'M'
		}
		fieldCount, inputCount := bw[sc-'A'], in[sc-'A']
		if fieldCount == count && inputCount > fieldCount {
			b.WriteString(strings.Repeat(string([]byte{prev}), inputCount-fieldCount))
		}
	}
	for i := 0; i < len(adjusted); i++ {
		ch := adjusted[i]
		if ch != prev && count > 0 {
			extend()
			count = 0
		}
		if ch == '\'' {
			if i+1 < len(adjusted) && adjusted[i+1] == '\'' {
				b.WriteByte(ch)
				i++
			} else {
				inQuote = !inQuote
			}
		} else if !inQuote && isASCIILetter(ch) {
			prev = ch
			count++
		}
		b.WriteByte(ch)
	}
	if count > 0 {
		extend()
	}
	return b.String()
}

// findReplaceInPattern replaces outside quoted literals.
func findReplaceInPattern(s, old, repl string) string {
	q := strings.IndexByte(s, '\'')
	if q < 0 {
		return strings.ReplaceAll(s, old, repl)
	}
	var b strings.Builder
	src := s
	for q >= 0 {
		q2 := strings.IndexByte(src[q+1:], '\'')
		if q2 < 0 {
			q2 = len(src) - 1
		} else {
			q2 += q + 1
		}
		b.WriteString(strings.ReplaceAll(src[:q], old, repl))
		b.WriteString(src[q : q2+1])
		src = src[q2+1:]
		q = strings.IndexByte(src, '\'')
	}
	b.WriteString(strings.ReplaceAll(src, old, repl))
	return b.String()
}

// format is DateIntervalFormat::formatImpl; it reports whether a range (not a single date) was written.
func (it *intervalFormat) format(b *strings.Builder, from, to time.Time) bool {
	f := it.f
	a, c := f.fields(from), f.fields(to)
	idx := -1
	switch {
	case a.era != c.era:
		idx = ipiEra
	case a.year != c.year:
		idx = ipiYear
	case a.month != c.month:
		idx = ipiMonth
	case a.day != c.day:
		idx = ipiDate
	case (a.hour >= 12) != (c.hour >= 12):
		idx = ipiAmPm
	case a.hour%12 != c.hour%12:
		idx = ipiHour
	case a.minute != c.minute:
		idx = ipiMinute
	case a.second != c.second:
		idx = ipiSecond
	case a.ms != c.ms:
		idx = ipiMillisecond
	}
	if idx < 0 {
		return false
	}
	sameDay := idx >= ipiAmPm
	info := it.patterns[idx]
	if info.first == "" && info.second == "" {
		if patternFieldIgnored(it.pattern, intervalFieldLevel[idx]) {
			return false
		}
		it.fallbackFormat(b, it.pattern, from, to, sameDay)
		return true
	}
	if info.first == "" {
		it.fallbackFormat(b, info.second, from, to, sameDay)
		return true
	}
	first, second := from, to
	if info.laterFirst {
		first, second = to, from
	}
	f.formatPattern(b, info.first, first)
	if info.second != "" {
		f.formatPattern(b, info.second, second)
	}
	return true
}

func (it *intervalFormat) fallbackRange(b *strings.Builder, pattern string, from, to time.Time) {
	fb := it.fallback
	i0, i1 := strings.Index(fb, "{0}"), strings.Index(fb, "{1}")
	f := it.f
	if i0 < i1 {
		b.WriteString(fb[:i0])
		f.formatPattern(b, pattern, from)
		b.WriteString(fb[i0+3 : i1])
		f.formatPattern(b, pattern, to)
		b.WriteString(fb[i1+3:])
	} else {
		b.WriteString(fb[:i1])
		f.formatPattern(b, pattern, to)
		b.WriteString(fb[i1+3 : i0])
		f.formatPattern(b, pattern, from)
		b.WriteString(fb[i0+3:])
	}
}

func (it *intervalFormat) fallbackFormat(b *strings.Builder, pattern string, from, to time.Time, sameDay bool) {
	if sameDay && it.datePattern != "" && it.timePattern != "" && it.hasDateTime {
		glue := it.dateTimeGlue
		i0, i1 := strings.Index(glue, "{0}"), strings.Index(glue, "{1}")
		if i0 < i1 {
			b.WriteString(glue[:i0])
			it.fallbackRange(b, it.timePattern, from, to)
			b.WriteString(glue[i0+3 : i1])
			it.f.formatPattern(b, it.datePattern, from)
			b.WriteString(glue[i1+3:])
		} else {
			b.WriteString(glue[:i1])
			it.f.formatPattern(b, it.datePattern, from)
			b.WriteString(glue[i1+3 : i0])
			it.fallbackRange(b, it.timePattern, from, to)
			b.WriteString(glue[i0+3:])
		}
		return
	}
	it.fallbackRange(b, pattern, from, to)
}
