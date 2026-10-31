package goja

import (
	"strings"

	"github.com/dop251/goja/unistring"
)

// ---------- Intl namespace ----------

func (r *Runtime) getIntl() *Object {
	ret := r.global.Intl
	if ret == nil {
		ret = &Object{runtime: r}
		r.global.Intl = ret
		o := r.newBaseObject(r.global.ObjectPrototype, classObject)
		ret.self = o
		o._putProp("Locale", r.getIntlLocale(), true, false, true)
		o._putProp("PluralRules", r.getIntlPluralRules(), true, false, true)
		o._putProp("ListFormat", r.getIntlListFormat(), true, false, true)
		o._putProp("RelativeTimeFormat", r.getIntlRelativeTimeFormat(), true, false, true)
		o._putProp("NumberFormat", r.getIntlNumberFormat(), true, false, true)
		o._putProp("Collator", r.getIntlCollator(), true, false, true)
		o._putProp("getCanonicalLocales", r.newNativeFunc(r.intlGetCanonicalLocales, "getCanonicalLocales", 1), true, false, true)
		o._putProp("supportedValuesOf", r.newNativeFunc(r.intlSupportedValuesOf, "supportedValuesOf", 1), true, false, true)
		o.defineOwnPropertySym(SymToStringTag, PropertyDescriptor{
			Value:        asciiString("Intl"),
			Writable:     FLAG_FALSE,
			Enumerable:   FLAG_FALSE,
			Configurable: FLAG_TRUE,
		}, false)
	}
	return ret
}

// intlCanonicalizeLocaleList implements CanonicalizeLocaleList.
func (r *Runtime) intlCanonicalizeLocaleList(locales Value) []string {
	if locales == nil || IsUndefined(locales) {
		return nil
	}
	if s, ok := locales.(String); ok {
		tag, ok := intlCanonicalizeTagString(s.String())
		if !ok {
			panic(r.intlRangeError("Incorrect Language Tag: " + s.String()))
		}
		return []string{tag}
	}
	var o *Object
	if obj, ok := locales.(*Object); ok {
		if lo, ok := obj.self.(*intlLocaleObject); ok {
			return []string{lo.tag.String()}
		}
		o = obj
	} else {
		o = locales.ToObject(r)
	}
	length := int64(toLength(o.self.getStr("length", nil)))
	seen := make(map[string]bool, length)
	var result []string
	for k := int64(0); k < length; k++ {
		key := intToValue(k)
		if o.self.hasPropertyStr(key.string()) {
			v := nilSafe(o.self.getStr(key.string(), nil))
			switch sv := v.(type) {
			case String:
			case *Object:
				if l, ok := sv.self.(*intlLocaleObject); ok {
					s := l.tag.String()
					if !seen[s] {
						seen[s] = true
						result = append(result, s)
					}
					continue
				}
			default:
				panic(r.NewTypeError("Language ID should be string or object."))
			}
			tag := v.toString().String()
			canonical, ok := intlCanonicalizeTagString(tag)
			if !ok {
				panic(r.intlRangeError("Incorrect Language Tag: " + tag))
			}
			if !seen[canonical] {
				seen[canonical] = true
				result = append(result, canonical)
			}
		}
	}
	return result
}

func (r *Runtime) intlGetCanonicalLocales(call FunctionCall) Value {
	list := r.intlCanonicalizeLocaleList(call.Argument(0))
	vals := make([]Value, len(list))
	for i, s := range list {
		vals[i] = asciiString(s)
	}
	return r.newArrayValues(vals)
}

func (r *Runtime) intlSupportedValuesOf(call FunctionCall) Value {
	key := call.Argument(0).toString().String()
	var values []string
	switch key {
	case "calendar":
		values = intlSupportedCalendars()
	case "collation":
		values = intlCollationValues
	case "currency":
		values = intlCurrencyValuesUpper
	case "numberingSystem":
		values = intlNumberingSystemValues
	case "timeZone":
		values = append(append([]string(nil), intlAllTimeZones...), "UTC")
	case "unit":
		values = intlUnitValues
	default:
		panic(r.intlRangeError("Invalid key: " + key))
	}
	vals := make([]Value, len(values))
	for i, s := range values {
		vals[i] = asciiString(s)
	}
	return r.newArrayValues(vals)
}

// intlSupportedCalendars returns the calendars required by the era/monthCode
// proposal.
func intlSupportedCalendars() []string {
	return []string{
		"buddhist", "chinese", "coptic", "dangi", "ethioaa", "ethiopic",
		"gregory", "hebrew", "indian", "islamic-civil", "islamic-tbla",
		"islamic-umalqura", "iso8601", "japanese", "persian", "roc",
	}
}

// ---------- helpers ----------

func (r *Runtime) intlGetOption(options *Object, property string, typ string, values []string, fallback Value) Value {
	v := nilSafe(options.self.getStr(unistring.NewFromString(property), nil))
	if IsUndefined(v) {
		return fallback
	}
	switch typ {
	case "boolean":
		return r.ToValue(v.ToBoolean())
	case "string":
		s := v.toString()
		if len(values) > 0 {
			str := s.String()
			for _, allowed := range values {
				if str == allowed {
					return s
				}
			}
			panic(r.intlRangeError("Value " + str + " out of range for option " + property))
		}
		return s
	}
	return v
}

func (r *Runtime) intlDefaultLocale() string {
	// goja has no host locale; the default is "en".
	return "en"
}

// intlResolveLocale resolves the requested locales against the supported set
// (which, for now, is every canonicalized tag).
func (r *Runtime) intlResolveLocale(requested []string) *intlTag {
	if len(requested) == 0 {
		t, _ := intlParseTag(r.intlDefaultLocale())
		return intlCanonicalizeTag(t)
	}
	t, ok := intlParseTag(requested[0])
	if !ok {
		t, _ = intlParseTag(r.intlDefaultLocale())
	}
	return intlCanonicalizeTag(t)
}

func (r *Runtime) intlRangeError(msg string) Value {
	return r.newErrorf(r.getRangeError(), "%s", msg)
}

func intlContains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func intlIsUnicodeExtensionValue(s string) bool {
	if s == "" {
		return false
	}
	for _, part := range strings.Split(s, "-") {
		if len(part) < 3 || len(part) > 8 {
			return false
		}
		if !intlIsAlnum(part) {
			return false
		}
	}
	return true
}

// intlDefaultNumberingSystems maps a language to its default numbering system.
var intlDefaultNumberingSystems = map[string]string{
	"ar": "arab", "fa": "arabext", "hi": "deva", "mr": "deva", "ne": "deva",
	"bn": "beng", "as": "beng", "th": "thai", "my": "mymr", "km": "khmr",
	"lo": "laoo", "ta": "tamldec", "te": "telu", "kn": "knda", "ml": "mlym",
	"pa": "guru", "gu": "gujr", "or": "orya", "he": "hebr", "am": "ethi",
	"ur": "arabext", "ps": "arabext", "sd": "arab", "ug": "arab",
}
