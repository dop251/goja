package goja

import (
	"fmt"
	"reflect"
	"testing"
)

func TestArray1(t *testing.T) {
	r := &Runtime{}
	a := r.newArray(nil)
	a.setOwnIdx(valueInt(0), asciiString("test"), true)
	if l := a.getStr("length", nil).ToInteger(); l != 1 {
		t.Fatalf("Unexpected length: %d", l)
	}
}

func TestDefineProperty(t *testing.T) {
	r := New()
	o := r.NewObject()

	err := o.DefineDataProperty("data", r.ToValue(42), FLAG_TRUE, FLAG_TRUE, FLAG_TRUE)
	if err != nil {
		t.Fatal(err)
	}

	err = o.DefineAccessorProperty("accessor_ro", r.ToValue(func() int {
		return 1
	}), nil, FLAG_TRUE, FLAG_TRUE)
	if err != nil {
		t.Fatal(err)
	}

	err = o.DefineAccessorProperty("accessor_rw",
		r.ToValue(func(call FunctionCall) Value {
			return o.Get("__hidden")
		}),
		r.ToValue(func(call FunctionCall) (ret Value) {
			o.Set("__hidden", call.Argument(0))
			return
		}),
		FLAG_TRUE, FLAG_TRUE)

	if err != nil {
		t.Fatal(err)
	}

	if v := o.Get("accessor_ro"); v.ToInteger() != 1 {
		t.Fatalf("Unexpected accessor value: %v", v)
	}

	err = o.Set("accessor_ro", r.ToValue(2))
	if err == nil {
		t.Fatal("Expected an error")
	}
	if ex, ok := err.(*Exception); ok {
		if msg := ex.Error(); msg != "TypeError: Cannot assign to read only property 'accessor_ro'" {
			t.Fatalf("Unexpected error: '%s'", msg)
		}
	} else {
		t.Fatalf("Unexected error type: %T", err)
	}

	err = o.Set("accessor_rw", 42)
	if err != nil {
		t.Fatal(err)
	}

	if v := o.Get("accessor_rw"); v.ToInteger() != 42 {
		t.Fatalf("Unexpected value: %v", v)
	}
}

func TestPropertyOrder(t *testing.T) {
	const SCRIPT = `
	var o = {};
	var sym1 = Symbol(1);
	var sym2 = Symbol(2);
	o[sym2] = 1;
	o[4294967294] = 1;
	o[2] = 1;
	o[1] = 1;
	o[0] = 1;
	o["02"] = 1;
	o[4294967295] = 1;
	o["01"] = 1;
	o["00"] = 1;
	o[sym1] = 1;
	var expected = ["0", "1", "2", "4294967294", "02", "4294967295", "01", "00", sym2, sym1];
	var actual = Reflect.ownKeys(o);
	if (actual.length !== expected.length) {
		throw new Error("Unexpected length: "+actual.length);
	}
	for (var i = 0; i < actual.length; i++) {
		if (actual[i] !== expected[i]) {
			throw new Error("Unexpected list: " + actual);
		}
	}
	`

	testScript1(SCRIPT, _undefined, t)
}

func TestDefinePropertiesSymbol(t *testing.T) {
	const SCRIPT = `
	var desc = {};
	desc[Symbol.toStringTag] = {value: "Test"};
	var o = {};
	Object.defineProperties(o, desc);
	o[Symbol.toStringTag] === "Test";
	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestObjectShorthandProperties(t *testing.T) {
	const SCRIPT = `
	var b = 1;
	var a = {b, get() {return "c"}};

	assert.sameValue(a.b, b, "#1");
	assert.sameValue(a.get(), "c", "#2");

	var obj = {
		w\u0069th() { return 42; }
    };

	assert.sameValue(obj['with'](), 42, 'property exists');
	`
	testScript1(TESTLIB+SCRIPT, _undefined, t)
}

func TestObjectAssign(t *testing.T) {
	const SCRIPT = `
	assert.sameValue(Object.assign({ b: 1 }, { get a() {
          Object.defineProperty(this, "b", {
            value: 3,
            enumerable: false
          });
        }, b: 2 }).b, 1, "#1");

	assert.sameValue(Object.assign({ b: 1 }, { get a() {
          delete this.b;
        }, b: 2 }).b, 1, "#2");
	`
	testScript1(TESTLIB+SCRIPT, _undefined, t)
}

func TestExportCircular(t *testing.T) {
	vm := New()
	o := vm.NewObject()
	o.Set("o", o)
	v := o.Export()
	if m, ok := v.(map[string]interface{}); ok {
		if reflect.ValueOf(m["o"]).Pointer() != reflect.ValueOf(v).Pointer() {
			t.Fatal("Unexpected value")
		}
	} else {
		t.Fatal("Unexpected type")
	}

	res, err := vm.RunString(`var a = []; a[0] = a;`)
	if err != nil {
		t.Fatal(err)
	}
	v = res.Export()
	if a, ok := v.([]interface{}); ok {
		if reflect.ValueOf(a[0]).Pointer() != reflect.ValueOf(v).Pointer() {
			t.Fatal("Unexpected value")
		}
	} else {
		t.Fatal("Unexpected type")
	}
}

type test_s struct {
	S *test_s1
}
type test_s1 struct {
	S *test_s
}

func TestExportToCircular(t *testing.T) {
	vm := New()
	o := vm.NewObject()
	o.Set("o", o)
	var m map[string]interface{}
	err := vm.ExportTo(o, &m)
	if err != nil {
		t.Fatal(err)
	}

	type K string
	type T map[K]T
	var m1 T
	err = vm.ExportTo(o, &m1)
	if err != nil {
		t.Fatal(err)
	}

	type A []A
	var a A
	res, err := vm.RunString("var a = []; a[0] = a;")
	if err != nil {
		t.Fatal(err)
	}
	err = vm.ExportTo(res, &a)
	if err != nil {
		t.Fatal(err)
	}
	if &a[0] != &a[0][0] {
		t.Fatal("values do not match")
	}

	o = vm.NewObject()
	o.Set("S", o)
	var s test_s
	err = vm.ExportTo(o, &s)
	if err != nil {
		t.Fatal(err)
	}
	if s.S.S != &s {
		t.Fatalf("values do not match: %v, %v", s.S.S, &s)
	}

	type test_s2 struct {
		S  interface{}
		S1 *test_s2
	}

	var s2 test_s2
	o.Set("S1", o)

	err = vm.ExportTo(o, &s2)
	if err != nil {
		t.Fatal(err)
	}

	if m, ok := s2.S.(map[string]interface{}); ok {
		if reflect.ValueOf(m["S"]).Pointer() != reflect.ValueOf(m).Pointer() {
			t.Fatal("Unexpected m.S")
		}
	} else {
		t.Fatalf("Unexpected s2.S type: %T", s2.S)
	}
	if s2.S1 != &s2 {
		t.Fatal("Unexpected s2.S1")
	}

	o1 := vm.NewObject()
	o1.Set("S", o)
	o1.Set("S1", o)
	err = vm.ExportTo(o1, &s2)
	if err != nil {
		t.Fatal(err)
	}
	if s2.S1.S1 != s2.S1 {
		t.Fatal("Unexpected s2.S1.S1")
	}
}

func TestExportWrappedMap(t *testing.T) {
	vm := New()
	m := map[string]interface{}{
		"test": "failed",
	}
	exported := vm.ToValue(m).Export()
	if exportedMap, ok := exported.(map[string]interface{}); ok {
		exportedMap["test"] = "passed"
		if v := m["test"]; v != "passed" {
			t.Fatalf("Unexpected m[\"test\"]: %v", v)
		}
	} else {
		t.Fatalf("Unexpected export type: %T", exported)
	}
}

func TestExportToWrappedMap(t *testing.T) {
	vm := New()
	m := map[string]interface{}{
		"test": "failed",
	}
	var exported map[string]interface{}
	err := vm.ExportTo(vm.ToValue(m), &exported)
	if err != nil {
		t.Fatal(err)
	}
	exported["test"] = "passed"
	if v := m["test"]; v != "passed" {
		t.Fatalf("Unexpected m[\"test\"]: %v", v)
	}
}

func TestExportToWrappedMapCustom(t *testing.T) {
	type CustomMap map[string]bool
	vm := New()
	m := CustomMap{}
	var exported CustomMap
	err := vm.ExportTo(vm.ToValue(m), &exported)
	if err != nil {
		t.Fatal(err)
	}
	exported["test"] = true
	if v := m["test"]; v != true {
		t.Fatalf("Unexpected m[\"test\"]: %v", v)
	}
}

func ExampleObject_Delete() {
	vm := New()
	obj := vm.NewObject()
	_ = obj.Set("test", true)
	before := obj.Get("test")
	_ = obj.Delete("test")
	after := obj.Get("test")
	fmt.Printf("before: %v, after: %v", before, after)
	// Output: before: true, after: <nil>
}

func BenchmarkPut(b *testing.B) {
	v := &Object{}

	o := &baseObject{
		val:        v,
		extensible: true,
	}
	v.self = o

	o.init()

	var key Value = asciiString("test")
	var val Value = valueInt(123)

	for i := 0; i < b.N; i++ {
		v.setOwn(key, val, false)
	}
}

func BenchmarkPutStr(b *testing.B) {
	v := &Object{}

	o := &baseObject{
		val:        v,
		extensible: true,
	}

	o.init()

	v.self = o

	var val Value = valueInt(123)

	for i := 0; i < b.N; i++ {
		o.setOwnStr("test", val, false)
	}
}

func BenchmarkGet(b *testing.B) {
	v := &Object{}

	o := &baseObject{
		val:        v,
		extensible: true,
	}

	o.init()

	v.self = o
	var n Value = asciiString("test")

	for i := 0; i < b.N; i++ {
		v.get(n, nil)
	}

}

func BenchmarkGetStr(b *testing.B) {
	v := &Object{}

	o := &baseObject{
		val:        v,
		extensible: true,
	}
	v.self = o

	o.init()

	for i := 0; i < b.N; i++ {
		o.getStr("test", nil)
	}
}

func _toString(v Value) string {
	switch v := v.(type) {
	case asciiString:
		return string(v)
	default:
		return ""
	}
}

func BenchmarkToString1(b *testing.B) {
	v := asciiString("test")

	for i := 0; i < b.N; i++ {
		v.toString()
	}
}

func BenchmarkToString2(b *testing.B) {
	v := asciiString("test")

	for i := 0; i < b.N; i++ {
		_toString(v)
	}
}

func BenchmarkConv(b *testing.B) {
	count := int64(0)
	for i := 0; i < b.N; i++ {
		count += valueInt(123).ToInteger()
	}
	if count == 0 {
		b.Fatal("zero")
	}
}

func BenchmarkArrayGetStr(b *testing.B) {
	b.StopTimer()
	r := New()
	v := &Object{runtime: r}

	a := &arrayObject{
		baseObject: baseObject{
			val:        v,
			extensible: true,
		},
	}
	v.self = a

	a.init()

	v.setOwn(valueInt(0), asciiString("test"), false)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		a.getStr("0", nil)
	}

}

func BenchmarkArrayGet(b *testing.B) {
	b.StopTimer()
	r := New()
	v := &Object{runtime: r}

	a := &arrayObject{
		baseObject: baseObject{
			val:        v,
			extensible: true,
		},
	}
	v.self = a

	a.init()

	var idx Value = valueInt(0)

	v.setOwn(idx, asciiString("test"), false)

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		v.get(idx, nil)
	}

}

func BenchmarkArrayPut(b *testing.B) {
	b.StopTimer()
	r := New()

	v := &Object{runtime: r}

	a := &arrayObject{
		baseObject: baseObject{
			val:        v,
			extensible: true,
		},
	}

	v.self = a

	a.init()

	var idx Value = valueInt(0)
	var val Value = asciiString("test")

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		v.setOwn(idx, val, false)
	}

}

func BenchmarkToUTF8String(b *testing.B) {
	var s valueString = asciiString("test")
	for i := 0; i < b.N; i++ {
		_ = s.String()
	}
}

func BenchmarkAdd(b *testing.B) {
	var x, y Value
	x = valueInt(2)
	y = valueInt(2)

	for i := 0; i < b.N; i++ {
		if xi, ok := x.(valueInt); ok {
			if yi, ok := y.(valueInt); ok {
				x = xi + yi
			}
		}
	}
}

func BenchmarkAddString(b *testing.B) {
	var x, y Value

	tst := asciiString("22")
	x = asciiString("2")
	y = asciiString("2")

	for i := 0; i < b.N; i++ {
		var z Value
		if xi, ok := x.(valueString); ok {
			if yi, ok := y.(valueString); ok {
				z = xi.concat(yi)
			}
		}
		if !z.StrictEquals(tst) {
			b.Fatalf("Unexpected result %v", x)
		}
	}
}
