package goja

import (
	"math"
	"strconv"
	"strings"

	"github.com/dop251/goja/unistring"
)

// intlServiceBase holds the state shared by the Intl service constructors.
type intlServiceBase struct {
	locale     string
	tag        *intlTag
	dataLocale string
}

// intlSupportedLocaleData reports whether any locale data exists for tag.
func intlHasLocaleData(tag *intlTag) bool {
	if tag.language == "" {
		return false
	}
	for _, key := range intlLocaleLookupKeys(tag) {
		if _, ok := intlNumberLocales[key]; ok {
			return true
		}
	}
	return false
}

func intlLocaleLookupKeys(tag *intlTag) []string {
	var keys []string
	base := tag.baseName()
	keys = append(keys, base)
	if tag.script != "" && tag.region != "" {
		keys = append(keys, tag.language+"-"+tag.script)
		keys = append(keys, tag.language+"-"+tag.region)
	} else if tag.script != "" {
		keys = append(keys, tag.language)
	} else if tag.region != "" {
		keys = append(keys, tag.language)
	}
	keys = append(keys, tag.language)
	return keys
}

func intlDataLocale(tag *intlTag) (string, bool) {
	for _, key := range intlLocaleLookupKeys(tag) {
		if _, ok := intlNumberLocales[key]; ok {
			return key, true
		}
	}
	return "", false
}

// intlResolveServiceLocale resolves the requested locales and returns the
// resolved locale string and the tag describing it.
func (r *Runtime) intlResolveServiceLocale(requested []string, relevantKeys []string, options *Object) (string, *intlTag) {
	if len(requested) == 0 {
		requested = []string{r.intlDefaultLocale()}
	}
	for _, req := range requested {
		tag, ok := intlParseTag(req)
		if !ok {
			continue
		}
		tag = intlCanonicalizeTag(tag)
		if tag.language == "und" {
			tag = intlMaximize(tag)
		}
		if _, ok := intlDataLocale(tag); ok {
			base := tag.baseName()
			ext := intlBuildRelevantExtension(tag, relevantKeys, options)
			if ext != "" {
				return base + "-u-" + ext, tag
			}
			return base, tag
		}
	}
	// Fall back to the default locale.
	def, _ := intlParseTag(r.intlDefaultLocale())
	def = intlCanonicalizeTag(def)
	return def.baseName(), def
}

// intlBuildRelevantExtension returns the canonical u-extension keyword portion
// for the relevant keys of the requested locale.
func intlBuildRelevantExtension(tag *intlTag, keys []string, options *Object) string {
	var sb strings.Builder
	for _, k := range keys {
		v, ok := tag.uKw[k]
		if !ok {
			continue
		}
		if !intlRelevantExtensionValueValid(k, v) {
			continue
		}
		if sb.Len() > 0 {
			sb.WriteString("-")
		}
		sb.WriteString(k)
		if v != "" {
			sb.WriteString("-")
			sb.WriteString(v)
		}
	}
	return sb.String()
}

func intlRelevantExtensionValueValid(k, v string) bool {
	switch k {
	case "ca":
		return intlContains(intlCalendarValues, v)
	case "nu":
		return intlValidNumberingSystem(v)
	case "hc":
		return intlContains(intlHourCycleValues, v)
	case "co":
		return intlContains(intlCollationValues, v)
	}
	return true
}

// intlSupportedLocalesOf implements the shared supportedLocalesOf behaviour.
func (r *Runtime) intlSupportedLocalesOf(call FunctionCall, supported func(*intlTag) bool) Value {
	requested := r.intlCanonicalizeLocaleList(call.Argument(0))
	var result []string
	for _, req := range requested {
		tag, ok := intlParseTag(req)
		if !ok {
			continue
		}
		if supported(intlCanonicalizeTag(tag)) {
			result = append(result, req)
		}
	}
	vals := make([]Value, len(result))
	for i, s := range result {
		vals[i] = asciiString(s)
	}
	return r.newArrayValues(vals)
}

// intlGetOptionsObject returns the options argument as an object, or nil.
func (r *Runtime) intlGetOptionsObject(v Value) *Object {
	if v == nil || IsUndefined(v) {
		return nil
	}
	if IsNull(v) {
		return nil
	}
	if _, ok := v.(*Object); !ok {
		panic(r.NewTypeError("Options must be an object"))
	}
	return v.ToObject(r)
}

func intlGetOptionString(options *Object, name string, r *Runtime) Value {
	if options == nil {
		return _undefined
	}
	return r.intlGetOption(options, name, "string", nil, _undefined)
}

func intlGetOptionInt(options *Object, name string, min, max, def int, r *Runtime) int {
	if options == nil {
		return def
	}
	v := nilSafe(options.self.getStr(unistring.NewFromString(name), nil))
	if IsUndefined(v) {
		return def
	}
	n := int(v.ToInteger())
	if n < min || n > max {
		panic(r.intlRangeError("Value out of range for option " + name))
	}
	return n
}

func intlGetOptionBool(options *Object, name string, r *Runtime) bool {
	if options == nil {
		return false
	}
	v := nilSafe(options.self.getStr(unistring.NewFromString(name), nil))
	return v.ToBoolean()
}

// ---------- decimal formatting primitives ----------

// intlRoundDecimal rounds the decimal representation produced by
// strconv.FormatFloat to the given number of fraction digits.
func intlDecimalDigits(n float64) (neg bool, intPart, fracPart string) {
	if math.IsNaN(n) {
		return false, "", ""
	}
	if n < 0 || math.Signbit(n) {
		neg = true
		n = -n
	}
	s := strconv.FormatFloat(n, 'f', -1, 64)
	if i := strings.IndexByte(s, '.'); i >= 0 {
		intPart, fracPart = s[:i], s[i+1:]
	} else {
		intPart = s
	}
	return
}

// intlRoundDecimalString rounds a non-negative decimal digit string
// (intPart.fracPart) to the requested number of fraction digits.
func intlRoundDigits(intPart, fracPart string, maxFrac int, mode string) (string, string) {
	if maxFrac < 0 {
		maxFrac = 0
	}
	if len(fracPart) <= maxFrac {
		return intPart, fracPart
	}
	keep := fracPart[:maxFrac]
	rest := fracPart[maxFrac:]
	up := false
	switch mode {
	case "ceil", "floor":
		// Sign handling is done by the caller; here positive numbers only.
		if mode == "ceil" {
			up = intlHasNonZero(rest)
		}
	case "trunc":
		up = false
	case "expand":
		up = intlHasNonZero(rest)
	case "halfEven":
		first := rest[0]
		up = first > '5' || (first == '5' && (intlHasNonZero(rest[1:]) || intlLastDigitOdd(keep)))
	case "halfCeil":
		up = rest[0] >= '5'
	case "halfFloor":
		up = rest[0] > '5'
	case "halfTrunc":
		first := rest[0]
		up = first > '5'
	default: // halfExpand
		up = rest[0] >= '5'
	}
	if up {
		keep = intlIncrementDigits(keep)
		if len(keep) > maxFrac {
			// Carry into the integer part.
			carry := keep[:len(keep)-maxFrac]
			keep = keep[len(keep)-maxFrac:]
			intPart = intlAddDigits(intPart, carry)
		}
	}
	return intPart, keep
}

func intlHasNonZero(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] != '0' {
			return true
		}
	}
	return false
}

func intlLastDigitOdd(s string) bool {
	if len(s) == 0 {
		return false
	}
	d := s[len(s)-1]
	return d >= '1' && d <= '9' && (d-'0')%2 == 1
}

// intlIncrementDigits adds one to a decimal digit string.
func intlIncrementDigits(s string) string {
	b := []byte(s)
	i := len(b) - 1
	for i >= 0 {
		if b[i] == '9' {
			b[i] = '0'
			i--
			continue
		}
		b[i]++
		return string(b)
	}
	return "1" + string(b)
}

func intlAddDigits(a, b string) string {
	// Add two non-negative decimal integer strings.
	ia, _ := strconv.ParseUint(a, 10, 64)
	ib, _ := strconv.ParseUint(b, 10, 64)
	return strconv.FormatUint(ia+ib, 10)
}

func intlGroupDigits(s string, group string, primary, secondary int) string {
	if group == "" || primary <= 0 {
		return s
	}
	n := len(s)
	var parts []string
	first := n % primary
	if first == 0 {
		first = primary
	}
	if first > 0 && first <= n {
		parts = append(parts, s[:first])
	}
	rest := s[first:]
	if secondary <= 0 {
		secondary = primary
	}
	for len(rest) > 0 {
		if len(rest) <= secondary {
			parts = append(parts, rest)
			break
		}
		parts = append(parts, rest[len(rest)-secondary:])
		rest = rest[:len(rest)-secondary]
	}
	// parts are in reverse order after the first
	res := []string{parts[0]}
	for i := len(parts) - 1; i >= 1; i-- {
		res = append(res, parts[i])
	}
	return strings.Join(res, group)
}

// intlSignSymbol returns the sign prefix and whether the number is negative.
func intlNumberIsNegative(n float64) bool {
	return n < 0 || (n == 0 && math.Signbit(n))
}

// intlStr converts a Go string into a JS string value.
func intlStr(s string) Value {
	if isASCII(s) {
		return asciiString(s)
	}
	return newStringValue(s)
}

// intlPartsValue converts parts into an array of {type, value} objects.
func (r *Runtime) intlPartsArray(parts []intlStringPart) Value {
	vals := make([]Value, len(parts))
	for i, p := range parts {
		obj := r.newBaseObject(r.global.ObjectPrototype, classObject)
		obj.setOwnStr("type", asciiString(p.typ), false)
		obj.setOwnStr("value", intlStr(p.value), false)
		vals[i] = obj.val
	}
	return r.newArrayValues(vals)
}
