package i18n

import (
	"strconv"
	"strings"
	"time"
)

// This file ports the formatting half of ICU's SimpleDateFormat (smpdtfmt.cpp, subFormat) for the
// Gregorian calendar: pattern letters, quoting, day periods and zone names.

// calFields are the calendar fields of an instant in a zone (proleptic Gregorian, like V8).
type calFields struct {
	t            time.Time // the instant
	local        time.Time // wall clock in the zone (location UTC)
	era, year    int       // era 0 = BC; year of era
	extYear      int       // astronomical year
	month        int       // 0–11
	day, dow     int       // day of month; day of week 1 = Sunday … 7
	doy          int
	hour, minute int
	second, ms   int

	hasMin, hasSec bool // the pattern shows minutes / seconds (SimpleDateFormat::parsePattern)
}

func (f *DateTimeFormat) fields(t time.Time) calFields {
	off := f.zone.state(t).offset
	lt := t.UTC().Add(time.Duration(off) * time.Second)
	c := calFields{t: t, local: lt}
	c.extYear = lt.Year()
	if c.extYear <= 0 {
		c.era, c.year = 0, 1-c.extYear
	} else {
		c.era, c.year = 1, c.extYear
	}
	c.month = int(lt.Month()) - 1
	c.day = lt.Day()
	c.dow = int(lt.Weekday()) + 1
	c.doy = lt.YearDay()
	c.hour, c.minute, c.second = lt.Hour(), lt.Minute(), lt.Second()
	c.ms = lt.Nanosecond() / 1e6
	return c
}

// formatPattern formats t with an ICU date pattern.
func (f *DateTimeFormat) formatPattern(b *strings.Builder, pattern string, t time.Time) {
	c := f.fields(t)
	c.hasMin, c.hasSec = patternHasMinSec(pattern)
	inQuote := false
	var prev byte
	count := 0
	for i := 0; i < len(pattern); i++ {
		ch := pattern[i]
		if ch != prev && count > 0 {
			f.subFormat(b, prev, count, &c)
			count = 0
		}
		if ch == '\'' {
			if i+1 < len(pattern) && pattern[i+1] == '\'' {
				b.WriteByte('\'')
				i++
			} else {
				inQuote = !inQuote
			}
			continue
		}
		if !inQuote && isASCIILetter(ch) {
			prev = ch
			count++
			continue
		}
		b.WriteByte(ch)
	}
	if count > 0 {
		f.subFormat(b, prev, count, &c)
	}
}

func patternHasMinSec(pattern string) (hasMin, hasSec bool) {
	inQuote := false
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '\'':
			inQuote = !inQuote
		case 'm':
			hasMin = hasMin || !inQuote
		case 's':
			hasSec = hasSec || !inQuote
		}
	}
	return
}

// zeroPad is SimpleDateFormat::zeroPaddingNumber with the format's digits.
func (f *DateTimeFormat) zeroPad(b *strings.Builder, value, minDigits, maxDigits int) {
	if value < 0 {
		b.WriteString(f.sys.Symbols.Minus)
		value = -value
	}
	s := strconv.Itoa(value)
	if len(s) > maxDigits {
		s = s[len(s)-maxDigits:]
	}
	for k := len(s); k < minDigits; k++ {
		b.WriteString(f.digits[0])
	}
	for i := 0; i < len(s); i++ {
		b.WriteString(f.digits[s[i]-'0'])
	}
}

// width index into DateTimeData name arrays: 0 abbreviated, 1 wide, 2 narrow, 3 short.
func textWidth(count int) int {
	switch count {
	case 4:
		return 1
	case 5:
		return 2
	case 6:
		return 3
	}
	return 0
}

func (f *DateTimeFormat) subFormat(b *strings.Builder, ch byte, count int, c *calFields) {
	dt := &f.loc.Data.DateTime
	const maxInt = 10
	switch ch {
	case 'G':
		w := 0
		switch count {
		case 4:
			w = 1
		case 5:
			w = 2
		}
		b.WriteString(dt.Eras[w][c.era])
	case 'y', 'Y', 'U':
		if count == 2 {
			f.zeroPad(b, c.year, 2, 2)
		} else {
			f.zeroPad(b, c.year, count, maxInt)
		}
	case 'u', 'r':
		f.zeroPad(b, c.extYear, count, maxInt)
	case 'M', 'L':
		ctx := 0
		if ch == 'L' {
			ctx = 1
		}
		switch {
		case count == 5:
			b.WriteString(dt.Months[ctx][2][c.month])
		case count == 4:
			b.WriteString(dt.Months[ctx][1][c.month])
		case count == 3:
			b.WriteString(dt.Months[ctx][0][c.month])
		default:
			f.zeroPad(b, c.month+1, count, maxInt)
		}
	case 'd':
		f.zeroPad(b, c.day, count, maxInt)
	case 'k':
		h := c.hour
		if h == 0 {
			h = 24
		}
		f.zeroPad(b, h, count, maxInt)
	case 'H':
		f.zeroPad(b, c.hour, count, maxInt)
	case 'h':
		h := c.hour % 12
		if h == 0 {
			h = 12
		}
		f.zeroPad(b, h, count, maxInt)
	case 'K':
		f.zeroPad(b, c.hour%12, count, maxInt)
	case 'm':
		f.zeroPad(b, c.minute, count, maxInt)
	case 's':
		f.zeroPad(b, c.second, count, maxInt)
	case 'S':
		v := c.ms
		switch count {
		case 1:
			v /= 100
		case 2:
			v /= 10
		}
		f.zeroPad(b, v, min(count, 3), maxInt)
		if count > 3 {
			f.zeroPad(b, 0, count-3, maxInt)
		}
	case 'e':
		if count < 3 {
			f.zeroPad(b, f.localDayOfWeek(c.dow), count, maxInt)
			return
		}
		b.WriteString(dt.Days[0][textWidth(count)][c.dow-1])
	case 'E':
		b.WriteString(dt.Days[0][textWidth(count)][c.dow-1])
	case 'c':
		if count < 3 {
			f.zeroPad(b, f.localDayOfWeek(c.dow), 1, maxInt)
			return
		}
		b.WriteString(dt.Days[1][textWidth(count)][c.dow-1])
	case 'a':
		f.amPm(b, count, c)
	case 'b':
		if c.hour == 12 && (!c.hasMin || c.minute == 0) && (!c.hasSec || c.second == 0) {
			if s := dt.DayPeriods[periodWidth(count)][1]; s != "" {
				b.WriteString(s)
				return
			}
		}
		f.amPm(b, count, c)
	case 'B':
		f.flexibleDayPeriod(b, count, c)
	case 'D':
		f.zeroPad(b, c.doy, count, maxInt)
	case 'F':
		f.zeroPad(b, (c.day-1)/7+1, count, maxInt)
	case 'A':
		f.zeroPad(b, ((c.hour*60+c.minute)*60+c.second)*1000+c.ms, count, maxInt)
	case 'g':
		days := c.local.Unix() / 86400
		if c.local.Unix() < 0 && c.local.Unix()%86400 != 0 {
			days--
		}
		f.zeroPad(b, int(days)+2440588, count, maxInt)
	case 'Q', 'q':
		f.zeroPad(b, c.month/3+1, count, maxInt)
	case 'z':
		style := tzSpecificShort
		if count >= 4 {
			style = tzSpecificLong
		}
		b.WriteString(f.zf.format(style, f.zone, c.t))
	case 'v':
		switch count {
		case 1:
			b.WriteString(f.zf.format(tzGenericShort, f.zone, c.t))
		case 4:
			b.WriteString(f.zf.format(tzGenericLong, f.zone, c.t))
		}
	case 'V':
		switch count {
		case 1:
			b.WriteString(f.zf.format(tzShortID, f.zone, c.t))
		case 2:
			b.WriteString(f.zf.format(tzID, f.zone, c.t))
		case 3:
			b.WriteString(f.zf.format(tzExemplar, f.zone, c.t))
		case 4:
			b.WriteString(f.zf.format(tzGenericLocation, f.zone, c.t))
		}
	case 'O':
		switch count {
		case 1:
			b.WriteString(f.zf.localizedGMT(f.zone.state(c.t).offset, true))
		case 4:
			b.WriteString(f.zf.localizedGMT(f.zone.state(c.t).offset, false))
		}
	case 'Z':
		off := f.zone.state(c.t).offset
		switch {
		case count < 4:
			b.WriteString(iso8601(off, true, false, false, false))
		case count == 5:
			b.WriteString(iso8601(off, false, true, false, false))
		default:
			b.WriteString(f.zf.localizedGMT(off, false))
		}
	case 'X', 'x':
		off := f.zone.state(c.t).offset
		utc := ch == 'X'
		switch count {
		case 1:
			b.WriteString(iso8601(off, true, utc, true, true))
		case 2:
			b.WriteString(iso8601(off, true, utc, false, true))
		case 3:
			b.WriteString(iso8601(off, false, utc, false, true))
		case 4:
			b.WriteString(iso8601(off, true, utc, false, false))
		case 5:
			b.WriteString(iso8601(off, false, utc, false, false))
		}
	case 'w', 'W':
		week := (c.doy-1)/7 + 1
		if ch == 'W' {
			week = (c.day-1)/7 + 1
		}
		f.zeroPad(b, week, count, maxInt)
	}
}

// localDayOfWeek is ICU's UCAL_DOW_LOCAL: 1 on the region's first day of the week.
func (f *DateTimeFormat) localDayOfWeek(dow int) int {
	first := 0
	if d := f.loc.set; d != nil {
		region := f.loc.region()
		for _, key := range []string{region, "001"} {
			found := false
			for _, w := range d.Weeks {
				if w.Region == key {
					first, found = int(w.FirstDay), true
					break
				}
			}
			if found {
				break
			}
		}
	}
	return (dow-1-first+14)%7 + 1
}

func (f *DateTimeFormat) amPm(b *strings.Builder, count int, c *calFields) {
	w := 0
	switch count {
	case 4:
		w = 1
	case 5:
		w = 2
	}
	pm := 0
	if c.hour >= 12 {
		pm = 1
	}
	b.WriteString(f.loc.Data.DateTime.AmPm[w][pm])
}

// periodWidth maps a b/B field length to the DayPeriods width (abbreviated for 1–3, narrow for 5, wide
// otherwise).
func periodWidth(count int) int {
	switch {
	case count <= 3:
		return 0
	case count == 5:
		return 2
	}
	return 1
}

// ICU day period indexes (DayPeriodRules::DayPeriod).
const (
	periodMidnight = 0
	periodNoon     = 1
	periodAM       = 10
	periodPM       = 11
)

func (f *DateTimeFormat) flexibleDayPeriod(b *strings.Builder, count int, c *calFields) {
	dt := &f.loc.Data.DateTime
	rules := &dt.DayPeriodRules
	if rules.Hours[0] < 0 {
		f.amPm(b, count, c)
		return
	}
	minute, second := 0, 0
	if c.hasMin {
		minute = c.minute
	}
	if c.hasSec {
		second = c.second
	}
	period := int(rules.Hours[c.hour])
	switch {
	case c.hour == 0 && minute == 0 && second == 0 && rules.Midnight:
		period = periodMidnight
	case c.hour == 12 && minute == 0 && second == 0 && rules.Noon:
		period = periodNoon
	}
	w := periodWidth(count)
	text := ""
	if period != periodAM && period != periodPM && period != periodMidnight {
		text = dt.DayPeriods[w][period]
	}
	if text == "" && (period == periodMidnight || period == periodNoon) {
		period = int(rules.Hours[c.hour])
		if period < periodAM {
			text = dt.DayPeriods[w][period]
		}
	}
	if period == periodAM || period == periodPM || text == "" {
		f.amPm(b, count, c)
		return
	}
	b.WriteString(text)
}
