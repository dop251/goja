package goja

import (
	"math"
	"strings"

	"github.com/dop251/goja/unistring"
)

type intlDurationFormatObject struct {
	baseObject
	locale           string
	numberingSystem  string
	style            string
	unitStyles       [10]string
	unitDisplays     [10]string
	fractionalDigits int // -1 if not set
}

var intlDurationUnits = [10]string{
	"years", "months", "weeks", "days", "hours",
	"minutes", "seconds", "milliseconds", "microseconds", "nanoseconds",
}

func (r *Runtime) getIntlDurationFormat() *Object {
	ret := r.global.IntlDurationFormat
	if ret == nil {
		ret = &Object{runtime: r}
		r.global.IntlDurationFormat = ret
		proto := r.getIntlDurationFormatPrototype()
		r.newNativeFuncAndConstruct(ret, func(call FunctionCall) Value {
			panic(r.NewTypeError("Constructor Intl.DurationFormat requires 'new'"))
		}, r.wrapNativeConstruct(r.builtin_newIntlDurationFormat, ret, proto), proto, "DurationFormat", intToValue(0))
		ret.self._putProp("supportedLocalesOf", r.newNativeFunc(r.intlDurationFormatSupportedLocalesOf, "supportedLocalesOf", 1), true, false, true)
	}
	return ret
}

func (r *Runtime) getIntlDurationFormatPrototype() *Object {
	ret := r.global.IntlDurationFormatPrototype
	if ret == nil {
		ret = r.newBaseObject(r.global.ObjectPrototype, classObject).val
		r.global.IntlDurationFormatPrototype = ret
		o := ret.self
		o._putProp("constructor", r.getIntlDurationFormat(), true, false, true)
		o._putProp("format", r.newNativeFunc(r.intlDurationFormatProtoFormat, "format", 1), true, false, true)
		o._putProp("formatToParts", r.newNativeFunc(r.intlDurationFormatProtoFormatToParts, "formatToParts", 1), true, false, true)
		o._putProp("resolvedOptions", r.newNativeFunc(r.intlDurationFormatProtoResolvedOptions, "resolvedOptions", 0), true, false, true)
		o.defineOwnPropertySym(SymToStringTag, PropertyDescriptor{
			Value:        intlStr("Intl.DurationFormat"),
			Writable:     FLAG_FALSE,
			Enumerable:   FLAG_FALSE,
			Configurable: FLAG_TRUE,
		}, false)
	}
	return ret
}

func (r *Runtime) intlDurationFormatThis(call FunctionCall) *intlDurationFormatObject {
	o, ok := r.toObject(call.This).self.(*intlDurationFormatObject)
	if !ok {
		panic(r.NewTypeError("Intl.DurationFormat method called on incompatible receiver"))
	}
	return o
}

func (r *Runtime) builtin_newIntlDurationFormat(args []Value, proto *Object) *Object {
	var localesArg, optionsArg Value
	if len(args) > 0 {
		localesArg = args[0]
	}
	if len(args) > 1 {
		optionsArg = args[1]
	}
	requested := r.intlCanonicalizeLocaleList(localesArg)
	if IsNull(optionsArg) {
		panic(r.NewTypeError("Options must be an object"))
	}
	options := r.intlGetOptionsObject(optionsArg)
	obj := &intlDurationFormatObject{fractionalDigits: -1}
	obj.style = "short"
	numberingSystem := ""
	if options != nil {
		r.intlGetOption(options, "localeMatcher", "string", []string{"best fit", "lookup"}, _undefined)
		if v := r.intlGetOption(options, "numberingSystem", "string", nil, _undefined); !IsUndefined(v) {
			ns := strings.ToLower(v.String())
			if !intlIsUnicodeExtensionValue(ns) || !intlValidNumberingSystem(ns) {
				panic(r.intlRangeError("Invalid numberingSystem"))
			}
			numberingSystem = ns
		}
		if v := r.intlGetOption(options, "style", "string", []string{"long", "short", "narrow", "digital"}, _undefined); !IsUndefined(v) {
			obj.style = v.String()
		}
		if obj.style == "digital" {
			for _, unit := range intlDurationUnits {
				if v := r.intlGetOption(options, unit, "string", []string{"long", "short", "narrow", "numeric", "2-digit"}, _undefined); !IsUndefined(v) {
					st := v.String()
					if st == "long" || st == "short" || st == "narrow" {
						panic(r.NewTypeError("Invalid unit style for digital style"))
					}
				}
			}
		}
		unitStyles := []string{"long", "short", "narrow", "numeric", "2-digit"}
		for i, unit := range intlDurationUnits {
			if v := r.intlGetOption(options, unit, "string", unitStyles, _undefined); !IsUndefined(v) {
				obj.unitStyles[i] = v.String()
			} else if obj.style == "digital" {
				switch unit {
				case "hours":
					obj.unitStyles[i] = "numeric"
				case "minutes", "seconds":
					obj.unitStyles[i] = "2-digit"
				case "milliseconds", "microseconds", "nanoseconds":
					obj.unitStyles[i] = "numeric"
				default:
					obj.unitStyles[i] = "short"
				}
			} else {
				obj.unitStyles[i] = obj.style
			}
			if v := r.intlGetOption(options, unit+"Display", "string", []string{"auto", "always"}, _undefined); !IsUndefined(v) {
				obj.unitDisplays[i] = v.String()
			} else if obj.style == "digital" {
				switch unit {
				case "hours", "minutes", "seconds":
					obj.unitDisplays[i] = "always"
				default:
					obj.unitDisplays[i] = "auto"
				}
			} else {
				obj.unitDisplays[i] = "auto"
			}
		}
		if v := nilSafe(options.self.getStr("fractionalDigits", nil)); !IsUndefined(v) {
			n := int(v.ToInteger())
			if n < 0 || n > 9 {
				panic(r.intlRangeError("Invalid fractionalDigits"))
			}
			obj.fractionalDigits = n
		}
	} else {
		for i, unit := range intlDurationUnits {
			if obj.style == "digital" {
				switch unit {
				case "hours":
					obj.unitStyles[i] = "numeric"
					obj.unitDisplays[i] = "always"
				case "minutes", "seconds":
					obj.unitStyles[i] = "2-digit"
					obj.unitDisplays[i] = "always"
				case "milliseconds", "microseconds", "nanoseconds":
					obj.unitStyles[i] = "numeric"
					obj.unitDisplays[i] = "auto"
				default:
					obj.unitStyles[i] = "short"
					obj.unitDisplays[i] = "auto"
				}
			} else {
				obj.unitStyles[i] = obj.style
				obj.unitDisplays[i] = "auto"
			}
		}
	}
	locale, tag := r.intlResolveServiceLocale(requested, []string{"nu"}, options)
	if numberingSystem == "" {
		if ns, ok := tag.uKw["nu"]; ok && ns != "" {
			numberingSystem = ns
		} else if key, ok := intlDataLocale(tag); ok {
			numberingSystem = intlNumberLocaleFor(key).DefaultNS
		} else {
			numberingSystem = "latn"
		}
	}
	obj.locale = locale
	obj.numberingSystem = numberingSystem

	o := &Object{runtime: r}
	obj.class = classObject
	obj.val = o
	obj.extensible = true
	obj.prototype = proto
	obj.init()
	o.self = obj
	return o
}

type intlDurationValue struct {
	years, months, weeks, days, hours, minutes, seconds float64
	milliseconds, microseconds, nanoseconds             float64
}

func (r *Runtime) intlToDurationRecord(arg Value) intlDurationValue {
	if arg == nil || IsUndefined(arg) {
		panic(r.NewTypeError("Duration is required"))
	}
	o := arg.ToObject(r)
	var d intlDurationValue
	fields := []*float64{&d.years, &d.months, &d.weeks, &d.days, &d.hours, &d.minutes, &d.seconds, &d.milliseconds, &d.microseconds, &d.nanoseconds}
	for i, name := range intlDurationUnits {
		v := nilSafe(o.self.getStr(unistring.NewFromString(name), nil))
		n := v.ToFloat()
		if math.IsNaN(n) || math.IsInf(n, 0) {
			panic(r.intlRangeError("Invalid duration value for " + name))
		}
		if math.Abs(n) >= 4294967296 {
			panic(r.intlRangeError("Duration out of range"))
		}
		*fields[i] = n
	}
	neg, pos := false, false
	for _, f := range fields {
		if *f < 0 || (*f == 0 && math.Signbit(*f)) {
			neg = true
		} else if *f > 0 {
			pos = true
		}
	}
	if neg && pos {
		panic(r.intlRangeError("Mixed-sign duration"))
	}
	return d
}

func (d *intlDurationFormatObject) format(duration intlDurationValue) []intlStringPart {
	values := []float64{duration.years, duration.months, duration.weeks, duration.days,
		duration.hours, duration.minutes, duration.seconds, duration.milliseconds,
		duration.microseconds, duration.nanoseconds}
	anyNegative := false
	for _, v := range values {
		if v < 0 || (v == 0 && math.Signbit(v)) {
			anyNegative = true
		}
	}
	var result [][]intlStringPart
	needSeparator := false
	displayNegativeSign := true
	for i, unit := range intlDurationUnits {
		value := values[i]
		style := d.unitStyles[i]
		display := d.unitDisplays[i]
		nfUnit := strings.TrimSuffix(unit, "s")
		done := false
		maxFrac, minFrac := -1, 0
		if unit == "seconds" || unit == "milliseconds" || unit == "microseconds" {
			nextStyle := d.unitStyles[i+1]
			if nextStyle == "numeric" {
				switch unit {
				case "seconds":
					value = duration.seconds + duration.milliseconds/1e3 + duration.microseconds/1e6 + duration.nanoseconds/1e9
				case "milliseconds":
					value = duration.milliseconds + duration.microseconds/1e3 + duration.nanoseconds/1e6
				case "microseconds":
					value = duration.microseconds + duration.nanoseconds/1e3
				}
				if d.fractionalDigits >= 0 {
					maxFrac = d.fractionalDigits
					minFrac = d.fractionalDigits
				} else {
					maxFrac = 9
					minFrac = 0
				}
				done = true
			}
		}
		displayRequired := false
		if unit == "minutes" && needSeparator {
			displayRequired = d.unitDisplays[6] == "always" || duration.seconds != 0 ||
				duration.milliseconds != 0 || duration.microseconds != 0 || duration.nanoseconds != 0
		}
		if !(value != 0 || display != "auto" || displayRequired) {
			if done {
				break
			}
			continue
		}
		isFirst := displayNegativeSign
		if isFirst {
			displayNegativeSign = false
			if value == 0 && anyNegative {
				value = math.Copysign(0, -1)
			}
		}
		opts := intlDefaultNumberOptions()
		opts.numberingSystem = d.numberingSystem
		if !isFirst {
			opts.signDisplay = "never"
		}
		if maxFrac >= 0 {
			opts.maxFrac = maxFrac
			opts.minFrac = minFrac
			opts.roundingMode = "trunc"
			opts.fractionSpecified = true
		}
		if style == "2-digit" {
			opts.minInt = 2
		}
		if style != "numeric" && style != "2-digit" {
			opts.style = "unit"
			opts.unit = nfUnit
			opts.unitDisplay = style
		} else {
			opts.useGrouping = "never"
		}
		loc := intlNumberLocaleFor(d.dataKey())
		parts := intlFormatNumberParts(value, opts, loc)

		var list []intlStringPart
		if needSeparator {
			if len(result) > 0 {
				list = result[len(result)-1]
			}
			list = append(list, intlStringPart{"literal", ":"})
		}
		for _, p := range parts {
			list = append(list, intlStringPart{p.typ, p.value})
		}
		if !needSeparator {
			if style == "2-digit" || style == "numeric" {
				needSeparator = true
			}
			result = append(result, list)
		} else {
			result[len(result)-1] = list
		}
		if done {
			break
		}
	}
	if len(result) == 0 {
		return nil
	}
	// Join groups with the unit list format.
	lfStyle := d.style
	if lfStyle == "digital" {
		lfStyle = "short"
	}
	lf := &intlListFormatObject{locale: d.locale, typ: "unit", style: lfStyle}
	p := lf.patterns(lf.dataLocale())
	strings := make([]string, len(result))
	for i, parts := range result {
		strings[i] = intlPartsToString(parts)
	}
	listParts := intlFormatListParts(strings, p)
	var out []intlStringPart
	idx := 0
	for _, lp := range listParts {
		if lp.typ == "element" {
			out = append(out, result[idx]...)
			idx++
		} else {
			out = append(out, intlStringPart{"literal", lp.value})
		}
	}
	return out
}

func (d *intlDurationFormatObject) dataKey() string {
	if t, ok := intlParseTag(d.locale); ok {
		if k, ok := intlDataLocale(t); ok {
			return k
		}
	}
	return "en"
}

func (r *Runtime) intlDurationFormatProtoFormat(call FunctionCall) Value {
	d := r.intlDurationFormatThis(call)
	parts := d.format(r.intlToDurationRecord(call.Argument(0)))
	return intlStr(intlPartsToString(parts))
}

func (r *Runtime) intlDurationFormatProtoFormatToParts(call FunctionCall) Value {
	d := r.intlDurationFormatThis(call)
	parts := d.format(r.intlToDurationRecord(call.Argument(0)))
	vals := make([]Value, len(parts))
	for i, p := range parts {
		obj := r.newBaseObject(r.global.ObjectPrototype, classObject)
		obj.setOwnStr("type", asciiString(p.typ), false)
		obj.setOwnStr("value", intlStr(p.value), false)
		vals[i] = obj.val
	}
	return r.newArrayValues(vals)
}

func (r *Runtime) intlDurationFormatProtoResolvedOptions(call FunctionCall) Value {
	d := r.intlDurationFormatThis(call)
	o := r.newBaseObject(r.global.ObjectPrototype, classObject)
	o.setOwnStr("locale", intlStr(d.locale), false)
	o.setOwnStr("numberingSystem", intlStr(d.numberingSystem), false)
	o.setOwnStr("style", intlStr(d.style), false)
	for i, unit := range intlDurationUnits {
		o.setOwnStr(unistring.NewFromString(unit), intlStr(d.unitStyles[i]), false)
		o.setOwnStr(unistring.NewFromString(unit+"Display"), intlStr(d.unitDisplays[i]), false)
	}
	if d.fractionalDigits >= 0 {
		o.setOwnStr("fractionalDigits", intToValue(int64(d.fractionalDigits)), false)
	}
	return o.val
}

func (r *Runtime) intlDurationFormatSupportedLocalesOf(call FunctionCall) Value {
	return r.intlSupportedLocalesOf(call, intlHasLocaleData)
}
