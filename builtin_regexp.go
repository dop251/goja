package goja

import (
	"fmt"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/auvred/regonaut"
)

func (r *Runtime) newRegexpObject(proto *Object) *regexpObject {
	v := &Object{runtime: r}

	o := &regexpObject{}
	o.class = classRegExp
	o.val = v
	o.extensible = true
	v.self = o
	o.prototype = proto
	o.init()
	return o
}

func (r *Runtime) newRegExpp(pattern *regexpPattern, patternStr String, proto *Object) *regexpObject {
	o := r.newRegexpObject(proto)

	o.pattern = pattern
	o.source = patternStr

	return o
}

func writeHex4(b *strings.Builder, i int) {
	b.WriteByte(hex[i>>12])
	b.WriteByte(hex[(i>>8)&0xF])
	b.WriteByte(hex[(i>>4)&0xF])
	b.WriteByte(hex[i&0xF])
}

// convert any broken UTF-16 surrogate pairs to \uXXXX
func escapeInvalidUtf16(s String) string {
	if imported, ok := s.(*importedString); ok {
		return imported.s
	}
	if ascii, ok := s.(asciiString); ok {
		return ascii.String()
	}
	var sb strings.Builder
	rd := &lenientUtf16Decoder{utf16Reader: s.utf16Reader()}
	pos := 0
	utf8Size := 0
	var utf8Buf [utf8.UTFMax]byte
	for {
		c, size, err := rd.ReadRune()
		if err != nil {
			break
		}
		if utf16.IsSurrogate(c) {
			if sb.Len() == 0 {
				sb.Grow(utf8Size + 7)
				hrd := s.Reader()
				var c rune
				for p := 0; p < pos; {
					var size int
					var err error
					c, size, err = hrd.ReadRune()
					if err != nil {
						// will not happen
						panic(fmt.Errorf("error while reading string head %q, pos: %d: %w", s.String(), pos, err))
					}
					sb.WriteRune(c)
					p += size
				}
				if c == '\\' {
					sb.WriteRune(c)
				}
			}
			sb.WriteString(`\u`)
			writeHex4(&sb, int(c))
		} else {
			if sb.Len() > 0 {
				sb.WriteRune(c)
			} else {
				utf8Size += utf8.EncodeRune(utf8Buf[:], c)
				pos += size
			}
		}
	}
	if sb.Len() > 0 {
		return sb.String()
	}
	return s.String()
}

func compileRegexp(patternStr String, flags string) (p *regexpPattern, err error) {
	patternUtf16 := patternStr.toUnicode()

	var global, ignoreCase, multiline, dotAll, sticky, unicode bool

	reFlags := regonaut.FlagAnnexB

	if flags != "" {
		invalidFlags := func() {
			err = fmt.Errorf("Invalid flags supplied to RegExp constructor '%s'", flags)
		}
		for _, chr := range flags {
			switch chr {
			case 'g':
				if global {
					invalidFlags()
					return
				}
				global = true
			case 'm':
				if multiline {
					invalidFlags()
					return
				}
				reFlags |= regonaut.FlagMultiline
				multiline = true
			case 's':
				if dotAll {
					invalidFlags()
					return
				}
				reFlags |= regonaut.FlagDotAll
				dotAll = true
			case 'i':
				if ignoreCase {
					invalidFlags()
					return
				}
				reFlags |= regonaut.FlagIgnoreCase
				ignoreCase = true
			case 'y':
				if sticky {
					invalidFlags()
					return
				}
				reFlags |= regonaut.FlagSticky
				sticky = true
			case 'u':
				if unicode {
					invalidFlags()
				}
				reFlags |= regonaut.FlagUnicode
				unicode = true
			default:
				invalidFlags()
				return
			}
		}
	}

	var re *regonaut.RegExpUtf16
	re, err = regonaut.CompileUtf16(patternUtf16[1:], reFlags)
	if err != nil {
		return
	}

	p = &regexpPattern{
		src:        patternUtf16,
		re:         re,
		global:     global,
		ignoreCase: ignoreCase,
		multiline:  multiline,
		dotAll:     dotAll,
		sticky:     sticky,
		unicode:    unicode,
	}
	return
}

func (r *Runtime) _newRegExp(patternStr String, flags string, proto *Object) *regexpObject {
	pattern, err := compileRegexp(patternStr, flags)
	if err != nil {
		panic(r.newSyntaxError(err.Error(), -1))
	}
	return r.newRegExpp(pattern, patternStr, proto)
}

func (r *Runtime) builtin_newRegExp(args []Value, proto *Object) *Object {
	var patternVal, flagsVal Value
	if len(args) > 0 {
		patternVal = args[0]
	}
	if len(args) > 1 {
		flagsVal = args[1]
	}
	return r.newRegExp(patternVal, flagsVal, proto).val
}

func (r *Runtime) newRegExp(patternVal, flagsVal Value, proto *Object) *regexpObject {
	var pattern String
	var flags string
	if isRegexp(patternVal) { // this may have side effects so need to call it anyway
		if obj, ok := patternVal.(*Object); ok {
			if rx, ok := obj.self.(*regexpObject); ok {
				if flagsVal == nil || flagsVal == _undefined {
					return rx.clone()
				} else {
					return r._newRegExp(rx.source, flagsVal.toString().String(), proto)
				}
			} else {
				pattern = nilSafe(obj.self.getStr("source", nil)).toString()
				if flagsVal == nil || flagsVal == _undefined {
					flags = nilSafe(obj.self.getStr("flags", nil)).toString().String()
				} else {
					flags = flagsVal.toString().String()
				}
				goto exit
			}
		}
	}

	if patternVal != nil && patternVal != _undefined {
		pattern = patternVal.toString()
	}
	if flagsVal != nil && flagsVal != _undefined {
		flags = flagsVal.toString().String()
	}

	if pattern == nil {
		pattern = stringEmpty
	}
exit:
	return r._newRegExp(pattern, flags, proto)
}

func (r *Runtime) builtin_RegExp(call FunctionCall) Value {
	pattern := call.Argument(0)
	patternIsRegExp := isRegexp(pattern)
	flags := call.Argument(1)
	if patternIsRegExp && flags == _undefined {
		if obj, ok := call.Argument(0).(*Object); ok {
			patternConstructor := obj.self.getStr("constructor", nil)
			if patternConstructor == r.global.RegExp {
				return pattern
			}
		}
	}
	return r.newRegExp(pattern, flags, r.getRegExpPrototype()).val
}

func (r *Runtime) regexpproto_compile(call FunctionCall) Value {
	if this, ok := r.toObject(call.This).self.(*regexpObject); ok {
		var (
			pattern *regexpPattern
			source  String
			flags   string
			err     error
		)
		patternVal := call.Argument(0)
		flagsVal := call.Argument(1)
		if o, ok := patternVal.(*Object); ok {
			if p, ok := o.self.(*regexpObject); ok {
				if flagsVal != _undefined {
					panic(r.NewTypeError("Cannot supply flags when constructing one RegExp from another"))
				}
				this.pattern = p.pattern
				this.source = p.source
				goto exit
			}
		}
		if patternVal != _undefined {
			source = patternVal.toString()
		} else {
			source = stringEmpty
		}
		if flagsVal != _undefined {
			flags = flagsVal.toString().String()
		}
		pattern, err = compileRegexp(source, flags)
		if err != nil {
			panic(r.newSyntaxError(err.Error(), -1))
		}
		this.pattern = pattern
		this.source = source
	exit:
		this.setOwnStr("lastIndex", intToValue(0), true)
		return call.This
	}

	panic(r.NewTypeError("Method RegExp.prototype.compile called on incompatible receiver %s", r.objectproto_toString(FunctionCall{This: call.This})))
}

func (r *Runtime) regexpproto_exec(call FunctionCall) Value {
	if this, ok := r.toObject(call.This).self.(*regexpObject); ok {
		return this.exec(call.Argument(0).toString())
	} else {
		r.typeErrorResult(true, "Method RegExp.prototype.exec called on incompatible receiver %s", r.objectproto_toString(FunctionCall{This: call.This}))
		return nil
	}
}

func (r *Runtime) regexpproto_test(call FunctionCall) Value {
	if this, ok := r.toObject(call.This).self.(*regexpObject); ok {
		if this.test(call.Argument(0).toString()) {
			return valueTrue
		} else {
			return valueFalse
		}
	} else {
		panic(r.NewTypeError("Method RegExp.prototype.test called on incompatible receiver %s", r.objectproto_toString(FunctionCall{This: call.This})))
	}
}

func (r *Runtime) regexpproto_toString(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if this := r.checkStdRegexp(obj); this != nil {
		var sb StringBuilder
		sb.WriteRune('/')
		if !this.writeEscapedSource(&sb) {
			sb.WriteString(this.source)
		}
		sb.WriteRune('/')
		if this.pattern.global {
			sb.WriteRune('g')
		}
		if this.pattern.ignoreCase {
			sb.WriteRune('i')
		}
		if this.pattern.multiline {
			sb.WriteRune('m')
		}
		if this.pattern.dotAll {
			sb.WriteRune('s')
		}
		if this.pattern.unicode {
			sb.WriteRune('u')
		}
		if this.pattern.sticky {
			sb.WriteRune('y')
		}
		return sb.String()
	}
	pattern := nilSafe(obj.self.getStr("source", nil)).toString()
	flags := nilSafe(obj.self.getStr("flags", nil)).toString()
	var sb StringBuilder
	sb.WriteRune('/')
	sb.WriteString(pattern)
	sb.WriteRune('/')
	sb.WriteString(flags)
	return sb.String()
}

func (r *regexpObject) writeEscapedSource(sb *StringBuilder) bool {
	if r.source.Length() == 0 {
		sb.WriteString(asciiString("(?:)"))
		return true
	}
	pos := 0
	lastPos := 0
	rd := &lenientUtf16Decoder{utf16Reader: r.source.utf16Reader()}
L:
	for {
		c, size, err := rd.ReadRune()
		if err != nil {
			break
		}
		switch c {
		case '\\':
			pos++
			_, size, err = rd.ReadRune()
			if err != nil {
				break L
			}
		case '/', '\u000a', '\u000d', '\u2028', '\u2029':
			sb.WriteSubstring(r.source, lastPos, pos)
			sb.WriteRune('\\')
			switch c {
			case '\u000a':
				sb.WriteRune('n')
			case '\u000d':
				sb.WriteRune('r')
			default:
				sb.WriteRune('u')
				sb.WriteRune(rune(hex[c>>12]))
				sb.WriteRune(rune(hex[(c>>8)&0xF]))
				sb.WriteRune(rune(hex[(c>>4)&0xF]))
				sb.WriteRune(rune(hex[c&0xF]))
			}
			lastPos = pos + size
		}
		pos += size
	}
	if lastPos > 0 {
		sb.WriteSubstring(r.source, lastPos, r.source.Length())
		return true
	}
	return false
}

func (r *Runtime) regexpproto_getSource(call FunctionCall) Value {
	if this, ok := r.toObject(call.This).self.(*regexpObject); ok {
		var sb StringBuilder
		if this.writeEscapedSource(&sb) {
			return sb.String()
		}
		return this.source
	} else if call.This == r.global.RegExpPrototype {
		return asciiString("(?:)")
	} else {
		panic(r.NewTypeError("Method RegExp.prototype.source getter called on incompatible receiver"))
	}
}

func (r *Runtime) regexpproto_getGlobal(call FunctionCall) Value {
	if this, ok := r.toObject(call.This).self.(*regexpObject); ok {
		if this.pattern.global {
			return valueTrue
		} else {
			return valueFalse
		}
	} else if call.This == r.global.RegExpPrototype {
		return _undefined
	} else {
		panic(r.NewTypeError("Method RegExp.prototype.global getter called on incompatible receiver %s", r.objectproto_toString(FunctionCall{This: call.This})))
	}
}

func (r *Runtime) regexpproto_getMultiline(call FunctionCall) Value {
	if this, ok := r.toObject(call.This).self.(*regexpObject); ok {
		if this.pattern.multiline {
			return valueTrue
		} else {
			return valueFalse
		}
	} else if call.This == r.global.RegExpPrototype {
		return _undefined
	} else {
		panic(r.NewTypeError("Method RegExp.prototype.multiline getter called on incompatible receiver %s", r.objectproto_toString(FunctionCall{This: call.This})))
	}
}

func (r *Runtime) regexpproto_getDotAll(call FunctionCall) Value {
	if this, ok := r.toObject(call.This).self.(*regexpObject); ok {
		if this.pattern.dotAll {
			return valueTrue
		} else {
			return valueFalse
		}
	} else if call.This == r.global.RegExpPrototype {
		return _undefined
	} else {
		panic(r.NewTypeError("Method RegExp.prototype.dotAll getter called on incompatible receiver %s", r.objectproto_toString(FunctionCall{This: call.This})))
	}
}

func (r *Runtime) regexpproto_getIgnoreCase(call FunctionCall) Value {
	if this, ok := r.toObject(call.This).self.(*regexpObject); ok {
		if this.pattern.ignoreCase {
			return valueTrue
		} else {
			return valueFalse
		}
	} else if call.This == r.global.RegExpPrototype {
		return _undefined
	} else {
		panic(r.NewTypeError("Method RegExp.prototype.ignoreCase getter called on incompatible receiver %s", r.objectproto_toString(FunctionCall{This: call.This})))
	}
}

func (r *Runtime) regexpproto_getUnicode(call FunctionCall) Value {
	if this, ok := r.toObject(call.This).self.(*regexpObject); ok {
		if this.pattern.unicode {
			return valueTrue
		} else {
			return valueFalse
		}
	} else if call.This == r.global.RegExpPrototype {
		return _undefined
	} else {
		panic(r.NewTypeError("Method RegExp.prototype.unicode getter called on incompatible receiver %s", r.objectproto_toString(FunctionCall{This: call.This})))
	}
}

func (r *Runtime) regexpproto_getSticky(call FunctionCall) Value {
	if this, ok := r.toObject(call.This).self.(*regexpObject); ok {
		if this.pattern.sticky {
			return valueTrue
		} else {
			return valueFalse
		}
	} else if call.This == r.global.RegExpPrototype {
		return _undefined
	} else {
		panic(r.NewTypeError("Method RegExp.prototype.sticky getter called on incompatible receiver %s", r.objectproto_toString(FunctionCall{This: call.This})))
	}
}

func (r *Runtime) regexpproto_getFlags(call FunctionCall) Value {
	var global, ignoreCase, multiline, dotAll, sticky, unicode bool

	thisObj := r.toObject(call.This)
	size := 0
	if v := thisObj.self.getStr("global", nil); v != nil {
		global = v.ToBoolean()
		if global {
			size++
		}
	}
	if v := thisObj.self.getStr("ignoreCase", nil); v != nil {
		ignoreCase = v.ToBoolean()
		if ignoreCase {
			size++
		}
	}
	if v := thisObj.self.getStr("multiline", nil); v != nil {
		multiline = v.ToBoolean()
		if multiline {
			size++
		}
	}
	if v := thisObj.self.getStr("dotAll", nil); v != nil {
		dotAll = v.ToBoolean()
		if dotAll {
			size++
		}
	}
	if v := thisObj.self.getStr("sticky", nil); v != nil {
		sticky = v.ToBoolean()
		if sticky {
			size++
		}
	}
	if v := thisObj.self.getStr("unicode", nil); v != nil {
		unicode = v.ToBoolean()
		if unicode {
			size++
		}
	}

	var sb strings.Builder
	sb.Grow(size)
	if global {
		sb.WriteByte('g')
	}
	if ignoreCase {
		sb.WriteByte('i')
	}
	if multiline {
		sb.WriteByte('m')
	}
	if dotAll {
		sb.WriteByte('s')
	}
	if unicode {
		sb.WriteByte('u')
	}
	if sticky {
		sb.WriteByte('y')
	}

	return asciiString(sb.String())
}

func (r *Runtime) regExpExec(execFn func(FunctionCall) Value, rxObj *Object, arg Value) Value {
	res := execFn(FunctionCall{
		This:      rxObj,
		Arguments: []Value{arg},
	})

	if res != _null {
		if _, ok := res.(*Object); !ok {
			panic(r.NewTypeError("RegExp exec method returned something other than an Object or null"))
		}
	}

	return res
}

func (r *Runtime) getGlobalRegexpMatches(rxObj *Object, s String, fullUnicode bool) []Value {
	rxObj.self.setOwnStr("lastIndex", intToValue(0), true)
	execFn, ok := r.toObject(rxObj.self.getStr("exec", nil)).self.assertCallable()
	if !ok {
		panic(r.NewTypeError("exec is not a function"))
	}
	var a []Value
	for {
		res := r.regExpExec(execFn, rxObj, s)
		if res == _null {
			break
		}
		a = append(a, res)
		matchStr := nilSafe(r.toObject(res).self.getIdx(valueInt(0), nil)).toString()
		if matchStr.Length() == 0 {
			thisIndex := toLength(rxObj.self.getStr("lastIndex", nil))
			rxObj.self.setOwnStr("lastIndex", valueInt(advanceStringIndex64(s, thisIndex, fullUnicode)), true)
		}
	}

	return a
}

func (r *Runtime) regexpproto_stdMatcherGeneric(rxObj *Object, s String) Value {
	rx := rxObj.self
	flags := nilSafe(rx.getStr("flags", nil)).String()
	global := strings.ContainsRune(flags, 'g')
	if global {
		a := r.getGlobalRegexpMatches(rxObj, s, strings.ContainsRune(flags, 'u'))
		if len(a) == 0 {
			return _null
		}
		ar := make([]Value, 0, len(a))
		for _, result := range a {
			obj := r.toObject(result)
			matchStr := nilSafe(obj.self.getIdx(valueInt(0), nil)).ToString()
			ar = append(ar, matchStr)
		}
		return r.newArrayValues(ar)
	}

	execFn, ok := r.toObject(rx.getStr("exec", nil)).self.assertCallable()
	if !ok {
		panic(r.NewTypeError("exec is not a function"))
	}

	return r.regExpExec(execFn, rxObj, s)
}

func (r *Runtime) checkStdRegexp(rxObj *Object) *regexpObject {
	if deoptimiseRegexp {
		return nil
	}

	rx, ok := rxObj.self.(*regexpObject)
	if !ok {
		return nil
	}

	if !rx.standard || rx.prototype == nil || rx.prototype.self != r.global.stdRegexpProto {
		return nil
	}

	return rx
}

func (r *Runtime) regexpproto_stdMatcher(call FunctionCall) Value {
	thisObj := r.toObject(call.This)
	s := call.Argument(0).toString()
	rx := r.checkStdRegexp(thisObj)
	if rx == nil {
		return r.regexpproto_stdMatcherGeneric(thisObj, s)
	}
	if rx.pattern.global {
		sUtf16 := s.toUnicode()
		var a []Value
		rx.setOwnStr("lastIndex", valueInt(0), true)
		for {
			match := rx.execRegexp(rx.pattern, sUtf16, false)
			if match == nil {
				break
			}
			a = append(a, regexpGroupToValue(sUtf16, match.Groups[0]))
			if match.Groups[0].Start == match.Groups[0].End {
				thisIndex := toLength(rx.getStr("lastIndex", nil))
				rx.setOwnStr("lastIndex", valueInt(advanceStringIndex64(s, thisIndex, rx.pattern.unicode)), true)
			}
		}

		if len(a) == 0 {
			return _null
		}
		return r.newArrayValues(a)
	} else {
		return rx.exec(s)
	}
}

func (r *Runtime) regexpproto_stdSearchGeneric(rxObj *Object, arg String) Value {
	rx := rxObj.self
	previousLastIndex := nilSafe(rx.getStr("lastIndex", nil))
	zero := intToValue(0)
	if !previousLastIndex.SameAs(zero) {
		rx.setOwnStr("lastIndex", zero, true)
	}
	execFn, ok := r.toObject(rx.getStr("exec", nil)).self.assertCallable()
	if !ok {
		panic(r.NewTypeError("exec is not a function"))
	}

	result := r.regExpExec(execFn, rxObj, arg)
	currentLastIndex := nilSafe(rx.getStr("lastIndex", nil))
	if !currentLastIndex.SameAs(previousLastIndex) {
		rx.setOwnStr("lastIndex", previousLastIndex, true)
	}

	if result == _null {
		return intToValue(-1)
	}

	return r.toObject(result).self.getStr("index", nil)
}

func (r *Runtime) regexpproto_stdMatcherAll(call FunctionCall) Value {
	thisObj := r.toObject(call.This)
	s := call.Argument(0).toString()
	flags := nilSafe(thisObj.self.getStr("flags", nil)).toString()
	c := r.speciesConstructorObj(call.This.(*Object), r.getRegExp())
	matcher := r.toConstructor(c)([]Value{call.This, flags}, nil)
	matcher.self.setOwnStr("lastIndex", valueInt(toLength(thisObj.self.getStr("lastIndex", nil))), true)
	flagsStr := flags.String()
	global := strings.ContainsRune(flagsStr, 'g')
	fullUnicode := strings.ContainsRune(flagsStr, 'u') || strings.ContainsRune(flagsStr, 'v')
	return r.createRegExpStringIterator(matcher, s, global, fullUnicode)
}

func (r *Runtime) createRegExpStringIterator(matcher *Object, s String, global, fullUnicode bool) Value {
	o := &Object{runtime: r}

	ri := &regExpStringIterObject{
		matcher:     matcher,
		s:           s,
		global:      global,
		fullUnicode: fullUnicode,
	}
	ri.class = classObject
	ri.val = o
	ri.extensible = true
	o.self = ri
	ri.prototype = r.getRegExpStringIteratorPrototype()
	ri.init()

	return o
}

type regExpStringIterObject struct {
	baseObject
	matcher                   *Object
	s                         String
	global, fullUnicode, done bool
}

// RegExpExec as defined in 21.2.5.2.1
func regExpExec(r *Object, s String) Value {
	exec := r.self.getStr("exec", nil)
	if execObject, ok := exec.(*Object); ok {
		if execFn, ok := execObject.self.assertCallable(); ok {
			return r.runtime.regExpExec(execFn, r, s)
		}
	}
	if rx, ok := r.self.(*regexpObject); ok {
		return rx.exec(s)
	}
	panic(r.runtime.NewTypeError("no RegExpMatcher internal slot"))
}

func (ri *regExpStringIterObject) next() (v Value) {
	if ri.done {
		return ri.val.runtime.createIterResultObject(_undefined, true)
	}

	match := regExpExec(ri.matcher, ri.s)
	if IsNull(match) {
		ri.done = true
		return ri.val.runtime.createIterResultObject(_undefined, true)
	}
	if !ri.global {
		ri.done = true
		return ri.val.runtime.createIterResultObject(match, false)
	}

	matchStr := nilSafe(ri.val.runtime.toObject(match).self.getIdx(valueInt(0), nil)).toString()
	if matchStr.Length() == 0 {
		thisIndex := toLength(ri.matcher.self.getStr("lastIndex", nil))
		ri.matcher.self.setOwnStr("lastIndex", valueInt(advanceStringIndex64(ri.s, thisIndex, ri.fullUnicode)), true)
	}
	return ri.val.runtime.createIterResultObject(match, false)
}

func (r *Runtime) regexpproto_stdSearch(call FunctionCall) Value {
	thisObj := r.toObject(call.This)
	s := call.Argument(0).toString()
	rx := r.checkStdRegexp(thisObj)
	if rx == nil {
		return r.regexpproto_stdSearchGeneric(thisObj, s)
	}

	previousLastIndex := rx.getStr("lastIndex", nil)
	rx.setOwnStr("lastIndex", intToValue(0), true)

	match := rx.execRegexp(rx.pattern, s, false)
	rx.setOwnStr("lastIndex", previousLastIndex, true)

	if match == nil {
		return intToValue(-1)
	}
	return intToValue(int64(match.Groups[0].Start))
}

func (r *Runtime) regexpproto_stdSplitterGeneric(splitter *Object, s String, limit Value, unicodeMatching bool) Value {
	var a []Value
	var lim int64
	if limit == nil || limit == _undefined {
		lim = maxInt - 1
	} else {
		lim = toLength(limit)
	}
	if lim == 0 {
		return r.newArrayValues(a)
	}
	size := s.Length()
	p := 0
	execFn := toMethod(splitter.ToObject(r).self.getStr("exec", nil)) // must be non-nil

	if size == 0 {
		if r.regExpExec(execFn, splitter, s) == _null {
			a = append(a, s)
		}
		return r.newArrayValues(a)
	}

	q := p
	for q < size {
		splitter.self.setOwnStr("lastIndex", intToValue(int64(q)), true)
		z := r.regExpExec(execFn, splitter, s)
		if z == _null {
			q = advanceStringIndex(s, q, unicodeMatching)
		} else {
			z := r.toObject(z)
			e := toLength(splitter.self.getStr("lastIndex", nil))
			if e == int64(p) {
				q = advanceStringIndex(s, q, unicodeMatching)
			} else {
				a = append(a, s.Substring(p, q))
				if int64(len(a)) == lim {
					return r.newArrayValues(a)
				}
				if e > int64(size) {
					p = size
				} else {
					p = int(e)
				}
				numberOfCaptures := max(toLength(z.self.getStr("length", nil))-1, 0)
				for i := int64(1); i <= numberOfCaptures; i++ {
					a = append(a, nilSafe(z.self.getIdx(valueInt(i), nil)))
					if int64(len(a)) == lim {
						return r.newArrayValues(a)
					}
				}
				q = p
			}
		}
	}
	a = append(a, s.Substring(p, size))
	return r.newArrayValues(a)
}

func advanceStringIndex(s String, pos int, unicode bool) int {
	next := pos + 1
	if !unicode {
		return next
	}
	l := s.Length()
	if next >= l {
		return next
	}
	if !isUTF16FirstSurrogate(s.CharAt(pos)) {
		return next
	}
	if !isUTF16SecondSurrogate(s.CharAt(next)) {
		return next
	}
	return next + 1
}

func advanceStringIndex64(s String, pos int64, unicode bool) int64 {
	next := pos + 1
	if !unicode {
		return next
	}
	l := int64(s.Length())
	if next >= l {
		return next
	}
	if !isUTF16FirstSurrogate(s.CharAt(int(pos))) {
		return next
	}
	if !isUTF16SecondSurrogate(s.CharAt(int(next))) {
		return next
	}
	return next + 1
}

func (r *Runtime) regexpproto_stdSplitter(call FunctionCall) Value {
	rxObj := r.toObject(call.This)
	s := call.Argument(0).toString()
	limitValue := call.Argument(1)
	var splitter *Object
	search := r.checkStdRegexp(rxObj)
	c := r.speciesConstructorObj(rxObj, r.getRegExp())
	if search == nil || c != r.global.RegExp {
		flags := nilSafe(rxObj.self.getStr("flags", nil)).toString()
		flagsStr := flags.String()

		// Add 'y' flag if missing
		if !strings.Contains(flagsStr, "y") {
			flags = flags.Concat(asciiString("y"))
		}
		splitter = r.toConstructor(c)([]Value{rxObj, flags}, nil)
		search = r.checkStdRegexp(splitter)
		if search == nil {
			return r.regexpproto_stdSplitterGeneric(splitter, s, limitValue, strings.ContainsRune(flagsStr, 'u') || strings.ContainsRune(flagsStr, 'v'))
		}
	}

	pattern := search.pattern // toUint32() may recompile the pattern, but we still need to use the original

	var lim int64
	if limitValue == nil || limitValue == _undefined {
		lim = maxInt - 1
	} else {
		lim = int64(toUint32(limitValue))
	}

	if lim == 0 {
		return r.newArrayValues(nil)
	}

	size := s.Length()
	var a []Value

	sUtf16 := s.toUnicode()
	p := 0
	q := p

	if size == 0 {
		if search.execRegexp(pattern, s, true) == nil {
			a = append(a, s)
		}
		goto RETURN
	}

	for q < size {
		search.setOwnStr("lastIndex", intToValue(int64(q)), true)
		z := search.execRegexp(pattern, s, true)
		if z == nil {
			q = advanceStringIndex(s, q, search.pattern.unicode)
		} else {
			e := toLength(search.getStr("lastIndex", nil))
			if e == int64(p) {
				q = advanceStringIndex(s, q, search.pattern.unicode)
			} else {
				a = append(a, s.Substring(p, q))
				if int64(len(a)) == lim {
					return r.newArrayValues(a)
				}
				if e > int64(size) {
					p = size
				} else {
					p = int(e)
				}
				numberOfCaptures := max(int64(len(z.Groups))-1, 0)
				for i := int64(1); i <= numberOfCaptures; i++ {
					a = append(a, regexpGroupToValue(sUtf16, z.Groups[i]))
					if int64(len(a)) == lim {
						return r.newArrayValues(a)
					}
				}
				q = p
			}
		}
	}
	a = append(a, s.Substring(p, size))

RETURN:
	return r.newArrayValues(a)
}

func (r *Runtime) regexpproto_stdReplacerGeneric(rxObj *Object, s, replaceStr String, rcall func(FunctionCall) Value) Value {
	var results []Value
	flags := nilSafe(rxObj.self.getStr("flags", nil)).String()
	isGlobal := strings.ContainsRune(flags, 'g')
	isUnicode := strings.ContainsRune(flags, 'u') || strings.ContainsRune(flags, 'v')
	if isGlobal {
		results = r.getGlobalRegexpMatches(rxObj, s, isUnicode)
	} else {
		execFn := toMethod(rxObj.self.getStr("exec", nil)) // must be non-nil
		result := r.regExpExec(execFn, rxObj, s)
		if result != _null {
			results = append(results, result)
		}
	}
	lengthS := s.Length()
	nextSourcePosition := 0
	var resultBuf StringBuilder
	for _, result := range results {
		obj := r.toObject(result)
		nCaptures := max(toLength(obj.self.getStr("length", nil))-1, 0)
		matched := nilSafe(obj.self.getIdx(valueInt(0), nil)).toString()
		matchLength := matched.Length()
		position := toIntStrict(max(min(nilSafe(obj.self.getStr("index", nil)).ToInteger(), int64(lengthS)), 0))
		var captures []Value
		if rcall != nil {
			captures = make([]Value, 0, nCaptures+3)
		} else {
			captures = make([]Value, 0, nCaptures+1)
		}
		captures = append(captures, matched)
		for n := int64(1); n <= nCaptures; n++ {
			capN := nilSafe(obj.self.getIdx(valueInt(n), nil))
			if capN != _undefined {
				capN = capN.ToString()
			}
			captures = append(captures, capN)
		}
		var replacement String
		if rcall != nil {
			captures = append(captures, intToValue(int64(position)), s)
			replacement = rcall(FunctionCall{
				This:      _undefined,
				Arguments: captures,
			}).toString()
			if position >= nextSourcePosition {
				resultBuf.WriteString(s.Substring(nextSourcePosition, position))
				resultBuf.WriteString(replacement)
				nextSourcePosition = position + matchLength
			}
		} else {
			if position >= nextSourcePosition {
				resultBuf.WriteString(s.Substring(nextSourcePosition, position))
				writeSubstitution(s, position, len(captures), func(idx int) String {
					capture := captures[idx]
					if capture != _undefined {
						return capture.toString()
					}
					return stringEmpty
				}, replaceStr, &resultBuf)
				nextSourcePosition = position + matchLength
			}
		}
	}
	if nextSourcePosition < lengthS {
		resultBuf.WriteString(s.Substring(nextSourcePosition, lengthS))
	}
	return resultBuf.String()
}

func writeSubstitution(s String, position int, numCaptures int, getCapture func(int) String, replaceStr String, buf *StringBuilder) {
	l := s.Length()
	rl := replaceStr.Length()
	matched := getCapture(0)
	tailPos := position + matched.Length()

	for i := 0; i < rl; i++ {
		c := replaceStr.CharAt(i)
		if c == '$' && i < rl-1 {
			ch := replaceStr.CharAt(i + 1)
			switch ch {
			case '$':
				buf.WriteRune('$')
			case '`':
				buf.WriteString(s.Substring(0, position))
			case '\'':
				if tailPos < l {
					buf.WriteString(s.Substring(tailPos, l))
				}
			case '&':
				buf.WriteString(matched)
			default:
				matchNumber := 0
				j := i + 1
				for j < rl {
					ch := replaceStr.CharAt(j)
					if ch >= '0' && ch <= '9' {
						m := matchNumber*10 + int(ch-'0')
						if m >= numCaptures {
							break
						}
						matchNumber = m
						j++
					} else {
						break
					}
				}
				if matchNumber > 0 {
					buf.WriteString(getCapture(matchNumber))
					i = j - 1
					continue
				} else {
					buf.WriteRune('$')
					buf.WriteRune(rune(ch))
				}
			}
			i++
		} else {
			buf.WriteRune(rune(c))
		}
	}
}

func (r *Runtime) regexpproto_stdReplacer(call FunctionCall) Value {
	rxObj := r.toObject(call.This)
	s := call.Argument(0).toString()
	replaceStr, rcall := getReplaceValue(call.Argument(1))

	rx := r.checkStdRegexp(rxObj)
	if rx == nil {
		return r.regexpproto_stdReplacerGeneric(rxObj, s, replaceStr, rcall)
	}

	if rx.pattern.global {
		rx.setOwnStr("lastIndex", intToValue(0), true)
	}
	sUtf16 := s.toUnicode()
	a := [][]int{}
	for {
		match := rx.execRegexp(rx.pattern, sUtf16, false)
		if match == nil {
			break
		}
		result := make([]int, len(match.Groups)<<1)
		for i, group := range match.Groups {
			if group.Start == -1 {
				result[i*2] = -1
				result[i*2+1] = 0
			} else {
				result[i*2] = group.Start
				result[i*2+1] = group.End
			}
		}
		a = append(a, result)
		if !rx.pattern.global {
			break
		}

		if match.Groups[0].Start == match.Groups[0].End {
			thisIndex := toLength(rx.getStr("lastIndex", nil))
			rx.setOwnStr("lastIndex", valueInt(advanceStringIndex64(s, thisIndex, rx.pattern.unicode)), true)
		}

	}
	return stringReplace(s, a, replaceStr, rcall)
}

func (r *Runtime) regExpStringIteratorProto_next(call FunctionCall) Value {
	thisObj := r.toObject(call.This)
	if iter, ok := thisObj.self.(*regExpStringIterObject); ok {
		return iter.next()
	}
	panic(r.NewTypeError("Method RegExp String Iterator.prototype.next called on incompatible receiver %s", r.objectproto_toString(FunctionCall{This: thisObj})))
}

func (r *Runtime) createRegExpStringIteratorPrototype(val *Object) objectImpl {
	o := newBaseObjectObj(val, r.getIteratorPrototype(), classObject)

	o._putProp("next", r.newNativeFunc(r.regExpStringIteratorProto_next, "next", 0), true, false, true)
	o._putSym(SymToStringTag, valueProp(asciiString(classRegExpStringIterator), false, false, true))

	return o
}

func (r *Runtime) getRegExpStringIteratorPrototype() *Object {
	var o *Object
	if o = r.global.RegExpStringIteratorPrototype; o == nil {
		o = &Object{runtime: r}
		r.global.RegExpStringIteratorPrototype = o
		o.self = r.createRegExpStringIteratorPrototype(o)
	}
	return o
}

func (r *Runtime) getRegExp() *Object {
	ret := r.global.RegExp
	if ret == nil {
		ret = &Object{runtime: r}
		r.global.RegExp = ret
		proto := r.getRegExpPrototype()
		r.newNativeFuncAndConstruct(ret, r.builtin_RegExp,
			r.wrapNativeConstruct(r.builtin_newRegExp, ret, proto), proto, "RegExp", intToValue(2))
		rx := ret.self
		r.putSpeciesReturnThis(rx)
	}
	return ret
}

func (r *Runtime) getRegExpPrototype() *Object {
	ret := r.global.RegExpPrototype
	if ret == nil {
		o := r.newGuardedObject(r.global.ObjectPrototype, classObject)
		ret = o.val
		r.global.RegExpPrototype = ret
		r.global.stdRegexpProto = o

		o._putProp("constructor", r.getRegExp(), true, false, true)
		o._putProp("compile", r.newNativeFunc(r.regexpproto_compile, "compile", 2), true, false, true)
		o._putProp("exec", r.newNativeFunc(r.regexpproto_exec, "exec", 1), true, false, true)
		o._putProp("test", r.newNativeFunc(r.regexpproto_test, "test", 1), true, false, true)
		o._putProp("toString", r.newNativeFunc(r.regexpproto_toString, "toString", 0), true, false, true)
		o.setOwnStr("source", &valueProperty{
			configurable: true,
			getterFunc:   r.newNativeFunc(r.regexpproto_getSource, "get source", 0),
			accessor:     true,
		}, false)
		o.setOwnStr("global", &valueProperty{
			configurable: true,
			getterFunc:   r.newNativeFunc(r.regexpproto_getGlobal, "get global", 0),
			accessor:     true,
		}, false)
		o.setOwnStr("multiline", &valueProperty{
			configurable: true,
			getterFunc:   r.newNativeFunc(r.regexpproto_getMultiline, "get multiline", 0),
			accessor:     true,
		}, false)
		o.setOwnStr("dotAll", &valueProperty{
			configurable: true,
			getterFunc:   r.newNativeFunc(r.regexpproto_getDotAll, "get dotAll", 0),
			accessor:     true,
		}, false)
		o.setOwnStr("ignoreCase", &valueProperty{
			configurable: true,
			getterFunc:   r.newNativeFunc(r.regexpproto_getIgnoreCase, "get ignoreCase", 0),
			accessor:     true,
		}, false)
		o.setOwnStr("unicode", &valueProperty{
			configurable: true,
			getterFunc:   r.newNativeFunc(r.regexpproto_getUnicode, "get unicode", 0),
			accessor:     true,
		}, false)
		o.setOwnStr("sticky", &valueProperty{
			configurable: true,
			getterFunc:   r.newNativeFunc(r.regexpproto_getSticky, "get sticky", 0),
			accessor:     true,
		}, false)
		o.setOwnStr("flags", &valueProperty{
			configurable: true,
			getterFunc:   r.newNativeFunc(r.regexpproto_getFlags, "get flags", 0),
			accessor:     true,
		}, false)

		o._putSym(SymMatch, valueProp(r.newNativeFunc(r.regexpproto_stdMatcher, "[Symbol.match]", 1), true, false, true))
		o._putSym(SymMatchAll, valueProp(r.newNativeFunc(r.regexpproto_stdMatcherAll, "[Symbol.matchAll]", 1), true, false, true))
		o._putSym(SymSearch, valueProp(r.newNativeFunc(r.regexpproto_stdSearch, "[Symbol.search]", 1), true, false, true))
		o._putSym(SymSplit, valueProp(r.newNativeFunc(r.regexpproto_stdSplitter, "[Symbol.split]", 2), true, false, true))
		o._putSym(SymReplace, valueProp(r.newNativeFunc(r.regexpproto_stdReplacer, "[Symbol.replace]", 2), true, false, true))
		o.guard("exec", "global", "multiline", "ignoreCase", "unicode", "sticky")
	}
	return ret
}
