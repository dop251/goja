package goja

import (
	"strings"

	"github.com/dop251/goja/unistring"
)

type intlListFormatObject struct {
	baseObject
	locale string
	typ    string
	style  string
}

func (r *Runtime) getIntlListFormat() *Object {
	ret := r.global.IntlListFormat
	if ret == nil {
		ret = &Object{runtime: r}
		r.global.IntlListFormat = ret
		proto := r.getIntlListFormatPrototype()
		r.newNativeFuncAndConstruct(ret, func(call FunctionCall) Value {
			panic(r.NewTypeError("Constructor Intl.ListFormat requires 'new'"))
		}, r.wrapNativeConstruct(r.builtin_newIntlListFormat, ret, proto), proto, "ListFormat", intToValue(0))
		ret.self._putProp("supportedLocalesOf", r.newNativeFunc(r.intlListFormatSupportedLocalesOf, "supportedLocalesOf", 1), true, false, true)
	}
	return ret
}

func (r *Runtime) getIntlListFormatPrototype() *Object {
	ret := r.global.IntlListFormatPrototype
	if ret == nil {
		ret = r.newBaseObject(r.global.ObjectPrototype, classObject).val
		r.global.IntlListFormatPrototype = ret
		o := ret.self
		o._putProp("constructor", r.getIntlListFormat(), true, false, true)
		o._putProp("format", r.newNativeFunc(r.intlListFormatProtoFormat, "format", 1), true, false, true)
		o._putProp("formatToParts", r.newNativeFunc(r.intlListFormatProtoFormatToParts, "formatToParts", 1), true, false, true)
		o._putProp("resolvedOptions", r.newNativeFunc(r.intlListFormatProtoResolvedOptions, "resolvedOptions", 0), true, false, true)
		o.defineOwnPropertySym(SymToStringTag, PropertyDescriptor{
			Value:        intlStr("Intl.ListFormat"),
			Writable:     FLAG_FALSE,
			Enumerable:   FLAG_FALSE,
			Configurable: FLAG_TRUE,
		}, false)
	}
	return ret
}

func (r *Runtime) intlListFormatThis(call FunctionCall) *intlListFormatObject {
	o, ok := r.toObject(call.This).self.(*intlListFormatObject)
	if !ok {
		panic(r.NewTypeError("Intl.ListFormat method called on incompatible receiver"))
	}
	return o
}

func (r *Runtime) builtin_newIntlListFormat(args []Value, proto *Object) *Object {
	var localesArg, optionsArg Value
	if len(args) > 0 {
		localesArg = args[0]
	}
	if len(args) > 1 {
		optionsArg = args[1]
	}
	requested := r.intlCanonicalizeLocaleList(localesArg)
	options := r.intlGetOptionsObject(optionsArg)
	typ := "conjunction"
	style := "long"
	if options != nil {
		r.intlGetOption(options, "localeMatcher", "string", []string{"best fit", "lookup"}, _undefined)
		if v := r.intlGetOption(options, "type", "string", []string{"conjunction", "disjunction", "unit"}, _undefined); !IsUndefined(v) {
			typ = v.String()
		}
		if v := r.intlGetOption(options, "style", "string", []string{"long", "short", "narrow"}, _undefined); !IsUndefined(v) {
			style = v.String()
		}
	}
	locale, _ := r.intlResolveServiceLocale(requested, nil, options)
	o := &Object{runtime: r}
	obj := &intlListFormatObject{locale: locale, typ: typ, style: style}
	obj.class = classObject
	obj.val = o
	obj.extensible = true
	obj.prototype = proto
	obj.init()
	o.self = obj
	return o
}

func (lf *intlListFormatObject) patterns(dataLocale string) [4]string {
	data, ok := intlListLocaleData[dataLocale]
	if !ok {
		data = intlListLocaleData["en"]
	}
	key := "standard"
	switch lf.typ {
	case "disjunction":
		key = "or"
	case "unit":
		key = "unit"
	}
	if lf.style != "long" {
		if p, ok := data[key+"-"+lf.style]; ok {
			return p
		}
	}
	if p, ok := data[key]; ok {
		return p
	}
	return [4]string{"{0}, {1}", "{0}, {1}", "{0}, {1}", "{0}, {1}"}
}

func (r *Runtime) intlListFormatProtoFormat(call FunctionCall) Value {
	lf := r.intlListFormatThis(call)
	list := r.intlIterateStringList(call.Argument(0))
	return intlStr(intlFormatListPattern(list, lf.patterns(lf.dataLocale())))
}

func (r *Runtime) intlListFormatProtoFormatToParts(call FunctionCall) Value {
	lf := r.intlListFormatThis(call)
	list := r.intlIterateStringList(call.Argument(0))
	parts := intlFormatListParts(list, lf.patterns(lf.dataLocale()))
	return r.intlPartsArray(parts)
}

func (lf *intlListFormatObject) dataLocale() string {
	tag, ok := intlParseTag(lf.locale)
	if !ok {
		return "en"
	}
	if key, ok := intlDataLocale(tag); ok {
		return key
	}
	return "en"
}

type intlStringPart struct {
	typ, value string
}

func (r *Runtime) intlIterateStringList(v Value) []string {
	if v == nil || IsUndefined(v) {
		panic(r.NewTypeError("Cannot convert undefined to object"))
	}
	if method := toMethod(r.getV(v, SymIterator)); method != nil {
		iter := r.getIterator(v, method)
		var list []string
		for {
			item, ok := iter.stepValue()
			if !ok {
				break
			}
			list = append(list, item.toString().String())
		}
		return list
	}
	o := v.ToObject(r)
	length := int64(toLength(o.self.getStr("length", nil)))
	list := make([]string, length)
	for i := int64(0); i < length; i++ {
		list[i] = nilSafe(o.self.getStr(intToValue(i).string(), nil)).toString().String()
	}
	return list
}

func intlFormatListPattern(list []string, p [4]string) string {
	switch len(list) {
	case 0:
		return ""
	case 1:
		return list[0]
	case 2:
		return strings.ReplaceAll(strings.ReplaceAll(p[3], "{0}", list[0]), "{1}", list[1])
	}
	res := strings.ReplaceAll(strings.ReplaceAll(p[0], "{0}", list[0]), "{1}", list[1])
	for i := 2; i < len(list)-1; i++ {
		res = strings.ReplaceAll(strings.ReplaceAll(p[1], "{0}", res), "{1}", list[i])
	}
	return strings.ReplaceAll(strings.ReplaceAll(p[2], "{0}", res), "{1}", list[len(list)-1])
}

func intlFormatListParts(list []string, p [4]string) []intlStringPart {
	if len(list) == 0 {
		return nil
	}
	if len(list) == 1 {
		return []intlStringPart{{"element", list[0]}}
	}
	full := intlFormatListPattern(list, p)
	var parts []intlStringPart
	pos := 0
	for i, el := range list {
		idx := strings.Index(full[pos:], el)
		if idx < 0 {
			parts = append(parts, intlStringPart{"element", el})
			continue
		}
		if idx > 0 {
			parts = append(parts, intlStringPart{"literal", full[pos : pos+idx]})
		}
		parts = append(parts, intlStringPart{"element", el})
		pos += idx + len(el)
		_ = i
	}
	if pos < len(full) {
		parts = append(parts, intlStringPart{"literal", full[pos:]})
	}
	return parts
}

func (r *Runtime) intlListFormatProtoResolvedOptions(call FunctionCall) Value {
	lf := r.intlListFormatThis(call)
	o := r.newBaseObject(r.global.ObjectPrototype, classObject)
	o.setOwnStr("locale", intlStr(lf.locale), false)
	o.setOwnStr("type", intlStr(lf.typ), false)
	o.setOwnStr("style", intlStr(lf.style), false)
	return o.val
}

func (r *Runtime) intlListFormatSupportedLocalesOf(call FunctionCall) Value {
	return r.intlSupportedLocalesOf(call, intlHasLocaleData)
}

var _ = unistring.NewFromString
