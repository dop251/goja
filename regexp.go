package goja

import (
	"github.com/auvred/regonaut"
	"github.com/dop251/goja/unistring"
)

// Not goroutine-safe. Use regexpPattern.clone()
type regexpPattern struct {
	src unicodeString

	global, ignoreCase, multiline, dotAll, sticky, unicode, unicodeSets bool

	re *regonaut.RegExpUtf16
}

// TODO: regonaut's RegExp is safe for concurrent use
// clone creates a copy of the regexpPattern which can be used concurrently.
func (p *regexpPattern) clone() *regexpPattern {
	return p
}

type regexpObject struct {
	baseObject
	pattern *regexpPattern
	source  String

	standard bool
}

func regexpGroupToValue(str unicodeString, group regonaut.GroupUtf16) Value {
	if group.Start >= 0 {
		return str.Substring(group.Start, group.End)
	} else {
		return _undefined
	}
}
func (r *regexpObject) execResultToArray(target String, targetUtf16 unicodeString, match *regonaut.MatchUtf16) Value {
	valueArray := make([]Value, len(match.Groups))
	matchIndex := match.Groups[0].Start
	for index, group := range match.Groups {
		valueArray[index] = regexpGroupToValue(targetUtf16, group)
	}
	result := r.val.runtime.newArrayValues(valueArray)
	result.self.setOwnStr("input", target, false)
	result.self.setOwnStr("index", intToValue(int64(matchIndex)), false)
	return result
}

func (r *regexpObject) execRegexp(pattern *regexpPattern, target String, sticky bool) *regonaut.MatchUtf16 {
	globalOrSticky := pattern.global || sticky || pattern.sticky
	index := toLength(r.getStr("lastIndex", nil))

	var match *regonaut.MatchUtf16

	if !globalOrSticky || index < 0 {
		index = 0
	}

	if sticky {
		match = pattern.re.FindMatchStartingAtSticky(target.toUnicode()[1:], int(index))
	} else {
		match = pattern.re.FindMatchStartingAt(target.toUnicode()[1:], int(index))
	}
	if globalOrSticky {
		if match == nil {
			index = 0
		} else {
			index = int64(match.Groups[0].End)
		}
		r.setOwnStr("lastIndex", intToValue(index), true)
	}

	return match
}

func (r *regexpObject) exec(target String) Value {
	targetUtf16 := target.toUnicode()
	match := r.execRegexp(r.pattern, targetUtf16, false)
	if match == nil {
		return _null
	}
	return r.execResultToArray(target, targetUtf16, match)
}

func (r *regexpObject) test(target String) bool {
	return r.execRegexp(r.pattern, target, false) != nil
}

func (r *regexpObject) clone() *regexpObject {
	r1 := r.val.runtime.newRegexpObject(r.prototype)
	r1.source = r.source
	r1.pattern = r.pattern

	return r1
}

func (r *regexpObject) init() {
	r.baseObject.init()
	r.standard = true
	r._putProp("lastIndex", intToValue(0), true, false, false)
}

func (r *regexpObject) setProto(proto *Object, throw bool) bool {
	res := r.baseObject.setProto(proto, throw)
	if res {
		r.standard = false
	}
	return res
}

func (r *regexpObject) defineOwnPropertyStr(name unistring.String, desc PropertyDescriptor, throw bool) bool {
	res := r.baseObject.defineOwnPropertyStr(name, desc, throw)
	if res {
		r.standard = false
	}
	return res
}

func (r *regexpObject) defineOwnPropertySym(name *Symbol, desc PropertyDescriptor, throw bool) bool {
	res := r.baseObject.defineOwnPropertySym(name, desc, throw)
	if res && r.standard {
		switch name {
		case SymMatch, SymMatchAll, SymSearch, SymSplit, SymReplace:
			r.standard = false
		}
	}
	return res
}

func (r *regexpObject) deleteStr(name unistring.String, throw bool) bool {
	res := r.baseObject.deleteStr(name, throw)
	if res {
		r.standard = false
	}
	return res
}

func (r *regexpObject) setOwnStr(name unistring.String, value Value, throw bool) bool {
	res := r.baseObject.setOwnStr(name, value, throw)
	if res && r.standard && name == "exec" {
		r.standard = false
	}
	return res
}

func (r *regexpObject) setOwnSym(name *Symbol, value Value, throw bool) bool {
	res := r.baseObject.setOwnSym(name, value, throw)
	if res && r.standard {
		switch name {
		case SymMatch, SymMatchAll, SymSearch, SymSplit, SymReplace:
			r.standard = false
		}
	}
	return res
}
