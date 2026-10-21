package intl

import (
	"context"

	"github.com/gsxhq/gsx"
)

// Render is Value's gsx renderer (gsx.toml [renderers], the gsx Package renderers pattern): the result in the
// request's language, time zone and clock, in an element whose data-intl, data-method, data-opts and data-args are the
// call js/intl-browser.mjs redoes in the viewer's own zone: <time> for instants, <data> for numbers and durations,
// else <span>. Since and Zone are data-intl "since" and "zone".
func Render(ctx context.Context, v Value) gsx.Node {
	text, call := v.text(ctx), v.Attrs()
	switch {
	case v.kind == "since": // its call carries its datetime
		return <time { call... }>{ text }</time>
	case v.tag == "time":
		return <time datetime={v.machine} { call... }>{ text }</time>
	case v.tag == "data":
		return <data value={v.machine} { call... }>{ text }</data>
	}
	return <span { call... }>{ text }</span>
}

// Attrs is v's call as js/intl-browser.mjs reads it, for an element that can't hold Render's (an <option>): spread it
// there with v.Plain() as the text, <option { v.Attrs()... }>{ v.Plain() }</option>.
func (v Value) Attrs() gsx.Attrs {
	switch v.kind {
	case "since":
		return gsx.Attrs{{Key: "datetime", Value: v.machine}, {Key: "data-intl", Value: "since"}}
	case "zone":
		return gsx.Attrs{{Key: "data-intl", Value: "zone"}, {Key: "data-args", Value: "[" + JSON(v.zone) + "]"}}
	}
	a := gsx.Attrs{{Key: "data-intl", Value: v.API}}
	if v.Method != "" {
		a = append(a, gsx.Attr{Key: "data-method", Value: v.Method})
	}
	if v.Opts != "" {
		a = append(a, gsx.Attr{Key: "data-opts", Value: v.Opts})
	}
	return append(a, gsx.Attr{Key: "data-args", Value: v.ArgsJSON()})
}
