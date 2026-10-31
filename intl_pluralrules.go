package goja

import (
	"math"
	"strconv"
	"strings"

	"golang.org/x/text/feature/plural"
	"golang.org/x/text/language"
)

type intlPluralRulesObject struct {
	baseObject
	locale              string
	tag                 *intlTag
	typ                 string
	notation            string
	minInt              int
	minFrac, maxFrac    int
	minSig, maxSig      int // 0 == unused
	roundingIncrement   int
	roundingMode        string
	roundingPriority    string
	trailingZeroDisplay string
}

func (r *Runtime) getIntlPluralRules() *Object {
	ret := r.global.IntlPluralRules
	if ret == nil {
		ret = &Object{runtime: r}
		r.global.IntlPluralRules = ret
		proto := r.getIntlPluralRulesPrototype()
		r.newNativeFuncAndConstruct(ret, func(call FunctionCall) Value {
			panic(r.NewTypeError("Constructor Intl.PluralRules requires 'new'"))
		}, r.wrapNativeConstruct(r.builtin_newIntlPluralRules, ret, proto), proto, "PluralRules", intToValue(0))
		ret.self._putProp("supportedLocalesOf", r.newNativeFunc(r.intlPluralRulesSupportedLocalesOf, "supportedLocalesOf", 1), true, false, true)
	}
	return ret
}

func (r *Runtime) getIntlPluralRulesPrototype() *Object {
	ret := r.global.IntlPluralRulesPrototype
	if ret == nil {
		ret = r.newBaseObject(r.global.ObjectPrototype, classObject).val
		r.global.IntlPluralRulesPrototype = ret
		o := ret.self
		o._putProp("constructor", r.getIntlPluralRules(), true, false, true)
		o._putProp("select", r.newNativeFunc(r.intlPluralRulesProtoSelect, "select", 1), true, false, true)
		o._putProp("selectRange", r.newNativeFunc(r.intlPluralRulesProtoSelectRange, "selectRange", 2), true, false, true)
		o._putProp("resolvedOptions", r.newNativeFunc(r.intlPluralRulesProtoResolvedOptions, "resolvedOptions", 0), true, false, true)
		o.defineOwnPropertySym(SymToStringTag, PropertyDescriptor{
			Value:        intlStr("Intl.PluralRules"),
			Writable:     FLAG_FALSE,
			Enumerable:   FLAG_FALSE,
			Configurable: FLAG_TRUE,
		}, false)
	}
	return ret
}

func (r *Runtime) intlPluralRulesThis(call FunctionCall) *intlPluralRulesObject {
	o, ok := r.toObject(call.This).self.(*intlPluralRulesObject)
	if !ok {
		panic(r.NewTypeError("Intl.PluralRules method called on incompatible receiver"))
	}
	return o
}

func (r *Runtime) builtin_newIntlPluralRules(args []Value, proto *Object) *Object {
	var localesArg, optionsArg Value
	if len(args) > 0 {
		localesArg = args[0]
	}
	if len(args) > 1 {
		optionsArg = args[1]
	}
	requested := r.intlCanonicalizeLocaleList(localesArg)
	options := r.intlGetOptionsObject(optionsArg)

	typ := "cardinal"
	notation := "standard"
	minInt := 1
	minFrac, maxFrac := -1, -1
	minSig, maxSig := 0, 0
	roundingIncrement := 1
	roundingMode := "halfExpand"
	roundingPriority := "auto"
	trailingZeroDisplay := "auto"
	if options != nil {
		r.intlGetOption(options, "localeMatcher", "string", []string{"best fit", "lookup"}, _undefined)
		if v := r.intlGetOption(options, "type", "string", []string{"cardinal", "ordinal"}, _undefined); !IsUndefined(v) {
			typ = v.String()
		}
		if v := r.intlGetOption(options, "notation", "string", []string{"standard", "scientific", "engineering", "compact"}, _undefined); !IsUndefined(v) {
			notation = v.String()
		}
		minInt = intlGetOptionInt(options, "minimumIntegerDigits", 1, 21, 1, r)
		if v := nilSafe(options.self.getStr("minimumFractionDigits", nil)); !IsUndefined(v) {
			minFrac = intlCheckDigitOption(v.ToInteger(), "minimumFractionDigits", r)
		}
		if v := nilSafe(options.self.getStr("maximumFractionDigits", nil)); !IsUndefined(v) {
			maxFrac = intlCheckDigitOption(v.ToInteger(), "maximumFractionDigits", r)
		}
		if v := nilSafe(options.self.getStr("minimumSignificantDigits", nil)); !IsUndefined(v) {
			minSig = intlCheckDigitOption(v.ToInteger(), "minimumSignificantDigits", r)
			if minSig < 1 {
				minSig = 1
			}
		}
		if v := nilSafe(options.self.getStr("maximumSignificantDigits", nil)); !IsUndefined(v) {
			maxSig = intlCheckDigitOption(v.ToInteger(), "maximumSignificantDigits", r)
			if maxSig < 1 {
				maxSig = 1
			}
		}
		if v := nilSafe(options.self.getStr("roundingIncrement", nil)); !IsUndefined(v) {
			roundingIncrement = int(v.ToInteger())
			if roundingIncrement < 1 || roundingIncrement > 5000 {
				panic(r.intlRangeError("Invalid roundingIncrement"))
			}
		}
		if v := r.intlGetOption(options, "roundingMode", "string", []string{"ceil", "floor", "expand", "trunc", "halfCeil", "halfFloor", "halfExpand", "halfTrunc", "halfEven"}, _undefined); !IsUndefined(v) {
			roundingMode = v.String()
		}
		if v := r.intlGetOption(options, "roundingPriority", "string", []string{"auto", "morePrecision", "lessPrecision"}, _undefined); !IsUndefined(v) {
			roundingPriority = v.String()
		}
		if v := r.intlGetOption(options, "trailingZeroDisplay", "string", []string{"auto", "stripIfInteger"}, _undefined); !IsUndefined(v) {
			trailingZeroDisplay = v.String()
		}
	}
	if minSig == 0 && maxSig == 0 && (minFrac < 0 && maxFrac < 0) {
		minFrac, maxFrac = 0, 3
	}
	if minSig == 0 && maxSig != 0 {
		minSig = maxSig
	}
	if maxSig == 0 && minSig != 0 {
		maxSig = 21
	}
	if minFrac < 0 {
		minFrac = 0
	}
	if maxFrac < 0 {
		maxFrac = 3
		if maxFrac < minFrac {
			maxFrac = minFrac
		}
	}
	if maxFrac < minFrac {
		maxFrac = minFrac
	}

	locale, tag := r.intlResolveServiceLocale(requested, nil, options)

	o := &Object{runtime: r}
	obj := &intlPluralRulesObject{
		locale:              locale,
		tag:                 tag,
		typ:                 typ,
		notation:            notation,
		minInt:              minInt,
		minFrac:             minFrac,
		maxFrac:             maxFrac,
		minSig:              minSig,
		maxSig:              maxSig,
		roundingIncrement:   roundingIncrement,
		roundingMode:        roundingMode,
		roundingPriority:    roundingPriority,
		trailingZeroDisplay: trailingZeroDisplay,
	}
	obj.class = classObject
	obj.val = o
	obj.extensible = true
	obj.prototype = proto
	obj.init()
	o.self = obj
	return o
}

func intlCheckDigitOption(v int64, name string, r *Runtime) int {
	n := int(v)
	if n < 0 || n > 100 {
		panic(r.intlRangeError("Invalid value for option " + name))
	}
	return n
}

func (r *Runtime) intlPluralRulesProtoSelect(call FunctionCall) Value {
	pr := r.intlPluralRulesThis(call)
	n := call.Argument(0).ToFloat()
	if math.IsNaN(n) {
		return intlStr("other")
	}
	return intlStr(pr.selectPlural(n))
}

func (r *Runtime) intlPluralRulesProtoSelectRange(call FunctionCall) Value {
	pr := r.intlPluralRulesThis(call)
	if len(call.Arguments) < 2 || IsUndefined(call.Arguments[0]) || IsUndefined(call.Arguments[1]) {
		panic(r.NewTypeError("selectRange requires two arguments"))
	}
	start := call.Argument(0).ToFloat()
	end := call.Argument(1).ToFloat()
	if math.IsNaN(start) || math.IsNaN(end) {
		panic(r.intlRangeError("Invalid range"))
	}
	startCat := pr.selectPlural(start)
	endCat := pr.selectPlural(end)
	if startCat == endCat {
		return intlStr(startCat)
	}
	return intlStr("other")
}

func (pr *intlPluralRulesObject) selectPlural(n float64) string {
	i, v, w, f, t := intlPluralOperands(n, pr.minInt, pr.minFrac, pr.maxFrac, pr.minSig, pr.maxSig, pr.roundingMode, pr.roundingIncrement)
	tag := language.Make(pr.tag.language)
	var form plural.Form
	if pr.typ == "ordinal" {
		form = plural.Ordinal.MatchPlural(tag, i, v, w, f, t)
	} else {
		form = plural.Cardinal.MatchPlural(tag, i, v, w, f, t)
	}
	return intlPluralFormName(form)
}

// intlPluralOperands computes the plural rule operands after formatting.
func intlPluralOperands(n float64, minInt, minFrac, maxFrac, minSig, maxSig int, roundingMode string, roundingIncrement int) (i, v, w, f, t int) {
	neg, intPart, fracPart := intlDecimalDigits(n)
	_ = neg
	if intPart == "" {
		intPart = "0"
	}
	if minSig > 0 || maxSig > 0 {
		intPart, fracPart = intlRoundToSignificant(intPart, fracPart, minSig, maxSig, roundingMode)
	} else {
		if roundingIncrement > 1 && maxFrac > 0 {
			// Round to a multiple of roundingIncrement.
			scale := math.Pow10(maxFrac)
			total, _ := strconv.ParseFloat(intPart+"."+fracPart, 64)
			inc := float64(roundingIncrement) / scale
			total = math.Round(total/inc) * inc
			_, intPart, fracPart = intlDecimalDigits(total)
		} else {
			intPart, fracPart = intlRoundDigits(intPart, fracPart, maxFrac, roundingMode)
		}
	}
	// Strip trailing zeroes, but keep at least minFrac digits.
	for len(fracPart) > minFrac && strings.HasSuffix(fracPart, "0") {
		fracPart = fracPart[:len(fracPart)-1]
	}
	for len(fracPart) < minFrac {
		fracPart += "0"
	}
	iv, _ := strconv.Atoi(intPart)
	v = len(fracPart)
	fv, _ := strconv.Atoi(fracPart)
	tv, _ := strconv.Atoi(strings.TrimRight(fracPart, "0"))
	return iv, v, 0, fv, tv
}

// intlRoundToSignificant rounds a decimal to the given significant digits.
func intlRoundToSignificant(intPart, fracPart string, minSig, maxSig int, mode string) (string, string) {
	digits := strings.TrimLeft(intPart, "0") + fracPart
	if digits == "" {
		digits = "0"
	}
	// Position of the first significant digit.
	lead := 0
	for lead < len(digits)-1 && digits[lead] == '0' {
		lead++
	}
	digits = digits[lead:]
	keep := maxSig
	if keep <= 0 || keep > len(digits) {
		keep = len(digits)
	}
	rest := digits[keep:]
	up := false
	switch mode {
	case "trunc":
	case "expand":
		up = intlHasNonZero(rest)
	case "ceil":
		up = intlHasNonZero(rest)
	case "floor":
	default:
		if len(rest) > 0 {
			first := rest[0]
			up = first > '5' || (first == '5' && (intlHasNonZero(rest[1:]) || mode == "halfExpand" || mode == "halfEven" && intlLastDigitOdd(digits[:keep])))
		}
	}
	if up {
		digits = intlIncrementDigits(digits[:keep])
		if len(digits) > keep {
			lead--
			if lead < 0 {
				lead = 0
				// value grew by one digit
				intPart = digits
				digits = ""
			}
		} else {
			digits = digits[:keep]
		}
	} else {
		digits = digits[:keep]
	}
	// Reassemble based on the original integer part length.
	intLen := len(strings.TrimLeft(intPart, "0"))
	if intLen == 0 {
		intLen = 1
	}
	total := intLen - lead + len(digits)
	if total <= 0 {
		return "0", strings.Repeat("0", -total) + digits
	}
	if total >= len(digits) {
		return digits + strings.Repeat("0", total-len(digits)), ""
	}
	return digits[:total], digits[total:]
}

func (r *Runtime) intlPluralRulesProtoResolvedOptions(call FunctionCall) Value {
	pr := r.intlPluralRulesThis(call)
	o := r.newBaseObject(r.global.ObjectPrototype, classObject)
	o.setOwnStr("locale", intlStr(pr.locale), false)
	o.setOwnStr("type", intlStr(pr.typ), false)
	o.setOwnStr("notation", intlStr(pr.notation), false)
	o.setOwnStr("minimumIntegerDigits", intToValue(int64(pr.minInt)), false)
	if pr.minSig == 0 && pr.maxSig == 0 {
		o.setOwnStr("minimumFractionDigits", intToValue(int64(pr.minFrac)), false)
		o.setOwnStr("maximumFractionDigits", intToValue(int64(pr.maxFrac)), false)
	} else {
		o.setOwnStr("minimumSignificantDigits", intToValue(int64(pr.minSig)), false)
		o.setOwnStr("maximumSignificantDigits", intToValue(int64(pr.maxSig)), false)
	}
	cats := intlPluralCategoriesFor(pr.tag.language, pr.typ)
	vals := make([]Value, len(cats))
	for i, c := range cats {
		vals[i] = intlStr(c)
	}
	o.setOwnStr("pluralCategories", r.newArrayValues(vals), false)
	o.setOwnStr("roundingIncrement", intToValue(int64(pr.roundingIncrement)), false)
	o.setOwnStr("roundingMode", intlStr(pr.roundingMode), false)
	o.setOwnStr("roundingPriority", intlStr(pr.roundingPriority), false)
	o.setOwnStr("trailingZeroDisplay", intlStr(pr.trailingZeroDisplay), false)
	return o.val
}

func (r *Runtime) intlPluralRulesSupportedLocalesOf(call FunctionCall) Value {
	return r.intlSupportedLocalesOf(call, intlHasLocaleData)
}

func intlPluralFormName(f plural.Form) string {
	switch f {
	case plural.Zero:
		return "zero"
	case plural.One:
		return "one"
	case plural.Two:
		return "two"
	case plural.Few:
		return "few"
	case plural.Many:
		return "many"
	}
	return "other"
}

func intlPluralCategoriesFor(lang, typ string) []string {
	src := intlCardinalCategories
	if typ == "ordinal" {
		src = intlOrdinalCategories
	}
	if cats, ok := src[lang]; ok {
		order := []string{"zero", "one", "two", "few", "many", "other"}
		var res []string
		for _, c := range order {
			for _, have := range cats {
				if have == c {
					res = append(res, c)
					break
				}
			}
		}
		return res
	}
	return []string{"other"}
}
