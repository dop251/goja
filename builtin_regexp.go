package goja

import (
	"fmt"
	"github.com/dlclark/regexp2"
	"github.com/dop251/goja/parser"
	"regexp"
	"strings"
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

func (r *Runtime) newRegExpp(pattern regexpPattern, patternStr valueString, global, ignoreCase, multiline, sticky bool, proto *Object) *Object {
	o := r.newRegexpObject(proto)

	o.pattern = pattern
	o.source = patternStr
	o.global = global
	o.ignoreCase = ignoreCase
	o.multiline = multiline
	o.sticky = sticky

	return o.val
}

func compileRegexp(patternStr, flags string) (p regexpPattern, global, ignoreCase, multiline, sticky bool, err error) {

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
				multiline = true
			case 'i':
				if ignoreCase {
					invalidFlags()
					return
				}
				ignoreCase = true
			case 'y':
				if sticky {
					invalidFlags()
					return
				}
				sticky = true
			default:
				invalidFlags()
				return
			}
		}
	}

	re2Str, err1 := parser.TransformRegExp(patternStr)
	if /*false &&*/ err1 == nil {
		re2flags := ""
		if multiline {
			re2flags += "m"
		}
		if ignoreCase {
			re2flags += "i"
		}
		if len(re2flags) > 0 {
			re2Str = fmt.Sprintf("(?%s:%s)", re2flags, re2Str)
		}

		pattern, err1 := regexp.Compile(re2Str)
		if err1 != nil {
			err = fmt.Errorf("Invalid regular expression (re2): %s (%v)", re2Str, err1)
			return
		}

		p = (*regexpWrapper)(pattern)
	} else {
		var opts regexp2.RegexOptions = regexp2.ECMAScript
		if multiline {
			opts |= regexp2.Multiline
		}
		if ignoreCase {
			opts |= regexp2.IgnoreCase
		}
		regexp2Pattern, err1 := regexp2.Compile(patternStr, opts)
		if err1 != nil {
			err = fmt.Errorf("Invalid regular expression (regexp2): %s (%v)", patternStr, err1)
			return
		}
		p = (*regexp2Wrapper)(regexp2Pattern)
	}
	return
}

func (r *Runtime) newRegExp(patternStr valueString, flags string, proto *Object) *Object {
	pattern, global, ignoreCase, multiline, sticky, err := compileRegexp(patternStr.String(), flags)
	if err != nil {
		panic(r.newSyntaxError(err.Error(), -1))
	}
	return r.newRegExpp(pattern, patternStr, global, ignoreCase, multiline, sticky, proto)
}

func (r *Runtime) builtin_newRegExp(args []Value, proto *Object) *Object {
	var pattern valueString
	var flags string
	if len(args) > 0 {
		if obj, ok := args[0].(*Object); ok {
			if rx, ok := obj.self.(*regexpObject); ok {
				if len(args) < 2 || args[1] == _undefined {
					return rx.clone()
				} else {
					return r.newRegExp(rx.source, args[1].String(), proto)
				}
			}
		}
		if args[0] != _undefined {
			pattern = args[0].toString()
		}
	}
	if len(args) > 1 {
		if a := args[1]; a != _undefined {
			flags = a.String()
		}
	}
	if pattern == nil {
		pattern = stringEmpty
	}
	return r.newRegExp(pattern, flags, proto)
}

func (r *Runtime) builtin_RegExp(call FunctionCall) Value {
	flags := call.Argument(1)
	if flags == _undefined {
		if obj, ok := call.Argument(0).(*Object); ok {
			if _, ok := obj.self.(*regexpObject); ok {
				return call.Arguments[0]
			}
		}
	}
	return r.builtin_newRegExp(call.Arguments, r.global.RegExpPrototype)
}

func (r *Runtime) regexpproto_exec(call FunctionCall) Value {
	if this, ok := r.toObject(call.This).self.(*regexpObject); ok {
		return this.exec(call.Argument(0).toString())
	} else {
		r.typeErrorResult(true, "Method RegExp.prototype.exec called on incompatible receiver %s", call.This.toString())
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
		r.typeErrorResult(true, "Method RegExp.prototype.test called on incompatible receiver %s", call.This.toString())
		return nil
	}
}

func (r *Runtime) regexpproto_toString(call FunctionCall) Value {
	if this, ok := r.toObject(call.This).self.(*regexpObject); ok {
		var g, i, m, y string
		if this.global {
			g = "g"
		}
		if this.ignoreCase {
			i = "i"
		}
		if this.multiline {
			m = "m"
		}
		if this.sticky {
			y = "y"
		}
		return newStringValue(fmt.Sprintf("/%s/%s%s%s%s", this.source.String(), g, i, m, y))
	} else {
		r.typeErrorResult(true, "Method RegExp.prototype.toString called on incompatible receiver %s", call.This)
		return nil
	}
}

func (r *Runtime) regexpproto_getSource(call FunctionCall) Value {
	if this, ok := r.toObject(call.This).self.(*regexpObject); ok {
		return this.source
	} else {
		r.typeErrorResult(true, "Method RegExp.prototype.source getter called on incompatible receiver %s", call.This.toString())
		return nil
	}
}

func (r *Runtime) regexpproto_getGlobal(call FunctionCall) Value {
	if this, ok := r.toObject(call.This).self.(*regexpObject); ok {
		if this.global {
			return valueTrue
		} else {
			return valueFalse
		}
	} else {
		r.typeErrorResult(true, "Method RegExp.prototype.global getter called on incompatible receiver %s", call.This.toString())
		return nil
	}
}

func (r *Runtime) regexpproto_getMultiline(call FunctionCall) Value {
	if this, ok := r.toObject(call.This).self.(*regexpObject); ok {
		if this.multiline {
			return valueTrue
		} else {
			return valueFalse
		}
	} else {
		r.typeErrorResult(true, "Method RegExp.prototype.multiline getter called on incompatible receiver %s", call.This.toString())
		return nil
	}
}

func (r *Runtime) regexpproto_getIgnoreCase(call FunctionCall) Value {
	if this, ok := r.toObject(call.This).self.(*regexpObject); ok {
		if this.ignoreCase {
			return valueTrue
		} else {
			return valueFalse
		}
	} else {
		r.typeErrorResult(true, "Method RegExp.prototype.ignoreCase getter called on incompatible receiver %s", call.This.toString())
		return nil
	}
}

func (r *Runtime) regexpproto_getSticky(call FunctionCall) Value {
	if this, ok := r.toObject(call.This).self.(*regexpObject); ok {
		if this.sticky {
			return valueTrue
		} else {
			return valueFalse
		}
	} else {
		r.typeErrorResult(true, "Method RegExp.prototype.sticky getter called on incompatible receiver %s", call.This.toString())
		return nil
	}
}

func (r *Runtime) regexpproto_getFlags(call FunctionCall) Value {
	var global, ignoreCase, multiline, sticky bool

	thisObj := r.toObject(call.This)
	if this, ok := thisObj.self.(*regexpObject); ok {
		global, ignoreCase, multiline, sticky = this.global, this.ignoreCase, this.multiline, this.sticky
	} else {
		if v := thisObj.self.getStr("global", nil); v != nil {
			global = v.ToBoolean()
		}
		if v := thisObj.self.getStr("ignoreCase", nil); v != nil {
			ignoreCase = v.ToBoolean()
		}
		if v := thisObj.self.getStr("multiline", nil); v != nil {
			multiline = v.ToBoolean()
		}
		if v := thisObj.self.getStr("sticky", nil); v != nil {
			sticky = v.ToBoolean()
		}
	}

	var sb strings.Builder
	if global {
		sb.WriteByte('g')
	}
	if ignoreCase {
		sb.WriteByte('i')
	}
	if multiline {
		sb.WriteByte('m')
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

func (r *Runtime) regexpproto_stdMatcherGeneric(rxObj *Object, arg Value) Value {
	rx := rxObj.self
	global := rx.getStr("global", nil)
	if global != nil && global.ToBoolean() {
		rx.setOwnStr("lastIndex", intToValue(0), true)
		execFn, ok := r.toObject(rx.getStr("exec", nil)).self.assertCallable()
		if !ok {
			panic(r.NewTypeError("exec is not a function"))
		}
		var a []Value
		for {
			res := r.regExpExec(execFn, rxObj, arg)
			if res == _null {
				break
			}
			matchStr := nilSafe(r.toObject(res).self.getIdx(valueInt(0), nil)).toString()
			a = append(a, matchStr)
			if matchStr.length() == 0 {
				thisIndex := rx.getStr("lastIndex", nil).ToInteger()
				rx.setOwnStr("lastIndex", intToValue(thisIndex+1), true) // TODO fullUnicode
			}
		}
		if len(a) == 0 {
			return _null
		}
		return r.newArrayValues(a)
	}

	execFn, ok := r.toObject(rx.getStr("exec", nil)).self.assertCallable()
	if !ok {
		panic(r.NewTypeError("exec is not a function"))
	}

	return r.regExpExec(execFn, rxObj, arg)
}

func (r *Runtime) checkStdRegexp(rxObj *Object) *regexpObject {
	rx, ok := rxObj.self.(*regexpObject)
	if !ok {
		return nil
	}
	execFn := rx.getStr("exec", nil)
	if execFn != nil && execFn != r.global.regexpProtoExec {
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
	if rx.global {
		rx.setOwnStr("lastIndex", intToValue(0), true)
		var a []Value
		var previousLastIndex int64
		for {
			match, result := rx.execRegexp(s)
			if !match {
				break
			}
			thisIndex := rx.getStr("lastIndex", nil).ToInteger()
			if thisIndex == previousLastIndex {
				previousLastIndex++
				rx.setOwnStr("lastIndex", intToValue(previousLastIndex), true)
			} else {
				previousLastIndex = thisIndex
			}
			a = append(a, s.substring(result[0], result[1]))
		}
		if len(a) == 0 {
			return _null
		}
		return r.newArrayValues(a)
	} else {
		return rx.exec(s)
	}
}

func (r *Runtime) regexpproto_stdSearchGeneric(rxObj *Object, arg valueString) Value {
	rx := rxObj.self
	previousLastIndex := rx.getStr("lastIndex", nil)
	rx.setOwnStr("lastIndex", intToValue(0), true)
	execFn, ok := r.toObject(rx.getStr("exec", nil)).self.assertCallable()
	if !ok {
		panic(r.NewTypeError("exec is not a function"))
	}

	result := r.regExpExec(execFn, rxObj, arg)
	rx.setOwnStr("lastIndex", previousLastIndex, true)

	if result == _null {
		return intToValue(-1)
	}

	return r.toObject(result).self.getStr("index", nil)
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

	match, result := rx.execRegexp(s)
	rx.setOwnStr("lastIndex", previousLastIndex, true)

	if !match {
		return intToValue(-1)
	}
	return intToValue(int64(result[0]))
}

func (r *Runtime) regexpproto_stdSplitterGeneric(splitter *Object, s valueString, limit Value) Value {
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
	size := s.length()
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
			q++
		} else {
			z := r.toObject(z)
			e := toLength(splitter.self.getStr("lastIndex", nil))
			if e == int64(p) {
				q++
			} else {
				a = append(a, s.substring(p, q))
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
					a = append(a, z.self.getIdx(valueInt(i), nil))
					if int64(len(a)) == lim {
						return r.newArrayValues(a)
					}
				}
				q = p
			}
		}
	}
	a = append(a, s.substring(p, size))
	return r.newArrayValues(a)
}

func (r *Runtime) regexpproto_stdSplitter(call FunctionCall) Value {
	rxObj := r.toObject(call.This)
	c := r.speciesConstructor(rxObj, r.global.RegExp)
	flags := nilSafe(rxObj.self.getStr("flags", nil)).toString()

	// Add 'y' flag if missing
	if flagsStr := flags.String(); !strings.Contains(flagsStr, "y") {
		flags = newStringValue(flagsStr + "y")
	}
	splitter := c([]Value{rxObj, flags}, nil)

	s := call.Argument(0).toString()
	limitValue := call.Argument(1)
	search := r.checkStdRegexp(splitter)
	if search == nil {
		return r.regexpproto_stdSplitterGeneric(splitter, s, limitValue)
	}

	limit := -1
	if limitValue != _undefined {
		limit = int(toUint32(limitValue))
	}

	if limit == 0 {
		return r.newArrayValues(nil)
	}

	targetLength := s.length()
	var valueArray []Value
	result := search.pattern.FindAllSubmatchIndex(s, -1)
	lastIndex := 0
	found := 0

	for _, match := range result {
		if match[0] == match[1] {
			// FIXME Ugh, this is a hack
			if match[0] == 0 || match[0] == targetLength {
				continue
			}
		}

		if lastIndex != match[0] {
			valueArray = append(valueArray, s.substring(lastIndex, match[0]))
			found++
		} else if lastIndex == match[0] {
			if lastIndex != -1 {
				valueArray = append(valueArray, stringEmpty)
				found++
			}
		}

		lastIndex = match[1]
		if found == limit {
			goto RETURN
		}

		captureCount := len(match) / 2
		for index := 1; index < captureCount; index++ {
			offset := index * 2
			var value Value
			if match[offset] != -1 {
				value = s.substring(match[offset], match[offset+1])
			} else {
				value = _undefined
			}
			valueArray = append(valueArray, value)
			found++
			if found == limit {
				goto RETURN
			}
		}
	}

	if found != limit {
		if lastIndex != targetLength {
			valueArray = append(valueArray, s.substring(lastIndex, targetLength))
		} else {
			valueArray = append(valueArray, stringEmpty)
		}
	}

RETURN:
	return r.newArrayValues(valueArray)
}

func (r *Runtime) initRegExp() {
	r.global.RegExpPrototype = r.NewObject()
	o := r.global.RegExpPrototype.self
	r.global.regexpProtoExec = valueProp(r.newNativeFunc(r.regexpproto_exec, nil, "exec", nil, 1), true, false, true)
	o.setOwnStr("exec", r.global.regexpProtoExec, true)
	o._putProp("test", r.newNativeFunc(r.regexpproto_test, nil, "test", nil, 1), true, false, true)
	o._putProp("toString", r.newNativeFunc(r.regexpproto_toString, nil, "toString", nil, 0), true, false, true)
	o.setOwnStr("source", &valueProperty{
		configurable: true,
		getterFunc:   r.newNativeFunc(r.regexpproto_getSource, nil, "get source", nil, 0),
		accessor:     true,
	}, false)
	o.setOwnStr("global", &valueProperty{
		configurable: true,
		getterFunc:   r.newNativeFunc(r.regexpproto_getGlobal, nil, "get global", nil, 0),
		accessor:     true,
	}, false)
	o.setOwnStr("multiline", &valueProperty{
		configurable: true,
		getterFunc:   r.newNativeFunc(r.regexpproto_getMultiline, nil, "get multiline", nil, 0),
		accessor:     true,
	}, false)
	o.setOwnStr("ignoreCase", &valueProperty{
		configurable: true,
		getterFunc:   r.newNativeFunc(r.regexpproto_getIgnoreCase, nil, "get ignoreCase", nil, 0),
		accessor:     true,
	}, false)
	o.setOwnStr("sticky", &valueProperty{
		configurable: true,
		getterFunc:   r.newNativeFunc(r.regexpproto_getSticky, nil, "get sticky", nil, 0),
		accessor:     true,
	}, false)
	o.setOwnStr("flags", &valueProperty{
		configurable: true,
		getterFunc:   r.newNativeFunc(r.regexpproto_getFlags, nil, "get flags", nil, 0),
		accessor:     true,
	}, false)

	o._putSym(symMatch, valueProp(r.newNativeFunc(r.regexpproto_stdMatcher, nil, "[Symbol.match]", nil, 1), true, false, true))
	o._putSym(symSearch, valueProp(r.newNativeFunc(r.regexpproto_stdSearch, nil, "[Symbol.search]", nil, 1), true, false, true))
	o._putSym(symSplit, valueProp(r.newNativeFunc(r.regexpproto_stdSplitter, nil, "[Symbol.split]", nil, 2), true, false, true))

	r.global.RegExp = r.newNativeFunc(r.builtin_RegExp, r.builtin_newRegExp, "RegExp", r.global.RegExpPrototype, 2)
	o = r.global.RegExp.self
	o._putSym(symSpecies, &valueProperty{
		getterFunc:   r.newNativeFunc(r.returnThis, nil, "get [Symbol.species]", nil, 0),
		accessor:     true,
		configurable: true,
	})
	r.addToGlobal("RegExp", r.global.RegExp)
}
