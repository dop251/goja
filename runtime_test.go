package goja

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestGlobalObjectProto(t *testing.T) {
	const SCRIPT = `
	this instanceof Object
	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestArrayProtoProp(t *testing.T) {
	const SCRIPT = `
	Object.defineProperty(Array.prototype, '0', {value: 42, configurable: true, writable: false})
	var a = []
	a[0] = 1
	a[0]
	`

	testScript1(SCRIPT, valueInt(42), t)
}

func TestArrayDelete(t *testing.T) {
	const SCRIPT = `
	var a = [1, 2];
	var deleted = delete a[0];
	var undef = a[0] === undefined;
	var len = a.length;

	deleted && undef && len === 2;
	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestArrayDeleteNonexisting(t *testing.T) {
	const SCRIPT = `
	Array.prototype[0] = 42;
	var a = [];
	delete a[0] && a[0] === 42;
	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestArraySetLength(t *testing.T) {
	const SCRIPT = `
	var a = [1, 2];
	var assert0 = a.length == 2;
	a.length = "1";
	a.length = 1.0;
	a.length = 1;
	var assert1 = a.length == 1;
	a.length = 2;
	var assert2 = a.length == 2;
	assert0 && assert1 && assert2 && a[1] === undefined;

	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestUnicodeString(t *testing.T) {
	const SCRIPT = `
	var s = "Тест";
	s.length === 4 && s[1] === "е";

	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestArrayReverseNonOptimisable(t *testing.T) {
	const SCRIPT = `
	var a = [];
	Object.defineProperty(a, "0", {get: function() {return 42}, set: function(v) {Object.defineProperty(a, "0", {value: v + 1, writable: true, configurable: true})}, configurable: true})
	a[1] = 43;
	a.reverse();

	a.length === 2 && a[0] === 44 && a[1] === 42;
	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestArrayPushNonOptimisable(t *testing.T) {
	const SCRIPT = `
	Object.defineProperty(Object.prototype, "0", {value: 42});
	var a = [];
	var thrown = false;
	try {
		a.push(1);
	} catch (e) {
		thrown = e instanceof TypeError;
	}
	thrown;
	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestArraySetLengthWithPropItems(t *testing.T) {
	const SCRIPT = `
	var a = [1,2,3,4];
	var thrown = false;

	Object.defineProperty(a, "2", {value: 42, configurable: false, writable: false});
	try {
		Object.defineProperty(a, "length", {value: 0, writable: false});
	} catch (e) {
		thrown = e instanceof TypeError;
	}
	thrown && a.length === 3;
	`

	testScript1(SCRIPT, valueTrue, t)
}

func Test2TierHierarchyProp(t *testing.T) {
	const SCRIPT = `
	var a = {};
	Object.defineProperty(a, "test", {
		value: 42,
		writable: false,
		enumerable: false,
		configurable: true
	});
	var b = Object.create(a);
	var c = Object.create(b);
	c.test = 43;
	c.test === 42 && !b.hasOwnProperty("test");

	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestConstStringIter(t *testing.T) {
	const SCRIPT = `

	var count = 0;

	for (var i in "1234") {
    		for (var j in "1234567") {
        		count++
    		}
	}

	count;
	`

	testScript1(SCRIPT, intToValue(28), t)
}

func TestUnicodeConcat(t *testing.T) {
	const SCRIPT = `

	var s = "тест";
	var s1 = "test";
	var s2 = "абвгд";

	s.concat(s1) === "тестtest" && s.concat(s1, s2) === "тестtestабвгд" && s1.concat(s, s2) === "testтестабвгд"
		&& s.concat(s2) === "тестабвгд";

	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestIndexOf(t *testing.T) {
	const SCRIPT = `

	"abc".indexOf("", 4)
	`

	testScript1(SCRIPT, intToValue(3), t)
}

func TestUnicodeIndexOf(t *testing.T) {
	const SCRIPT = `

	"абвгд".indexOf("вг", 1)

	`

	testScript1(SCRIPT, intToValue(2), t)
}

func TestLastIndexOf(t *testing.T) {
	const SCRIPT = `

	"abcabab".lastIndexOf("ab", 3)
	`

	testScript1(SCRIPT, intToValue(3), t)
}

func TestUnicodeLastIndexOf(t *testing.T) {
	const SCRIPT = `
	"абвабаб".lastIndexOf("аб", 3)
	`

	testScript1(SCRIPT, intToValue(3), t)
}

func TestUnicodeLastIndexOf1(t *testing.T) {
	const SCRIPT = `
	"abꞐcde".lastIndexOf("cd");
	`

	testScript1(SCRIPT, intToValue(3), t)
}

func TestNumber(t *testing.T) {
	const SCRIPT = `
	(new Number(100111122133144155)).toString()
	`

	testScript1(SCRIPT, asciiString("100111122133144160"), t)
}

func TestFractionalNumberToStringRadix(t *testing.T) {
	const SCRIPT = `
	(new Number(123.456)).toString(36)
	`

	testScript1(SCRIPT, asciiString("3f.gez4w97ry"), t)
}

func TestSetFunc(t *testing.T) {
	const SCRIPT = `
	sum(40, 2);
	`
	r := New()
	r.Set("sum", func(call FunctionCall) Value {
		return r.ToValue(call.Argument(0).ToInteger() + call.Argument(1).ToInteger())
	})
	v, err := r.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}
	if i := v.ToInteger(); i != 42 {
		t.Fatalf("Expected 42, got: %d", i)
	}
}

func TestObjectGetSet(t *testing.T) {
	const SCRIPT = `
		input.test++;
		input;
	`
	r := New()
	o := r.NewObject()
	o.Set("test", 42)
	r.Set("input", o)

	v, err := r.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}
	if o1, ok := v.(*Object); ok {
		if v1 := o1.Get("test"); v1.Export() != int64(43) {
			t.Fatalf("Unexpected test value: %v (%T)", v1, v1.Export())
		}
	}
}

func TestThrowFromNativeFunc(t *testing.T) {
	const SCRIPT = `
	var thrown;
	try {
		f();
	} catch (e) {
		thrown = e;
	}
	thrown;
	`
	r := New()
	r.Set("f", func(call FunctionCall) Value {
		panic(r.ToValue("testError"))
	})

	v, err := r.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	if !v.Equals(asciiString("testError")) {
		t.Fatalf("Unexpected result: %v", v)
	}
}

func TestSetGoFunc(t *testing.T) {
	const SCRIPT = `
	f(40, 2)
	`
	r := New()
	r.Set("f", func(a, b int) int {
		return a + b
	})

	v, err := r.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	if v.ToInteger() != 42 {
		t.Fatalf("Unexpected result: %v", v)
	}
}

func TestArgsKeys(t *testing.T) {
	const SCRIPT = `
	function testArgs2(x, y, z) {
    		// Properties of the arguments object are enumerable.
    		return Object.keys(arguments);
	}

	testArgs2(1,2).length
	`

	testScript1(SCRIPT, intToValue(2), t)
}

func TestIPowOverflow(t *testing.T) {
	const SCRIPT = `
	Math.pow(65536, 6)
	`

	testScript1(SCRIPT, floatToValue(7.922816251426434e+28), t)
}

func TestIPowZero(t *testing.T) {
	const SCRIPT = `
	Math.pow(0, 0)
	`

	testScript1(SCRIPT, intToValue(1), t)
}

func TestInterrupt(t *testing.T) {
	const SCRIPT = `
	var i = 0;
	for (;;) {
		i++;
	}
	`

	vm := New()
	time.AfterFunc(200*time.Millisecond, func() {
		vm.Interrupt("halt")
	})

	_, err := vm.RunString(SCRIPT)
	if err == nil {
		t.Fatal("Err is nil")
	}
}

func TestRuntime_ExportToSlice(t *testing.T) {
	const SCRIPT = `
	var a = [1, 2, 3];
	a;
	`

	vm := New()
	v, err := vm.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}
	var a []string
	err = vm.ExportTo(v, &a)
	if err != nil {
		t.Fatal(err)
	}
	if l := len(a); l != 3 {
		t.Fatalf("Unexpected len: %d", l)
	}
	if a[0] != "1" || a[1] != "2" || a[2] != "3" {
		t.Fatalf("Unexpected value: %+v", a)
	}
}

func TestRuntime_ExportToMap(t *testing.T) {
	const SCRIPT = `
	var m = {
		"0": 1,
		"1": 2,
		"2": 3,
	}
	m;
	`

	vm := New()
	v, err := vm.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}
	var m map[int]string
	err = vm.ExportTo(v, &m)
	if err != nil {
		t.Fatal(err)
	}
	if l := len(m); l != 3 {
		t.Fatalf("Unexpected len: %d", l)
	}
	if m[0] != "1" || m[1] != "2" || m[2] != "3" {
		t.Fatalf("Unexpected value: %+v", m)
	}
}

func TestRuntime_ExportToMap1(t *testing.T) {
	const SCRIPT = `
	var m = {
		"0": 1,
		"1": 2,
		"2": 3,
	}
	m;
	`

	vm := New()
	v, err := vm.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]string
	err = vm.ExportTo(v, &m)
	if err != nil {
		t.Fatal(err)
	}
	if l := len(m); l != 3 {
		t.Fatalf("Unexpected len: %d", l)
	}
	if m["0"] != "1" || m["1"] != "2" || m["2"] != "3" {
		t.Fatalf("Unexpected value: %+v", m)
	}
}

func TestRuntime_ExportToStruct(t *testing.T) {
	const SCRIPT = `
	var m = {
		Test: 1,
	}
	m;
	`
	vm := New()
	v, err := vm.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	var o testGoReflectMethod_O
	err = vm.ExportTo(v, &o)
	if err != nil {
		t.Fatal(err)
	}

	if o.Test != "1" {
		t.Fatalf("Unexpected value: '%s'", o.Test)
	}

}

func TestRuntime_ExportToFunc(t *testing.T) {
	const SCRIPT = `
	function f(param) {
		return +param + 2;
	}
	`

	vm := New()
	_, err := vm.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	var fn func(string) string
	vm.ExportTo(vm.Get("f"), &fn)

	if res := fn("40"); res != "42" {
		t.Fatalf("Unexpected value: %q", res)
	}
}

func TestRuntime_ExportToFuncThrow(t *testing.T) {
	const SCRIPT = `
	function f(param) {
		throw new Error("testing");
	}
	`

	vm := New()
	_, err := vm.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	var fn func(string) (string, error)
	err = vm.ExportTo(vm.Get("f"), &fn)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := fn("40"); err != nil {
		if ex, ok := err.(*Exception); ok {
			if msg := ex.Error(); msg != "Error: testing at f (<eval>:3:9(4))" {
				t.Fatalf("Msg: %q", msg)
			}
		} else {
			t.Fatalf("Error is not *Exception (%T): %v", err, err)
		}
	} else {
		t.Fatal("Expected error")
	}
}

func TestRuntime_ExportToFuncFail(t *testing.T) {
	const SCRIPT = `
	function f(param) {
		return +param + 2;
	}
	`

	type T struct {
		Field1 int
	}

	var fn func(string) (T, error)

	vm := New()
	_, err := vm.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	err = vm.ExportTo(vm.Get("f"), &fn)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := fn("40"); err == nil {
		t.Fatal("Expected error")
	}
}

func TestRuntime_ExportToCallable(t *testing.T) {
	const SCRIPT = `
	function f(param) {
		return +param + 2;
	}
	`
	vm := New()
	_, err := vm.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	var c Callable
	err = vm.ExportTo(vm.Get("f"), &c)
	if err != nil {
		t.Fatal(err)
	}

	res, err := c(Undefined(), vm.ToValue("40"))
	if err != nil {
		t.Fatal(err)
	} else if !res.StrictEquals(vm.ToValue(42)) {
		t.Fatalf("Unexpected value: %v", res)
	}
}

func TestRuntime_ExportToObject(t *testing.T) {
	const SCRIPT = `
	var o = {"test": 42};
	o;
	`
	vm := New()
	_, err := vm.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	var o *Object
	err = vm.ExportTo(vm.Get("o"), &o)
	if err != nil {
		t.Fatal(err)
	}

	if v := o.Get("test"); !v.StrictEquals(vm.ToValue(42)) {
		t.Fatalf("Unexpected value: %v", v)
	}
}

func TestGoFuncError(t *testing.T) {
	const SCRIPT = `
	try {
		f();
	} catch (e) {
		if (!(e instanceof GoError)) {
			throw(e);
		}
		if (e.value.Error() !== "Test") {
			throw("Unexpected value: " + e.value.Error());
		}
	}
	`

	f := func() error {
		return errors.New("Test")
	}

	vm := New()
	vm.Set("f", f)
	_, err := vm.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}
}

func TestToValueNil(t *testing.T) {
	type T struct{}
	var a *T
	vm := New()

	if v := vm.ToValue(a); !IsNull(v) {
		t.Fatalf("Expected null, got: %v", v)
	}
}

func TestJSONEscape(t *testing.T) {
	const SCRIPT = `
	var a = "\\+1";
	JSON.stringify(a);
	`

	testScript1(SCRIPT, asciiString(`"\\+1"`), t)
}

func TestJSONObjectInArray(t *testing.T) {
	const SCRIPT = `
	var a = "[{\"a\":1},{\"a\":2}]";
	JSON.stringify(JSON.parse(a)) == a;
	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestJSONQuirkyNumbers(t *testing.T) {
	const SCRIPT = `
	var s;
	s = JSON.stringify(NaN);
	if (s != "null") {
		throw new Error("NaN: " + s);
	}

	s = JSON.stringify(Infinity);
	if (s != "null") {
		throw new Error("Infinity: " + s);
	}

	s = JSON.stringify(-Infinity);
	if (s != "null") {
		throw new Error("-Infinity: " + s);
	}

	`

	testScript1(SCRIPT, _undefined, t)
}

func TestJSONNil(t *testing.T) {
	const SCRIPT = `
	JSON.stringify(i);
	`

	vm := New()
	var i interface{}
	vm.Set("i", i)
	ret, err := vm.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	if ret.String() != "null" {
		t.Fatalf("Expected 'null', got: %v", ret)
	}
}

type customJsonEncodable struct{}

func (*customJsonEncodable) JsonEncodable() interface{} {
	return "Test"
}

func TestJsonEncodable(t *testing.T) {
	var s customJsonEncodable

	vm := New()
	vm.Set("s", &s)

	ret, err := vm.RunString("JSON.stringify(s)")
	if err != nil {
		t.Fatal(err)
	}
	if !ret.StrictEquals(vm.ToValue("\"Test\"")) {
		t.Fatalf("Expected \"Test\", got: %v", ret)
	}
}

func TestSortComparatorReturnValues(t *testing.T) {
	const SCRIPT = `
	var a = [];
	for (var i = 0; i < 12; i++) {
	    a[i] = i;
	}

	a.sort(function(x, y) { return y - x });

	for (var i = 0; i < 12; i++) {
	    if (a[i] !== 11-i) {
		throw new Error("Value at index " + i + " is incorrect: " + a[i]);
	    }
	}
	`

	testScript1(SCRIPT, _undefined, t)
}

func TestNilApplyArg(t *testing.T) {
	const SCRIPT = `
	(function x(a, b) {
		return a === undefined && b === 1;
        }).apply(this, [,1])
	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestNilCallArg(t *testing.T) {
	const SCRIPT = `
	"use strict";
	function f(a) {
		return this === undefined && a === undefined;
	}
	`
	vm := New()
	prg := MustCompile("test.js", SCRIPT, false)
	vm.RunProgram(prg)
	if f, ok := AssertFunction(vm.Get("f")); ok {
		v, err := f(nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !v.StrictEquals(valueTrue) {
			t.Fatalf("Unexpected result: %v", v)
		}
	}
}

func TestNullCallArg(t *testing.T) {
	const SCRIPT = `
	f(null);
	`
	vm := New()
	prg := MustCompile("test.js", SCRIPT, false)
	vm.Set("f", func(x *int) bool {
		return x == nil
	})

	v, err := vm.RunProgram(prg)
	if err != nil {
		t.Fatal(err)
	}

	if !v.StrictEquals(valueTrue) {
		t.Fatalf("Unexpected result: %v", v)
	}
}

func TestObjectKeys(t *testing.T) {
	const SCRIPT = `
	var o = { a: 1, b: 2, c: 3, d: 4 };
	o;
	`

	vm := New()
	prg := MustCompile("test.js", SCRIPT, false)

	res, err := vm.RunProgram(prg)
	if err != nil {
		t.Fatal(err)
	}

	if o, ok := res.(*Object); ok {
		keys := o.Keys()
		if !reflect.DeepEqual(keys, []string{"a", "b", "c", "d"}) {
			t.Fatalf("Unexpected keys: %v", keys)
		}
	}
}

func TestReflectCallExtraArgs(t *testing.T) {
	const SCRIPT = `
	f(41, "extra")
	`
	f := func(x int) int {
		return x + 1
	}

	vm := New()
	vm.Set("f", f)

	prg := MustCompile("test.js", SCRIPT, false)

	res, err := vm.RunProgram(prg)
	if err != nil {
		t.Fatal(err)
	}
	if !res.StrictEquals(intToValue(42)) {
		t.Fatalf("Unexpected result: %v", res)
	}
}

func TestReflectCallNotEnoughArgs(t *testing.T) {
	const SCRIPT = `
	f(42)
	`
	vm := New()

	f := func(x, y int, z *int, s string) (int, error) {
		if z != nil {
			return 0, fmt.Errorf("z is not nil")
		}
		if s != "" {
			return 0, fmt.Errorf("s is not \"\"")
		}
		return x + y, nil
	}

	vm.Set("f", f)

	prg := MustCompile("test.js", SCRIPT, false)

	res, err := vm.RunProgram(prg)
	if err != nil {
		t.Fatal(err)
	}
	if !res.StrictEquals(intToValue(42)) {
		t.Fatalf("Unexpected result: %v", res)
	}
}

func TestReflectCallVariadic(t *testing.T) {
	const SCRIPT = `
	var r = f("Hello %s, %d", "test", 42);
	if (r !== "Hello test, 42") {
		throw new Error("test 1 has failed: " + r);
	}

	r = f("Hello %s, %d", ["test", 42]);
	if (r !== "Hello test, 42") {
		throw new Error("test 2 has failed: " + r);
	}

	r = f("Hello %s, %s", "test");
	if (r !== "Hello test, %!s(MISSING)") {
		throw new Error("test 3 has failed: " + r);
	}

	r = f();
	if (r !== "") {
		throw new Error("test 4 has failed: " + r);
	}

	`

	vm := New()
	vm.Set("f", fmt.Sprintf)

	prg := MustCompile("test.js", SCRIPT, false)

	_, err := vm.RunProgram(prg)
	if err != nil {
		t.Fatal(err)
	}
}

func TestReflectNullValueArgument(t *testing.T) {
	rt := New()
	rt.Set("fn", func(v Value) {
		if v == nil {
			t.Error("null becomes nil")
		}
		if !IsNull(v) {
			t.Error("null is not null")
		}
	})
	rt.RunString(`fn(null);`)
}

/*
func TestArrayConcatSparse(t *testing.T) {
function foo(a,b,c)
  {
    arguments[0] = 1; arguments[1] = 'str'; arguments[2] = 2.1;
    if(1 === a && 'str' === b && 2.1 === c)
      return true;
  }


	const SCRIPT = `
	var a1 = [];
	var a2 = [];
	a1[500000] = 1;
	a2[1000000] = 2;
	var a3 = a1.concat(a2);
	a3.length === 1500002 && a3[500000] === 1 && a3[1500001] == 2;
	`

	testScript1(SCRIPT, valueTrue, t)
}
*/

func BenchmarkCallReflect(b *testing.B) {
	vm := New()
	vm.Set("f", func(v Value) {

	})

	prg := MustCompile("test.js", "f(null)", true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vm.RunProgram(prg)
	}
}

func BenchmarkCallNative(b *testing.B) {
	vm := New()
	vm.Set("f", func(call FunctionCall) (ret Value) {
		return
	})

	prg := MustCompile("test.js", "f(null)", true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vm.RunProgram(prg)
	}
}
