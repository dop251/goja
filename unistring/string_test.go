package unistring

import "testing"

func TestString_AsUtf16(t *testing.T) {
	const str = "más"
	s := NewFromString(str)

	if b := s.AsUtf16(); len(b) != 4 || b[0] != BOM {
		t.Fatal(b)
	}

	if s.String() != str {
		t.Fatal(s)
	}
}
