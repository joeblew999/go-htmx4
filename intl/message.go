package intl

import (
	"context"
	"math"
	"strconv"
)

// Msg is a translated message, written in markup where its text goes: gsx renders it (Translate, registered in
// gsx.toml [renderers]) in the request's language. go tool intl gen writes one per catalog key, so a missing key is a
// compile error: a value for a message without arguments, a constructor for one with.
//
//	<h1>{ locales.HomeHeading }</h1>
//	<p title={ locales.GreetHello(name) }>{ locales.CartItems(len(items)) }</p>
//	<title>{ locales.PageTitle(title) }</title>   (title a Msg: a message can be another's argument)
type Msg struct {
	index int
	args  []Arg
}

// Arg is a message argument: text (Str), or a number (Num).
type Arg struct {
	s    string
	num  bool
	text Textual // a Msg or Value argument, made text in the request's language when the message renders
}

// TextArg is what a message's text argument can be: a string, another message, or an Intl Value ({ zone } in
// "Automatic ({zone})" as the viewer's zone's name), the last two in the request's language.
type TextArg interface {
	string | Msg | Value | Plain
}

// NumberArg is what a message's number argument can be: any Go number.
type NumberArg interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~float32 | ~float64
}

// Str is a text argument. js/intl.mjs wraps it in a first-strong isolate (FSI … PDI) where it lands in the message, so
// an Arabic name can't reorder the English words around it; select matches the value itself.
func Str[T TextArg](v T) Arg {
	switch x := any(v).(type) {
	case Msg:
		return Arg{text: x}
	case Value:
		return Arg{text: x}
	case Plain:
		return Arg{text: Value(x)}
	case string:
		return Arg{s: x}
	}
	return Arg{}
}

// Num is a number argument, formatted by Intl.NumberFormat in the message's language.
func Num[N NumberArg](n N) Arg {
	switch x := any(n).(type) {
	case int:
		return Arg{s: strconv.Itoa(x), num: true}
	case int64:
		return Arg{s: strconv.FormatInt(x, 10), num: true}
	}
	f := float64(n)
	s := strconv.FormatFloat(f, 'f', -1, 64)
	switch {
	case math.IsInf(f, 1):
		s = "Infinity"
	case math.IsInf(f, -1):
		s = "-Infinity"
	}
	return Arg{s: s, num: true}
}

// M is message index of the catalogs with args: what the generated constructors return.
func M(index int, args ...Arg) Msg { return Msg{index: index, args: args} }

// Pseudo makes every Msg render from the en-XA pseudo-localized catalog (accented, bracketed, ~40% longer): tests
// render pages with it to find text that doesn't come from a catalog.
var Pseudo bool

// Translate is Msg's gsx renderer: the message in the request's language (the source's without a request).
func Translate(ctx context.Context, m Msg) string {
	loc := Current(ctx)
	catalog := shipped[0].id
	switch {
	case Pseudo:
		catalog = "en-XA"
	case shipped.Find(loc.id) == loc:
		catalog = loc.id
	}
	args := make([]string, len(m.args))
	for i, a := range m.args {
		args[i] = a.s
		if a.text != nil {
			args[i] = a.text.text(ctx)
		}
	}
	return loc.Message(catalog, m.index, args...)
}

// Message formats message index of catalog (a shipped locale id, or "en-XA") in the locale's language with
// js/intl.mjs: ICU MessageFormat over Intl.PluralRules and Intl.NumberFormat. A message without arguments is the
// same for every request: asked once, then kept.
func (l *Locale) Message(catalog string, index int, args ...string) string {
	if len(args) > 0 {
		s, _ := call("message", append([]string{l.id, catalog, strconv.Itoa(index)}, args...)...)
		return s
	}
	key := messageKey{catalog, index}
	l.mu.Lock()
	s, ok := l.plain[key]
	l.mu.Unlock()
	if !ok {
		s, _ = call("message", l.id, catalog, strconv.Itoa(index))
		l.mu.Lock()
		if l.plain == nil {
			l.plain = map[messageKey]string{}
		}
		l.plain[key] = s
		l.mu.Unlock()
	}
	return s
}

type messageKey struct {
	catalog string
	index   int
}
