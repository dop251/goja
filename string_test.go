package goja

import (
	"strings"
	"testing"
	"unicode/utf16"
)

func TestStringOOBProperties(t *testing.T) {
	const SCRIPT = `
	var string = new String("str");
	
	string[4] = 1;
	string[4];
	`

	testScript(SCRIPT, valueInt(1), t)
}

func TestImportedString(t *testing.T) {
	vm := New()

	testUnaryOp := func(a, expr string, result interface{}, t *testing.T) {
		v, err := vm.RunString("a => " + expr)
		if err != nil {
			t.Fatal(err)
		}
		var fn func(a Value) (Value, error)
		err = vm.ExportTo(v, &fn)
		if err != nil {
			t.Fatal(err)
		}
		for _, aa := range []Value{newStringValue(a), vm.ToValue(a)} {
			res, err := fn(aa)
			if err != nil {
				t.Fatal(err)
			}
			if res.Export() != result {
				t.Fatalf("%s, a:%v(%T). expected: %v, actual: %v", expr, aa, aa, result, res)
			}
		}
	}

	testBinaryOp := func(a, b, expr string, result interface{}, t *testing.T) {
		v, err := vm.RunString("(a, b) => " + expr)
		if err != nil {
			t.Fatal(err)
		}
		var fn func(a, b Value) (Value, error)
		err = vm.ExportTo(v, &fn)
		if err != nil {
			t.Fatal(err)
		}
		for _, aa := range []Value{newStringValue(a), vm.ToValue(a)} {
			for _, bb := range []Value{newStringValue(b), vm.ToValue(b)} {
				res, err := fn(aa, bb)
				if err != nil {
					t.Fatal(err)
				}
				if res.Export() != result {
					t.Fatalf("%s, a:%v(%T), b:%v(%T). expected: %v, actual: %v", expr, aa, aa, bb, bb, result, res)
				}
			}
		}
	}

	strs := []string{"shortAscii", "longlongAscii1234567890123456789", "short юникод", "long юникод 1234567890 юникод \U0001F600", "юникод", "Ascii", "long", "код"}
	indexOfResults := [][]int{
		/*
			const strs = ["shortAscii", "longlongAscii1234567890123456789", "short юникод", "long юникод 1234567890 юникод \u{1F600}", "юникод", "Ascii", "long", "код"];

			strs.forEach(a => {
			    console.log("{", strs.map(b => a.indexOf(b)).join(", "), "},");
			});
		*/
		{0, -1, -1, -1, -1, 5, -1, -1},
		{-1, 0, -1, -1, -1, 8, 0, -1},
		{-1, -1, 0, -1, 6, -1, -1, 9},
		{-1, -1, -1, 0, 5, -1, 0, 8},
		{-1, -1, -1, -1, 0, -1, -1, 3},
		{-1, -1, -1, -1, -1, 0, -1, -1},
		{-1, -1, -1, -1, -1, -1, 0, -1},
		{-1, -1, -1, -1, -1, -1, -1, 0},
	}

	lastIndexOfResults := [][]int{
		/*
			strs.forEach(a => {
			    console.log("{", strs.map(b => a.lastIndexOf(b)).join(", "), "},");
			});
		*/
		{0, -1, -1, -1, -1, 5, -1, -1},
		{-1, 0, -1, -1, -1, 8, 4, -1},
		{-1, -1, 0, -1, 6, -1, -1, 9},
		{-1, -1, -1, 0, 23, -1, 0, 26},
		{-1, -1, -1, -1, 0, -1, -1, 3},
		{-1, -1, -1, -1, -1, 0, -1, -1},
		{-1, -1, -1, -1, -1, -1, 0, -1},
		{-1, -1, -1, -1, -1, -1, -1, 0},
	}

	pad := func(s, p string, n int, start bool) string {
		if n == 0 {
			return s
		}
		if p == "" {
			p = " "
		}
		var b strings.Builder
		ss := utf16.Encode([]rune(s))
		b.Grow(n)
		n -= len(ss)
		if !start {
			b.WriteString(s)
		}
		if n > 0 {
			pp := utf16.Encode([]rune(p))
			for n > 0 {
				if n > len(pp) {
					b.WriteString(p)
					n -= len(pp)
				} else {
					b.WriteString(string(utf16.Decode(pp[:n])))
					n = 0
				}
			}
		}
		if start {
			b.WriteString(s)
		}
		return b.String()
	}

	for i, a := range strs {
		testUnaryOp(a, "JSON.parse(JSON.stringify(a))", a, t)
		testUnaryOp(a, "a.length", int64(len(utf16.Encode([]rune(a)))), t)
		for j, b := range strs {
			testBinaryOp(a, b, "a === b", a == b, t)
			testBinaryOp(a, b, "a == b", a == b, t)
			testBinaryOp(a, b, "a + b", a+b, t)
			testBinaryOp(a, b, "a > b", strings.Compare(a, b) > 0, t)
			testBinaryOp(a, b, "`A${a}B${b}C`", "A"+a+"B"+b+"C", t)
			testBinaryOp(a, b, "a.indexOf(b)", int64(indexOfResults[i][j]), t)
			testBinaryOp(a, b, "a.lastIndexOf(b)", int64(lastIndexOfResults[i][j]), t)
			testBinaryOp(a, b, "a.padStart(32, b)", pad(a, b, 32, true), t)
			testBinaryOp(a, b, "a.padEnd(32, b)", pad(a, b, 32, false), t)
			testBinaryOp(a, b, "a.replace(b, '')", strings.Replace(a, b, "", 1), t)
		}
	}
}

func TestStringFromUTF16(t *testing.T) {
	s := StringFromUTF16([]uint16{})
	if s.Length() != 0 || !s.SameAs(asciiString("")) {
		t.Fatal(s)
	}

	s = StringFromUTF16([]uint16{0xD800})
	if s.Length() != 1 || s.CharAt(0) != 0xD800 {
		t.Fatal(s)
	}

	s = StringFromUTF16([]uint16{'A', 'B'})
	if !s.SameAs(asciiString("AB")) {
		t.Fatal(s)
	}
}

func TestStringBuilder(t *testing.T) {
	t.Run("writeUTF8String-switch", func(t *testing.T) {
		var sb StringBuilder
		sb.WriteUTF8String("Head")
		sb.WriteUTF8String("1ábc")
		if res := sb.String().String(); res != "Head1ábc" {
			t.Fatal(res)
		}
	})
}

func BenchmarkASCIIConcat(b *testing.B) {
	vm := New()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := vm.RunString(`{let result = "ab";
		for (let i = 0 ; i < 10;i++) {
			result += result;
		}}`)
		if err != nil {
			b.Fatalf("Unexpected errors %s", err)
		}
	}
}
