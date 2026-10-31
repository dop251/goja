package goja

import (
	"strings"
)

type intlDisplayNamesObject struct {
	baseObject
	locale          string
	style           string
	typ             string
	fallback        string
	languageDisplay string
}

var intlDisplayNameTypes = []string{"language", "region", "script", "currency", "calendar", "dateTimeField"}

func (r *Runtime) getIntlDisplayNames() *Object {
	ret := r.global.IntlDisplayNames
	if ret == nil {
		ret = &Object{runtime: r}
		r.global.IntlDisplayNames = ret
		proto := r.getIntlDisplayNamesPrototype()
		r.newNativeFuncAndConstruct(ret, func(call FunctionCall) Value {
			panic(r.NewTypeError("Constructor Intl.DisplayNames requires 'new'"))
		}, r.wrapNativeConstruct(r.builtin_newIntlDisplayNames, ret, proto), proto, "DisplayNames", intToValue(2))
		ret.self._putProp("supportedLocalesOf", r.newNativeFunc(r.intlDisplayNamesSupportedLocalesOf, "supportedLocalesOf", 1), true, false, true)
	}
	return ret
}

func (r *Runtime) getIntlDisplayNamesPrototype() *Object {
	ret := r.global.IntlDisplayNamesPrototype
	if ret == nil {
		ret = r.newBaseObject(r.global.ObjectPrototype, classObject).val
		r.global.IntlDisplayNamesPrototype = ret
		o := ret.self
		o._putProp("constructor", r.getIntlDisplayNames(), true, false, true)
		o._putProp("of", r.newNativeFunc(r.intlDisplayNamesProtoOf, "of", 1), true, false, true)
		o._putProp("resolvedOptions", r.newNativeFunc(r.intlDisplayNamesProtoResolvedOptions, "resolvedOptions", 0), true, false, true)
		o.defineOwnPropertySym(SymToStringTag, PropertyDescriptor{
			Value:        intlStr("Intl.DisplayNames"),
			Writable:     FLAG_FALSE,
			Enumerable:   FLAG_FALSE,
			Configurable: FLAG_TRUE,
		}, false)
	}
	return ret
}

func (r *Runtime) intlDisplayNamesThis(call FunctionCall) *intlDisplayNamesObject {
	o, ok := r.toObject(call.This).self.(*intlDisplayNamesObject)
	if !ok {
		panic(r.NewTypeError("Intl.DisplayNames method called on incompatible receiver"))
	}
	return o
}

func (r *Runtime) builtin_newIntlDisplayNames(args []Value, proto *Object) *Object {
	var localesArg, optionsArg Value
	if len(args) > 0 {
		localesArg = args[0]
	}
	if len(args) > 1 {
		optionsArg = args[1]
	}
	requested := r.intlCanonicalizeLocaleList(localesArg)
	if optionsArg == nil || IsUndefined(optionsArg) {
		panic(r.NewTypeError("Intl.DisplayNames requires an options object"))
	}
	options := r.intlGetOptionsObject(optionsArg)
	if options == nil {
		panic(r.NewTypeError("Intl.DisplayNames requires an options object"))
	}
	style := "long"
	fallback := "code"
	languageDisplay := "dialect"
	typ := ""
	r.intlGetOption(options, "localeMatcher", "string", []string{"best fit", "lookup"}, _undefined)
	if v := r.intlGetOption(options, "style", "string", []string{"narrow", "short", "long"}, _undefined); !IsUndefined(v) {
		style = v.String()
	}
	if v := r.intlGetOption(options, "type", "string", intlDisplayNameTypes, _undefined); !IsUndefined(v) {
		typ = v.String()
	}
	if typ == "" {
		panic(r.NewTypeError("Intl.DisplayNames requires a type option"))
	}
	if v := r.intlGetOption(options, "fallback", "string", []string{"code", "none"}, _undefined); !IsUndefined(v) {
		fallback = v.String()
	}
	if v := r.intlGetOption(options, "languageDisplay", "string", []string{"dialect", "standard"}, _undefined); !IsUndefined(v) {
		languageDisplay = v.String()
	}

	locale, tag := r.intlResolveServiceLocale(requested, nil, options)
	key := tag.baseName()
	if _, ok := intlDisplayLocales[key]; !ok {
		if k, ok := intlDataLocale(tag); ok {
			if _, ok := intlDisplayLocales[k]; ok {
				key = k
			} else {
				key = "en"
			}
		} else {
			key = "en"
		}
	}
	o := &Object{runtime: r}
	obj := &intlDisplayNamesObject{locale: locale, style: style, typ: typ, fallback: fallback, languageDisplay: languageDisplay}
	obj.class = classObject
	obj.val = o
	obj.extensible = true
	obj.prototype = proto
	obj.init()
	o.self = obj
	return o
}

func (d *intlDisplayNamesObject) data() *intlDisplayLocaleData {
	if t, ok := intlParseTag(d.locale); ok {
		if data, ok := intlDisplayLocales[t.baseName()]; ok {
			return data
		}
		if k, ok := intlDataLocale(t); ok {
			if data, ok := intlDisplayLocales[k]; ok {
				return data
			}
		}
	}
	return intlDisplayLocales["en"]
}

func (r *Runtime) intlDisplayNamesProtoOf(call FunctionCall) Value {
	d := r.intlDisplayNamesThis(call)
	code := call.Argument(0).toString().String()
	data := d.data()
	var name string
	var found bool
	switch d.typ {
	case "language":
		name, found = r.intlDisplayLanguageName(d, data, code)
	case "region":
		up := strings.ToUpper(code)
		if !intlIsRegionSubtag(up) {
			panic(r.intlRangeError("Invalid region code: " + code))
		}
		name, found = data.Territories[up]
	case "script":
		if !intlIsScriptSubtag(code) {
			panic(r.intlRangeError("Invalid script code: " + code))
		}
		sc := intlTitleCase(code)
		name, found = data.Scripts[sc]
	case "currency":
		cur := strings.ToUpper(code)
		if !intlIsWellFormedCurrencyCode(cur) {
			panic(r.intlRangeError("Invalid currency code: " + code))
		}
		loc := intlNumberLocaleFor(d.dataKey())
		if ci, ok := loc.Currencies[cur]; ok && ci.Name != "" {
			name, found = ci.Name, true
		}
	case "calendar":
		if !intlIsUnicodeExtensionValue(code) {
			panic(r.intlRangeError("Invalid calendar code: " + code))
		}
		name, found = intlCalendarDisplayName(code, d.locale)
	case "dateTimeField":
		if !intlContains(intlDateTimeFieldValues, code) {
			panic(r.intlRangeError("Invalid dateTimeField: " + code))
		}
		name, found = data.DateFields[code]
	}
	if !found || name == "" {
		if d.fallback == "none" {
			return _undefined
		}
		return intlStr(code)
	}
	return intlStr(name)
}

func (d *intlDisplayNamesObject) dataKey() string {
	if t, ok := intlParseTag(d.locale); ok {
		if k, ok := intlDataLocale(t); ok {
			return k
		}
	}
	return "en"
}

func (r *Runtime) intlDisplayLanguageName(d *intlDisplayNamesObject, data *intlDisplayLocaleData, code string) (string, bool) {
	tag, ok := intlParseTag(code)
	if !ok || len(tag.private) > 0 || len(tag.ext) > 0 || len(tag.uKw) > 0 || len(tag.uAttr) > 0 {
		panic(r.intlRangeError("Invalid language code: " + code))
	}
	canon := intlCanonicalizeTag(tag)
	if d.languageDisplay == "dialect" {
		if n, ok := data.Languages[canon.String()]; ok && n != "" {
			return n, true
		}
	}
	base, ok := data.Languages[canon.language]
	if !ok || base == "" {
		return "", false
	}
	if canon.region != "" {
		if rn, ok := data.Territories[canon.region]; ok && rn != "" {
			if d.languageDisplay == "standard" || canon.String() != canon.language {
				return base + " (" + rn + ")", true
			}
		}
	}
	return base, true
}

func intlCalendarDisplayName(code, locale string) (string, bool) {
	if locale == "en" || !strings.HasPrefix(locale, "en") {
		if n, ok := intlCalendarDisplayNames[code]; ok {
			return n, true
		}
	}
	if n, ok := intlCalendarDisplayNames[code]; ok {
		return n, true
	}
	return "", false
}

var intlCalendarDisplayNames = map[string]string{
	"buddhist":         "Buddhist Calendar",
	"chinese":          "Chinese Calendar",
	"coptic":           "Coptic Calendar",
	"dangi":            "Dangi Calendar",
	"ethioaa":          "Ethiopic Amete Alem Calendar",
	"ethiopic":         "Ethiopic Calendar",
	"gregory":          "Gregorian Calendar",
	"hebrew":           "Hebrew Calendar",
	"indian":           "Indian National Calendar",
	"islamic":          "Islamic Calendar",
	"islamic-civil":    "Islamic Civil Calendar",
	"islamic-rgsa":     "Islamic Calendar (Saudi Arabia Sighted)",
	"islamic-tbla":     "Islamic Tabular Calendar",
	"islamic-umalqura": "Islamic Umm al-Qura Calendar",
	"iso8601":          "ISO-8601 Calendar",
	"japanese":         "Japanese Calendar",
	"persian":          "Persian Calendar",
	"roc":              "Minguo Calendar",
	"gregorian":        "Gregorian Calendar",
}

var intlDateTimeFieldValues = []string{
	"era", "year", "quarter", "month", "week", "weekOfMonth", "day", "dayOfYear",
	"weekday", "weekdayOfMonth", "dayperiod", "hour", "minute", "second", "zone", "weekOfYear",
}

func (r *Runtime) intlDisplayNamesProtoResolvedOptions(call FunctionCall) Value {
	d := r.intlDisplayNamesThis(call)
	o := r.newBaseObject(r.global.ObjectPrototype, classObject)
	o.setOwnStr("locale", intlStr(d.locale), false)
	o.setOwnStr("style", intlStr(d.style), false)
	o.setOwnStr("type", intlStr(d.typ), false)
	o.setOwnStr("fallback", intlStr(d.fallback), false)
	if d.typ == "language" {
		o.setOwnStr("languageDisplay", intlStr(d.languageDisplay), false)
	}
	return o.val
}

func (r *Runtime) intlDisplayNamesSupportedLocalesOf(call FunctionCall) Value {
	return r.intlSupportedLocalesOf(call, func(tag *intlTag) bool {
		if _, ok := intlDisplayLocales[tag.baseName()]; ok {
			return true
		}
		_, ok := intlDataLocale(tag)
		return ok
	})
}
