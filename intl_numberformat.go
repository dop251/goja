package goja

import (
	"strings"

	"github.com/dop251/goja/unistring"
)

type intlNumberFormatObject struct {
	baseObject
	locale     string
	tag        *intlTag
	dataLocale string
	opts       *intlNumberFormatOptions
	bound      *Object
}

func (r *Runtime) getIntlNumberFormat() *Object {
	ret := r.global.IntlNumberFormat
	if ret == nil {
		ret = &Object{runtime: r}
		r.global.IntlNumberFormat = ret
		proto := r.getIntlNumberFormatPrototype()
		r.newNativeFuncAndConstruct(ret, func(call FunctionCall) Value {
			return r.builtin_newIntlNumberFormat(call.Arguments, r.getIntlNumberFormatPrototype())
		}, r.wrapNativeConstruct(r.builtin_newIntlNumberFormat, ret, proto), proto, "NumberFormat", intToValue(0))
		ret.self._putProp("supportedLocalesOf", r.newNativeFunc(r.intlNumberFormatSupportedLocalesOf, "supportedLocalesOf", 1), true, false, true)
	}
	return ret
}

func (r *Runtime) getIntlNumberFormatPrototype() *Object {
	ret := r.global.IntlNumberFormatPrototype
	if ret == nil {
		ret = r.newBaseObject(r.global.ObjectPrototype, classObject).val
		r.global.IntlNumberFormatPrototype = ret
		o := ret.self
		o._putProp("constructor", r.getIntlNumberFormat(), true, false, true)
		o.setOwnStr("format", &valueProperty{
			getterFunc:   r.newNativeFunc(r.intlNumberFormatGetFormat, "get format", 0),
			accessor:     true,
			configurable: true,
		}, false)
		o._putProp("formatToParts", r.newNativeFunc(r.intlNumberFormatProtoFormatToParts, "formatToParts", 1), true, false, true)
		o._putProp("formatRange", r.newNativeFunc(r.intlNumberFormatProtoFormatRange, "formatRange", 2), true, false, true)
		o._putProp("formatRangeToParts", r.newNativeFunc(r.intlNumberFormatProtoFormatRangeToParts, "formatRangeToParts", 2), true, false, true)
		o._putProp("resolvedOptions", r.newNativeFunc(r.intlNumberFormatProtoResolvedOptions, "resolvedOptions", 0), true, false, true)
		o.defineOwnPropertySym(SymToStringTag, PropertyDescriptor{
			Value:        intlStr("Intl.NumberFormat"),
			Writable:     FLAG_FALSE,
			Enumerable:   FLAG_FALSE,
			Configurable: FLAG_TRUE,
		}, false)
	}
	return ret
}

func (r *Runtime) intlNumberFormatThis(call FunctionCall) *intlNumberFormatObject {
	o, ok := r.toObject(call.This).self.(*intlNumberFormatObject)
	if !ok {
		panic(r.NewTypeError("Intl.NumberFormat method called on incompatible receiver"))
	}
	return o
}

func (r *Runtime) builtin_newIntlNumberFormat(args []Value, proto *Object) *Object {
	var localesArg, optionsArg Value
	if len(args) > 0 {
		localesArg = args[0]
	}
	if len(args) > 1 {
		optionsArg = args[1]
	}
	requested := r.intlCanonicalizeLocaleList(localesArg)
	options := r.intlGetOptionsObject(optionsArg)
	opts := intlDefaultNumberOptions()

	if options != nil {
		r.intlGetOption(options, "localeMatcher", "string", []string{"best fit", "lookup"}, _undefined)
		if v := r.intlGetOption(options, "numberingSystem", "string", nil, _undefined); !IsUndefined(v) {
			ns := strings.ToLower(v.String())
			if !intlIsUnicodeExtensionValue(ns) || !intlValidNumberingSystem(ns) {
				panic(r.intlRangeError("Invalid numberingSystem"))
			}
			opts.numberingSystem = ns
		}
		if v := r.intlGetOption(options, "style", "string", []string{"decimal", "percent", "currency", "unit"}, _undefined); !IsUndefined(v) {
			opts.style = v.String()
		}
		currency := r.intlGetOption(options, "currency", "string", nil, _undefined)
		if !IsUndefined(currency) {
			cur := currency.String()
			if !intlIsWellFormedCurrencyCode(cur) {
				panic(r.intlRangeError("Invalid currency code: " + cur))
			}
			opts.currency = strings.ToUpper(cur)
		}
		if opts.style == "currency" && opts.currency == "" {
			panic(r.NewTypeError("Currency code is required with currency style."))
		}
		if v := r.intlGetOption(options, "currencyDisplay", "string", []string{"code", "symbol", "narrowSymbol", "name"}, _undefined); !IsUndefined(v) {
			opts.currencyDisplay = v.String()
		}
		if v := r.intlGetOption(options, "currencySign", "string", []string{"standard", "accounting"}, _undefined); !IsUndefined(v) {
			opts.currencySign = v.String()
		}
		unit := r.intlGetOption(options, "unit", "string", nil, _undefined)
		if !IsUndefined(unit) {
			u := unit.String()
			if !intlIsWellFormedUnit(u) {
				panic(r.intlRangeError("Invalid unit: " + u))
			}
			opts.unit = u
		}
		if opts.style == "unit" && opts.unit == "" {
			panic(r.NewTypeError("Unit is required with unit style."))
		}
		if v := r.intlGetOption(options, "unitDisplay", "string", []string{"long", "short", "narrow"}, _undefined); !IsUndefined(v) {
			opts.unitDisplay = v.String()
		}
		if v := r.intlGetOption(options, "notation", "string", []string{"standard", "scientific", "engineering", "compact"}, _undefined); !IsUndefined(v) {
			opts.notation = v.String()
		}
		if v := r.intlGetOption(options, "compactDisplay", "string", []string{"short", "long"}, _undefined); !IsUndefined(v) {
			opts.compactDisplay = v.String()
		}
		if v := nilSafe(options.self.getStr("useGrouping", nil)); !IsUndefined(v) {
			switch sv := v.(type) {
			case valueBool:
				if bool(sv) {
					opts.useGrouping = "always"
				} else {
					opts.useGrouping = "never"
				}
			default:
				s := v.toString().String()
				switch s {
				case "always", "auto", "never", "min2", "min5":
					opts.useGrouping = s
				default:
					panic(r.intlRangeError("Invalid useGrouping"))
				}
			}
		}
		if v := r.intlGetOption(options, "signDisplay", "string", []string{"auto", "never", "always", "exceptZero", "negative", "negativeExceptZero"}, _undefined); !IsUndefined(v) {
			opts.signDisplay = v.String()
		}
		opts.minInt = intlGetOptionInt(options, "minimumIntegerDigits", 1, 21, 1, r)
		minFracSet := false
		maxFracSet := false
		if v := nilSafe(options.self.getStr("minimumFractionDigits", nil)); !IsUndefined(v) {
			opts.minFrac = intlCheckDigitOption(v.ToInteger(), "minimumFractionDigits", r)
			minFracSet = true
		}
		if v := nilSafe(options.self.getStr("maximumFractionDigits", nil)); !IsUndefined(v) {
			opts.maxFrac = intlCheckDigitOption(v.ToInteger(), "maximumFractionDigits", r)
			maxFracSet = true
		}
		if v := nilSafe(options.self.getStr("minimumSignificantDigits", nil)); !IsUndefined(v) {
			opts.minSig = intlCheckDigitOption(v.ToInteger(), "minimumSignificantDigits", r)
			if opts.minSig < 1 {
				opts.minSig = 1
			}
		}
		if v := nilSafe(options.self.getStr("maximumSignificantDigits", nil)); !IsUndefined(v) {
			opts.maxSig = intlCheckDigitOption(v.ToInteger(), "maximumSignificantDigits", r)
			if opts.maxSig < 1 {
				opts.maxSig = 1
			}
		}
		if v := nilSafe(options.self.getStr("roundingIncrement", nil)); !IsUndefined(v) {
			opts.roundingIncrement = int(v.ToInteger())
		}
		if v := r.intlGetOption(options, "roundingMode", "string", []string{"ceil", "floor", "expand", "trunc", "halfCeil", "halfFloor", "halfExpand", "halfTrunc", "halfEven"}, _undefined); !IsUndefined(v) {
			opts.roundingMode = v.String()
		}
		if v := r.intlGetOption(options, "roundingPriority", "string", []string{"auto", "morePrecision", "lessPrecision"}, _undefined); !IsUndefined(v) {
			opts.roundingPriority = v.String()
		}
		if v := r.intlGetOption(options, "trailingZeroDisplay", "string", []string{"auto", "stripIfInteger"}, _undefined); !IsUndefined(v) {
			opts.trailingZeroDisplay = v.String()
		}
		opts.fractionSpecified = minFracSet || maxFracSet
		_ = maxFracSet
	}

	if opts.style != "currency" && opts.fractionSpecified {
		// keep user digits
	} else if opts.style == "currency" {
		d := intlCurrencyDefaultDigitsValue
		if dd, ok := intlCurrencyDefaultDigits[opts.currency]; ok {
			d = dd
		}
		if !opts.fractionSpecified {
			opts.minFrac, opts.maxFrac = d, d
		}
	} else if opts.style == "percent" && !opts.fractionSpecified {
		opts.minFrac, opts.maxFrac = 0, 0
	}
	if opts.minSig == 0 && opts.maxSig != 0 {
		opts.minSig = opts.maxSig
	}
	if opts.maxSig == 0 && opts.minSig != 0 {
		opts.maxSig = 21
	}
	if opts.maxFrac < opts.minFrac {
		opts.maxFrac = opts.minFrac
	}

	locale, tag := r.intlResolveServiceLocale(requested, []string{"nu"}, options)
	dataLocale := "en"
	if key, ok := intlDataLocale(tag); ok {
		dataLocale = key
	}
	if opts.numberingSystem == "" {
		if ns, ok := tag.uKw["nu"]; ok && ns != "" {
			opts.numberingSystem = ns
		} else {
			opts.numberingSystem = intlNumberLocaleFor(dataLocale).DefaultNS
		}
	}

	o := &Object{runtime: r}
	obj := &intlNumberFormatObject{locale: locale, tag: tag, dataLocale: dataLocale, opts: opts}
	obj.class = classObject
	obj.val = o
	obj.extensible = true
	obj.prototype = proto
	obj.init()
	o.self = obj
	return o
}

func intlValidNumberingSystem(ns string) bool {
	_, ok := intlNumberingSystemDigits[ns]
	return ok
}

func intlIsWellFormedCurrencyCode(s string) bool {
	if len(s) != 3 || !intlIsAlpha(s) {
		return false
	}
	return true
}

func intlIsWellFormedUnit(s string) bool {
	parts := strings.Split(s, "-per-")
	if len(parts) > 2 {
		return false
	}
	for _, p := range parts {
		if p == "" {
			return false
		}
		for _, sub := range strings.Split(p, "-") {
			if len(sub) < 3 || len(sub) > 8 || !intlIsAlnum(sub) {
				return false
			}
		}
	}
	return true
}

func (nf *intlNumberFormatObject) formatParts(n float64) []intlStringPart {
	return intlFormatNumberParts(n, nf.opts, intlNumberLocaleFor(nf.dataLocale))
}

func (r *Runtime) intlNumberFormatGetFormat(call FunctionCall) Value {
	nf := r.intlNumberFormatThis(call)
	if nf.bound == nil {
		nf.bound = r.newNativeFunc(func(call FunctionCall) Value {
			return intlStr(intlPartsToString(nf.formatParts(call.Argument(0).ToFloat())))
		}, "format", 1)
	}
	return nf.bound
}

func intlPartsToString(parts []intlStringPart) string {
	var sb strings.Builder
	for _, p := range parts {
		sb.WriteString(p.value)
	}
	return sb.String()
}

func (r *Runtime) intlNumberFormatProtoFormatToParts(call FunctionCall) Value {
	nf := r.intlNumberFormatThis(call)
	parts := nf.formatParts(call.Argument(0).ToFloat())
	return r.intlPartsArray(parts)
}

func (r *Runtime) intlNumberFormatProtoFormatRange(call FunctionCall) Value {
	nf := r.intlNumberFormatThis(call)
	start := call.Argument(0).ToFloat()
	end := call.Argument(1).ToFloat()
	return intlStr(intlPartsToString(nf.formatParts(start)) + "–" + intlPartsToString(nf.formatParts(end)))
}

func (r *Runtime) intlNumberFormatProtoFormatRangeToParts(call FunctionCall) Value {
	nf := r.intlNumberFormatThis(call)
	start := call.Argument(0).ToFloat()
	end := call.Argument(1).ToFloat()
	var vals []Value
	add := func(typ, val string, source string) {
		obj := r.newBaseObject(r.global.ObjectPrototype, classObject)
		obj.setOwnStr("type", intlStr(typ), false)
		obj.setOwnStr("value", intlStr(val), false)
		obj.setOwnStr("source", intlStr(source), false)
		vals = append(vals, obj.val)
	}
	for _, p := range nf.formatParts(start) {
		add(p.typ, p.value, "startRange")
	}
	add("literal", "–", "shared")
	for _, p := range nf.formatParts(end) {
		add(p.typ, p.value, "endRange")
	}
	return r.newArrayValues(vals)
}

func (r *Runtime) intlNumberFormatProtoResolvedOptions(call FunctionCall) Value {
	nf := r.intlNumberFormatThis(call)
	opts := nf.opts
	o := r.newBaseObject(r.global.ObjectPrototype, classObject)
	o.setOwnStr("locale", intlStr(nf.locale), false)
	o.setOwnStr("numberingSystem", intlStr(opts.numberingSystem), false)
	o.setOwnStr("style", intlStr(opts.style), false)
	if opts.style == "currency" {
		o.setOwnStr("currency", intlStr(opts.currency), false)
		o.setOwnStr("currencyDisplay", intlStr(opts.currencyDisplay), false)
		o.setOwnStr("currencySign", intlStr(opts.currencySign), false)
	} else if opts.style == "unit" {
		o.setOwnStr("unit", intlStr(opts.unit), false)
		o.setOwnStr("unitDisplay", intlStr(opts.unitDisplay), false)
	}
	o.setOwnStr("minimumIntegerDigits", intToValue(int64(opts.minInt)), false)
	if opts.minSig == 0 && opts.maxSig == 0 {
		o.setOwnStr("minimumFractionDigits", intToValue(int64(opts.minFrac)), false)
		o.setOwnStr("maximumFractionDigits", intToValue(int64(opts.maxFrac)), false)
	} else {
		o.setOwnStr("minimumSignificantDigits", intToValue(int64(opts.minSig)), false)
		o.setOwnStr("maximumSignificantDigits", intToValue(int64(opts.maxSig)), false)
	}
	o.setOwnStr("useGrouping", intlStr(opts.useGrouping), false)
	o.setOwnStr("notation", intlStr(opts.notation), false)
	if opts.notation == "compact" {
		o.setOwnStr("compactDisplay", intlStr(opts.compactDisplay), false)
	}
	o.setOwnStr("signDisplay", intlStr(opts.signDisplay), false)
	o.setOwnStr("roundingIncrement", intToValue(int64(opts.roundingIncrement)), false)
	o.setOwnStr("roundingMode", intlStr(opts.roundingMode), false)
	o.setOwnStr("roundingPriority", intlStr(opts.roundingPriority), false)
	o.setOwnStr("trailingZeroDisplay", intlStr(opts.trailingZeroDisplay), false)
	return o.val
}

func (r *Runtime) intlNumberFormatSupportedLocalesOf(call FunctionCall) Value {
	return r.intlSupportedLocalesOf(call, intlHasLocaleData)
}

var _ = unistring.NewFromString
