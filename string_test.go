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

func TestUnicodeRepeat(t *testing.T) {
	testCases := []struct {
		name     string
		script   string
		expected string
	}{
		{
			name: "Unicode character only",
			script: `
				var str = "★";
				str.repeat(3);
			`,
			expected: "★★★",
		},
		{
			name: "Mixed unicode",
			script: `
				var str = "a★b★c";
				str.repeat(2);
			`,
			expected: "a★b★ca★b★c",
		},
		{
			name: "Single unicode char repeated once",
			script: `
				var str = "★";
				str.repeat(1);
			`,
			expected: "★",
		},
		{
			name: "Unicode with surrogate pairs",
			script: `
				var str = "𝐀𝐁";
				str.repeat(2);
			`,
			expected: "𝐀𝐁𝐀𝐁",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vm := New()
			v, err := vm.RunString(tc.script)
			if err != nil {
				t.Fatal(err)
			}
			result := v.String()
			if result != tc.expected {
				t.Fatalf("Expected '%s' but got '%s'. Length expected: %d, got: %d",
					tc.expected, result, len([]rune(tc.expected)), len([]rune(result)))
			}
		})
	}
}

func TestImportedString_CompareTo(t *testing.T) {
	vm := New()
	s1 := StringFromUTF16([]uint16{0xFFFF})
	s2 := vm.ToValue("\U00010000") //  "equivalent to "\uD800\uDC00"
	if s2.(String).CompareTo(s1) >= 0 {
		t.Fatal("Imported string does did not compare by code units")
	}
}

func TestStringCaseConversionLoneSurrogates(t *testing.T) {
	tests := []struct {
		name     string
		input    []uint16
		expLower []uint16
		expUpper []uint16
	}{
		{
			name:     "lone high surrogate",
			input:    []uint16{0xD800},
			expLower: []uint16{0xD800},
			expUpper: []uint16{0xD800},
		},
		{
			name:     "lone low surrogate",
			input:    []uint16{0xDC00},
			expLower: []uint16{0xDC00},
			expUpper: []uint16{0xDC00},
		},
		{
			name:     "consecutive lone high surrogates",
			input:    []uint16{0xD800, 0xD801},
			expLower: []uint16{0xD800, 0xD801},
			expUpper: []uint16{0xD800, 0xD801},
		},
		{
			name:     "consecutive lone low surrogates",
			input:    []uint16{0xDC00, 0xDC01},
			expLower: []uint16{0xDC00, 0xDC01},
			expUpper: []uint16{0xDC00, 0xDC01},
		},
		{
			name:     "inverted surrogates",
			input:    []uint16{0xDC00, 0xD800},
			expLower: []uint16{0xDC00, 0xD800},
			expUpper: []uint16{0xDC00, 0xD800},
		},
		{
			name:     "mixed ASCII and lone surrogates",
			input:    []uint16{'A', 0xD800, 'b', 0xDC00, 'C'},
			expLower: []uint16{'a', 0xD800, 'b', 0xDC00, 'c'},
			expUpper: []uint16{'A', 0xD800, 'B', 0xDC00, 'C'},
		},
		{
			name:     "valid surrogate pair surrounded by lone surrogates",
			input:    []uint16{0xD800, 0xD83D, 0xDE00, 0xDFFF},
			expLower: []uint16{0xD800, 0xD83D, 0xDE00, 0xDFFF},
			expUpper: []uint16{0xD800, 0xD83D, 0xDE00, 0xDFFF},
		},
		{
			name: "supplementary casing with lone surrogates",
			// U+10400 (Deseret capital long I) -> U+10428 (Deseret small long I)
			input:    []uint16{0xD800, 0xD801, 0xDC00, 0xDC00},
			expLower: []uint16{0xD800, 0xD801, 0xDC28, 0xDC00},
			expUpper: []uint16{0xD800, 0xD801, 0xDC00, 0xDC00},
		},
	}

	assertCodeUnits := func(t *testing.T, actual String, expected []uint16) {
		t.Helper()
		if actual.Length() != len(expected) {
			t.Fatalf("length mismatch: got %d, want %d", actual.Length(), len(expected))
		}
		for i, exp := range expected {
			if actual.CharAt(i) != exp {
				t.Fatalf("at index %d: got 0x%04X, want 0x%04X", i, actual.CharAt(i), exp)
			}
		}
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := StringFromUTF16(tc.input)
			assertCodeUnits(t, s.toLower(), tc.expLower)
			assertCodeUnits(t, s.toUpper(), tc.expUpper)
		})
	}

	t.Run("JS runtime evaluation", func(t *testing.T) {
		vm := New()
		scripts := []string{
			`"\uD800".toLowerCase() === "\uD800"`,
			`"\uD800".toUpperCase() === "\uD800"`,
			`"\uDC00".toLowerCase() === "\uDC00"`,
			`"\uDC00".toUpperCase() === "\uDC00"`,
			`"\uD800\uD800".toLowerCase() === "\uD800\uD800"`,
			`"\uDC00\uD800".toUpperCase() === "\uDC00\uD800"`,
			`"Hello \uD800 World".toLowerCase().charCodeAt(6) === 0xD800`,
			`"Hello \uD800 World".toUpperCase().charCodeAt(6) === 0xD800`,
			`"Hello \uDC00 World".toLowerCase().charCodeAt(6) === 0xDC00`,
			`"Hello \uDC00 World".toUpperCase().charCodeAt(6) === 0xDC00`,
			`"ABC\uD800def".toLowerCase() === "abc\uD800def"`,
			`"ABC\uD800def".toUpperCase() === "ABC\uD800DEF"`,
		}
		for _, script := range scripts {
			v, err := vm.RunString(script)
			if err != nil {
				t.Fatalf("script %q failed: %v", script, err)
			}
			if !v.ToBoolean() {
				t.Fatalf("script %q evaluated to false", script)
			}
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

func BenchmarkCase(b *testing.B) {
	benchmarks := []struct {
		name string
		s    String
	}{
		{
			name: "ASCII_Short",
			s:    newStringValue("The quick brown fox jumps over the lazy dog."),
		},
		{
			name: "BMP_NoSurrogates_Short",
			s:    newStringValue("Привет, Мир! Καλημέρα κόσμε!"),
		},
		{
			name: "BMP_NoSurrogates_Long",
			s:    newStringValue(strings.Repeat("Привет, Мир! Καλημέρα κόσμε! ", 20)),
		},
		{
			name: "ValidSurrogates",
			s:    newStringValue("Hello 🌍 World 🚀 Deseret: 𐐀𐐁𐐂"),
		},
		{
			name: "LoneSurrogates_Short",
			s:    StringFromUTF16([]uint16{'H', 'e', 'l', 'l', 'o', ' ', 0xD800, ' ', 'W', 'o', 'r', 'l', 'd', ' ', 0xDC00}),
		},
		{
			name: "LoneSurrogates_Dense",
			s:    StringFromUTF16([]uint16{0xD800, 'a', 0xDC00, 'b', 0xD801, 'c', 0xDC01}),
		},
	}

	for _, bm := range benchmarks {
		b.Run("ToLower/"+bm.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = bm.s.toLower()
			}
		})
		b.Run("ToUpper/"+bm.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = bm.s.toUpper()
			}
		})
	}
}
