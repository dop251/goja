package goja

import (
	"strings"
	"unicode"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"
	"golang.org/x/text/unicode/norm"
)

type intlCollatorObject struct {
	baseObject
	locale            string
	tag               *intlTag
	usage             string
	sensitivity       string
	ignorePunctuation bool
	collation         string
	numeric           bool
	caseFirst         string
	collator          *collate.Collator
	bound             *Object
}

func (r *Runtime) getIntlCollator() *Object {
	ret := r.global.IntlCollator
	if ret == nil {
		ret = &Object{runtime: r}
		r.global.IntlCollator = ret
		proto := r.getIntlCollatorPrototype()
		r.newNativeFuncAndConstruct(ret, func(call FunctionCall) Value {
			return r.builtin_newIntlCollator(call.Arguments, r.getIntlCollatorPrototype())
		}, r.wrapNativeConstruct(r.builtin_newIntlCollator, ret, proto), proto, "Collator", intToValue(0))
		ret.self._putProp("supportedLocalesOf", r.newNativeFunc(r.intlCollatorSupportedLocalesOf, "supportedLocalesOf", 1), true, false, true)
	}
	return ret
}

func (r *Runtime) getIntlCollatorPrototype() *Object {
	ret := r.global.IntlCollatorPrototype
	if ret == nil {
		ret = r.newBaseObject(r.global.ObjectPrototype, classObject).val
		r.global.IntlCollatorPrototype = ret
		o := ret.self
		o._putProp("constructor", r.getIntlCollator(), true, false, true)
		o.setOwnStr("compare", &valueProperty{
			getterFunc:   r.newNativeFunc(r.intlCollatorGetCompare, "get compare", 0),
			accessor:     true,
			configurable: true,
		}, false)
		o._putProp("resolvedOptions", r.newNativeFunc(r.intlCollatorProtoResolvedOptions, "resolvedOptions", 0), true, false, true)
		o.defineOwnPropertySym(SymToStringTag, PropertyDescriptor{
			Value:        intlStr("Intl.Collator"),
			Writable:     FLAG_FALSE,
			Enumerable:   FLAG_FALSE,
			Configurable: FLAG_TRUE,
		}, false)
	}
	return ret
}

func (r *Runtime) intlCollatorThis(call FunctionCall) *intlCollatorObject {
	o, ok := r.toObject(call.This).self.(*intlCollatorObject)
	if !ok {
		panic(r.NewTypeError("Intl.Collator method called on incompatible receiver"))
	}
	return o
}

func (r *Runtime) builtin_newIntlCollator(args []Value, proto *Object) *Object {
	var localesArg, optionsArg Value
	if len(args) > 0 {
		localesArg = args[0]
	}
	if len(args) > 1 {
		optionsArg = args[1]
	}
	requested := r.intlCanonicalizeLocaleList(localesArg)
	options := r.intlGetOptionsObject(optionsArg)
	usage := "sort"
	sensitivity := ""
	ignorePunctuation := false
	ignorePunctuationSet := false
	collation := ""
	numeric := false
	caseFirst := "false"
	numericSet := false
	caseFirstSet := false
	collationSet := false
	if options != nil {
		r.intlGetOption(options, "localeMatcher", "string", []string{"best fit", "lookup"}, _undefined)
		if v := r.intlGetOption(options, "usage", "string", []string{"sort", "search"}, _undefined); !IsUndefined(v) {
			usage = v.String()
		}
		if v := r.intlGetOption(options, "collation", "string", nil, _undefined); !IsUndefined(v) {
			c := strings.ToLower(v.String())
			if !intlIsUnicodeExtensionValue(c) {
				panic(r.intlRangeError("Invalid collation"))
			}
			collation = c
			collationSet = true
		}
		if v := r.intlGetOption(options, "numeric", "boolean", nil, _undefined); !IsUndefined(v) {
			numeric = v.ToBoolean()
			numericSet = true
		}
		if v := r.intlGetOption(options, "caseFirst", "string", []string{"upper", "lower", "false"}, _undefined); !IsUndefined(v) {
			caseFirst = v.String()
			caseFirstSet = true
		}
		if v := r.intlGetOption(options, "sensitivity", "string", []string{"base", "accent", "case", "variant"}, _undefined); !IsUndefined(v) {
			sensitivity = v.String()
		}
		if v := r.intlGetOption(options, "ignorePunctuation", "boolean", nil, _undefined); !IsUndefined(v) {
			ignorePunctuation = v.ToBoolean()
			ignorePunctuationSet = true
		}
	}
	if sensitivity == "" {
		sensitivity = "variant"
	}

	baseLocale, tag := r.intlResolveServiceLocale(requested, nil, options)
	if !ignorePunctuationSet && tag.language == "th" {
		ignorePunctuation = true
	}
	if !collationSet {
		if c, ok := tag.uKw["co"]; ok {
			collation = c
		}
	}
	if collation == "" || collation == "standard" || collation == "search" || !intlContains(intlCollationValues, collation) {
		collation = "default"
	}
	if !numericSet {
		if v, ok := tag.uKw["kn"]; ok {
			if v == "" || v == "true" {
				numeric = true
			} else if v == "false" {
				numeric = false
			}
		}
	}
	if !caseFirstSet {
		if v, ok := tag.uKw["kf"]; ok && v != "" {
			caseFirst = v
		}
	}

	// Only extension keywords that were used (and match the resolved value)
	// are reflected in the resolved locale.
	var ext []string
	if collation != "default" {
		if v, ok := tag.uKw["co"]; ok && v == collation {
			ext = append(ext, "co-"+collation)
		}
	}
	if v, ok := tag.uKw["kn"]; ok {
		extVal := v == "" || v == "true"
		if extVal == numeric {
			if numeric {
				ext = append(ext, "kn")
			} else {
				ext = append(ext, "kn-false")
			}
		}
	}
	if v, ok := tag.uKw["kf"]; ok && v == caseFirst {
		ext = append(ext, "kf-"+caseFirst)
	}
	locale := baseLocale
	if len(ext) > 0 {
		locale = baseLocale + "-u-" + strings.Join(ext, "-")
	}

	o := &Object{runtime: r}
	obj := &intlCollatorObject{
		locale:            locale,
		tag:               tag,
		usage:             usage,
		sensitivity:       sensitivity,
		ignorePunctuation: ignorePunctuation,
		collation:         collation,
		numeric:           numeric,
		caseFirst:         caseFirst,
	}
	obj.class = classObject
	obj.val = o
	obj.extensible = true
	obj.prototype = proto
	obj.init()
	o.self = obj
	return o
}

func (c *intlCollatorObject) getCollator() *collate.Collator {
	if c.collator != nil {
		return c.collator
	}
	lang := "und"
	if c.tag != nil && c.tag.language != "" {
		lang = c.tag.language
	}
	var opts []collate.Option
	switch c.sensitivity {
	case "base":
		opts = append(opts, collate.IgnoreCase, collate.IgnoreDiacritics)
	case "accent":
		opts = append(opts, collate.IgnoreCase)
	case "case":
		opts = append(opts, collate.IgnoreDiacritics)
	}
	if c.numeric {
		opts = append(opts, collate.Numeric)
	}
	c.collator = collate.New(language.Make(lang), opts...)
	return c.collator
}

func (c *intlCollatorObject) compare(a, b string) int {
	a = norm.NFD.String(a)
	b = norm.NFD.String(b)
	if c.ignorePunctuation {
		a = intlStripPunctuation(a)
		b = intlStripPunctuation(b)
	}
	res := c.getCollator().CompareString(a, b)
	if res < 0 {
		return -1
	}
	if res > 0 {
		return 1
	}
	return 0
}

func intlStripPunctuation(s string) string {
	var sb strings.Builder
	for _, r := range s {
		if unicode.IsPunct(r) || unicode.IsSymbol(r) || unicode.IsSpace(r) {
			continue
		}
		sb.WriteRune(r)
	}
	return sb.String()
}

func (r *Runtime) intlCollatorGetCompare(call FunctionCall) Value {
	c := r.intlCollatorThis(call)
	if c.bound == nil {
		c.bound = r.newNativeFunc(func(call FunctionCall) Value {
			a := call.Argument(0).toString().String()
			b := call.Argument(1).toString().String()
			return intToValue(int64(c.compare(a, b)))
		}, "", 2)
	}
	return c.bound
}

func (r *Runtime) intlCollatorProtoResolvedOptions(call FunctionCall) Value {
	c := r.intlCollatorThis(call)
	o := r.newBaseObject(r.global.ObjectPrototype, classObject)
	o.setOwnStr("locale", intlStr(c.locale), false)
	o.setOwnStr("usage", intlStr(c.usage), false)
	o.setOwnStr("sensitivity", intlStr(c.sensitivity), false)
	o.setOwnStr("ignorePunctuation", r.ToValue(c.ignorePunctuation), false)
	o.setOwnStr("collation", intlStr(c.collation), false)
	o.setOwnStr("numeric", r.ToValue(c.numeric), false)
	o.setOwnStr("caseFirst", intlStr(c.caseFirst), false)
	return o.val
}

func (r *Runtime) intlCollatorSupportedLocalesOf(call FunctionCall) Value {
	return r.intlSupportedLocalesOf(call, intlHasLocaleData)
}

// intlDefaultCompareString compares two strings using the default collation.
func intlDefaultCompareString(r *Runtime, a, b string) int {
	c := &intlCollatorObject{sensitivity: "variant", caseFirst: "false", collation: "default"}
	return c.compare(a, b)
}
