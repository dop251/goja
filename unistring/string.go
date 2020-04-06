package unistring

import (
	"reflect"
	"unicode/utf16"
	"unicode/utf8"
	"unsafe"
)

const (
	BOM = 0xFEFF
)

type String string

func NewFromString(s string) String {
	ascii := true
	size := 0
	for _, c := range s {
		if c >= utf8.RuneSelf {
			ascii = false
			if c > 0xFFFF {
				size++
			}
		}
		size++
	}
	if ascii {
		return String(s)
	}
	b := make([]uint16, size+1)
	b[0] = BOM
	i := 1
	for _, c := range s {
		if c <= 0xFFFF {
			b[i] = uint16(c)
		} else {
			first, second := utf16.EncodeRune(c)
			b[i] = uint16(first)
			i++
			b[i] = uint16(second)
		}
		i++
	}
	return FromUtf16(b)
}

func NewFromRunes(s []rune) String {
	ascii := true
	size := 0
	for _, c := range s {
		if c >= utf8.RuneSelf {
			ascii = false
			if c > 0xFFFF {
				size++
			}
		}
		size++
	}
	if ascii {
		return String(s)
	}
	b := make([]uint16, size+1)
	b[0] = BOM
	i := 1
	for _, c := range s {
		if c <= 0xFFFF {
			b[i] = uint16(c)
		} else {
			first, second := utf16.EncodeRune(c)
			b[i] = uint16(first)
			i++
			b[i] = uint16(second)
		}
		i++
	}
	return FromUtf16(b)
}

func FromUtf16(b []uint16) String {
	var str string
	hdr := (*reflect.StringHeader)(unsafe.Pointer(&str))
	hdr.Data = uintptr(unsafe.Pointer(&b[0]))
	hdr.Len = len(b) * 2

	return String(str)
}

func (s String) String() string {
	if b := s.AsUtf16(); b != nil {
		return string(utf16.Decode(b[1:]))
	}

	return string(s)
}

func (s String) AsUtf16() []uint16 {
	if len(s) < 4 || len(s)&1 != 0 {
		return nil
	}
	l := len(s) / 2
	raw := string(s)
	hdr := (*reflect.StringHeader)(unsafe.Pointer(&raw))
	a := *(*[]uint16)(unsafe.Pointer(&reflect.SliceHeader{
		Data: hdr.Data,
		Len:  l,
		Cap:  l,
	}))
	if a[0] == BOM {
		return a
	}

	return nil
}
