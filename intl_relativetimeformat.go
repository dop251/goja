package goja

import (
	"math"
	"strconv"
	"strings"
)

type intlRelativeTimeFormatObject struct {
	baseObject
	locale  string
	tag     *intlTag
	style   string
	numeric string
}

var intlRTFUnits = map[string]bool{
	"year": true, "quarter": true, "month": true, "week": true,
	"day": true, "hour": true, "minute": true, "second": true,
}

func (r *Runtime) getIntlRelativeTimeFormat() *Object {
	ret := r.global.IntlRelativeTimeFormat
	if ret == nil {
		ret = &Object{runtime: r}
		r.global.IntlRelativeTimeFormat = ret
		proto := r.getIntlRelativeTimeFormatPrototype()
		r.newNativeFuncAndConstruct(ret, func(call FunctionCall) Value {
			panic(r.NewTypeError("Constructor Intl.RelativeTimeFormat requires 'new'"))
		}, r.wrapNativeConstruct(r.builtin_newIntlRelativeTimeFormat, ret, proto), proto, "RelativeTimeFormat", intToValue(0))
		ret.self._putProp("supportedLocalesOf", r.newNativeFunc(r.intlRTFSupportedLocalesOf, "supportedLocalesOf", 1), true, false, true)
	}
	return ret
}

func (r *Runtime) getIntlRelativeTimeFormatPrototype() *Object {
	ret := r.global.IntlRelativeTimeFormatPrototype
	if ret == nil {
		ret = r.newBaseObject(r.global.ObjectPrototype, classObject).val
		r.global.IntlRelativeTimeFormatPrototype = ret
		o := ret.self
		o._putProp("constructor", r.getIntlRelativeTimeFormat(), true, false, true)
		o._putProp("format", r.newNativeFunc(r.intlRTFProtoFormat, "format", 2), true, false, true)
		o._putProp("formatToParts", r.newNativeFunc(r.intlRTFProtoFormatToParts, "formatToParts", 2), true, false, true)
		o._putProp("resolvedOptions", r.newNativeFunc(r.intlRTFProtoResolvedOptions, "resolvedOptions", 0), true, false, true)
		o.defineOwnPropertySym(SymToStringTag, PropertyDescriptor{
			Value:        intlStr("Intl.RelativeTimeFormat"),
			Writable:     FLAG_FALSE,
			Enumerable:   FLAG_FALSE,
			Configurable: FLAG_TRUE,
		}, false)
	}
	return ret
}

func (r *Runtime) intlRTFThis(call FunctionCall) *intlRelativeTimeFormatObject {
	o, ok := r.toObject(call.This).self.(*intlRelativeTimeFormatObject)
	if !ok {
		panic(r.NewTypeError("Intl.RelativeTimeFormat method called on incompatible receiver"))
	}
	return o
}

func (r *Runtime) builtin_newIntlRelativeTimeFormat(args []Value, proto *Object) *Object {
	var localesArg, optionsArg Value
	if len(args) > 0 {
		localesArg = args[0]
	}
	if len(args) > 1 {
		optionsArg = args[1]
	}
	requested := r.intlCanonicalizeLocaleList(localesArg)
	options := r.intlGetOptionsObject(optionsArg)
	style := "long"
	numeric := "always"
	if options != nil {
		r.intlGetOption(options, "localeMatcher", "string", []string{"best fit", "lookup"}, _undefined)
		if v := r.intlGetOption(options, "numberingSystem", "string", nil, _undefined); !IsUndefined(v) {
			ns := v.String()
			if !intlIsUnicodeExtensionValue(ns) || !intlValidNumberingSystem(ns) {
				panic(r.intlRangeError("Invalid numberingSystem"))
			}
		}
		if v := r.intlGetOption(options, "style", "string", []string{"long", "short", "narrow"}, _undefined); !IsUndefined(v) {
			style = v.String()
		}
		if v := r.intlGetOption(options, "numeric", "string", []string{"always", "auto"}, _undefined); !IsUndefined(v) {
			numeric = v.String()
		}
	}
	locale, tag := r.intlResolveServiceLocale(requested, []string{"nu"}, options)
	o := &Object{runtime: r}
	obj := &intlRelativeTimeFormatObject{locale: locale, tag: tag, style: style, numeric: numeric}
	obj.class = classObject
	obj.val = o
	obj.extensible = true
	obj.prototype = proto
	obj.init()
	o.self = obj
	return o
}

func (rtf *intlRelativeTimeFormatObject) data() map[string]*intlRelativeFieldData {
	key := "en"
	if t, ok := intlParseTag(rtf.locale); ok {
		if k, ok := intlDataLocale(t); ok {
			key = k
		}
	}
	if d, ok := intlRelativeLocaleData[key]; ok {
		return d
	}
	return intlRelativeLocaleData["en"]
}

func (rtf *intlRelativeTimeFormatObject) fieldPattern(n float64, unit string) (string, bool) {
	d := rtf.data()
	field := unit
	if rtf.style == "short" {
		field = unit + "-short"
	} else if rtf.style == "narrow" {
		field = unit + "-narrow"
	}
	f, ok := d[field]
	if !ok {
		f, ok = d[unit]
	}
	if !ok {
		return "", false
	}
	if rtf.numeric == "auto" {
		if math.Trunc(n) == n {
			if p, ok := f.Relative[int(n)]; ok {
				return p, true
			}
		}
	}
	cat := rtf.pluralCategory(n)
	var p string
	if n < 0 {
		p = f.Past[cat]
		if p == "" {
			p = f.Past["other"]
		}
	} else {
		p = f.Future[cat]
		if p == "" {
			p = f.Future["other"]
		}
	}
	return p, p != ""
}

func (rtf *intlRelativeTimeFormatObject) pluralCategory(n float64) string {
	if n < 0 {
		n = -n
	}
	pr := &intlPluralRulesObject{
		locale:       rtf.locale,
		tag:          rtf.tag,
		typ:          "cardinal",
		minInt:       1,
		minFrac:      0,
		maxFrac:      0,
		roundingMode: "halfExpand",
	}
	return pr.selectPlural(n)
}

func (r *Runtime) intlRTFProtoFormat(call FunctionCall) Value {
	rtf := r.intlRTFThis(call)
	parts := r.intlRTFFormatParts(rtf, call.Argument(0), call.Argument(1))
	var sb strings.Builder
	for _, p := range parts {
		sb.WriteString(p.value)
	}
	return intlStr(sb.String())
}

func (r *Runtime) intlRTFFormatParts(rtf *intlRelativeTimeFormatObject, value, unitArg Value) []intlStringPart {
	n := value.ToFloat()
	if math.IsNaN(n) || math.IsInf(n, 0) {
		panic(r.intlRangeError("Invalid value"))
	}
	unit := unitArg.toString().String()
	if !intlRTFUnits[unit] {
		panic(r.intlRangeError("Invalid unit: " + unit))
	}
	pattern, ok := rtf.fieldPattern(n, unit)
	if !ok {
		pattern = "{0} " + unit
	}
	loc := intlNumberLocaleFor(rtf.dataKey())
	opts := intlDefaultNumberOptions()
	opts.minInt = 1
	opts.minFrac = 0
	opts.maxFrac = 3
	opts.numberingSystem = rtf.numberingSystem()
	numParts := intlFormatNumberParts(math.Abs(n), opts, loc)
	var parts []intlStringPart
	idx := strings.Index(pattern, "{0}")
	if idx < 0 {
		return []intlStringPart{{"literal", pattern}}
	}
	if idx > 0 {
		parts = append(parts, intlStringPart{"literal", pattern[:idx]})
	}
	parts = append(parts, numParts...)
	if rest := pattern[idx+3:]; rest != "" {
		parts = append(parts, intlStringPart{"literal", rest})
	}
	return parts
}

func (rtf *intlRelativeTimeFormatObject) dataKey() string {
	if t, ok := intlParseTag(rtf.locale); ok {
		if k, ok := intlDataLocale(t); ok {
			return k
		}
	}
	return "en"
}

func (rtf *intlRelativeTimeFormatObject) numberingSystem() string {
	if t, ok := intlParseTag(rtf.locale); ok {
		if ns, ok := t.uKw["nu"]; ok && ns != "" {
			return ns
		}
	}
	return intlNumberLocaleFor(rtf.dataKey()).DefaultNS
}

func (r *Runtime) intlRTFProtoFormatToParts(call FunctionCall) Value {
	rtf := r.intlRTFThis(call)
	parts := r.intlRTFFormatParts(rtf, call.Argument(0), call.Argument(1))
	return r.intlPartsArray(parts)
}

func (r *Runtime) intlRTFProtoResolvedOptions(call FunctionCall) Value {
	rtf := r.intlRTFThis(call)
	o := r.newBaseObject(r.global.ObjectPrototype, classObject)
	o.setOwnStr("locale", intlStr(rtf.locale), false)
	o.setOwnStr("style", intlStr(rtf.style), false)
	o.setOwnStr("numeric", intlStr(rtf.numeric), false)
	o.setOwnStr("numberingSystem", intlStr(rtf.numberingSystem()), false)
	return o.val
}

func (r *Runtime) intlRTFSupportedLocalesOf(call FunctionCall) Value {
	return r.intlSupportedLocalesOf(call, intlHasLocaleData)
}

var _ = strconv.Itoa
