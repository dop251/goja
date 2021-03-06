package goja

import (
	"testing"
)

func TestGoSliceReflectBasic(t *testing.T) {
	const SCRIPT = `
	var sum = 0;
	for (var i = 0; i < a.length; i++) {
		sum += a[i];
	}
	sum;
	`
	r := New()
	r.Set("a", []int{1, 2, 3, 4})
	v, err := r.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}
	if i := v.ToInteger(); i != 10 {
		t.Fatalf("Expected 10, got: %d", i)
	}

}

func TestGoSliceReflectIn(t *testing.T) {
	const SCRIPT = `
	var idx = "";
	for (var i in a) {
		idx += i;
	}
	idx;
	`
	r := New()
	r.Set("a", []int{1, 2, 3, 4})
	v, err := r.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}
	if i := v.String(); i != "0123" {
		t.Fatalf("Expected '0123', got: '%s'", i)
	}
}

func TestGoSliceReflectSet(t *testing.T) {
	const SCRIPT = `
	a[0] = 33;
	a[1] = 333;
	a[2] = "42";
	a[3] = {};
	a[4] = 0;
	`
	r := New()
	a := []int8{1, 2, 3, 4}
	r.Set("a", a)
	_, err := r.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	if a[0] != 33 {
		t.Fatalf("a[0] = %d, expected 33", a[0])
	}
	if a[1] != 77 {
		t.Fatalf("a[1] = %d, expected 77", a[1])
	}
	if a[2] != 42 {
		t.Fatalf("a[2] = %d, expected 42", a[2])
	}
	if a[3] != 0 {
		t.Fatalf("a[3] = %d, expected 0", a[3])
	}
}

func TestGoSliceReflectPush(t *testing.T) {

	r := New()

	t.Run("Can push to array by array ptr", func(t *testing.T) {
		a := []int8{1}
		r.Set("a", &a)
		_, err := r.RunString(`a.push (10)`)
		if err != nil {
			t.Fatal(err)
		}

		if a[1] != 10 {
			t.Fatalf("a[1] = %d, expected 10", a[1])
		}
	})

	t.Run("Can push to array by struct ptr", func(t *testing.T) {
		type testStr struct {
			A []int
		}
		a := testStr{
			A: []int{2},
		}

		r.Set("a", &a)
		_, err := r.RunString(`a.A.push (10)`)
		if err != nil {
			t.Fatal(err)
		}

		if a.A[1] != 10 {
			t.Fatalf("a[1] = %v, expected 10", a)
		}
	})

}

func TestGoSliceReflectProtoMethod(t *testing.T) {
	const SCRIPT = `
	a.join(",")
	`

	r := New()
	a := []int8{1, 2, 3, 4}
	r.Set("a", a)
	ret, err := r.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}
	if s := ret.String(); s != "1,2,3,4" {
		t.Fatalf("Unexpected result: '%s'", s)
	}
}

type gosliceReflect_withMethods []interface{}

func (s gosliceReflect_withMethods) Method() bool {
	return true
}

func TestGoSliceReflectMethod(t *testing.T) {
	const SCRIPT = `
	typeof a === "object" && a[0] === 42 && a.Method() === true;
	`

	vm := New()
	a := make(gosliceReflect_withMethods, 1)
	a[0] = 42
	vm.Set("a", a)
	v, err := vm.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}
	if !v.StrictEquals(valueTrue) {
		t.Fatalf("Expected true, got %v", v)
	}

}

func TestGoSliceReflectGetStr(t *testing.T) {
	r := New()
	v := r.ToValue([]string{"test"})
	if o, ok := v.(*Object); ok {
		if e := o.Get("0").Export(); e != "test" {
			t.Fatalf("Unexpected o.Get(\"0\"): %v", e)
		}
	}
}

func TestGoSliceReflectNilObjectIfaceVal(t *testing.T) {
	r := New()
	a := []Value{(*Object)(nil)}
	r.Set("a", a)
	ret, err := r.RunString(`
	""+a[0];
	`)
	if err != nil {
		t.Fatal(err)
	}
	if !asciiString("null").SameAs(ret) {
		t.Fatalf("ret: %v", ret)
	}
}

func TestGoSliceReflectSetLength(t *testing.T) {
	r := New()
	a := []int{1, 2, 3, 4}
	b := []testing.TB{&testing.T{}, &testing.T{}, (*testing.T)(nil)}
	r.Set("a", &a)
	r.Set("b", &b)
	_, err := r.RunString(`
	'use strict';
	a.length = 3;
	if (a.length !== 3) {
		throw new Error("length="+a.length);
	}
	if (a[3] !== undefined) {
		throw new Error("a[3]="+a[3]);
	}
	a.length = 5;
	if (a.length !== 5) {
		throw new Error("a.length="+a.length);
	}
	if (a[3] !== 0) {
		throw new Error("a[3]="+a[3]);
	}
	if (a[4] !== 0) {
		throw new Error("a[4]="+a[4]);
	}

	b.length = 3;
	if (b.length !== 3) {
		throw new Error("b.length="+b.length);
	}
	if (b[3] !== undefined) {
		throw new Error("b[3]="+b[3]);
	}
	b.length = 5;
	if (b.length !== 5) {
		throw new Error("length="+b.length);
	}
	if (b[3] !== null) {
		throw new Error("b[3]="+b[3]);
	}
	if (b[4] !== null) {
		throw new Error("b[4]="+b[4]);
	}
	if (b[2] !== null) {
		throw new Error("b[2]="+b[2]);
	}
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGoSliceReflectProto(t *testing.T) {
	r := New()
	a := []*Object{{}, nil, {}}
	r.Set("a", &a)
	_, err := r.RunString(TESTLIB + `
	var proto = [,2,,4];
	Object.setPrototypeOf(a, proto);
	assert.sameValue(a[1], null, "a[1]");
	assert.sameValue(a[3], 4, "a[3]");
	var desc = Object.getOwnPropertyDescriptor(a, "1");
	assert.sameValue(desc.value, null, "desc.value");
	assert(desc.writable, "writable");
	assert(desc.enumerable, "enumerable");
	assert(!desc.configurable, "configurable");
	var v5;
	Object.defineProperty(proto, "5", {
		set: function(v) {
			v5 = v;
		}
	});
	a[5] = "test";
	assert.sameValue(v5, "test", "v5");
	`)
	if err != nil {
		t.Fatal(err)
	}

}

func TestGoSliceReflectProtoProto(t *testing.T) {
	r := New()
	a := []*Object{{}, nil, {}}
	proto := []*Object{{}, {}, {}, {}}
	r.Set("a", &a)
	r.Set("proto", proto)
	_, err := r.RunString(`
	"use strict";
	var protoproto = {};
	Object.defineProperty(protoproto, "3", {
		value: 42
	});
	Object.setPrototypeOf(proto, protoproto);
	Object.setPrototypeOf(a, proto);
	if (a.hasOwnProperty("3")) {
		throw new Error("a.hasOwnProperty(\"3\")");
	}
	if (a[3] !== null) {
		throw new Error("a[3]="+a[3]);
	}
	a[3] = null;
	if (a[3] !== null) {
		throw new Error("a[3]=" + a[3]);
	}
	`)
	if err != nil {
		t.Fatal(err)
	}

}

func TestGoSliceReflectDelete(t *testing.T) {
	r := New()
	a := []*Object{{}, nil, {}}
	r.Set("a", a)
	v, err := r.RunString(`
	delete a[0] && delete a[1] && delete a[3];
	`)
	if err != nil {
		t.Fatal(err)
	}
	if v != valueTrue {
		t.Fatalf("not true: %v", v)
	}
}

func TestGoSliceReflectPop(t *testing.T) {
	r := New()
	a := []string{"1", "", "3"}
	r.Set("a", &a)
	v, err := r.RunString(`
	a.pop()
	`)
	if err != nil {
		t.Fatal(err)
	}
	if !v.SameAs(asciiString("3")) {
		t.Fatal(v)
	}
}
