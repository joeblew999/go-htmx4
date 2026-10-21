// Package filters is intl as gsx pipeline filters. Register the package in gsx.toml and every function here is a
// filter, its name with the first letter lowercased (gsx Configuration guide, filter_packages):
//
//	filter_packages = ["github.com/joeblew999/go-htmx4/intl/filters"]
//
//	{ note.Created |> since }                          vor 3 Minuten
//	{ order.Placed |> date(`{"dateStyle":"long"}`) }   4. Juli 2026
//	{ total |> number(`{"style":"currency","currency":"EUR"}`) }
//	{ "JP" |> intl("DisplayNames", "of", `{"type":"region"}`) }
//	title={ v |> plain }                               a Value's text, for an attribute
//	title={ locales.BoardTitle |> text }               a message's text, for a component's string input
//
// Each returns an intl.Value (rendered by intl.Render, in the request's language and redone by the browser in the
// viewer's zone), except plain and text. Options are an Intl options object as JSON, optional.
package filters

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/joeblew999/go-htmx4/intl"
)

// Date is t with Intl.DateTimeFormat: { t |> date } or { t |> date(`{"dateStyle":"full","timeStyle":"short"}`) }.
func Date(t time.Time, opts ...string) intl.Value { return intl.Date(t, first(opts)) }

// DateRange is t to end with Intl.DateTimeFormat's formatRange: { start |> dateRange(end) }.
func DateRange(t, end time.Time, opts ...string) intl.Value {
	return intl.DateRange(t, end, first(opts))
}

// Since is t relative to now with Intl.RelativeTimeFormat, kept fresh in the browser: { t |> since }.
func Since(t time.Time) intl.Value { return intl.Since(t) }

// Number is n with Intl.NumberFormat: an integer, a float, or a decimal string (exact: "1234.50").
// { n |> number }, { price |> number(`{"style":"currency","currency":"EUR"}`) }.
func Number(n any, opts ...string) intl.Value { return intl.Number(decimal(n), first(opts)) }

// Duration is d with Intl.DurationFormat: { uptime |> duration }, { d |> duration(`{"style":"long"}`) }.
func Duration(d time.Duration, opts ...string) intl.Value { return intl.Duration(d, first(opts)) }

// List is items with Intl.ListFormat: { names |> list }, { options |> list(`{"type":"disjunction"}`) }.
func List(items []string, opts ...string) intl.Value { return intl.List(items, first(opts)) }

// Zone is a time zone's name in the language: { "Europe/Berlin" |> zone }.
func Zone(id string) intl.Value { return intl.Zone(id) }

// Intl is any Intl call with v as its argument: new Intl[api](lang, opts)[method](v).
// { "JP" |> intl("DisplayNames", "of", `{"type":"region"}`) }, { 2 |> intl("PluralRules", "select") }.
func Intl(v any, api, method string, opts ...string) intl.Value {
	return intl.Of(api, method, first(opts), argJSON(v))
}

// Plain is a Value's text alone, for an attribute (which takes no markup): title={ v |> plain }.
func Plain(v intl.Value) intl.Plain { return v.Plain() }

// Text is a message's or a Value's text in the request's language, for where a renderer can't reach: a component
// input typed string (gsxui's Toast title={ locales.GreetToastTitle |> text }).
func Text(ctx context.Context, x intl.Textual) string { return intl.Text(ctx, x) }

func first(s []string) string {
	if len(s) > 0 {
		return s[0]
	}
	return ""
}

// decimal is n as a plain decimal string, exactly.
func decimal(n any) string {
	switch x := n.(type) {
	case string:
		return x
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case int32:
		return strconv.FormatInt(int64(x), 10)
	case uint64:
		return strconv.FormatUint(x, 10)
	case uint:
		return strconv.FormatUint(uint64(x), 10)
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(x), 'f', -1, 32)
	}
	b, _ := json.Marshal(n)
	return string(b)
}

// argJSON is v as an Intl argument: a string quoted, an instant in epoch milliseconds, a number as it is.
func argJSON(v any) string {
	switch x := v.(type) {
	case string:
		return intl.JSON(x)
	case time.Time:
		return intl.Millis(x)
	}
	return decimal(v)
}
