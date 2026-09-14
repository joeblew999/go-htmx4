// Package goemit writes Go values as composite-literal source for kit/i18n's generators: zero fields omitted,
// kit/i18n types qualified as i18n.T, so generated tables are static data TinyGo lays out at compile time.
package goemit

import (
	"reflect"
	"strconv"
	"strings"
)

// ImportPath is kit/i18n's package path; its named types are written as i18n.Name.
const ImportPath = "github.com/joeblew999/go-htmx4/kit/i18n"

// Value writes v as a Go expression. typed is false for elements of a composite literal whose type Go infers.
func Value(b *strings.Builder, v reflect.Value, typed bool) {
	t := v.Type()
	switch v.Kind() {
	case reflect.String:
		b.WriteString(strconv.Quote(v.String()))
	case reflect.Bool:
		b.WriteString(strconv.FormatBool(v.Bool()))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if typed && t.PkgPath() == ImportPath {
			b.WriteString(TypeName(t) + "(" + strconv.FormatInt(v.Int(), 10) + ")")
			return
		}
		b.WriteString(strconv.FormatInt(v.Int(), 10))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if typed && t.PkgPath() == ImportPath {
			b.WriteString(TypeName(t) + "(" + strconv.FormatUint(v.Uint(), 10) + ")")
			return
		}
		b.WriteString(strconv.FormatUint(v.Uint(), 10))
	case reflect.Pointer:
		if v.IsNil() {
			b.WriteString("nil")
			return
		}
		b.WriteString("&")
		Value(b, v.Elem(), true)
	case reflect.Struct:
		if typed {
			b.WriteString(TypeName(t))
		}
		b.WriteString("{")
		for i := 0; i < t.NumField(); i++ {
			fv := v.Field(i)
			if !t.Field(i).IsExported() || fv.IsZero() {
				continue
			}
			b.WriteString("\n" + t.Field(i).Name + ": ")
			Value(b, fv, true)
			b.WriteString(",")
		}
		b.WriteString("\n}")
	case reflect.Slice, reflect.Array:
		if typed {
			b.WriteString(TypeName(t))
		}
		b.WriteString("{")
		multiline := v.Len() > 0 && (t.Elem().Kind() == reflect.Struct || t.Elem().Kind() == reflect.Slice || t.Elem().Kind() == reflect.Pointer)
		for i := 0; i < v.Len(); i++ {
			if multiline {
				b.WriteString("\n")
			}
			Value(b, v.Index(i), false)
			b.WriteString(",")
			if !multiline && i < v.Len()-1 {
				b.WriteString(" ")
			}
		}
		if multiline {
			b.WriteString("\n")
		}
		b.WriteString("}")
	default:
		panic("goemit: cannot emit " + t.String())
	}
}

// TypeName is t as written in generated code.
func TypeName(t reflect.Type) string {
	if t.Name() != "" && t.PkgPath() == ImportPath {
		return "i18n." + t.Name()
	}
	switch t.Kind() {
	case reflect.Slice:
		return "[]" + TypeName(t.Elem())
	case reflect.Pointer:
		return "*" + TypeName(t.Elem())
	case reflect.Array:
		return "[" + strconv.Itoa(t.Len()) + "]" + TypeName(t.Elem())
	}
	return t.String()
}
