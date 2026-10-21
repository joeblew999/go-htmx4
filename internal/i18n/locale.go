package i18n

import "strings"

// Locale is a shipped locale, resolved for a request. It is immutable and safe to share.
type Locale struct {
	Data *LocaleData
}

// ID is the locale id, e.g. "pt-BR".
func (l *Locale) ID() string { return l.Data.ID }

// Lang is the locale id for <html lang> and hreflang.
func (l *Locale) Lang() string { return l.Data.ID }

// Dir is "rtl" or "ltr", for <html dir>.
func (l *Locale) Dir() string {
	if l.Data.RTL {
		return "rtl"
	}
	return "ltr"
}

// Find returns the shipped locale data with id, compared case-insensitively ("pt-br" finds pt-BR).
func (d *Data) Find(id string) *LocaleData {
	for _, ld := range d.Locales {
		if strings.EqualFold(ld.ID, id) {
			return ld
		}
	}
	return nil
}

// Locale returns the shipped locale id (case-insensitive); ok is false when it isn't shipped.
func (d *Data) Locale(id string) (l *Locale, ok bool) {
	if ld := d.Find(id); ld != nil {
		return &Locale{Data: ld}, true
	}
	return nil, false
}

// MustLocale is Locale for an id that must be shipped.
func (d *Data) MustLocale(id string) *Locale {
	l, ok := d.Locale(id)
	if !ok {
		panic("i18n: locale " + id + " is not shipped")
	}
	return l
}

// Parent returns the nearest shipped ancestor of the shipped locale id on CLDR's parent chain, or "root".
func (d *Data) Parent(id string) string {
	if ld := d.Find(id); ld != nil && ld.Parent != "" {
		return ld.Parent
	}
	return "root"
}
