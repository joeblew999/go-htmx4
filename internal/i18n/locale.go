package i18n

import (
	"strings"
	"sync"
)

// Locale is a shipped locale: its BCP 47 id, and what the runtime's Intl says about it (its name in its own
// language, text direction, likely subtags), asked once per runtime and kept: it is the same for every request.
// Make one per id with NewLocale and share it.
type Locale struct {
	id                 string
	once               sync.Once
	name, dir, maximal string
}

// NewLocale returns the locale with a BCP 47 id, e.g. "pt-BR".
func NewLocale(id string) *Locale { return &Locale{id: id} }

// ID is the locale id, e.g. "pt-BR".
func (l *Locale) ID() string { return l.id }

// Lang is the locale id for <html lang>, hreflang and Intl.
func (l *Locale) Lang() string { return l.id }

// Name is the locale's name in its own language (Intl.DisplayNames), e.g. "português (Brasil)".
func (l *Locale) Name() string { l.ask(); return l.name }

// Dir is "rtl" or "ltr" (Intl.Locale's text direction), for <html dir>.
func (l *Locale) Dir() string { l.ask(); return l.dir }

// Maximal is the id with its likely subtags (Intl.Locale maximize), e.g. "pt-Latn-BR".
func (l *Locale) Maximal() string { l.ask(); return l.maximal }

// ask fills the Intl facts. With no JS runtime (go test) they are the id and "ltr".
func (l *Locale) ask() {
	l.once.Do(func() {
		get := func(c Call, none string) string {
			if s, ok := l.Format(c, "", ""); ok {
				return s
			}
			return none
		}
		l.name = get(Call{Kind: "display-name", Value: l.id}, l.id)
		l.dir = get(Call{Kind: "locale", Value: "direction"}, "ltr")
		l.maximal = get(Call{Kind: "locale", Value: "maximize"}, l.id)
	})
}

// Locales are the locales an app ships, its source (default) locale first.
type Locales []*Locale

// Find returns the shipped locale with id, compared case-insensitively ("pt-br" finds pt-BR), or nil.
func (s Locales) Find(id string) *Locale {
	for _, l := range s {
		if strings.EqualFold(l.id, id) {
			return l
		}
	}
	return nil
}
