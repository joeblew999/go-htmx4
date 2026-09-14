package cldrgen

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/joeblew999/go-htmx4/kit/i18n"
)

var weekdayIndex = map[string]int8{"sun": 0, "mon": 1, "tue": 2, "wed": 3, "thu": 4, "fri": 5, "sat": 6}

// loadWeekData fills Data.Weeks from supplemental weekData: first day, minimal days and weekend per region.
func (g *gen) loadWeekData(d *i18n.Data) error {
	wd, err := g.src.read("cldr-core/supplemental/weekData.json")
	if err != nil {
		return err
	}
	w := mapAt(wd, "supplemental", "weekData")
	regions := map[string]*i18n.RegionWeek{}
	get := func(r string) *i18n.RegionWeek {
		if rw, ok := regions[r]; ok {
			return rw
		}
		rw := &i18n.RegionWeek{Region: r, FirstDay: -1, MinDays: -1}
		regions[r] = rw
		return rw
	}
	for r, v := range mapAt(w, "firstDay") {
		if strings.Contains(r, "-alt-") {
			continue
		}
		get(r).FirstDay = weekdayIndex[v.(string)]
	}
	for r, v := range mapAt(w, "minDays") {
		n, _ := strconv.Atoi(v.(string))
		get(r).MinDays = int8(n)
	}
	starts, ends := mapAt(w, "weekendStart"), mapAt(w, "weekendEnd")
	for r := range starts {
		get(r)
	}
	world := regions["001"]
	if world == nil {
		return fmt.Errorf("weekData has no 001")
	}
	for _, r := range sortedKeys(regions) {
		rw := regions[r]
		if rw.FirstDay < 0 {
			rw.FirstDay = weekdayIndex[str(w, "firstDay", "001")]
		}
		if rw.MinDays < 0 {
			n, _ := strconv.Atoi(str(w, "minDays", "001"))
			rw.MinDays = int8(n)
		}
		start, end := str(starts, r), str(ends, r)
		if start == "" {
			start = str(starts, "001")
		}
		if end == "" {
			end = str(ends, "001")
		}
		for day := weekdayIndex[start]; ; day = (day + 1) % 7 {
			rw.Weekend = append(rw.Weekend, day)
			if day == weekdayIndex[end] || len(rw.Weekend) == 7 {
				break
			}
		}
		d.Weeks = append(d.Weeks, *rw)
	}
	return nil
}

var widthSuffix = [i18n.WidthCount]string{i18n.Short: "-short", i18n.Long: "", i18n.Narrow: "-narrow"}

// buildRelative reads dateFields.json (relative phrases and "in {0} days" patterns).
func (g *gen) buildRelative(ld *i18n.LocaleData) error {
	file, err := g.localeFile("cldr-dates-full", "dateFields.json", ld.DataID)
	if err != nil {
		return err
	}
	fields := mapAt(file, "dates", "fields")
	for u := i18n.RelYear; u < i18n.RelUnitCount; u++ {
		for w := 0; w < i18n.WidthCount; w++ {
			f := mapAt(fields, u.String()+widthSuffix[w])
			if f == nil {
				return fmt.Errorf("no dateFields %s%s", u, widthSuffix[w])
			}
			ru := &ld.Relative.Units[u][w]
			for k, v := range f {
				if off, ok := strings.CutPrefix(k, "relative-type-"); ok {
					o, err := strconv.Atoi(off)
					if err != nil {
						return fmt.Errorf("dateFields %s: %q", u, k)
					}
					ru.Phrases = append(ru.Phrases, i18n.RelPhrase{Offset: int8(o), Text: v.(string)})
				}
			}
			slices.SortFunc(ru.Phrases, func(a, b i18n.RelPhrase) int { return int(a.Offset) - int(b.Offset) })
			for c := i18n.Zero; c < i18n.PluralCatCount; c++ {
				ru.Future[c] = str(f, "relativeTime-type-future", "relativeTimePattern-count-"+c.String())
				ru.Past[c] = str(f, "relativeTime-type-past", "relativeTimePattern-count-"+c.String())
			}
		}
	}
	return nil
}

// buildLists reads listPatterns.json.
func (g *gen) buildLists(ld *i18n.LocaleData) error {
	file, err := g.localeFile("cldr-misc-full", "listPatterns.json", ld.DataID)
	if err != nil {
		return err
	}
	lp := mapAt(file, "listPatterns")
	types := [i18n.ListTypeCount]string{"standard", "or", "unit"}
	for t, typ := range types {
		for w := 0; w < i18n.WidthCount; w++ {
			p := mapAt(lp, "listPattern-type-"+typ+widthSuffix[w])
			if p == nil {
				return fmt.Errorf("no listPattern %s%s", typ, widthSuffix[w])
			}
			ld.Lists.Patterns[t][w] = i18n.ListPattern{Two: str(p, "2"), Start: str(p, "start"), Middle: str(p, "middle"), End: str(p, "end")}
		}
	}
	switch strings.SplitN(ld.ID, "-", 2)[0] {
	case "es":
		ld.Lists.Rule = i18n.ListRuleEs
	case "he":
		ld.Lists.Rule = i18n.ListRuleHe
	}
	return nil
}

// sanctionedUnits are ECMA-402's simple unit identifiers (IsSanctionedSingleUnitIdentifier).
var sanctionedUnits = []string{"acre", "bit", "byte", "celsius", "centimeter", "day", "degree", "fahrenheit", "fluid-ounce", "foot", "gallon", "gigabit", "gigabyte", "gram", "hectare", "hour", "inch", "kilobit", "kilobyte", "kilogram", "kilometer", "liter", "megabit", "megabyte", "meter", "microsecond", "mile", "mile-scandinavian", "milliliter", "millimeter", "millisecond", "minute", "month", "nanosecond", "ounce", "percent", "petabyte", "pound", "second", "stone", "terabit", "terabyte", "week", "yard", "year"}

// buildDuration reads units.json: unit patterns for Intl.NumberFormat style "unit" (sanctioned units and
// CLDR's own X-per-Y units of sanctioned units) and the digital duration patterns.
func (g *gen) buildDuration(ld *i18n.LocaleData) error {
	file, err := g.localeFile("cldr-units-full", "units.json", ld.DataID)
	if err != nil {
		return err
	}
	units := mapAt(file, "units")
	ld.Duration = i18n.DurationData{
		HM:  str(units, "durationUnit-type-hm", "durationUnitPattern"),
		HMS: str(units, "durationUnit-type-hms", "durationUnitPattern"),
		MS:  str(units, "durationUnit-type-ms", "durationUnitPattern"),
	}
	isSanctioned := func(id string) bool { return slices.Contains(sanctionedUnits, id) }
	byID := map[string]*i18n.UnitPatterns{}
	widths := [i18n.WidthCount]string{i18n.Short: "short", i18n.Long: "long", i18n.Narrow: "narrow"}
	for w, name := range widths {
		table := mapAt(units, name)
		ld.Units.Per[w] = str(table, "per", "compoundUnitPattern")
		for _, key := range sortedKeys(table) {
			cat, id, ok := strings.Cut(key, "-")
			if !ok || cat == "per" || cat == "times" || strings.HasSuffix(key, "-imperial") || strings.HasSuffix(key, "-metric") {
				continue
			}
			if cat == "10p" || cat == "1024p" || strings.HasPrefix(cat, "power") {
				continue
			}
			x, y, per := strings.Cut(id, "-per-")
			if !isSanctioned(id) && !(per && isSanctioned(x) && isSanctioned(y)) {
				continue
			}
			up := byID[id]
			if up == nil {
				up = &i18n.UnitPatterns{ID: id}
				byID[id] = up
			}
			u := mapAt(table, key)
			for c := i18n.Zero; c < i18n.PluralCatCount; c++ {
				up.Patterns[w][c] = str(u, "unitPattern-count-"+c.String())
			}
			up.PerUnit[w] = str(u, "perUnitPattern")
		}
	}
	for _, id := range sanctionedUnits {
		if byID[id] == nil {
			return fmt.Errorf("units.json has no %s", id)
		}
	}
	for _, id := range sortedKeys(byID) {
		ld.Units.Units = append(ld.Units.Units, *byID[id])
	}
	return nil
}
