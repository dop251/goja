package goja

import (
	"testing"
)

func TestGoMapReflectGetSet(t *testing.T) {
	const SCRIPT = `
	m.c = m.a + m.b;
	`

	vm := New()
	m := map[string]string{
		"a": "4",
		"b": "2",
	}
	vm.Set("m", m)

	_, err := vm.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	if c := m["c"]; c != "42" {
		t.Fatalf("Unexpected value: '%s'", c)
	}
}

func TestGoMapReflectIntKey(t *testing.T) {
	const SCRIPT = `
	m[2] = m[0] + m[1];
	`

	vm := New()
	m := map[int]int{
		0: 40,
		1: 2,
	}
	vm.Set("m", m)

	_, err := vm.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	if c := m[2]; c != 42 {
		t.Fatalf("Unexpected value: '%d'", c)
	}
}

func TestGoMapReflectDelete(t *testing.T) {
	const SCRIPT = `
	delete m.a;
	`

	vm := New()
	m := map[string]string{
		"a": "4",
		"b": "2",
	}
	vm.Set("m", m)

	_, err := vm.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	if _, exists := m["a"]; exists {
		t.Fatal("a still exists")
	}

	if b := m["b"]; b != "2" {
		t.Fatalf("Unexpected b: '%s'", b)
	}
}

func TestGoMapReflectJSON(t *testing.T) {
	const SCRIPT = `
	function f(m) {
		return JSON.stringify(m);
	}
	`

	vm := New()
	m := map[string]string{
		"t": "42",
	}
	_, err := vm.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}
	f := vm.Get("f")
	if call, ok := AssertFunction(f); ok {
		v, err := call(nil, ([]Value{vm.ToValue(m)})...)
		if err != nil {
			t.Fatal(err)
		}
		if !v.StrictEquals(asciiString(`{"t":"42"}`)) {
			t.Fatalf("Unexpected value: %v", v)
		}
	} else {
		t.Fatalf("Not a function: %v", f)
	}
}

func TestGoMapReflectProto(t *testing.T) {
	const SCRIPT = `
	m.hasOwnProperty("t");
	`

	vm := New()
	m := map[string]string{
		"t": "42",
	}
	vm.Set("m", m)
	v, err := vm.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}
	if !v.StrictEquals(valueTrue) {
		t.Fatalf("Expected true, got %v", v)
	}
}

type gomapReflect_noMethods map[string]interface{}
type gomapReflect_withMethods map[string]interface{}

func (m gomapReflect_withMethods) Method() bool {
	return true
}

func TestGoMapReflectNoMethods(t *testing.T) {
	const SCRIPT = `
	typeof m === "object" && m.hasOwnProperty("t") && m.t === 42;
	`

	vm := New()
	m := make(gomapReflect_noMethods)
	m["t"] = 42
	vm.Set("m", m)
	v, err := vm.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}
	if !v.StrictEquals(valueTrue) {
		t.Fatalf("Expected true, got %v", v)
	}

}

func TestGoMapReflectWithMethods(t *testing.T) {
	const SCRIPT = `
	typeof m === "object" && !m.hasOwnProperty("t") && m.hasOwnProperty("Method") && m.Method();
	`

	vm := New()
	m := make(gomapReflect_withMethods)
	m["t"] = 42
	vm.Set("m", m)
	v, err := vm.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}
	if !v.StrictEquals(valueTrue) {
		t.Fatalf("Expected true, got %v", v)
	}

}

func TestGoMapReflectWithProto(t *testing.T) {
	vm := New()
	m := map[string]string{
		"t": "42",
	}
	vm.Set("m", m)
	_, err := vm.RunString(TESTLIB + `
	(function() {
	'use strict';
	var proto = {};
	var getterAllowed = false;
	var setterAllowed = false;
	var tHolder = "proto t";
	Object.defineProperty(proto, "t", {
		get: function() {
			if (!getterAllowed) throw new Error("getter is called");
			return tHolder;
		},
		set: function(v) {
			if (!setterAllowed) throw new Error("setter is called");
			tHolder = v;
		}
	});
	var t1Holder;
	Object.defineProperty(proto, "t1", {
		get: function() {
			return t1Holder;
		},
		set: function(v) {
			t1Holder = v;
		}
	});
	Object.setPrototypeOf(m, proto);
	assert.sameValue(m.t, "42");
	m.t = 43;
	assert.sameValue(m.t, "43");
	t1Holder = "test";
	assert.sameValue(m.t1, "test");
	m.t1 = "test1";
	assert.sameValue(m.t1, "test1");
	delete m.t;
	getterAllowed = true;
	assert.sameValue(m.t, "proto t", "after delete");
	setterAllowed = true;
	m.t = true;
	assert.sameValue(m.t, true, "m.t === true");
	assert.sameValue(tHolder, true, "tHolder === true");
	Object.preventExtensions(m);
	assert.throws(TypeError, function() {
		m.t2 = 1;
	});
	m.t1 = "test2";
	assert.sameValue(m.t1, "test2");
	})();
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGoMapReflectProtoProp(t *testing.T) {
	const SCRIPT = `
	(function() {
	"use strict";
	var proto = {};
	Object.defineProperty(proto, "ro", {value: 42});
	Object.setPrototypeOf(m, proto);
	assert.throws(TypeError, function() {
		m.ro = 43;
	});
	Object.defineProperty(m, "ro", {value: 43});
	assert.sameValue(m.ro, "43");
	})();
	`

	r := New()
	r.Set("m", map[string]string{})
	_, err := r.RunString(TESTLIB + SCRIPT)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGoMapReflectUnicode(t *testing.T) {
	const SCRIPT = `
	Object.setPrototypeOf(m, s);
	if (m.Тест !== "passed") {
		throw new Error("m.Тест: " + m.Тест);
	}
	m["é"];
	`
	type S struct {
		Тест string
	}
	vm := New()
	m := map[string]int{
		"é": 42,
	}
	s := S{
		Тест: "passed",
	}
	vm.Set("m", m)
	vm.Set("s", &s)
	res, err := vm.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}
	if res == nil || !res.StrictEquals(valueInt(42)) {
		t.Fatalf("Unexpected value: %v", res)
	}
}
