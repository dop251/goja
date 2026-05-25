package goja

import (
	"testing"
)

func BenchmarkStringConcatLoop(b *testing.B) {
	vm := New()
	for i := 0; i < b.N; i++ {
		_, err := vm.RunString(`
			var s = "";
			for (var i = 0; i < 10000; i++) {
				s += "x";
			}
			s;
		`)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStringConcatLoopLarge(b *testing.B) {
	vm := New()
	for i := 0; i < b.N; i++ {
		_, err := vm.RunString(`
			var s = "";
			for (var i = 0; i < 100000; i++) {
				s += "x";
			}
			s;
		`)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStringConcatLoopUnicode(b *testing.B) {
	vm := New()
	for i := 0; i < b.N; i++ {
		_, err := vm.RunString(`
			var s = "";
			for (var i = 0; i < 10000; i++) {
				s += "\u4e2d";
			}
			s;
		`)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestConcatStringBasic(t *testing.T) {
	vm := New()
	v, err := vm.RunString(`
		var s = "hello";
		s += " ";
		s += "world";
		s;
	`)
	if err != nil {
		t.Fatal(err)
	}
	if v.String() != "hello world" {
		t.Fatalf("unexpected result: %s", v.String())
	}
}

func TestConcatStringLength(t *testing.T) {
	vm := New()
	v, err := vm.RunString(`
		var s = "";
		for (var i = 0; i < 100; i++) {
			s += "ab";
		}
		s.length;
	`)
	if err != nil {
		t.Fatal(err)
	}
	if v.ToInteger() != 200 {
		t.Fatalf("unexpected length: %d", v.ToInteger())
	}
}

func TestConcatStringCharAt(t *testing.T) {
	vm := New()
	v, err := vm.RunString(`
		var s = "a";
		for (var i = 0; i < 100; i++) {
			s += "b";
		}
		s.charAt(50);
	`)
	if err != nil {
		t.Fatal(err)
	}
	if v.String() != "b" {
		t.Fatalf("unexpected char: %s", v.String())
	}
}

func TestConcatStringIndexOf(t *testing.T) {
	vm := New()
	v, err := vm.RunString(`
		var s = "hello";
		for (var i = 0; i < 100; i++) {
			s += "x";
		}
		s.indexOf("xx");
	`)
	if err != nil {
		t.Fatal(err)
	}
	if v.ToInteger() != 5 {
		t.Fatalf("unexpected index: %d", v.ToInteger())
	}
}

func TestConcatStringEquality(t *testing.T) {
	vm := New()
	v, err := vm.RunString(`
		var s1 = "";
		for (var i = 0; i < 100; i++) {
			s1 += "x";
		}
		var s2 = new Array(101).join("x");
		s1 === s2;
	`)
	if err != nil {
		t.Fatal(err)
	}
	if !v.ToBoolean() {
		t.Fatal("expected equal strings")
	}
}
