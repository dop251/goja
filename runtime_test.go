package goja

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/dop251/goja/parser"
)

func TestGlobalObjectProto(t *testing.T) {
	const SCRIPT = `
	this instanceof Object
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
	"абвгд".indexOf("вг", 1) === 2 && '中国'.indexOf('国') === 1
	`

	testScript1(SCRIPT, valueTrue, t)
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

func TestNumberFormatRounding(t *testing.T) {
	const SCRIPT = `
	assert.sameValue((123.456).toExponential(undefined), "1.23456e+2", "undefined");
	assert.sameValue((0.000001).toPrecision(2), "0.0000010")
	assert.sameValue((-7).toPrecision(1), "-7");
	assert.sameValue((-42).toPrecision(1), "-4e+1");
	assert.sameValue((0.000001).toPrecision(1), "0.000001");
	assert.sameValue((123.456).toPrecision(1), "1e+2", "1");
	assert.sameValue((123.456).toPrecision(2), "1.2e+2", "2");

	var n = new Number("0.000000000000000000001"); // 1e-21
	assert.sameValue((n).toPrecision(1), "1e-21");
	assert.sameValue((25).toExponential(0), "3e+1");
	assert.sameValue((-25).toExponential(0), "-3e+1");
	assert.sameValue((12345).toExponential(3), "1.235e+4");
	assert.sameValue((25.5).toFixed(0), "26");
	assert.sameValue((-25.5).toFixed(0), "-26");
	assert.sameValue((99.9).toFixed(0), "100");
	assert.sameValue((99.99).toFixed(1), "100.0");
	`
	testScript1(TESTLIB+SCRIPT, _undefined, t)
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

func TestRuntime_ExportToNumbers(t *testing.T) {
	vm := New()
	t.Run("int8/no overflow", func(t *testing.T) {
		var i8 int8
		err := vm.ExportTo(vm.ToValue(-123), &i8)
		if err != nil {
			t.Fatal(err)
		}
		if i8 != -123 {
			t.Fatalf("i8: %d", i8)
		}
	})

	t.Run("int8/overflow", func(t *testing.T) {
		var i8 int8
		err := vm.ExportTo(vm.ToValue(333), &i8)
		if err != nil {
			t.Fatal(err)
		}
		if i8 != 77 {
			t.Fatalf("i8: %d", i8)
		}
	})

	t.Run("int64/uint64", func(t *testing.T) {
		var ui64 uint64
		err := vm.ExportTo(vm.ToValue(-1), &ui64)
		if err != nil {
			t.Fatal(err)
		}
		if ui64 != math.MaxUint64 {
			t.Fatalf("ui64: %d", ui64)
		}
	})

	t.Run("int8/float", func(t *testing.T) {
		var i8 int8
		err := vm.ExportTo(vm.ToValue(333.9234), &i8)
		if err != nil {
			t.Fatal(err)
		}
		if i8 != 77 {
			t.Fatalf("i8: %d", i8)
		}
	})

	t.Run("int8/object", func(t *testing.T) {
		var i8 int8
		err := vm.ExportTo(vm.NewObject(), &i8)
		if err != nil {
			t.Fatal(err)
		}
		if i8 != 0 {
			t.Fatalf("i8: %d", i8)
		}
	})

	t.Run("int/object_cust_valueOf", func(t *testing.T) {
		var i int
		obj, err := vm.RunString(`
		({
			valueOf: function() { return 42; }
		})
		`)
		if err != nil {
			t.Fatal(err)
		}
		err = vm.ExportTo(obj, &i)
		if err != nil {
			t.Fatal(err)
		}
		if i != 42 {
			t.Fatalf("i: %d", i)
		}
	})

	t.Run("float32/no_trunc", func(t *testing.T) {
		var f float32
		err := vm.ExportTo(vm.ToValue(1.234567), &f)
		if err != nil {
			t.Fatal(err)
		}
		if f != 1.234567 {
			t.Fatalf("f: %f", f)
		}
	})

	t.Run("float32/trunc", func(t *testing.T) {
		var f float32
		err := vm.ExportTo(vm.ToValue(1.234567890), &f)
		if err != nil {
			t.Fatal(err)
		}
		if f != float32(1.234567890) {
			t.Fatalf("f: %f", f)
		}
	})

	t.Run("float64", func(t *testing.T) {
		var f float64
		err := vm.ExportTo(vm.ToValue(1.234567), &f)
		if err != nil {
			t.Fatal(err)
		}
		if f != 1.234567 {
			t.Fatalf("f: %f", f)
		}
	})

	t.Run("float32/object", func(t *testing.T) {
		var f float32
		err := vm.ExportTo(vm.NewObject(), &f)
		if err != nil {
			t.Fatal(err)
		}
		if f == f { // expecting NaN
			t.Fatalf("f: %f", f)
		}
	})

	t.Run("float64/object", func(t *testing.T) {
		var f float64
		err := vm.ExportTo(vm.NewObject(), &f)
		if err != nil {
			t.Fatal(err)
		}
		if f == f { // expecting NaN
			t.Fatalf("f: %f", f)
		}
	})

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

func TestRuntime_ExportToStructPtr(t *testing.T) {
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

	var o *testGoReflectMethod_O
	err = vm.ExportTo(v, &o)
	if err != nil {
		t.Fatal(err)
	}

	if o.Test != "1" {
		t.Fatalf("Unexpected value: '%s'", o.Test)
	}

}

func TestRuntime_ExportToStructAnonymous(t *testing.T) {
	type BaseTestStruct struct {
		A int64
		B int64
	}

	type TestStruct struct {
		BaseTestStruct
		C string
	}

	const SCRIPT = `
	var m = {
		A: 1,
		B: 2,
		C: "testC"
	}
	m;
	`
	vm := New()
	v, err := vm.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	test := &TestStruct{}
	err = vm.ExportTo(v, test)
	if err != nil {
		t.Fatal(err)
	}

	if test.A != 1 {
		t.Fatalf("Unexpected value: '%d'", test.A)
	}
	if test.B != 2 {
		t.Fatalf("Unexpected value: '%d'", test.B)
	}
	if test.C != "testC" {
		t.Fatalf("Unexpected value: '%s'", test.C)
	}

}

func TestRuntime_ExportToStructWithPtrValues(t *testing.T) {
	type BaseTestStruct struct {
		A int64
		B *int64
	}

	type TestStruct2 struct {
		E string
	}

	type TestStruct struct {
		BaseTestStruct
		C *string
		D *TestStruct2
	}

	const SCRIPT = `
	var m = {
		A: 1,
		B: 2,
		C: "testC",
		D: {
			E: "testE",
		}
	}
	m;
	`
	vm := New()
	v, err := vm.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	test := &TestStruct{}
	err = vm.ExportTo(v, test)
	if err != nil {
		t.Fatal(err)
	}

	if test.A != 1 {
		t.Fatalf("Unexpected value: '%d'", test.A)
	}
	if test.B == nil || *test.B != 2 {
		t.Fatalf("Unexpected value: '%v'", test.B)
	}
	if test.C == nil || *test.C != "testC" {
		t.Fatalf("Unexpected value: '%v'", test.C)
	}
	if test.D == nil || test.D.E != "testE" {
		t.Fatalf("Unexpected value: '%s'", test.D.E)
	}

}

func TestRuntime_ExportToTime(t *testing.T) {
	const SCRIPT = `
	var dateStr = "2018-08-13T15:02:13+02:00";
	var str = "test123";
	`

	vm := New()
	_, err := vm.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	var ti time.Time
	err = vm.ExportTo(vm.Get("dateStr"), &ti)
	if err != nil {
		t.Fatal(err)
	}
	if ti.Format(time.RFC3339) != "2018-08-13T15:02:13+02:00" {
		t.Fatalf("Unexpected value: '%s'", ti.Format(time.RFC3339))
	}

	err = vm.ExportTo(vm.Get("str"), &ti)
	if err == nil {
		t.Fatal("Expected err to not be nil")
	}

	var str string
	err = vm.ExportTo(vm.Get("dateStr"), &str)
	if err != nil {
		t.Fatal(err)
	}
	if str != "2018-08-13T15:02:13+02:00" {
		t.Fatalf("Unexpected value: '%s'", str)
	}

	d, err := vm.RunString(`new Date(1000)`)
	if err != nil {
		t.Fatal(err)
	}

	ti = time.Time{}
	err = vm.ExportTo(d, &ti)
	if err != nil {
		t.Fatal(err)
	}

	if ti.UnixNano() != 1000*1e6 {
		t.Fatal(ti)
	}
	if ti.Location() != time.Local {
		t.Fatalf("Wrong location: %v", ti)
	}
}

func ExampleRuntime_ExportTo_func() {
	const SCRIPT = `
	function f(param) {
		return +param + 2;
	}
	`

	vm := New()
	_, err := vm.RunString(SCRIPT)
	if err != nil {
		panic(err)
	}

	var fn func(string) string
	err = vm.ExportTo(vm.Get("f"), &fn)
	if err != nil {
		panic(err)
	}

	fmt.Println(fn("40")) // note, _this_ value in the function will be undefined.
	// Output: 42
}

func ExampleRuntime_ExportTo_funcThrow() {
	const SCRIPT = `
	function f(param) {
		throw new Error("testing");
	}
	`

	vm := New()
	_, err := vm.RunString(SCRIPT)
	if err != nil {
		panic(err)
	}

	var fn func(string) (string, error)
	err = vm.ExportTo(vm.Get("f"), &fn)
	if err != nil {
		panic(err)
	}
	_, err = fn("")

	fmt.Println(err)
	// Output: Error: testing at f (<eval>:3:9(4))
}

func ExampleRuntime_ExportTo_funcVariadic() {
	const SCRIPT = `
	function f() {
		return Array.prototype.join.call(arguments, ",");
	}
	`
	vm := New()
	_, err := vm.RunString(SCRIPT)
	if err != nil {
		panic(err)
	}

	var fn func(args ...interface{}) string
	err = vm.ExportTo(vm.Get("f"), &fn)
	if err != nil {
		panic(err)
	}
	fmt.Println(fn("a", "b", 42))
	// Output: a,b,42
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

func ExampleAssertFunction() {
	vm := New()
	_, err := vm.RunString(`
	function sum(a, b) {
		return a+b;
	}
	`)
	if err != nil {
		panic(err)
	}
	sum, ok := AssertFunction(vm.Get("sum"))
	if !ok {
		panic("Not a function")
	}

	res, err := sum(Undefined(), vm.ToValue(40), vm.ToValue(2))
	if err != nil {
		panic(err)
	}
	fmt.Println(res)
	// Output: 42
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

	if v := vm.ToValue(nil); !IsNull(v) {
		t.Fatalf("nil: %v", v)
	}

	if v := vm.ToValue(a); !IsNull(v) {
		t.Fatalf("struct ptr: %v", v)
	}

	var m map[string]interface{}
	if v := vm.ToValue(m); !IsNull(v) {
		t.Fatalf("map[string]interface{}: %v", v)
	}

	var ar []interface{}
	if v := vm.ToValue(ar); !IsNull(v) {
		t.Fatalf("[]interface{}: %v", v)
	}

	var arptr *[]interface{}
	if v := vm.ToValue(arptr); !IsNull(v) {
		t.Fatalf("*[]interface{}: %v", v)
	}
}

func TestToValueFloat(t *testing.T) {
	vm := New()
	vm.Set("f64", float64(123))
	vm.Set("f32", float32(321))

	v, err := vm.RunString("f64 === 123 && f32 === 321")
	if err != nil {
		t.Fatal(err)
	}
	if v.Export().(bool) != true {
		t.Fatalf("StrictEquals for golang float failed")
	}
}

func TestToValueInterface(t *testing.T) {

	f := func(i interface{}) bool {
		return i == t
	}
	vm := New()
	vm.Set("f", f)
	vm.Set("t", t)
	v, err := vm.RunString(`f(t)`)
	if err != nil {
		t.Fatal(err)
	}
	if v != valueTrue {
		t.Fatalf("v: %v", v)
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

func TestSortComparatorReturnValueFloats(t *testing.T) {
	const SCRIPT = `
	var a = [
		5.97,
		9.91,
		4.13,
		9.28,
		3.29,
	];
	a.sort( function(a, b) { return a - b; } );
	for (var i = 1; i < a.length; i++) {
		if (a[i] < a[i-1]) {
			throw new Error("Array is not sorted: " + a);
		}
	}
	`
	testScript1(SCRIPT, _undefined, t)
}

func TestSortComparatorReturnValueNegZero(t *testing.T) {
	const SCRIPT = `
	var a = [2, 1];
	a.sort( function(a, b) { return a > b ? 0 : -0; } );
	for (var i = 1; i < a.length; i++) {
		if (a[i] < a[i-1]) {
			throw new Error("Array is not sorted: " + a);
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

type testNativeConstructHelper struct {
	rt   *Runtime
	base int64
	// any other state
}

func (t *testNativeConstructHelper) calc(call FunctionCall) Value {
	return t.rt.ToValue(t.base + call.Argument(0).ToInteger())
}

func TestNativeConstruct(t *testing.T) {
	const SCRIPT = `
	var f = new F(40);
	f instanceof F && f.method() === 42 && f.calc(2) === 42;
	`

	rt := New()

	method := func(call FunctionCall) Value {
		return rt.ToValue(42)
	}

	rt.Set("F", func(call ConstructorCall) *Object { // constructor signature (as opposed to 'func(FunctionCall) Value')
		h := &testNativeConstructHelper{
			rt:   rt,
			base: call.Argument(0).ToInteger(),
		}
		call.This.Set("method", method)
		call.This.Set("calc", h.calc)
		return nil // or any other *Object which will be used instead of call.This
	})

	prg := MustCompile("test.js", SCRIPT, false)

	res, err := rt.RunProgram(prg)
	if err != nil {
		t.Fatal(err)
	}

	if !res.StrictEquals(valueTrue) {
		t.Fatalf("Unexpected result: %v", res)
	}

	if fn, ok := AssertFunction(rt.Get("F")); ok {
		v, err := fn(nil, rt.ToValue(42))
		if err != nil {
			t.Fatal(err)
		}
		if o, ok := v.(*Object); ok {
			if o.Get("method") == nil {
				t.Fatal("No method")
			}
		} else {
			t.Fatal("Not an object")
		}
	} else {
		t.Fatal("Not a function")
	}

	resp := &testNativeConstructHelper{}
	value := rt.ToValue(resp)
	if value.Export() != resp {
		t.Fatal("no")
	}
}

func TestCreateObject(t *testing.T) {
	const SCRIPT = `
	inst instanceof C;
	`

	r := New()
	c := r.ToValue(func(call ConstructorCall) *Object {
		return nil
	})

	proto := c.(*Object).Get("prototype").(*Object)

	inst := r.CreateObject(proto)

	r.Set("C", c)
	r.Set("inst", inst)

	prg := MustCompile("test.js", SCRIPT, false)

	res, err := r.RunProgram(prg)
	if err != nil {
		t.Fatal(err)
	}

	if !res.StrictEquals(valueTrue) {
		t.Fatalf("Unexpected result: %v", res)
	}
}

func TestInterruptInWrappedFunction(t *testing.T) {
	rt := New()
	v, err := rt.RunString(`
		var fn = function() {
			while (true) {}
		};
		fn;
	`)
	if err != nil {
		t.Fatal(err)
	}
	fn, ok := AssertFunction(v)
	if !ok {
		t.Fatal("Not a function")
	}
	go func() {
		<-time.After(10 * time.Millisecond)
		rt.Interrupt(errors.New("hi"))
	}()

	_, err = fn(nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if _, ok := err.(*InterruptedError); !ok {
		t.Fatalf("Wrong error type: %T", err)
	}
}

func TestRunLoopPreempt(t *testing.T) {
	vm := New()
	v, err := vm.RunString("(function() {for (;;) {}})")
	if err != nil {
		t.Fatal(err)
	}

	fn, ok := AssertFunction(v)
	if !ok {
		t.Fatal("Not a function")
	}

	go func() {
		<-time.After(100 * time.Millisecond)
		runtime.GC() // this hangs if the vm loop does not have any preemption points
		vm.Interrupt(errors.New("hi"))
	}()

	_, err = fn(nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if _, ok := err.(*InterruptedError); !ok {
		t.Fatalf("Wrong error type: %T", err)
	}
}

func TestNaN(t *testing.T) {
	if !IsNaN(_NaN) {
		t.Fatal("IsNaN() doesn't detect NaN")
	}
	if IsNaN(Undefined()) {
		t.Fatal("IsNaN() says undefined is a NaN")
	}
	if !IsNaN(NaN()) {
		t.Fatal("NaN() doesn't return NaN")
	}
}

func TestInf(t *testing.T) {
	if !IsInfinity(_positiveInf) {
		t.Fatal("IsInfinity() doesn't detect +Inf")
	}
	if !IsInfinity(_negativeInf) {
		t.Fatal("IsInfinity() doesn't detect -Inf")
	}
	if IsInfinity(Undefined()) {
		t.Fatal("IsInfinity() says undefined is a Infinity")
	}
	if !IsInfinity(PositiveInf()) {
		t.Fatal("PositiveInfinity() doesn't return Inf")
	}
	if !IsInfinity(NegativeInf()) {
		t.Fatal("NegativeInfinity() doesn't return Inf")
	}
}

func TestRuntimeNew(t *testing.T) {
	vm := New()
	v, err := vm.New(vm.Get("Number"), vm.ToValue("12345"))
	if err != nil {
		t.Fatal(err)
	}
	if n, ok := v.Export().(int64); ok {
		if n != 12345 {
			t.Fatalf("n: %v", n)
		}
	} else {
		t.Fatalf("v: %T", v)
	}
}

func TestAutoBoxing(t *testing.T) {
	const SCRIPT = `
	function f() {
		'use strict';
		var a = 1;
		var thrown1 = false;
		var thrown2 = false;
		try {
			a.test = 42;
		} catch (e) {
			thrown1 = e instanceof TypeError;
		}
		try {
			a["test1"] = 42;
		} catch (e) {
			thrown2 = e instanceof TypeError;
		}
		return thrown1 && thrown2;
	}
	var a = 1;
	a.test = 42; // should not throw
	a["test1"] = 42; // should not throw
	a.test === undefined && a.test1 === undefined && f();
	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestProtoGetter(t *testing.T) {
	const SCRIPT = `
	({}).__proto__ === Object.prototype && [].__proto__ === Array.prototype;
	`
	testScript1(SCRIPT, valueTrue, t)
}

func TestFuncProto(t *testing.T) {
	const SCRIPT = `
	"use strict";
	function A() {}
	A.__proto__ = Object;
	A.prototype = {};

	function B() {}
	B.__proto__ = Object.create(null);
	var thrown = false;
	try {
		delete B.prototype;
	} catch (e) {
		thrown = e instanceof TypeError;
	}
	thrown;
	`
	testScript1(SCRIPT, valueTrue, t)
}

func TestSymbol1(t *testing.T) {
	const SCRIPT = `
		Symbol.toPrimitive[Symbol.toPrimitive]() === Symbol.toPrimitive;
	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestFreezeSymbol(t *testing.T) {
	const SCRIPT = `
		var s = Symbol(1);
		var o = {};
		o[s] = 42;
		Object.freeze(o);
		o[s] = 43;
		o[s] === 42 && Object.isFrozen(o);
	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestToPropertyKey(t *testing.T) {
	const SCRIPT = `
	var sym = Symbol(42);
	var callCount = 0;

	var wrapper = {
	  toString: function() {
		callCount += 1;
		return sym;
	  },
	  valueOf: function() {
		$ERROR("valueOf() called");
	  }
	};

	var o = {};
	o[wrapper] = function() { return "test" };
	assert.sameValue(o[wrapper], o[sym], "o[wrapper] === o[sym]");
	assert.sameValue(o[wrapper](), "test", "o[wrapper]()");
	assert.sameValue(o[sym](), "test", "o[sym]()");

	var wrapper1 = {};
	wrapper1[Symbol.toPrimitive] = function(hint) {
		if (hint === "string" || hint === "default") {
			return "1";
		}
		if (hint === "number") {
			return 2;
		}
		$ERROR("Unknown hint value "+hint);
	};
	var a = [];
	a[wrapper1] = 42;
	assert.sameValue(a[1], 42, "a[1]");
	assert.sameValue(a[1], a[wrapper1], "a[1] === a[wrapper1]");
	`

	testScript1(TESTLIB+SCRIPT, _undefined, t)
}

func TestPrimThisValue(t *testing.T) {
	const SCRIPT = `
	function t() {
		'use strict';

		Boolean.prototype.toString = function() {
		  return typeof this;
		};

		assert.sameValue(true.toLocaleString(), "boolean");

		Boolean.prototype[Symbol.iterator] = function() {
			return [typeof this][Symbol.iterator]();
		}
		var s = new Set(true);
		assert.sameValue(s.size, 1, "size");
		assert.sameValue(s.has("boolean"), true, "s.has('boolean')");
	}
	t();
	`

	testScript1(TESTLIB+SCRIPT, _undefined, t)
}

func TestPrimThisValueGetter(t *testing.T) {
	const SCRIPT = `
	function t() {
		'use strict';
		Object.defineProperty(Boolean.prototype, "toString", {
		  get: function() {
			var v = typeof this;
			return function() {
			  return v;
			};
		  }
		});

		assert.sameValue(true.toLocaleString(), "boolean");
	}
	t();
	`

	testScript1(TESTLIB+SCRIPT, _undefined, t)
}

func TestObjSetSym(t *testing.T) {
	const SCRIPT = `
	'use strict';
	var sym = Symbol(true);
	var p1 = Object.create(null);
	var p2 = Object.create(p1);
	
	Object.defineProperty(p1, sym, {
	value: 42
	});
	
	Object.defineProperty(p2, sym, {
	value: 43,
	writable: true,
	});
	var o = Object.create(p2);
	o[sym] = 44;
	o[sym];
	`
	testScript1(SCRIPT, intToValue(44), t)
}

func TestObjSet(t *testing.T) {
	const SCRIPT = `
	'use strict';
	var p1 = Object.create(null);
	var p2 = Object.create(p1);
	
	Object.defineProperty(p1, "test", {
	value: 42
	});
	
	Object.defineProperty(p2, "test", {
	value: 43,
	writable: true,
	});
	var o = Object.create(p2);
	o.test = 44;
	o.test;
	`
	testScript1(SCRIPT, intToValue(44), t)
}

func TestToValueNilValue(t *testing.T) {
	r := New()
	var a Value
	r.Set("a", a)
	ret, err := r.RunString(`
	""+a;
	`)
	if err != nil {
		t.Fatal(err)
	}
	if !asciiString("null").SameAs(ret) {
		t.Fatalf("ret: %v", ret)
	}
}

func TestDateConversion(t *testing.T) {
	now := time.Now()
	vm := New()
	val, err := vm.New(vm.Get("Date").ToObject(vm), vm.ToValue(now.UnixNano()/1e6))
	if err != nil {
		t.Fatal(err)
	}
	vm.Set("d", val)
	res, err := vm.RunString(`+d`)
	if err != nil {
		t.Fatal(err)
	}
	if exp := res.Export(); exp != now.UnixNano()/1e6 {
		t.Fatalf("Value does not match: %v", exp)
	}
	vm.Set("goval", now)
	res, err = vm.RunString(`+(new Date(goval.UnixNano()/1e6))`)
	if err != nil {
		t.Fatal(err)
	}
	if exp := res.Export(); exp != now.UnixNano()/1e6 {
		t.Fatalf("Value does not match: %v", exp)
	}
}

func TestNativeCtorNewTarget(t *testing.T) {
	const SCRIPT = `
	function NewTarget() {
	}

	var o = Reflect.construct(Number, [1], NewTarget);
	o.__proto__ === NewTarget.prototype && o.toString() === "[object Number]";
	`
	testScript1(SCRIPT, valueTrue, t)
}

func TestNativeCtorNonNewCall(t *testing.T) {
	vm := New()
	vm.Set(`Animal`, func(call ConstructorCall) *Object {
		obj := call.This
		obj.Set(`name`, call.Argument(0).String())
		obj.Set(`eat`, func(call FunctionCall) Value {
			self := call.This.(*Object)
			return vm.ToValue(fmt.Sprintf("%s eat", self.Get(`name`)))
		})
		return nil
	})
	v, err := vm.RunString(`

	function __extends(d, b){
		function __() {
			this.constructor = d;
		}
		d.prototype = b === null ? Object.create(b) : (__.prototype = b.prototype, new __());
	}

	var Cat = (function (_super) {
		__extends(Cat, _super);
		function Cat() {
			return _super.call(this, "cat") || this;
		}
		return Cat;
	}(Animal));

	var cat = new Cat();
	cat instanceof Cat && cat.eat() === "cat eat";
	`)
	if err != nil {
		t.Fatal(err)
	}
	if v != valueTrue {
		t.Fatal(v)
	}
}

func ExampleNewSymbol() {
	sym1 := NewSymbol("66")
	sym2 := NewSymbol("66")
	fmt.Printf("%s %s %v", sym1, sym2, sym1.Equals(sym2))
	// Output: 66 66 false
}

func ExampleObject_SetSymbol() {
	type IterResult struct {
		Done  bool
		Value Value
	}

	vm := New()
	vm.SetFieldNameMapper(UncapFieldNameMapper()) // to use IterResult

	o := vm.NewObject()
	o.SetSymbol(SymIterator, func() *Object {
		count := 0
		iter := vm.NewObject()
		iter.Set("next", func() IterResult {
			if count < 10 {
				count++
				return IterResult{
					Value: vm.ToValue(count),
				}
			}
			return IterResult{
				Done: true,
			}
		})
		return iter
	})
	vm.Set("o", o)

	res, err := vm.RunString(`
	var acc = "";
	for (var v of o) {
		acc += v + " ";
	}
	acc;
	`)
	if err != nil {
		panic(err)
	}
	fmt.Println(res)
	// Output: 1 2 3 4 5 6 7 8 9 10
}

func ExampleRuntime_NewArray() {
	vm := New()
	array := vm.NewArray(1, 2, true)
	vm.Set("array", array)
	res, err := vm.RunString(`
	var acc = "";
	for (var v of array) {
		acc += v + " ";
	}
	acc;
	`)
	if err != nil {
		panic(err)
	}
	fmt.Println(res)
	// Output: 1 2 true
}

func ExampleRuntime_SetParserOptions() {
	vm := New()
	vm.SetParserOptions(parser.WithDisableSourceMaps)

	res, err := vm.RunString(`
	"I did not hang!";
//# sourceMappingURL=/dev/zero`)

	if err != nil {
		panic(err)
	}
	fmt.Println(res.String())
	// Output: I did not hang!
}

func TestRuntime_SetParserOptions_Eval(t *testing.T) {
	vm := New()
	vm.SetParserOptions(parser.WithDisableSourceMaps)

	_, err := vm.RunString(`
	eval("//# sourceMappingURL=/dev/zero");
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestNativeCallWithRuntimeParameter(t *testing.T) {
	vm := New()
	vm.Set("f", func(_ FunctionCall, r *Runtime) Value {
		if r == vm {
			return valueTrue
		}
		return valueFalse
	})
	ret, err := vm.RunString(`f()`)
	if err != nil {
		t.Fatal(err)
	}
	if ret != valueTrue {
		t.Fatal(ret)
	}
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

func BenchmarkMainLoop(b *testing.B) {
	vm := New()

	const SCRIPT = `
		for (var i=0; i<100000; i++) {
		}
	`

	prg := MustCompile("test.js", SCRIPT, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vm.RunProgram(prg)
	}
}

func BenchmarkStringMapGet(b *testing.B) {
	m := make(map[string]Value)
	for i := 0; i < 100; i++ {
		m[strconv.Itoa(i)] = intToValue(int64(i))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if m["50"] == nil {
			b.Fatal()
		}
	}
}

func BenchmarkValueStringMapGet(b *testing.B) {
	m := make(map[valueString]Value)
	for i := 0; i < 100; i++ {
		m[asciiString(strconv.Itoa(i))] = intToValue(int64(i))
	}
	b.ResetTimer()
	var key valueString = asciiString("50")
	for i := 0; i < b.N; i++ {
		if m[key] == nil {
			b.Fatal()
		}
	}
}

func BenchmarkAsciiStringMapGet(b *testing.B) {
	m := make(map[asciiString]Value)
	for i := 0; i < 100; i++ {
		m[asciiString(strconv.Itoa(i))] = intToValue(int64(i))
	}
	b.ResetTimer()
	var key = asciiString("50")
	for i := 0; i < b.N; i++ {
		if m[key] == nil {
			b.Fatal()
		}
	}
}
