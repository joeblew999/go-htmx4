package i18n

import (
	"errors"
	"slices"
	"strings"
)

// sanctionedUnits are ECMA-402's simple unit identifiers (IsSanctionedSingleUnitIdentifier).
var sanctionedUnits = []string{"acre", "bit", "byte", "celsius", "centimeter", "day", "degree", "fahrenheit", "fluid-ounce", "foot", "gallon", "gigabit", "gigabyte", "gram", "hectare", "hour", "inch", "kilobit", "kilobyte", "kilogram", "kilometer", "liter", "megabit", "megabyte", "meter", "microsecond", "mile", "mile-scandinavian", "milliliter", "millimeter", "millisecond", "minute", "month", "nanosecond", "ounce", "percent", "petabyte", "pound", "second", "stone", "terabit", "terabyte", "week", "yard", "year"}

// IsUnitIdentifier reports whether unit is a valid Intl.NumberFormat unit: a sanctioned simple unit or
// "<simple>-per-<simple>".
func IsUnitIdentifier(unit string) bool {
	if slices.Contains(sanctionedUnits, unit) {
		return true
	}
	x, y, ok := strings.Cut(unit, "-per-")
	return ok && slices.Contains(sanctionedUnits, x) && slices.Contains(sanctionedUnits, y)
}

func (d *UnitData) find(id string) *UnitPatterns {
	i, ok := slices.BinarySearchFunc(d.Units, id, func(u UnitPatterns, s string) int { return strings.Compare(u.ID, s) })
	if !ok {
		return nil
	}
	return &d.Units[i]
}

// compileUnit resolves the unit option: CLDR's own unit when it has one ("kilometer-per-hour"), else the
// numerator and denominator of an X-per-Y compound.
func (f *NumberFormat) compileUnit() error {
	unit := f.opts.Unit
	if !IsUnitIdentifier(unit) {
		return errors.New("i18n: unit style needs a sanctioned unit identifier, got " + quote(unit))
	}
	ud := &f.loc.Data.Units
	if u := ud.find(unit); u != nil {
		f.unit = u
		return nil
	}
	x, y, _ := strings.Cut(unit, "-per-")
	f.unit, f.perUnit = ud.find(x), ud.find(y)
	if f.unit == nil || f.perUnit == nil {
		return errors.New("i18n: no CLDR data for unit " + quote(unit))
	}
	return nil
}

// applyUnit puts a formatted number into the unit pattern for its plural category (ICU LongNameHandler):
// "{0} km"; for X-per-Y without a CLDR unit, the denominator's perUnitPattern ("{0}/h") or the locale's
// compoundUnitPattern with the denominator's bare name ("{0} per {1}").
func (f *NumberFormat) applyUnit(num string, cat PluralCat) string {
	w := f.opts.UnitDisplay
	if w >= WidthCount {
		w = Short
	}
	pats := &f.unit.Patterns[w]
	s := strings.Replace(cmpOr(pats[cat], pats[Other]), "{0}", num, 1)
	if f.perUnit == nil {
		return s
	}
	if per := f.perUnit.PerUnit[w]; per != "" {
		return strings.Replace(per, "{0}", s, 1)
	}
	dp := &f.perUnit.Patterns[w]
	name := strings.TrimSpace(strings.Replace(cmpOr(dp[One], dp[Other]), "{0}", "", 1))
	name = strings.Trim(name, "  ")
	compound := cmpOr(f.loc.Data.Units.Per[w], "{0}/{1}")
	return strings.NewReplacer("{0}", s, "{1}", name).Replace(compound)
}
