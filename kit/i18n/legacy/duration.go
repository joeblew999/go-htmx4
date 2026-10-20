package i18n

import (
	"errors"
	"strconv"
	"strings"
)

// DurationUnit indexes the units of a Duration, largest first (ECMA-402 DurationFormat table order).
type DurationUnit uint8

// Duration units.
const (
	DurYears DurationUnit = iota
	DurMonths
	DurWeeks
	DurDays
	DurHours
	DurMinutes
	DurSeconds
	DurMilliseconds
	DurMicroseconds
	DurNanoseconds
	DurationUnitCount
)

var durUnitNames = [DurationUnitCount]string{"years", "months", "weeks", "days", "hours", "minutes", "seconds", "milliseconds", "microseconds", "nanoseconds"}

// numberFormat units (singular) for each duration unit.
var durNumberUnits = [DurationUnitCount]string{"year", "month", "week", "day", "hour", "minute", "second", "millisecond", "microsecond", "nanosecond"}

func (u DurationUnit) String() string { return durUnitNames[u] }

// Duration is an Intl.DurationFormat duration record: whole numbers, all of the same sign.
type Duration [DurationUnitCount]int64

// DurationStyle is Intl.DurationFormat's style; the zero value is "short".
type DurationStyle uint8

// Duration styles.
const (
	DurationShort DurationStyle = iota
	DurationLong
	DurationNarrow
	DurationDigital
)

// UnitStyle is a per-unit style option of Intl.DurationFormat; the zero value means unset.
type UnitStyle uint8

// Unit styles. UnitFractional is resolved internally (numeric milli-, micro- and nanoseconds).
const (
	UnitUnset UnitStyle = iota
	UnitLong
	UnitShort
	UnitNarrow
	UnitNumeric
	Unit2Digit
	UnitFractional
)

// UnitDisplay is a per-unit display option; the zero value means unset.
type UnitDisplay uint8

// Unit displays.
const (
	DisplayUnset UnitDisplay = iota
	DisplayAuto
	DisplayAlways
)

// DurationOptions mirror Intl.DurationFormat's options.
type DurationOptions struct {
	Style            DurationStyle
	Units            [DurationUnitCount]UnitStyle   // e.g. Units[DurHours] = UnitNumeric
	Display          [DurationUnitCount]UnitDisplay // e.g. Display[DurHours] = DisplayAlways
	FractionalDigits *int                           // 0–9
	NumberingSystem  string
}

type durUnitOpts struct {
	style   UnitStyle
	display UnitDisplay
}

// DurationFormat formats durations for one locale and set of options (Intl.DurationFormat).
type DurationFormat struct {
	loc     *Locale
	style   DurationStyle
	units   [DurationUnitCount]durUnitOpts
	fracDig *int
	nu      string
}

// DurationFormat compiles options for this locale.
func (l *Locale) DurationFormat(o DurationOptions) (*DurationFormat, error) {
	f := &DurationFormat{loc: l, style: o.Style, nu: o.NumberingSystem}
	if o.FractionalDigits != nil {
		if *o.FractionalDigits < 0 || *o.FractionalDigits > 9 {
			return nil, rangeErr("fractionalDigits")
		}
		d := *o.FractionalDigits
		f.fracDig = &d
	}
	baseStyles := [...]UnitStyle{DurationShort: UnitShort, DurationLong: UnitLong, DurationNarrow: UnitNarrow}
	prev := UnitUnset
	for u := DurYears; u < DurationUnitCount; u++ {
		style := o.Units[u]
		if style == UnitFractional || u < DurHours && (style == UnitNumeric || style == Unit2Digit) {
			return nil, rangeErr(u.String() + " style")
		}
		displayDefault := DisplayAlways
		if style == UnitUnset {
			switch {
			case o.Style == DurationDigital:
				style = UnitShort
				if u >= DurHours {
					style = UnitNumeric
					if u == DurMinutes || u == DurSeconds {
						style = Unit2Digit
					}
				}
				if u != DurHours && u != DurMinutes && u != DurSeconds {
					displayDefault = DisplayAuto
				}
			case prev == UnitFractional || prev == UnitNumeric || prev == Unit2Digit:
				style = UnitNumeric
				if u != DurMinutes && u != DurSeconds {
					displayDefault = DisplayAuto
				}
			default:
				style = baseStyles[o.Style]
				displayDefault = DisplayAuto
			}
		}
		if style == UnitNumeric && u >= DurMilliseconds {
			style = UnitFractional
			displayDefault = DisplayAuto
		}
		display := o.Display[u]
		if display == DisplayUnset {
			display = displayDefault
		}
		if display == DisplayAlways && style == UnitFractional {
			return nil, rangeErr(u.String() + "Display always with fractional style")
		}
		if prev == UnitFractional && style != UnitFractional {
			return nil, rangeErr(u.String() + " style after a fractional unit")
		}
		if (prev == UnitNumeric || prev == Unit2Digit) && style != UnitFractional && style != UnitNumeric && style != Unit2Digit {
			return nil, rangeErr(u.String() + " style after a numeric unit")
		}
		if (u == DurMinutes || u == DurSeconds) && (prev == UnitNumeric || prev == Unit2Digit) {
			style = Unit2Digit
		}
		f.units[u] = durUnitOpts{style, display}
		if u >= DurHours && u <= DurMicroseconds {
			prev = style
		}
	}
	return f, nil
}

// ErrDuration is returned for an invalid duration (mixed signs).
var ErrDuration = errors.New("i18n: invalid duration")

func (d *Duration) sign() int {
	for _, v := range d {
		if v < 0 {
			return -1
		}
		if v > 0 {
			return 1
		}
	}
	return 0
}

// Format formats d. It follows V8's DurationRecordToListOfFormattedNumber (Chrome's output), which differs
// from the spec text in one visible way: a numeric or 2-digit seconds (or minutes after numeric hours) is
// appended to the previous item with the time separator, even after a unit-style minutes ("5 min:3.00").
func (f *DurationFormat) Format(d Duration) (string, error) {
	sign := d.sign()
	for _, v := range d {
		if v != 0 && (v < 0) != (sign < 0) {
			return "", ErrDuration
		}
	}
	w := &durWriter{f: f, negative: sign < 0, dns: true}
	for u := DurYears; u <= DurDays; u++ {
		w.longShortNarrow(u, Int(d[u]), false)
	}
	hs := f.units[DurHours].style
	w.numericOr2Digit(DurHours, Int(d[DurHours]), false, false)
	required := (hs == UnitNumeric || hs == Unit2Digit) &&
		(f.units[DurHours].display == DisplayAlways || d[DurHours] != 0) &&
		(f.units[DurSeconds].display == DisplayAlways || d[DurSeconds] != 0 || d[DurMilliseconds] != 0 || d[DurMicroseconds] != 0 || d[DurNanoseconds] != 0)
	w.numericOr2Digit(DurMinutes, Int(d[DurMinutes]), hs == UnitNumeric || hs == Unit2Digit, required)
	switch {
	case f.units[DurMilliseconds].style == UnitFractional:
		w.fractional(DurSeconds, decAdd(decAdd(decAdd(Int(d[DurSeconds]), Minor(d[DurMilliseconds], 3)), Minor(d[DurMicroseconds], 6)), Minor(d[DurNanoseconds], 9)))
	default:
		w.numericOr2Digit(DurSeconds, Int(d[DurSeconds]), true, false)
		switch {
		case f.units[DurMicroseconds].style == UnitFractional:
			w.fractional(DurMilliseconds, decAdd(decAdd(Int(d[DurMilliseconds]), Minor(d[DurMicroseconds], 3)), Minor(d[DurNanoseconds], 6)))
		default:
			w.longShortNarrowOrNumeric(DurMilliseconds, Int(d[DurMilliseconds]), false)
			switch {
			case f.units[DurNanoseconds].style == UnitFractional:
				w.fractional(DurMicroseconds, decAdd(Int(d[DurMicroseconds]), Minor(d[DurNanoseconds], 3)))
			default:
				w.longShortNarrowOrNumeric(DurMicroseconds, Int(d[DurMicroseconds]), false)
				w.longShortNarrowOrNumeric(DurNanoseconds, Int(d[DurNanoseconds]), false)
			}
		}
	}
	style := TextShort
	switch f.style {
	case DurationLong:
		style = TextLong
	case DurationNarrow:
		style = TextNarrow
	}
	return f.loc.ListFormat(ListOptions{Type: ListUnit, Style: style}).Format(w.strs), nil
}

type durWriter struct {
	f        *DurationFormat
	negative bool
	dns      bool // display negative sign (on the first output only)
	strs     []string
}

func (w *durWriter) output(value Decimal, o NumberOptions, addToLast bool) {
	if w.dns {
		w.dns = false
		if value.IsZero() && w.negative {
			value.neg = true
		}
	} else {
		o.SignDisplay = SignNever
	}
	o.NumberingSystem = w.f.nu
	s := w.f.loc.MustNumberFormat(o).Format(value)
	if addToLast && len(w.strs) > 0 {
		w.strs[len(w.strs)-1] += w.f.timeSep() + s
		return
	}
	w.strs = append(w.strs, s)
}

func unitOptions(u DurationUnit, style UnitStyle) NumberOptions {
	o := NumberOptions{Style: StyleUnit, Unit: durNumberUnits[u], UnitDisplay: Short}
	switch style {
	case UnitLong:
		o.UnitDisplay = Long
	case UnitNarrow:
		o.UnitDisplay = Narrow
	}
	return o
}

func (w *durWriter) longShortNarrow(u DurationUnit, value Decimal, addToLast bool) {
	uo := w.f.units[u]
	if value.IsZero() && uo.display == DisplayAuto {
		return
	}
	w.output(value, unitOptions(u, uo.style), addToLast)
}

func (w *durWriter) longShortNarrowOrNumeric(u DurationUnit, value Decimal, addToLast bool) {
	uo := w.f.units[u]
	if value.IsZero() && uo.display == DisplayAuto {
		return
	}
	if uo.style == UnitNumeric {
		w.output(value, NumberOptions{UseGrouping: GroupingOff}, addToLast)
		return
	}
	w.longShortNarrow(u, value, addToLast)
}

func (w *durWriter) numericOr2Digit(u DurationUnit, value Decimal, maybeAddToLast, required bool) {
	uo := w.f.units[u]
	if value.IsZero() && uo.display == DisplayAuto && !required {
		return
	}
	if uo.style == Unit2Digit {
		w.output(value, NumberOptions{MinimumIntegerDigits: 2, UseGrouping: GroupingOff}, maybeAddToLast)
		return
	}
	w.longShortNarrowOrNumeric(u, value, maybeAddToLast && uo.style == UnitNumeric)
}

// fractional outputs u with the following fractional units folded in (V8 OutputFractional).
func (w *durWriter) fractional(u DurationUnit, value Decimal) {
	uo := w.f.units[u]
	if value.IsZero() && uo.display == DisplayAuto {
		return
	}
	var o NumberOptions
	addToLast := false
	switch uo.style {
	case Unit2Digit:
		o = NumberOptions{MinimumIntegerDigits: 2, UseGrouping: GroupingOff}
		addToLast = true
	case UnitNumeric:
		o = NumberOptions{UseGrouping: GroupingOff}
		addToLast = true
	default:
		o = unitOptions(u, uo.style)
	}
	w.f.fractionOptions(&o)
	if w.dns {
		w.dns = false
	} else {
		o.SignDisplay = SignNever
	}
	o.NumberingSystem = w.f.nu
	s := w.f.loc.MustNumberFormat(o).Format(value)
	if addToLast && len(w.strs) > 0 {
		w.strs[len(w.strs)-1] += w.f.timeSep() + s
		return
	}
	w.strs = append(w.strs, s)
}

// timeSep is the locale's time separator for the numbering system in use, as V8 maps it: ":", ".", "：" or "٫".
func (f *DurationFormat) timeSep() string {
	sys, _ := f.loc.numberSystem(cmpOr(f.nu, cmpOr(f.loc.Tag.Keyword("nu"), f.loc.Data.Numbers.DefaultSystem)))
	switch sys.Symbols.TimeSeparator {
	case ".", "\uFF1A", "\u066B":
		return sys.Symbols.TimeSeparator
	}
	return ":"
}

func (f *DurationFormat) fractionOptions(o *NumberOptions) {
	if f.fracDig == nil {
		o.MaximumFractionDigits, o.MinimumFractionDigits = N(9), N(0)
	} else {
		o.MaximumFractionDigits, o.MinimumFractionDigits = N(*f.fracDig), N(*f.fracDig)
	}
	o.RoundingMode = Trunc
}

// decAdd adds two decimals of the same sign (duration fields always share one).
func decAdd(a, b Decimal) Decimal {
	if a.IsZero() {
		if b.IsZero() {
			return Decimal{neg: a.neg || b.neg}
		}
		return b
	}
	if b.IsZero() {
		return a
	}
	// align both to the smallest exponent of their last digit
	lowA, lowB := a.exp-len(a.digits), b.exp-len(b.digits)
	low := min(lowA, lowB)
	ia := a.digits + strings.Repeat("0", lowA-low)
	ib := b.digits + strings.Repeat("0", lowB-low)
	for len(ia) < len(ib) {
		ia = "0" + ia
	}
	for len(ib) < len(ia) {
		ib = "0" + ib
	}
	sum := make([]byte, len(ia)+1)
	carry := 0
	for i := len(ia) - 1; i >= 0; i-- {
		v := int(ia[i]-'0') + int(ib[i]-'0') + carry
		sum[i+1] = byte('0' + v%10)
		carry = v / 10
	}
	sum[0] = byte('0' + carry)
	out := Decimal{neg: a.neg, digits: string(sum), exp: len(sum) + low}
	out.normalize()
	return out
}

// ParseDuration reads Intl-style fields ("hours": 1, …) into a Duration; unknown keys are an error.
func ParseDuration(fields map[string]int64) (Duration, error) {
	var d Duration
	if len(fields) == 0 {
		return d, errors.New("i18n: duration has no fields")
	}
	for k, v := range fields {
		found := false
		for u, name := range durUnitNames {
			if k == name {
				d[u] = v
				found = true
			}
		}
		if !found {
			return d, errors.New("i18n: unknown duration field " + strconv.Quote(k))
		}
	}
	return d, nil
}
