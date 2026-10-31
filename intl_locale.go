package goja

import (
	"sort"
	"strings"

	"github.com/dop251/goja/unistring"
)

type intlLocaleObject struct {
	baseObject
	tag *intlTag
}

func (r *Runtime) getIntlLocale() *Object {
	ret := r.global.IntlLocale
	if ret == nil {
		ret = &Object{runtime: r}
		r.global.IntlLocale = ret
		proto := r.getIntlLocalePrototype()
		r.newNativeFuncAndConstruct(ret, r.builtin_IntlLocale,
			r.wrapNativeConstruct(r.builtin_newIntlLocale, ret, proto), proto, "Locale", intToValue(1))
	}
	return ret
}

func (r *Runtime) getIntlLocalePrototype() *Object {
	ret := r.global.IntlLocalePrototype
	if ret == nil {
		ret = r.newBaseObject(r.global.ObjectPrototype, classObject).val
		r.global.IntlLocalePrototype = ret
		o := ret.self
		o._putProp("constructor", r.getIntlLocale(), true, false, true)
		addGetter := func(name string, f func(*intlLocaleObject) Value) {
			o.setOwnStr(unistring.NewFromString(name), &valueProperty{
				getterFunc:   r.newNativeFunc(func(call FunctionCall) Value { return f(r.intlLocaleThis(call)) }, unistring.NewFromString("get "+name), 0),
				accessor:     true,
				configurable: true,
			}, false)
		}
		addGetter("baseName", func(l *intlLocaleObject) Value { return intlStr(l.tag.baseName()) })
		addGetter("calendar", func(l *intlLocaleObject) Value { return l.uKeyword("ca") })
		addGetter("caseFirst", func(l *intlLocaleObject) Value { return l.uKeyword("kf") })
		addGetter("collation", func(l *intlLocaleObject) Value { return l.uKeyword("co") })
		addGetter("hourCycle", func(l *intlLocaleObject) Value { return l.uKeyword("hc") })
		addGetter("language", func(l *intlLocaleObject) Value {
			if l.tag.language == "" {
				return _undefined
			}
			return intlStr(l.tag.language)
		})
		addGetter("numberingSystem", func(l *intlLocaleObject) Value { return l.uKeyword("nu") })
		addGetter("numeric", func(l *intlLocaleObject) Value {
			v, ok := l.tag.uKw["kn"]
			if !ok {
				return valueFalse
			}
			return r.ToValue(v != "false")
		})
		addGetter("firstDayOfWeek", func(l *intlLocaleObject) Value {
			return l.uKeyword("fw")
		})
		addGetter("region", func(l *intlLocaleObject) Value {
			if l.tag.region == "" {
				return _undefined
			}
			return intlStr(l.tag.region)
		})
		addGetter("script", func(l *intlLocaleObject) Value {
			if l.tag.script == "" {
				return _undefined
			}
			return intlStr(l.tag.script)
		})
		addGetter("variants", func(l *intlLocaleObject) Value {
			if len(l.tag.variants) == 0 {
				return _undefined
			}
			return intlStr(strings.Join(l.tag.variants, "-"))
		})
		o._putProp("toString", r.newNativeFunc(r.intlLocaleProtoToString, "toString", 0), true, false, true)
		o._putProp("maximize", r.newNativeFunc(r.intlLocaleProtoMaximize, "maximize", 0), true, false, true)
		o._putProp("minimize", r.newNativeFunc(r.intlLocaleProtoMinimize, "minimize", 0), true, false, true)
		o._putProp("getCalendars", r.newNativeFunc(r.intlLocaleProtoGetCalendars, "getCalendars", 0), true, false, true)
		o._putProp("getCollations", r.newNativeFunc(r.intlLocaleProtoGetCollations, "getCollations", 0), true, false, true)
		o._putProp("getHourCycles", r.newNativeFunc(r.intlLocaleProtoGetHourCycles, "getHourCycles", 0), true, false, true)
		o._putProp("getNumberingSystems", r.newNativeFunc(r.intlLocaleProtoGetNumberingSystems, "getNumberingSystems", 0), true, false, true)
		o._putProp("getTextInfo", r.newNativeFunc(r.intlLocaleProtoGetTextInfo, "getTextInfo", 0), true, false, true)
		o._putProp("getTimeZones", r.newNativeFunc(r.intlLocaleProtoGetTimeZones, "getTimeZones", 0), true, false, true)
		o._putProp("getWeekInfo", r.newNativeFunc(r.intlLocaleProtoGetWeekInfo, "getWeekInfo", 0), true, false, true)
		o.defineOwnPropertySym(SymToStringTag, PropertyDescriptor{
			Value:        intlStr("Intl.Locale"),
			Writable:     FLAG_FALSE,
			Enumerable:   FLAG_FALSE,
			Configurable: FLAG_TRUE,
		}, false)
	}
	return ret
}

func (l *intlLocaleObject) uKeyword(key string) Value {
	if v, ok := l.tag.uKw[key]; ok {
		return intlStr(v)
	}
	return _undefined
}

func (r *Runtime) intlLocaleThis(call FunctionCall) *intlLocaleObject {
	l, ok := r.toObject(call.This).self.(*intlLocaleObject)
	if !ok {
		panic(r.NewTypeError("Intl.Locale method called on incompatible receiver"))
	}
	return l
}

func (r *Runtime) builtin_IntlLocale(call FunctionCall) Value {
	panic(r.NewTypeError("Intl.Locale constructor must be called with new"))
}

func (r *Runtime) builtin_newIntlLocale(args []Value, proto *Object) *Object {
	if proto == nil {
		panic(r.NewTypeError("Intl.Locale constructor must be called with new"))
	}
	var tagVal Value
	if len(args) > 0 {
		tagVal = args[0]
	}
	if tagVal == nil || IsNull(tagVal) {
		panic(r.NewTypeError("Language ID should be string or object."))
	}
	if _, ok := tagVal.(*Object); !ok {
		if _, ok := tagVal.(String); !ok {
			panic(r.NewTypeError("Language ID should be string or object."))
		}
	}
	tagStr := tagVal.toString().String()
	tag, ok := intlParseTag(tagStr)
	if !ok {
		panic(r.intlRangeError("Incorrect Language Tag: " + tagStr))
	}
	tag = intlCanonicalizeTag(tag)

	if len(args) > 1 && !IsUndefined(args[1]) {
		options := r.toObject(args[1])
		tag = r.intlApplyOptionsToTag(tag, options)
	}

	o := &Object{runtime: r}
	lo := &intlLocaleObject{tag: tag}
	lo.class = classObject
	lo.val = o
	lo.extensible = true
	lo.prototype = proto
	lo.init()
	o.self = lo
	return o
}

func (r *Runtime) intlApplyOptionsToTag(tag *intlTag, options *Object) *intlTag {
	language := r.intlGetStringOption(options, "language")
	script := r.intlGetStringOption(options, "script")
	region := r.intlGetStringOption(options, "region")
	variants := r.intlGetStringOption(options, "variants")

	res := tag.clone()
	if !IsUndefined(language) {
		if !intlIsLanguageSubtag(language.String()) {
			panic(r.intlRangeError("Invalid language: " + language.String()))
		}
		res.language = strings.ToLower(language.String())
	}
	if !IsUndefined(script) {
		if !intlIsScriptSubtag(script.String()) {
			panic(r.intlRangeError("Invalid script: " + script.String()))
		}
		res.script = strings.ToUpper(script.String()[:1]) + strings.ToLower(script.String()[1:])
	}
	if !IsUndefined(region) {
		if !intlIsRegionSubtag(region.String()) {
			panic(r.intlRangeError("Invalid region: " + region.String()))
		}
		res.region = strings.ToUpper(region.String())
	}
	if !IsUndefined(variants) {
		vs := strings.Split(strings.ToLower(variants.String()), "-")
		for _, v := range vs {
			if !intlIsVariantSubtag(v) {
				panic(r.intlRangeError("Invalid variants: " + variants.String()))
			}
		}
		res.variants = vs
	}
	if mt, ok := intlParseTag(res.baseName()); !ok {
		panic(r.intlRangeError("Invalid language tag"))
	} else {
		mt.ext = res.ext
		mt.uAttr = res.uAttr
		mt.uKw = res.uKw
		mt.uKwOrder = res.uKwOrder
		mt.private = res.private
		res = mt
	}

	calendar := r.intlGetStringOption(options, "calendar")
	if !IsUndefined(calendar) {
		if !intlIsUnicodeExtensionValue(calendar.String()) {
			panic(r.intlRangeError("Invalid calendar: " + calendar.String()))
		}
		res = r.intlSetUnicodeKeyword(res, "ca", strings.ToLower(calendar.String()))
	}
	collation := r.intlGetStringOption(options, "collation")
	if !IsUndefined(collation) {
		if !intlIsUnicodeExtensionValue(collation.String()) {
			panic(r.intlRangeError("Invalid collation: " + collation.String()))
		}
		res = r.intlSetUnicodeKeyword(res, "co", strings.ToLower(collation.String()))
	}
	firstDay := r.intlGetOption(options, "firstDayOfWeek", "string", nil, _undefined)
	if !IsUndefined(firstDay) {
		v := strings.ToLower(firstDay.String())
		if n, ok := intlWeekdayNumbers[v]; ok {
			_ = n
		}
		switch v {
		case "0", "7":
			v = "sun"
		case "1":
			v = "mon"
		case "2":
			v = "tue"
		case "3":
			v = "wed"
		case "4":
			v = "thu"
		case "5":
			v = "fri"
		case "6":
			v = "sat"
		}
		if v == "true" {
			v = ""
		} else if !intlIsUnicodeExtensionValue(v) {
			panic(r.intlRangeError("Invalid firstDayOfWeek: " + firstDay.String()))
		}
		res = r.intlSetUnicodeKeyword(res, "fw", v)
	}
	hourCycle := r.intlGetOption(options, "hourCycle", "string", []string{"h11", "h12", "h23", "h24"}, _undefined)
	if !IsUndefined(hourCycle) {
		res = r.intlSetUnicodeKeyword(res, "hc", hourCycle.String())
	}
	caseFirst := r.intlGetOption(options, "caseFirst", "string", []string{"upper", "lower", "false"}, _undefined)
	if !IsUndefined(caseFirst) {
		res = r.intlSetUnicodeKeyword(res, "kf", caseFirst.String())
	}
	numeric := r.intlGetOption(options, "numeric", "boolean", nil, _undefined)
	if !IsUndefined(numeric) {
		if numeric.ToBoolean() {
			res = r.intlSetUnicodeKeyword(res, "kn", "")
		} else {
			res = r.intlSetUnicodeKeyword(res, "kn", "false")
		}
	}
	numberingSystem := r.intlGetStringOption(options, "numberingSystem")
	if !IsUndefined(numberingSystem) {
		if !intlIsUnicodeExtensionValue(numberingSystem.String()) {
			panic(r.intlRangeError("Invalid numberingSystem: " + numberingSystem.String()))
		}
		res = r.intlSetUnicodeKeyword(res, "nu", strings.ToLower(numberingSystem.String()))
	}
	return intlCanonicalizeTag(res)
}

func (r *Runtime) intlGetStringOption(options *Object, name string) Value {
	v := nilSafe(options.self.getStr(unistring.NewFromString(name), nil))
	if IsUndefined(v) {
		return _undefined
	}
	return v.toString()
}

func (r *Runtime) intlSetUnicodeKeyword(tag *intlTag, key, value string) *intlTag {
	res := tag.clone()
	if res.uKw == nil {
		res.uKw = map[string]string{}
	}
	if _, ok := res.uKw[key]; !ok {
		res.uKwOrder = append(res.uKwOrder, key)
	}
	res.uKw[key] = value
	return res
}

func (r *Runtime) intlLocaleProtoToString(call FunctionCall) Value {
	l := r.intlLocaleThis(call)
	return intlStr(l.tag.String())
}

func (r *Runtime) newIntlLocaleFromTag(tag *intlTag, proto *Object) Value {
	o := &Object{runtime: r}
	lo := &intlLocaleObject{tag: tag}
	lo.class = classObject
	lo.val = o
	lo.extensible = true
	lo.prototype = proto
	lo.init()
	o.self = lo
	return o
}

func (r *Runtime) intlLocaleProtoMaximize(call FunctionCall) Value {
	l := r.intlLocaleThis(call)
	max := intlMaximize(l.tag)
	return r.newIntlLocaleFromTag(max, l.prototype)
}

func (r *Runtime) intlLocaleProtoMinimize(call FunctionCall) Value {
	l := r.intlLocaleThis(call)
	min := intlMinimize(l.tag)
	return r.newIntlLocaleFromTag(min, l.prototype)
}

func (r *Runtime) intlLocaleProtoGetCalendars(call FunctionCall) Value {
	l := r.intlLocaleThis(call)
	var list []string
	if v, ok := l.tag.uKw["ca"]; ok && v != "" {
		list = []string{v}
	} else {
		region := l.tag.region
		if region == "" {
			region = "001"
		}
		list = append(list, intlCalendarPref[region]...)
		if len(list) == 0 {
			list = append(list, intlCalendarPref["001"]...)
		}
		if len(list) == 0 {
			list = []string{"gregory"}
		}
	}
	return intlStringArray(r, list)
}

func (r *Runtime) intlLocaleProtoGetCollations(call FunctionCall) Value {
	l := r.intlLocaleThis(call)
	var list []string
	if v, ok := l.tag.uKw["co"]; ok && v != "" {
		list = []string{v}
	} else {
		list = []string{"emoji", "eor"}
	}
	return intlStringArray(r, list)
}

func (r *Runtime) intlLocaleProtoGetHourCycles(call FunctionCall) Value {
	l := r.intlLocaleThis(call)
	var list []string
	if v, ok := l.tag.uKw["hc"]; ok && v != "" {
		list = []string{v}
	} else {
		region := l.tag.region
		if region == "" {
			region = "001"
		}
		if hc, ok := intlHourCyclePref[region]; ok {
			list = []string{hc}
		} else if hc, ok := intlHourCyclePref["001"]; ok {
			list = []string{hc}
		} else {
			list = []string{"h23"}
		}
	}
	return intlStringArray(r, list)
}

func (r *Runtime) intlLocaleProtoGetNumberingSystems(call FunctionCall) Value {
	l := r.intlLocaleThis(call)
	var list []string
	if v, ok := l.tag.uKw["nu"]; ok && v != "" {
		list = []string{v}
	} else {
		list = []string{intlDefaultNumberingSystem(l.tag)}
	}
	return intlStringArray(r, list)
}

func intlDefaultNumberingSystem(tag *intlTag) string {
	if ns, ok := intlDefaultNumberingSystems[tag.language]; ok {
		return ns
	}
	switch tag.script {
	case "Arab":
		return "arab"
	case "Deva":
		return "deva"
	case "Beng":
		return "beng"
	case "Thai":
		return "thai"
	case "Mymr":
		return "mymr"
	case "Khmr":
		return "khmr"
	case "Laoo":
		return "laoo"
	case "Taml":
		return "tamldec"
	case "Telu":
		return "telu"
	case "Knda":
		return "knda"
	case "Mlym":
		return "mlym"
	case "Guru":
		return "guru"
	case "Gujr":
		return "gujr"
	case "Orya":
		return "orya"
	case "Hebr":
		return "hebr"
	case "Ethi":
		return "ethi"
	}
	return "latn"
}

func (r *Runtime) intlLocaleProtoGetTimeZones(call FunctionCall) Value {
	l := r.intlLocaleThis(call)
	if v, ok := l.tag.uKw["tz"]; ok && v != "" {
		return intlStringArray(r, []string{v})
	}
	if l.tag.region == "" {
		return _undefined
	}
	list := intlTimeZonesByRegion[l.tag.region]
	if len(list) == 0 {
		return _undefined
	}
	return intlStringArray(r, append([]string(nil), list...))
}

func (r *Runtime) intlLocaleProtoGetTextInfo(call FunctionCall) Value {
	l := r.intlLocaleThis(call)
	direction := "ltr"
	if intlScriptRTL[l.tag.script] {
		direction = "rtl"
	}
	info := r.newBaseObject(r.global.ObjectPrototype, classObject)
	info.setOwnStr("direction", intlStr(direction), false)
	return info.val
}

func (r *Runtime) intlLocaleProtoGetWeekInfo(call FunctionCall) Value {
	l := r.intlLocaleThis(call)
	mt := l.tag
	if mt.region == "" {
		mt = intlMaximize(mt)
	}
	region := mt.region
	if region == "" {
		region = "001"
	}
	firstDay := intlFirstDay[region]
	if v, ok := l.tag.uKw["fw"]; ok && v != "" {
		if d, ok := intlWeekdayNumbers[v]; ok {
			firstDay = d
		}
	}
	if firstDay == 0 {
		firstDay = intlFirstDay["001"]
	}
	start := intlWeekendStart[region]
	if start == 0 {
		start = intlWeekendStart["001"]
	}
	end := intlWeekendEnd[region]
	if end == 0 {
		end = intlWeekendEnd["001"]
	}
	if end == 0 {
		end = start
	}
	var vv []Value
	vv = append(vv, intToValue(int64(firstDay)))
	weekendVals := []Value{intToValue(int64(start))}
	if end != start {
		weekendVals = append(weekendVals, intToValue(int64(end)))
	}
	_ = vv
	info := r.newBaseObject(r.global.ObjectPrototype, classObject)
	info.setOwnStr("firstDay", intToValue(int64(firstDay)), false)
	info.setOwnStr("weekend", r.newArrayValues(weekendVals), false)
	return info.val
}

var intlWeekdayNumbers = map[string]int{
	"mon": 1, "tue": 2, "wed": 3, "thu": 4, "fri": 5, "sat": 6, "sun": 7,
}

func intlStringArray(r *Runtime, list []string) Value {
	vals := make([]Value, len(list))
	for i, s := range list {
		vals[i] = intlStr(s)
	}
	return r.newArrayValues(vals)
}

// sortedStrings returns a sorted copy of list.
func sortedStrings(list []string) []string {
	res := append([]string(nil), list...)
	sort.Strings(res)
	return res
}
