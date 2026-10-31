package goja

import (
	"strings"
	"unicode"
)

type intlSegmenterObject struct {
	baseObject
	locale      string
	granularity string
}

type intlSegment struct {
	segment    string
	index      int // UTF-16 code unit offset
	isWordLike bool
}

type intlSegmentsObject struct {
	baseObject
	input       string
	granularity string
	segments    []intlSegment
}

type intlSegmentIteratorObject struct {
	baseObject
	segments *intlSegmentsObject
	pos      int
}

func (r *Runtime) getIntlSegmenter() *Object {
	ret := r.global.IntlSegmenter
	if ret == nil {
		ret = &Object{runtime: r}
		r.global.IntlSegmenter = ret
		proto := r.getIntlSegmenterPrototype()
		r.newNativeFuncAndConstruct(ret, func(call FunctionCall) Value {
			panic(r.NewTypeError("Constructor Intl.Segmenter requires 'new'"))
		}, r.wrapNativeConstruct(r.builtin_newIntlSegmenter, ret, proto), proto, "Segmenter", intToValue(0))
		ret.self._putProp("supportedLocalesOf", r.newNativeFunc(r.intlSegmenterSupportedLocalesOf, "supportedLocalesOf", 1), true, false, true)
	}
	return ret
}

func (r *Runtime) getIntlSegmenterPrototype() *Object {
	ret := r.global.IntlSegmenterPrototype
	if ret == nil {
		ret = r.newBaseObject(r.global.ObjectPrototype, classObject).val
		r.global.IntlSegmenterPrototype = ret
		o := ret.self
		o._putProp("constructor", r.getIntlSegmenter(), true, false, true)
		o._putProp("segment", r.newNativeFunc(r.intlSegmenterProtoSegment, "segment", 1), true, false, true)
		o._putProp("resolvedOptions", r.newNativeFunc(r.intlSegmenterProtoResolvedOptions, "resolvedOptions", 0), true, false, true)
		o.defineOwnPropertySym(SymToStringTag, PropertyDescriptor{
			Value:        intlStr("Intl.Segmenter"),
			Writable:     FLAG_FALSE,
			Enumerable:   FLAG_FALSE,
			Configurable: FLAG_TRUE,
		}, false)
	}
	return ret
}

func (r *Runtime) intlSegmenterThis(call FunctionCall) *intlSegmenterObject {
	o, ok := r.toObject(call.This).self.(*intlSegmenterObject)
	if !ok {
		panic(r.NewTypeError("Intl.Segmenter method called on incompatible receiver"))
	}
	return o
}

func (r *Runtime) builtin_newIntlSegmenter(args []Value, proto *Object) *Object {
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
	granularity := "grapheme"
	if options != nil {
		r.intlGetOption(options, "localeMatcher", "string", []string{"best fit", "lookup"}, _undefined)
		if v := r.intlGetOption(options, "granularity", "string", []string{"grapheme", "word", "sentence"}, _undefined); !IsUndefined(v) {
			granularity = v.String()
		}
	}
	locale, _ := r.intlResolveServiceLocale(requested, nil, options)
	o := &Object{runtime: r}
	obj := &intlSegmenterObject{locale: locale, granularity: granularity}
	obj.class = classObject
	obj.val = o
	obj.extensible = true
	obj.prototype = proto
	obj.init()
	o.self = obj
	return o
}

func (r *Runtime) intlSegmenterProtoSegment(call FunctionCall) Value {
	sg := r.intlSegmenterThis(call)
	input := call.Argument(0).toString().String()
	o := &Object{runtime: r}
	so := &intlSegmentsObject{input: input, granularity: sg.granularity}
	switch sg.granularity {
	case "word":
		so.segments = intlSplitWords(input)
	case "sentence":
		so.segments = intlSplitSentences(input)
	default:
		so.segments = intlSplitGraphemes(input)
	}
	so.class = classObject
	so.val = o
	so.extensible = true
	so.prototype = r.getIntlSegmentsPrototype()
	so.init()
	o.self = so
	return o
}

func (r *Runtime) intlSegmenterProtoResolvedOptions(call FunctionCall) Value {
	sg := r.intlSegmenterThis(call)
	o := r.newBaseObject(r.global.ObjectPrototype, classObject)
	o.setOwnStr("locale", intlStr(sg.locale), false)
	o.setOwnStr("granularity", intlStr(sg.granularity), false)
	return o.val
}

func (r *Runtime) intlSegmenterSupportedLocalesOf(call FunctionCall) Value {
	optsArg := call.Argument(1)
	if IsNull(optsArg) {
		panic(r.NewTypeError("Options must be an object"))
	}
	if opts := r.intlGetOptionsObject(optsArg); opts != nil {
		r.intlGetOption(opts, "localeMatcher", "string", []string{"best fit", "lookup"}, _undefined)
	}
	return r.intlSupportedLocalesOf(call, intlHasLocaleData)
}

// ---------- segments object ----------

func (r *Runtime) getIntlSegmentsPrototype() *Object {
	ret := r.global.IntlSegmentsPrototype
	if ret == nil {
		ret = r.newBaseObject(r.global.ObjectPrototype, classObject).val
		r.global.IntlSegmentsPrototype = ret
		o := ret.self
		o._putProp("containing", r.newNativeFunc(r.intlSegmentsContaining, "containing", 1), true, false, true)
		o.setOwnSym(SymIterator, r.newNativeFunc(r.intlSegmentsIterator, "[Symbol.iterator]", 0), false)
		o.defineOwnPropertySym(SymToStringTag, PropertyDescriptor{
			Value:        intlStr("Segments"),
			Writable:     FLAG_FALSE,
			Enumerable:   FLAG_FALSE,
			Configurable: FLAG_TRUE,
		}, false)
	}
	return ret
}

func (r *Runtime) intlSegmentsThis(call FunctionCall) *intlSegmentsObject {
	o, ok := r.toObject(call.This).self.(*intlSegmentsObject)
	if !ok {
		panic(r.NewTypeError("Intl.Segments method called on incompatible receiver"))
	}
	return o
}

func (r *Runtime) intlSegmentDataValue(so *intlSegmentsObject, seg intlSegment) Value {
	o := r.newBaseObject(r.global.ObjectPrototype, classObject)
	o.setOwnStr("segment", intlStr(seg.segment), false)
	o.setOwnStr("index", intToValue(int64(seg.index)), false)
	o.setOwnStr("input", intlStr(so.input), false)
	if so.granularity == "word" {
		o.setOwnStr("isWordLike", r.ToValue(seg.isWordLike), false)
	}
	return o.val
}

func (r *Runtime) intlSegmentsContaining(call FunctionCall) Value {
	so := r.intlSegmentsThis(call)
	idx := call.Argument(0).ToInteger()
	if idx < 0 || idx >= int64(intlUTF16Len(so.input)) {
		return _undefined
	}
	for _, seg := range so.segments {
		if int64(seg.index) <= idx && idx < int64(seg.index+intlUTF16Len(seg.segment)) {
			return r.intlSegmentDataValue(so, seg)
		}
	}
	return _undefined
}

func (r *Runtime) intlSegmentsIterator(call FunctionCall) Value {
	so := r.intlSegmentsThis(call)
	o := &Object{runtime: r}
	it := &intlSegmentIteratorObject{segments: so}
	it.class = classObject
	it.val = o
	it.extensible = true
	it.prototype = r.getIntlSegmentIteratorPrototype()
	it.init()
	o.self = it
	return o
}

func (r *Runtime) getIntlSegmentIteratorPrototype() *Object {
	ret := r.global.IntlSegmentIteratorPrototype
	if ret == nil {
		base := r.getIteratorPrototype()
		ret = r.newBaseObject(base, classObject).val
		r.global.IntlSegmentIteratorPrototype = ret
		o := ret.self
		o._putProp("next", r.newNativeFunc(r.intlSegmentIteratorNext, "next", 0), true, false, true)
		o.defineOwnPropertySym(SymToStringTag, PropertyDescriptor{
			Value:        intlStr("Segments Iterator"),
			Writable:     FLAG_FALSE,
			Enumerable:   FLAG_FALSE,
			Configurable: FLAG_TRUE,
		}, false)
	}
	return ret
}

func (r *Runtime) intlSegmentIteratorNext(call FunctionCall) Value {
	it, ok := r.toObject(call.This).self.(*intlSegmentIteratorObject)
	if !ok {
		panic(r.NewTypeError("Method Intl.Segmenter iterator next called on incompatible receiver"))
	}
	so := it.segments
	if it.pos >= len(so.segments) {
		return r.createIterResultObject(_undefined, true)
	}
	seg := so.segments[it.pos]
	it.pos++
	return r.createIterResultObject(r.intlSegmentDataValue(so, seg), false)
}

// ---------- segmentation ----------

func intlUTF16Len(s string) int {
	n := 0
	for _, r := range s {
		if r > 0xFFFF {
			n += 2
		} else {
			n++
		}
	}
	return n
}

func intlSplitGraphemes(s string) []intlSegment {
	var res []intlSegment
	var cur strings.Builder
	start := 0
	utf16pos := 0
	curStart := 0
	prevRI := false
	zwj := false
	flush := func() {
		if cur.Len() > 0 {
			res = append(res, intlSegment{segment: cur.String(), index: curStart})
			cur.Reset()
		}
	}
	for _, ru := range s {
		w := 1
		if ru > 0xFFFF {
			w = 2
		}
		join := false
		if cur.Len() == 0 {
			join = false
		} else if zwj || ru == 0x200D {
			join = true
		} else if intlIsGraphemeExtend(ru) {
			join = true
		} else if ru >= 0x1F3FB && ru <= 0x1F3FF {
			join = true
		} else if ru == 0x200C {
			join = true
		} else if intlIsRegionalIndicator(ru) && prevRI {
			join = true
		} else if ru == '\n' && strings.HasSuffix(cur.String(), "\r") {
			join = true
		}
		if !join {
			flush()
			curStart = utf16pos
		}
		cur.WriteRune(ru)
		if ru == 0x200D {
			zwj = true
		} else {
			zwj = false
		}
		prevRI = intlIsRegionalIndicator(ru)
		utf16pos += w
		start = utf16pos
	}
	flush()
	_ = start
	return res
}

func intlIsGraphemeExtend(r rune) bool {
	return unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Mc, r) || unicode.Is(unicode.Me, r) ||
		r == 0xFE0F || r == 0xFE0E
}

func intlIsRegionalIndicator(r rune) bool { return r >= 0x1F1E6 && r <= 0x1F1FF }

func intlIsWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.Is(unicode.Mn, r) || r == '_' || r == '\''
}

func intlSplitWords(s string) []intlSegment {
	var res []intlSegment
	runes := []rune(s)
	i := 0
	utf16pos := 0
	for i < len(runes) {
		start := i
		startU16 := utf16pos
		wordLike := intlIsWordRune(runes[i])
		if wordLike {
			for i < len(runes) && intlIsWordRune(runes[i]) {
				if runes[i] > 0xFFFF {
					utf16pos += 2
				} else {
					utf16pos++
				}
				i++
			}
		} else {
			// Group whitespace runs and punctuation runs separately.
			ws := unicode.IsSpace(runes[i])
			for i < len(runes) && !intlIsWordRune(runes[i]) && unicode.IsSpace(runes[i]) == ws {
				if runes[i] > 0xFFFF {
					utf16pos += 2
				} else {
					utf16pos++
				}
				i++
			}
		}
		res = append(res, intlSegment{segment: string(runes[start:i]), index: startU16, isWordLike: wordLike})
	}
	return res
}

func intlSplitSentences(s string) []intlSegment {
	var res []intlSegment
	runes := []rune(s)
	i := 0
	utf16pos := 0
	start := 0
	startU16 := 0
	for i < len(runes) {
		w := 1
		if runes[i] > 0xFFFF {
			w = 2
		}
		if runes[i] == '.' || runes[i] == '!' || runes[i] == '?' || runes[i] == 0x2026 {
			i++
			utf16pos += w
			// Include following terminator and whitespace.
			for i < len(runes) && (runes[i] == '.' || runes[i] == '!' || runes[i] == '?') {
				if runes[i] > 0xFFFF {
					utf16pos += 2
				} else {
					utf16pos++
				}
				i++
			}
			for i < len(runes) && unicode.IsSpace(runes[i]) {
				if runes[i] > 0xFFFF {
					utf16pos += 2
				} else {
					utf16pos++
				}
				i++
			}
			res = append(res, intlSegment{segment: string(runes[start:i]), index: startU16})
			start = i
			startU16 = utf16pos
			continue
		}
		utf16pos += w
		i++
	}
	if start < len(runes) {
		res = append(res, intlSegment{segment: string(runes[start:]), index: startU16})
	}
	return res
}
