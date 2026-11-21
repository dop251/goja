package goja

import (
	"errors"
	"hash/maphash"
	"io"
	"math"
	"math/big"
	"reflect"
	"strconv"
	"strings"

	"github.com/dop251/goja/parser"
	"github.com/dop251/goja/unistring"
)

type asciiString string

type asciiRuneReader struct {
	s   asciiString
	pos int
}

func (rr *asciiRuneReader) ReadRune() (r rune, size int, err error) {
	if rr.pos < len(rr.s) {
		r = rune(rr.s[rr.pos])
		size = 1
		rr.pos++
	} else {
		err = io.EOF
	}
	return
}

type asciiUtf16Reader struct {
	s   asciiString
	pos int
}

func (rr *asciiUtf16Reader) readChar() (c uint16, err error) {
	if rr.pos < len(rr.s) {
		c = uint16(rr.s[rr.pos])
		rr.pos++
	} else {
		err = io.EOF
	}
	return
}

func (rr *asciiUtf16Reader) ReadRune() (r rune, size int, err error) {
	if rr.pos < len(rr.s) {
		r = rune(rr.s[rr.pos])
		rr.pos++
		size = 1
	} else {
		err = io.EOF
	}
	return
}

func (s asciiString) Reader() io.RuneReader {
	return &asciiRuneReader{
		s: s,
	}
}

func (s asciiString) utf16Reader() utf16Reader {
	return &asciiUtf16Reader{
		s: s,
	}
}

func (s asciiString) utf16RuneReader() io.RuneReader {
	return &asciiUtf16Reader{
		s: s,
	}
}

func (s asciiString) utf16Runes() []rune {
	runes := make([]rune, len(s))
	for i := 0; i < len(s); i++ {
		runes[i] = rune(s[i])
	}
	return runes
}

func (s asciiString) utf16() []uint16 {
	u := make([]uint16, len(s))
	for i := 0; i < len(s); i++ {
		u[i] = uint16(s[i])
	}
	return u
}

func isRangeErr(err error) bool {
	return errors.Is(err, strconv.ErrRange)
}

func (s asciiString) ToInteger() int64 {
	return trimmedToInteger(strings.TrimSpace(string(s)))
}

func trimmedToInteger(ss string) int64 {
	f, n, isInt := trimmedToNumber(ss)
	if isInt {
		return n
	}
	return floatToIntClip(f)
}

func (s asciiString) toString() String {
	return s
}

func (s asciiString) ToString() Value {
	return s
}

func (s asciiString) String() string {
	return string(s)
}

func (s asciiString) ToFloat() float64 {
	return trimmedToFloat(strings.TrimSpace(string(s)))
}

func trimmedToFloat(s string) float64 {
	f, n, isInt := trimmedToNumber(s)
	if isInt {
		return float64(n)
	}
	return f
}

func (s asciiString) ToBoolean() bool {
	return s != ""
}

func trimmedToNumber(s string) (float64, int64, bool) {
	if s == "" {
		return 0, 0, true
	}
	if s == "Infinity" || s == "+Infinity" {
		return math.Inf(1), 0, false
	}
	if s == "-Infinity" {
		return math.Inf(-1), 0, false
	}
	base := 10
	sign := false
	if len(s) > 2 && s[0] == '0' {
		switch s[1] {
		case 'x', 'X':
			s, base = s[2:], 16
		case 'b', 'B':
			s, base = s[2:], 2
		case 'o', 'O':
			s, base = s[2:], 8
		}
	} else if s[0] == '-' {
		s, sign = s[1:], true
	} else if s[0] == '+' {
		s = s[1:]
	}

	if s == "" {
		return math.NaN(), 0, false
	}

	n, i, err := parser.ParseUint(s, base, false)
	var f float64
	if err == nil {
		if n <= math.MaxInt64 {
			nn := int64(n)
			if sign {
				if n == 0 {
					return negativeZero, 0, false
				}
				nn = -nn
			}
			return 0, nn, true
		}
		f = float64(n)
	} else {
		if base == 10 {
			if errors.Is(err, strconv.ErrSyntax) && i < len(s) {
				if c := s[i]; c != '.' && c != 'e' && c != 'E' {
					return math.NaN(), 0, false
				}
			}
			if strings.ContainsRune(s[i:], '_') {
				return math.NaN(), 0, false
			}
			f, err = strconv.ParseFloat(s, 64)
			if err != nil {
				if !errors.Is(err, strconv.ErrRange) {
					return math.NaN(), 0, false
				}
			}
		} else {
			if errors.Is(err, strconv.ErrRange) {
				f, err = parser.ParseOverflownPow2Radix(n, s[i:], base, false)
				if err != nil {
					return math.NaN(), 0, false
				}
			} else {
				return math.NaN(), 0, false
			}
		}
	}
	if sign {
		f = -f
	}
	if n, ok := floatToInt(f); ok {
		return 0, n, true
	}
	return f, 0, false
}

func (s asciiString) ToNumber() Value {
	f, n, isInt := trimmedToNumber(strings.TrimSpace(string(s)))
	if isInt {
		return intToValue(n)
	}
	return valueFloat(f)
}

func (s asciiString) ToObject(r *Runtime) *Object {
	return r._newString(s, r.getStringPrototype())
}

func (s asciiString) SameAs(other Value) bool {
	return s.StrictEquals(other)
}

func (s asciiString) Equals(other Value) bool {
	if s.StrictEquals(other) {
		return true
	}

	if o, ok := other.(valueInt); ok {
		f, n, isInt := trimmedToNumber(s.toTrimmedUTF8())
		return isInt && n == int64(o) || !isInt && f == float64(o)
	}

	if o, ok := other.(valueFloat); ok {
		return s.ToFloat() == float64(o)
	}

	if o, ok := other.(valueBool); ok {
		f, n, isInt := trimmedToNumber(s.toTrimmedUTF8())
		want := int64(0)
		if o {
			want = 1
		}
		if isInt {
			return n == want
		}
		return f == float64(want)
	}

	if o, ok := other.(*valueBigInt); ok {
		bigInt, err := stringToBigInt(s.toTrimmedUTF8())
		if err != nil {
			return false
		}
		return bigInt.Cmp((*big.Int)(o)) == 0
	}

	if o, ok := other.(*Object); ok {
		return s.Equals(o.toPrimitive())
	}
	return false
}

func (s asciiString) StrictEquals(other Value) bool {
	switch o := other.(type) {
	case asciiString:
		return s == o
	case *importedString:
		if o.u == nil {
			return string(s) == o.s
		}
	case *concatString:
		return s.StrictEquals(o.flatten())
	}
	return false
}

func (s asciiString) baseObject(r *Runtime) *Object {
	ss := r.getStringSingleton()
	ss.value = s
	ss.setLength()
	return ss.val
}

func (s asciiString) hash(hash *maphash.Hash) uint64 {
	_, _ = hash.WriteString(string(s))
	h := hash.Sum64()
	hash.Reset()
	return h
}

func (s asciiString) CharAt(idx int) uint16 {
	return uint16(s[idx])
}

func (s asciiString) Length() int {
	return len(s)
}

func (s asciiString) Concat(other String) String {
	switch o := other.(type) {
	case asciiString:
		totalLen := len(s) + len(o)
		if totalLen > concatThreshold {
			return &concatString{left: s, right: o, length: totalLen}
		}
		return s + o
	case unicodeString:
		totalLen := len(s) + o.Length()
		if totalLen > concatThreshold {
			return &concatString{left: s, right: o, length: totalLen}
		}
		b := make([]uint16, len(s)+len(o))
		b[0] = unistring.BOM
		for i := 0; i < len(s); i++ {
			b[i+1] = uint16(s[i])
		}
		copy(b[len(s)+1:], o[1:])
		return unicodeString(b)
	case *importedString:
		o.ensureScanned()
		if o.u != nil {
			totalLen := len(s) + o.u.Length()
			if totalLen > concatThreshold {
				return &concatString{left: s, right: o.u, length: totalLen}
			}
			b := make([]uint16, len(s)+len(o.u))
			b[0] = unistring.BOM
			for i := 0; i < len(s); i++ {
				b[i+1] = uint16(s[i])
			}
			copy(b[len(s)+1:], o.u[1:])
			return unicodeString(b)
		}
		totalLen := len(s) + len(o.s)
		if totalLen > concatThreshold {
			return &concatString{left: s, right: asciiString(o.s), length: totalLen}
		}
		return s + asciiString(o.s)
	case *concatString:
		return &concatString{left: s, right: o, length: len(s) + o.length}
	default:
		panic(unknownStringTypeErr(other))
	}
}

func (s asciiString) Substring(start, end int) String {
	return s[start:end]
}

func (s asciiString) CompareTo(other String) int {
	a, u := devirtualizeString(other)
	if u != nil {
		return -u.compareToAscii(s)
	}
	return strings.Compare(string(s), string(a))
}

func (s asciiString) index(substr String, start int) int {
	a, u := devirtualizeString(substr)
	if u == nil {
		if start > len(s) {
			return -1
		}
		p := strings.Index(string(s[start:]), string(a))
		if p >= 0 {
			return p + start
		}
	}
	return -1
}

func (s asciiString) lastIndex(substr String, pos int) int {
	a, u := devirtualizeString(substr)
	if u == nil {
		end := pos + len(a)
		var ss string
		if end > len(s) {
			ss = string(s)
		} else {
			ss = string(s[:end])
		}
		return strings.LastIndex(ss, string(a))
	}
	return -1
}

func (s asciiString) toLower() String {
	return asciiString(strings.ToLower(string(s)))
}

func (s asciiString) toUpper() String {
	return asciiString(strings.ToUpper(string(s)))
}

func (s asciiString) toTrimmedUTF8() string {
	return strings.TrimSpace(string(s))
}

func (s asciiString) string() unistring.String {
	return unistring.String(s)
}

func (s asciiString) Export() interface{} {
	return string(s)
}

func (s asciiString) ExportType() reflect.Type {
	return reflectTypeString
}
