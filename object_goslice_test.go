package goja

import (
	"testing"
)

func TestGoSliceBasic(t *testing.T) {
	const SCRIPT = `
	var sum = 0;
	for (var i = 0; i < a.length; i++) {
		sum += a[i];
	}
	sum;
	`
	r := New()
	r.Set("a", []interface{}{1, 2, 3, 4})
	v, err := r.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}
	if i := v.ToInteger(); i != 10 {
		t.Fatalf("Expected 10, got: %d", i)
	}
}

func TestGoSliceIn(t *testing.T) {
	const SCRIPT = `
	var idx = "";
	for (var i in a) {
		idx += i;
	}
	idx;
	`
	r := New()
	r.Set("a", []interface{}{1, 2, 3, 4})
	v, err := r.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}
	if i := v.String(); i != "0123" {
		t.Fatalf("Expected '0123', got: '%s'", i)
	}
}

func TestGoSliceExpand(t *testing.T) {
	const SCRIPT = `
	var l = a.length;
	for (var i = 0; i < l; i++) {
		a[l + i] = a[i] * 2;
	}

	var sum = 0;
	for (var i = 0; i < a.length; i++) {
		sum += a[i];
	}
	sum;
	`
	r := New()
	a := []interface{}{int64(1), int64(2), int64(3), int64(4)}
	r.Set("a", &a)
	v, err := r.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}
	sum := int64(0)
	for _, v := range a {
		sum += v.(int64)
	}
	if i := v.ToInteger(); i != sum {
		t.Fatalf("Expected %d, got: %d", sum, i)
	}
}

func TestGoSliceProtoMethod(t *testing.T) {
	const SCRIPT = `
	a.join(",")
	`

	r := New()
	a := []interface{}{1, 2, 3, 4}
	r.Set("a", a)
	ret, err := r.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}
	if s := ret.String(); s != "1,2,3,4" {
		t.Fatalf("Unexpected result: '%s'", s)
	}
}

func TestGoSliceSetLength(t *testing.T) {
	r := New()
	a := []interface{}{1, 2, 3, 4}
	r.Set("a", &a)
	_, err := r.RunString(`
	'use strict';
	a.length = 3;
	if (a.length !== 3) {
		throw new Error("length="+a.length);
	}
	if (a[3] !== undefined) {
		throw new Error("a[3](1)="+a[3]);
	}
	a.length = 5;
	if (a.length !== 5) {
		throw new Error("length="+a.length);
	}
	if (a[3] !== null) {
		throw new Error("a[3](2)="+a[3]);
	}
	if (a[4] !== null) {
		throw new Error("a[4]="+a[4]);
	}
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGoSliceProto(t *testing.T) {
	r := New()
	a := []interface{}{1, nil, 3}
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

func TestGoSliceProtoProto(t *testing.T) {
	r := New()
	a := []interface{}{1, nil, 3}
	proto := []interface{}{1, 2, 3, 4}
	r.Set("a", &a)
	r.Set("proto", proto)
	_, err := r.RunString(`
	"use strict";
	var protoproto = Object.create(null);
	Object.defineProperty(protoproto, "3", {
		value: 42
	});
	Object.setPrototypeOf(proto, protoproto);
	Object.setPrototypeOf(a, proto);
	a[3] = 11;
	if (a[3] !== 11) {
		throw new Error("a[3]=" + a[3]);
	}
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGoSliceDelete(t *testing.T) {
	r := New()
	a := []interface{}{1, nil, 3}
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

func TestGoSlicePop(t *testing.T) {
	r := New()
	a := []interface{}{1, nil, 3}
	r.Set("a", &a)
	v, err := r.RunString(`
	a.pop()
	`)
	if err != nil {
		t.Fatal(err)
	}
	if !v.SameAs(intToValue(3)) {
		t.Fatal(v)
	}
}

func TestGoSliceShift(t *testing.T) {
	r := New()
	a := []interface{}{1, nil, 3}
	r.Set("a", &a)
	v, err := r.RunString(`
	a.shift()
	`)
	if err != nil {
		t.Fatal(err)
	}
	if !v.SameAs(intToValue(1)) {
		t.Fatal(v)
	}
}
