package goja

import (
	"fmt"
	"github.com/dlclark/regexp2"
	"regexp"
	"unicode/utf16"
	"unicode/utf8"
)

type regexpPattern interface {
	FindSubmatchIndex(valueString, int) []int
	FindAllSubmatchIndex(valueString, int) [][]int
	FindAllSubmatchIndexUTF8(string, int) [][]int
	FindAllSubmatchIndexASCII(string, int) [][]int
	MatchString(valueString) bool
}

type regexp2Wrapper regexp2.Regexp
type regexpWrapper regexp.Regexp

type regexpObject struct {
	baseObject
	pattern regexpPattern
	source  valueString

	global, multiline, ignoreCase, sticky bool
}

func (r *regexp2Wrapper) FindSubmatchIndex(s valueString, start int) (result []int) {
	wrapped := (*regexp2.Regexp)(r)
	var match *regexp2.Match
	var err error
	switch s := s.(type) {
	case asciiString:
		match, err = wrapped.FindStringMatch(string(s)[start:])
	case unicodeString:
		match, err = wrapped.FindRunesMatch(utf16.Decode(s[start+1:]))
	default:
		panic(fmt.Errorf("Unknown string type: %T", s))
	}
	if err != nil {
		return
	}

	if match == nil {
		return
	}
	groups := match.Groups()

	result = make([]int, 0, len(groups)<<1)
	for _, group := range groups {
		if len(group.Captures) > 0 {
			result = append(result, group.Index, group.Index+group.Length)
		} else {
			result = append(result, -1, 0)
		}
	}
	return
}

func (r *regexp2Wrapper) FindAllSubmatchIndexUTF8(s string, n int) [][]int {
	wrapped := (*regexp2.Regexp)(r)
	if n < 0 {
		n = len(s) + 1
	}
	results := make([][]int, 0, n)

	idxMap := make([]int, 0, len(s))
	runes := make([]rune, 0, len(s))
	for pos, rr := range s {
		runes = append(runes, rr)
		idxMap = append(idxMap, pos)
	}
	idxMap = append(idxMap, len(s))

	match, err := wrapped.FindRunesMatch(runes)
	if err != nil {
		return nil
	}
	i := 0
	for match != nil && i < n {
		groups := match.Groups()

		result := make([]int, 0, len(groups)<<1)

		for _, group := range groups {
			if len(group.Captures) > 0 {
				result = append(result, idxMap[group.Index], idxMap[group.Index+group.Length])
			} else {
				result = append(result, -1, 0)
			}
		}

		results = append(results, result)
		match, err = wrapped.FindNextMatch(match)
		if err != nil {
			return nil
		}
		i++
	}
	return results
}

func (r *regexp2Wrapper) FindAllSubmatchIndexASCII(s string, n int) [][]int {
	wrapped := (*regexp2.Regexp)(r)
	if n < 0 {
		n = len(s) + 1
	}
	results := make([][]int, 0, n)

	match, err := wrapped.FindStringMatch(s)
	if err != nil {
		return nil
	}
	i := 0
	for match != nil && i < n {
		groups := match.Groups()

		result := make([]int, 0, len(groups)<<1)

		for _, group := range groups {
			if len(group.Captures) > 0 {
				result = append(result, group.Index, group.Index+group.Length)
			} else {
				result = append(result, -1, 0)
			}
		}

		results = append(results, result)
		match, err = wrapped.FindNextMatch(match)
		if err != nil {
			return nil
		}
		i++
	}
	return results
}

func (r *regexp2Wrapper) findAllSubmatchIndexUTF16(s unicodeString, n int) [][]int {
	wrapped := (*regexp2.Regexp)(r)
	if n < 0 {
		n = len(s) + 1
	}
	results := make([][]int, 0, n)

	rd := runeReaderReplace{s.reader(0)}
	posMap := make([]int, s.length()+1)
	curPos := 0
	curRuneIdx := 0
	runes := make([]rune, 0, s.length())
	for {
		rn, size, err := rd.ReadRune()
		if err != nil {
			break
		}
		runes = append(runes, rn)
		posMap[curRuneIdx] = curPos
		curRuneIdx++
		curPos += size
	}
	posMap[curRuneIdx] = curPos

	match, err := wrapped.FindRunesMatch(runes)
	if err != nil {
		return nil
	}
	for match != nil {
		groups := match.Groups()

		result := make([]int, 0, len(groups)<<1)

		for _, group := range groups {
			if len(group.Captures) > 0 {
				start := posMap[group.Index]
				end := posMap[group.Index+group.Length]
				result = append(result, start, end)
			} else {
				result = append(result, -1, 0)
			}
		}

		results = append(results, result)
		match, err = wrapped.FindNextMatch(match)
		if err != nil {
			return nil
		}
	}
	return results
}

func (r *regexp2Wrapper) FindAllSubmatchIndex(s valueString, n int) [][]int {
	switch s := s.(type) {
	case asciiString:
		return r.FindAllSubmatchIndexASCII(string(s), n)
	case unicodeString:
		return r.findAllSubmatchIndexUTF16(s, n)
	default:
		panic("Unsupported string type")
	}
}

func (r *regexp2Wrapper) MatchString(s valueString) bool {
	wrapped := (*regexp2.Regexp)(r)

	switch s := s.(type) {
	case asciiString:
		matched, _ := wrapped.MatchString(string(s))
		return matched
	case unicodeString:
		matched, _ := wrapped.MatchRunes(utf16.Decode(s[1:]))
		return matched
	default:
		panic(fmt.Errorf("Unknown string type: %T", s))
	}
}

func (r *regexpWrapper) FindSubmatchIndex(s valueString, start int) (result []int) {
	wrapped := (*regexp.Regexp)(r)
	return wrapped.FindReaderSubmatchIndex(runeReaderReplace{s.reader(start)})
}

func (r *regexpWrapper) MatchString(s valueString) bool {
	wrapped := (*regexp.Regexp)(r)
	return wrapped.MatchReader(runeReaderReplace{s.reader(0)})
}

func (r *regexpWrapper) FindAllSubmatchIndex(s valueString, n int) [][]int {
	wrapped := (*regexp.Regexp)(r)
	switch s := s.(type) {
	case asciiString:
		return wrapped.FindAllStringSubmatchIndex(string(s), n)
	case unicodeString:
		return r.findAllSubmatchIndexUTF16(s, n)
	default:
		panic("Unsupported string type")
	}
}

func (r *regexpWrapper) FindAllSubmatchIndexUTF8(s string, n int) [][]int {
	wrapped := (*regexp.Regexp)(r)
	return wrapped.FindAllStringSubmatchIndex(s, n)
}

func (r *regexpWrapper) FindAllSubmatchIndexASCII(s string, n int) [][]int {
	return r.FindAllSubmatchIndexUTF8(s, n)
}

func (r *regexpWrapper) findAllSubmatchIndexUTF16(s unicodeString, n int) [][]int {
	wrapped := (*regexp.Regexp)(r)
	utf8Bytes := make([]byte, 0, len(s)*2)
	posMap := make(map[int]int)
	curPos := 0
	rd := runeReaderReplace{s.reader(0)}
	for {
		rn, size, err := rd.ReadRune()
		if err != nil {
			break
		}
		l := len(utf8Bytes)
		utf8Bytes = append(utf8Bytes, 0, 0, 0, 0)
		n := utf8.EncodeRune(utf8Bytes[l:], rn)
		utf8Bytes = utf8Bytes[:l+n]
		posMap[l] = curPos
		curPos += size
	}
	posMap[len(utf8Bytes)] = curPos

	rr := wrapped.FindAllSubmatchIndex(utf8Bytes, n)
	for _, res := range rr {
		for j, pos := range res {
			mapped, exists := posMap[pos]
			if !exists {
				panic("Unicode match is not on rune boundary")
			}
			res[j] = mapped
		}
	}
	return rr
}

func (r *regexpObject) execResultToArray(target valueString, result []int) Value {
	captureCount := len(result) >> 1
	valueArray := make([]Value, captureCount)
	matchIndex := result[0]
	lowerBound := matchIndex
	for index := 0; index < captureCount; index++ {
		offset := index << 1
		if result[offset] >= lowerBound {
			valueArray[index] = target.substring(result[offset], result[offset+1])
			lowerBound = result[offset]
		} else {
			valueArray[index] = _undefined
		}
	}
	match := r.val.runtime.newArrayValues(valueArray)
	match.self.setOwnStr("input", target, false)
	match.self.setOwnStr("index", intToValue(int64(matchIndex)), false)
	return match
}

func (r *regexpObject) execRegexp(target valueString) (match bool, result []int) {
	lastIndex := int64(0)
	if p := r.getStr("lastIndex", nil); p != nil {
		lastIndex = p.ToInteger()
		if lastIndex < 0 {
			lastIndex = 0
		}
	}
	index := lastIndex
	if !r.global && !r.sticky {
		index = 0
	}
	if index >= 0 && index <= int64(target.length()) {
		result = r.pattern.FindSubmatchIndex(target, int(index))
	}
	if result == nil || r.sticky && result[0] != 0 {
		r.setOwnStr("lastIndex", intToValue(0), true)
		return
	}
	match = true
	// We do this shift here because the .FindStringSubmatchIndex above
	// was done on a local subordinate slice of the string, not the whole string
	for i := range result {
		result[i] += int(index)
	}
	if r.global || r.sticky {
		r.setOwnStr("lastIndex", intToValue(int64(result[1])), true)
	}
	return
}

func (r *regexpObject) exec(target valueString) Value {
	match, result := r.execRegexp(target)
	if match {
		return r.execResultToArray(target, result)
	}
	return _null
}

func (r *regexpObject) test(target valueString) bool {
	match, _ := r.execRegexp(target)
	return match
}

func (r *regexpObject) clone() *Object {
	r1 := r.val.runtime.newRegexpObject(r.prototype)
	r1.source = r.source
	r1.pattern = r.pattern
	r1.global = r.global
	r1.ignoreCase = r.ignoreCase
	r1.multiline = r.multiline
	r1.sticky = r.sticky
	return r1.val
}

func (r *regexpObject) init() {
	r.baseObject.init()
	r._putProp("lastIndex", intToValue(0), true, false, false)
}
