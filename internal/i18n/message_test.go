package i18n

import "testing"

// TestFormatMessage: the Go side of a message, with a stand-in for the runtime's Intl that records what it is asked.
// What a language answers (its plural categories, its digits) is tested on workerd: e2e/smoke and locales.
func TestFormatMessage(t *testing.T) {
	var asked []Call
	runtimeFormat = func(lang string, c Call, _, _ string) string {
		asked = append(asked, c)
		switch c.Kind {
		case "plural":
			return map[string]string{"1": "one", "3": "few"}[c.Value]
		case "number":
			return "<" + c.Value + c.Opts + ">"
		}
		return ""
	}
	t.Cleanup(func() { runtimeFormat = func(string, Call, string, string) string { return "" } })

	num := func(f float64) MsgArgValue { return FloatArg(f) }
	files := []MsgPart{{Kind: MsgPlural, Arg: 0, Cases: []MsgCase{
		{Key: "=0", Parts: []MsgPart{{Kind: MsgText, Text: "none"}}},
		{Key: "one", Parts: []MsgPart{{Kind: MsgPound}, {Kind: MsgText, Text: " file"}}},
		{Key: "few", Parts: []MsgPart{{Kind: MsgPound}, {Kind: MsgText, Text: " files (few)"}}},
		{Key: "other", Parts: []MsgPart{{Kind: MsgPound}, {Kind: MsgText, Text: " files"}}},
	}}}
	others := []MsgPart{{Kind: MsgPlural, Arg: 0, Offset: 1, Cases: []MsgCase{
		{Key: "=1", Parts: []MsgPart{{Kind: MsgText, Text: "Ada"}}},
		{Key: "one", Parts: []MsgPart{{Kind: MsgText, Text: "Ada and "}, {Kind: MsgPound}, {Kind: MsgText, Text: " other"}}},
		{Key: "other", Parts: []MsgPart{{Kind: MsgText, Text: "Ada and "}, {Kind: MsgPound}, {Kind: MsgText, Text: " others"}}},
	}}}
	ordinal := []MsgPart{{Kind: MsgSelectOrdinal, Arg: 0, Cases: []MsgCase{
		{Key: "other", Parts: []MsgPart{{Kind: MsgPound}, {Kind: MsgText, Text: "th"}}},
	}}}
	l := NewLocale("ru")
	for _, tc := range []struct {
		name  string
		parts []MsgPart
		arg   MsgArgValue
		want  string
		ask   Call // the plural question, if one is asked
	}{
		{"exact =0, no question", files, num(0), "none", Call{}},
		{"category one", files, num(1), "<1> file", Call{Kind: "plural", Value: "1"}},
		{"category few", files, num(3), "<3> files (few)", Call{Kind: "plural", Value: "3"}},
		{"no answer: other", files, num(7), "<7> files", Call{Kind: "plural", Value: "7"}},
		{"offset: exact on the value", others, num(1), "Ada", Call{}},
		{"offset: category and # of value-offset", others, num(2), "Ada and <1> other", Call{Kind: "plural", Value: "1"}},
		{"ordinal", ordinal, num(22), "<22>th", Call{Kind: "plural", Value: "22", Opts: `{"type":"ordinal"}`}},
		{"integer style", []MsgPart{{Kind: MsgNumber, Arg: 0, Style: "integer"}}, num(2.5), `<2.5{"maximumFractionDigits":0}>`, Call{}},
		{"select", []MsgPart{{Kind: MsgSelect, Arg: 0, Cases: []MsgCase{{Key: "female", Parts: []MsgPart{{Kind: MsgText, Text: "her"}}}, {Key: "other", Parts: []MsgPart{{Kind: MsgText, Text: "their"}}}}}}, StrArg("x"), "their", Call{}},
	} {
		asked = nil
		if got := l.FormatMessage(tc.parts, tc.arg); got != tc.want {
			t.Errorf("%s: %q, want %q", tc.name, got, tc.want)
		}
		var plural Call
		for _, c := range asked {
			if c.Kind == "plural" {
				plural = c
			}
		}
		if plural != tc.ask {
			t.Errorf("%s: asked %+v, want %+v", tc.name, plural, tc.ask)
		}
	}
}
