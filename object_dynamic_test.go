package goja

import "testing"

type testDynObject struct {
	r *Runtime
	m map[string]Value
}

func (t *testDynObject) Get(key string) Value {
	return t.m[key]
}

func (t *testDynObject) Set(key string, val Value) bool {
	t.m[key] = val
	return true
}

func (t *testDynObject) Has(key string) bool {
	_, exists := t.m[key]
	return exists
}

func (t *testDynObject) Delete(key string) bool {
	delete(t.m, key)
	return true
}

func (t *testDynObject) Keys() []string {
	keys := make([]string, 0, len(t.m))
	for k := range t.m {
		keys = append(keys, k)
	}
	return keys
}

type testDynArray struct {
	r *Runtime
	a []Value
}

func (t *testDynArray) Len() int {
	return len(t.a)
}

func (t *testDynArray) Get(idx int) Value {
	if idx < 0 {
		idx += len(t.a)
	}
	if idx >= 0 && idx < len(t.a) {
		return t.a[idx]
	}
	return nil
}

func (t *testDynArray) expand(newLen int) {
	if newLen > cap(t.a) {
		a := make([]Value, newLen)
		copy(a, t.a)
		t.a = a
	} else {
		t.a = t.a[:newLen]
	}
}

func (t *testDynArray) Set(idx int, val Value) bool {
	if idx < 0 {
		idx += len(t.a)
	}
	if idx < 0 {
		return false
	}
	if idx >= len(t.a) {
		t.expand(idx + 1)
	}
	t.a[idx] = val
	return true
}

func (t *testDynArray) SetLen(i int) bool {
	if i > len(t.a) {
		t.expand(i)
		return true
	}
	if i < 0 {
		return false
	}
	if i < len(t.a) {
		tail := t.a[i:len(t.a)]
		for j := range tail {
			tail[j] = nil
		}
		t.a = t.a[:i]
	}
	return true
}

func TestDynamicObject(t *testing.T) {
	vm := New()
	dynObj := &testDynObject{
		r: vm,
		m: make(map[string]Value),
	}
	o := vm.NewDynamicObject(dynObj)
	vm.Set("o", o)
	_, err := vm.RunString(TESTLIBX + `
	assert(o instanceof Object, "instanceof Object");
	assert(o === o, "self equality");
	assert(o !== {}, "non-equality");

	o.test = 42;
	assert("test" in o, "'test' in o");
	assert(deepEqual(Object.getOwnPropertyDescriptor(o, "test"), {value: 42, writable: true, enumerable: true, configurable: true}), "prop desc");

	assert.throws(TypeError, function() {
		"use strict";
		Object.defineProperty(o, "test1", {value: 0, writable: false, enumerable: false, configurable: true});
	}, "define prop");

	var keys = [];
	for (var key in o) {
		keys.push(key);
	}
	assert(compareArray(keys, ["test"]), "for-in");

	assert(delete o.test, "delete");
	assert(!("test" in o), "'test' in o after delete");

	assert("__proto__" in o, "__proto__ in o");
	assert.sameValue(o.__proto__, Object.prototype, "__proto__");
	o.__proto__ = null;
	assert(!("__proto__" in o), "__proto__ in o after setting to null");
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDynamicObjectCustomProto(t *testing.T) {
	vm := New()
	m := make(map[string]Value)
	dynObj := &testDynObject{
		r: vm,
		m: m,
	}
	o := vm.NewDynamicObject(dynObj)
	vm.Set("o", o)
	_, err := vm.RunString(TESTLIB + `
	var proto = {
		valueOf: function() {
			return this.num;
		}
	};
	proto[Symbol.toStringTag] = "GoObject";
	Object.setPrototypeOf(o, proto);
	o.num = 41;
	assert(o instanceof Object, "instanceof");
	assert.sameValue(o+1, 42);
	assert.sameValue(o.toString(), "[object GoObject]");
	`)
	if err != nil {
		t.Fatal(err)
	}
	if v := m["num"]; v.Export() != int64(41) {
		t.Fatal(v)
	}
}

func TestDynamicArray(t *testing.T) {
	vm := New()
	dynObj := &testDynArray{
		r: vm,
	}
	a := vm.NewDynamicArray(dynObj)
	vm.Set("a", a)
	_, err := vm.RunString(TESTLIBX + `
	assert(a instanceof Array, "instanceof Array");
	assert(a instanceof Object, "instanceof Object");
	assert(a === a, "self equality");
	assert(a !== [], "non-equality");
	assert(Array.isArray(a), "isArray()");
	assert("length" in a, "length in a");
	assert.sameValue(a.length, 0, "len == 0");
	assert.sameValue(a[0], undefined, "a[0] (1)");

	a[0] = 0;
	assert.sameValue(a[0], 0, "a[0] (2)");
	assert.sameValue(a.length, 1, "length");
	assert(deepEqual(Object.getOwnPropertyDescriptor(a, 0), {value: 0, writable: true, enumerable: true, configurable: true}), "prop desc");
	assert(deepEqual(Object.getOwnPropertyDescriptor(a, "length"), {value: 1, writable: true, enumerable: false, configurable: false}), "length prop desc");

	assert("__proto__" in a, "__proto__ in a");
	assert.sameValue(a.__proto__, Array.prototype, "__proto__");

	assert(compareArray(Object.keys(a), ["0"]), "Object.keys()");
	assert(compareArray(Reflect.ownKeys(a), ["0", "length"]), "Reflect.ownKeys()");

	a.length = 2;
	assert.sameValue(a.length, 2, "length after grow");
	assert.sameValue(a[1], undefined, "a[1]");

	a[1] = 1;
	assert.sameValue(a[1], 1, "a[1] after set");
	a.length = 1;
	assert.sameValue(a.length, 1, "length after shrink");
	assert.sameValue(a[1], undefined, "a[1] after shrink");
	a.length = 2;
	assert.sameValue(a.length, 2, "length after shrink and grow");
	assert.sameValue(a[1], undefined, "a[1] after grow");

	a[0] = 3; a[1] = 1; a[2] = 2;
	assert.sameValue(a.length, 3);
	var keys = [];
	for (var key in a) {
		keys.push(key);
	}
	assert(compareArray(keys, ["0","1","2"]), "for-in");

	var vals = [];
	for (var val of a) {
		vals.push(val);
	}
	assert(compareArray(vals, [3,1,2]), "for-of");

	a.sort();
	assert(compareArray(a, [1,2,3]), "sort: "+a);

	assert.sameValue(a[-1], 3);
	assert.sameValue(a[-4], undefined);

	assert.throws(TypeError, function() {
		"use strict";
		delete a.length;
	}, "delete length");

	assert.throws(TypeError, function() {
		"use strict";
		a.test = true;
	}, "set string prop");

	assert.throws(TypeError, function() {
		"use strict";
		Object.defineProperty(a, 0, {value: 0, writable: false, enumerable: false, configurable: true});
	}, "define prop");

	`)

	if err != nil {
		t.Fatal(err)
	}
}
