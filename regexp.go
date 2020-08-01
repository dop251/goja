package goja

import (
	"fmt"
	"github.com/dlclark/regexp2"
	"io"
	"regexp"
	"sort"
	"strings"
)

type regexp2Wrapper regexp2.Regexp
type regexpWrapper regexp.Regexp

type positionMapItem struct {
	src, dst int
}
type positionMap []positionMapItem

func (m positionMap) get(src int) int {
	if src == 0 {
		return 0
	}
	res := sort.Search(len(m), func(n int) bool { return m[n].src >= src })
	if res >= len(m) || m[res].src != src {
		panic("index not found")
	}
	return m[res].dst
}

type arrayRuneReader struct {
	runes []rune
	pos   int
}

func (rd *arrayRuneReader) ReadRune() (r rune, size int, err error) {
	if rd.pos < len(rd.runes) {
		r = rd.runes[rd.pos]
		size = 1
		rd.pos++
	} else {
		err = io.EOF
	}
	return
}

type regexpPattern struct {
	src string

	global, ignoreCase, multiline, sticky, unicode bool

	regexpWrapper  *regexpWrapper
	regexp2Wrapper *regexp2Wrapper
}

func compileRegexp2(src string, multiline, ignoreCase bool) (*regexp2Wrapper, error) {
	var opts regexp2.RegexOptions = regexp2.ECMAScript
	if multiline {
		opts |= regexp2.Multiline
	}
	if ignoreCase {
		opts |= regexp2.IgnoreCase
	}
	regexp2Pattern, err1 := regexp2.Compile(src, opts)
	if err1 != nil {
		return nil, fmt.Errorf("Invalid regular expression (regexp2): %s (%v)", src, err1)
	}

	return (*regexp2Wrapper)(regexp2Pattern), nil
}

func (p *regexpPattern) createRegexp2() {
	rx, err := compileRegexp2(p.src, p.multiline, p.ignoreCase)
	if err != nil {
		// At this point the regexp should have been successfully converted to re2, if it fails now, it's a bug.
		panic(err)
	}
	p.regexp2Wrapper = rx
}

func buildUTF8PosMap(s valueString) (positionMap, string) {
	pm := make(positionMap, 0, s.length())
	rd := s.reader(0)
	sPos, utf8Pos := 0, 0
	var sb strings.Builder
	for {
		r, size, err := rd.ReadRune()
		if err == io.EOF {
			break
		}
		if err != nil {
			// the string contains invalid UTF-16, bailing out
			return nil, ""
		}
		utf8Size, _ := sb.WriteRune(r)
		sPos += size
		utf8Pos += utf8Size
		pm = append(pm, positionMapItem{src: utf8Pos, dst: sPos})
	}
	return pm, sb.String()
}

func (p *regexpPattern) findSubmatchIndex(s valueString, start int) []int {
	if p.regexpWrapper == nil {
		return p.regexp2Wrapper.findSubmatchIndex(s, start, p.unicode)
	}
	if start != 0 {
		// Unfortunately Go's regexp library does not allow starting from an arbitrary position.
		// If we just drop the first _start_ characters of the string the assertions (^, $, \b and \B) will not
		// work correctly.
		p.createRegexp2()
		return p.regexp2Wrapper.findSubmatchIndex(s, start, p.unicode)
	}
	return p.regexpWrapper.findSubmatchIndex(s, p.unicode)
}

func (p *regexpPattern) findAllSubmatchIndex(s valueString, limit int) [][]int {
	if p.regexpWrapper == nil {
		return p.regexp2Wrapper.findAllSubmatchIndex(s, limit, p.unicode)
	}
	if s, ok := s.(asciiString); ok {
		return p.regexpWrapper.findAllSubmatchIndex(s.String(), limit)
	}

	if limit == 1 {
		result := p.regexpWrapper.findSubmatchIndex(s, p.unicode)
		if result == nil {
			return nil
		}
		return [][]int{result}
	}

	// Unfortunately Go's regexp library lacks FindAllReaderSubmatchIndex(), so we have to use a UTF-8 string as an
	// input.
	if p.unicode {
		// Try to convert s to UTF-8. If it does not contain any invalid UTF-16 we can do the matching in UTF-8.
		pm, str := buildUTF8PosMap(s)
		if pm != nil {
			res := p.regexpWrapper.findAllSubmatchIndex(str, limit)
			for _, result := range res {
				for i, idx := range result {
					result[i] = pm.get(idx)
				}
			}
			return res
		}
	}

	p.createRegexp2()
	return p.regexp2Wrapper.findAllSubmatchIndex(s, limit, p.unicode)
}

type regexpObject struct {
	baseObject
	pattern *regexpPattern
	source  valueString
}

func (r *regexp2Wrapper) findSubmatchIndex(s valueString, start int, fullUnicode bool) (result []int) {
	if fullUnicode {
		return r.findSubmatchIndexUnicode(s, start)
	}
	return r.findSubmatchIndexUTF16(s, start)
}

func (r *regexp2Wrapper) findSubmatchIndexUTF16(s valueString, start int) (result []int) {
	wrapped := (*regexp2.Regexp)(r)
	match, err := wrapped.FindRunesMatchStartingAt(s.utf16Runes(), start)
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

func (r *regexp2Wrapper) findSubmatchIndexUnicode(s valueString, start int) (result []int) {
	wrapped := (*regexp2.Regexp)(r)
	posMap, runes := buildPosMap(&lenientUtf16Decoder{utf16Reader: s.utf16Reader(0)}, s.length())
	match, err := wrapped.FindRunesMatchStartingAt(runes, start)
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
			result = append(result, posMap[group.Index], posMap[group.Index+group.Length])
		} else {
			result = append(result, -1, 0)
		}
	}
	return
}

func (r *regexp2Wrapper) findAllSubmatchIndexUTF16(s valueString, n int) [][]int {
	wrapped := (*regexp2.Regexp)(r)
	runes := s.utf16Runes()
	match, err := wrapped.FindRunesMatch(runes)
	if err != nil {
		return nil
	}
	if n < 0 {
		n = len(runes) + 1
	}
	results := make([][]int, 0, n)
	for match != nil {
		groups := match.Groups()

		result := make([]int, 0, len(groups)<<1)

		for _, group := range groups {
			if len(group.Captures) > 0 {
				start := group.Index
				end := group.Index + group.Length
				result = append(result, start, end)
			} else {
				result = append(result, -1, 0)
			}
		}

		results = append(results, result)
		n--
		if n <= 0 {
			break
		}
		match, err = wrapped.FindNextMatch(match)
		if err != nil {
			return nil
		}
	}
	return results
}

func buildPosMap(rd io.RuneReader, l int) (posMap []int, runes []rune) {
	posMap = make([]int, 0, l+1)
	curPos := 0
	runes = make([]rune, 0, l)
	for {
		rn, size, err := rd.ReadRune()
		if err != nil {
			break
		}
		runes = append(runes, rn)
		posMap = append(posMap, curPos)
		curPos += size
	}
	posMap = append(posMap, curPos)
	return
}

func (r *regexp2Wrapper) findAllSubmatchIndexUnicode(s unicodeString, n int) [][]int {
	wrapped := (*regexp2.Regexp)(r)
	if n < 0 {
		n = len(s) + 1
	}
	results := make([][]int, 0, n)
	posMap, runes := buildPosMap(&lenientUtf16Decoder{utf16Reader: s.utf16Reader(0)}, s.length())

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

func (r *regexp2Wrapper) findAllSubmatchIndex(s valueString, n int, fullUnicode bool) [][]int {
	switch s := s.(type) {
	case asciiString:
		return r.findAllSubmatchIndexUTF16(s, n)
	case unicodeString:
		if fullUnicode {
			return r.findAllSubmatchIndexUnicode(s, n)
		}
		return r.findAllSubmatchIndexUTF16(s, n)
	default:
		panic("Unsupported string type")
	}
}

func (r *regexpWrapper) findAllSubmatchIndex(s string, limit int) (results [][]int) {
	wrapped := (*regexp.Regexp)(r)
	return wrapped.FindAllStringSubmatchIndex(s, limit)
}

func (r *regexpWrapper) findSubmatchIndex(s valueString, fullUnicode bool) (result []int) {
	wrapped := (*regexp.Regexp)(r)
	if fullUnicode {
		posMap, runes := buildPosMap(&lenientUtf16Decoder{utf16Reader: s.utf16Reader(0)}, s.length())
		res := wrapped.FindReaderSubmatchIndex(&arrayRuneReader{runes: runes})
		for i, item := range res {
			res[i] = posMap[item]
		}
		return res
	}
	return wrapped.FindReaderSubmatchIndex(s.utf16Reader(0))
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
	if !r.pattern.global && !r.pattern.sticky {
		index = 0
	}
	if index >= 0 && index <= int64(target.length()) {
		result = r.pattern.findSubmatchIndex(target, int(index))
	}
	if result == nil || r.pattern.sticky && result[0] != int(index) {
		r.setOwnStr("lastIndex", intToValue(0), true)
		return
	}
	match = true
	if r.pattern.global || r.pattern.sticky {
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

	return r1.val
}

func (r *regexpObject) init() {
	r.baseObject.init()
	r._putProp("lastIndex", intToValue(0), true, false, false)
}
