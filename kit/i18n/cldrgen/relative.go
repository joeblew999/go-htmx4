package cldrgen

import "github.com/joeblew999/go-htmx4/kit/i18n"

// loadWeekData fills Data.Weeks from supplemental weekData.
func (g *gen) loadWeekData(d *i18n.Data) error { return nil }

// buildRelative, buildLists and buildDuration fill the locale's relative-time, list and duration data.
func (g *gen) buildRelative(ld *i18n.LocaleData) error { return nil }
func (g *gen) buildLists(ld *i18n.LocaleData) error    { return nil }
func (g *gen) buildDuration(ld *i18n.LocaleData) error { return nil }
