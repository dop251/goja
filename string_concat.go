package goja

import (
	"hash/maphash"
	"io"
	"reflect"
	"strings"
	"sync"

	"github.com/dop251/goja/parser"
	"github.com/dop251/goja/unistring"
)

const concatThreshold = 32

type concatString struct {
	left   String
	right  String
	length int

	flatOnce sync.Once
	flat     String
}

func (s *concatString) flatten() String {
	s.flatOnce.Do(func() {
		sb := &StringBuilder{}
		sb.Grow(s.length)
		s.writeTo(sb)
		s.flat = sb.String()
	})
	return s.flat
}

func (s *concatString) writeTo(sb *StringBuilder) {
	stack := make([]String, 0, 64)
	stack = append(stack, s.right, s.left)
	for len(stack) > 0 {
		n := len(stack) - 1
		str := stack[n]
		stack = stack[:n]

		if cs, ok := str.(*concatString); ok {
			if cs.flat != nil {
				sb.WriteString(cs.flat)
			} else {
				stack = append(stack, cs.right, cs.left)
			}
		} else {
			sb.WriteString(str)
		}
	}
}

func (s *concatString) ToInteger() int64 {
	return s.flatten().ToInteger()
}

func (s *concatString) toString() String {
	return s
}

func (s *concatString) string() unistring.String {
	return s.flatten().string()
}

func (s *concatString) ToString() Value {
	return s
}

func (s *concatString) String() string {
	return s.flatten().String()
}

func (s *concatString) ToFloat() float64 {
	return s.flatten().ToFloat()
}

func (s *concatString) ToNumber() Value {
	return s.flatten().ToNumber()
}

func (s *concatString) ToBoolean() bool {
	return s.length > 0
}

func (s *concatString) ToObject(r *Runtime) *Object {
	return r._newString(s, r.getStringPrototype())
}

func (s *concatString) SameAs(other Value) bool {
	return s.StrictEquals(other)
}

func (s *concatString) Equals(other Value) bool {
	if s.StrictEquals(other) {
		return true
	}
	if o, ok := other.(*Object); ok {
		return s.Equals(o.toPrimitive())
	}
	return false
}

func (s *concatString) StrictEquals(other Value) bool {
	if s == other {
		return true
	}
	return s.flatten().StrictEquals(other)
}

func (s *concatString) baseObject(r *Runtime) *Object {
	return s.flatten().baseObject(r)
}

func (s *concatString) Export() interface{} {
	return s.String()
}

func (s *concatString) ExportType() reflect.Type {
	return reflectTypeString
}

func (s *concatString) hash(h *maphash.Hash) uint64 {
	return s.flatten().hash(h)
}

func (s *concatString) CharAt(idx int) uint16 {
	for {
		leftLen := s.left.Length()
		if idx < leftLen {
			if cs, ok := s.left.(*concatString); ok {
				s = cs
				continue
			}
			return s.left.CharAt(idx)
		}
		idx -= leftLen
		if cs, ok := s.right.(*concatString); ok {
			s = cs
			continue
		}
		return s.right.CharAt(idx)
	}
}

func (s *concatString) Length() int {
	return s.length
}

func (s *concatString) Concat(other String) String {
	totalLen := s.length + other.Length()
	if totalLen <= concatThreshold {
		return s.flatten().Concat(other)
	}
	return &concatString{
		left:   s,
		right:  other,
		length: totalLen,
	}
}

func (s *concatString) Substring(start, end int) String {
	return s.flatten().Substring(start, end)
}

func (s *concatString) CompareTo(other String) int {
	return strings.Compare(s.String(), other.String())
}

func (s *concatString) Reader() io.RuneReader {
	return s.flatten().Reader()
}

func (s *concatString) utf16Reader() utf16Reader {
	return s.flatten().utf16Reader()
}

func (s *concatString) utf16RuneReader() io.RuneReader {
	return s.flatten().utf16RuneReader()
}

func (s *concatString) utf16Runes() []rune {
	return s.flatten().utf16Runes()
}

func (s *concatString) index(substr String, start int) int {
	return s.flatten().index(substr, start)
}

func (s *concatString) lastIndex(substr String, pos int) int {
	return s.flatten().lastIndex(substr, pos)
}

func (s *concatString) toLower() String {
	return s.flatten().toLower()
}

func (s *concatString) toUpper() String {
	return s.flatten().toUpper()
}

func (s *concatString) toTrimmedUTF8() string {
	return strings.Trim(s.String(), parser.WhitespaceChars)
}
