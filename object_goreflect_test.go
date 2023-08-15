package goja

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestGoReflectGet(t *testing.T) {
	const SCRIPT = `
	o.X + o.Y;
	`
	type O struct {
		X int
		Y string
	}
	r := New()
	o := O{X: 4, Y: "2"}
	r.Set("o", o)

	v, err := r.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	if s, ok := v.(String); ok {
		if s.String() != "42" {
			t.Fatalf("Unexpected string: %s", s)
		}
	} else {
		t.Fatalf("Unexpected type: %s", v)
	}
}

func TestGoReflectSet(t *testing.T) {
	const SCRIPT = `
	o.X++;
	o.Y += "P";
	`
	type O struct {
		X int
		Y string
	}
	r := New()
	o := O{X: 4, Y: "2"}
	r.Set("o", &o)

	_, err := r.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	if o.X != 5 {
		t.Fatalf("Unexpected X: %d", o.X)
	}

	if o.Y != "2P" {
		t.Fatalf("Unexpected Y: %s", o.Y)
	}

	r.Set("o", o)
	_, err = r.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	if res, ok := r.Get("o").Export().(O); ok {
		if res.X != 6 {
			t.Fatalf("Unexpected res.X: %d", res.X)
		}

		if res.Y != "2PP" {
			t.Fatalf("Unexpected res.Y: %s", res.Y)
		}
	}
}

func TestGoReflectEnumerate(t *testing.T) {
	const SCRIPT = `
	var hasX = false;
	var hasY = false;
	for (var key in o) {
		switch (key) {
		case "X":
			if (hasX) {
				throw "Already have X";
			}
			hasX = true;
			break;
		case "Y":
			if (hasY) {
				throw "Already have Y";
			}
			hasY = true;
			break;
		default:
			throw "Unexpected property: " + key;
		}
	}
	hasX && hasY;
	`

	type S struct {
		X, Y int
	}

	r := New()
	r.Set("o", S{X: 40, Y: 2})
	v, err := r.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	if !v.StrictEquals(valueTrue) {
		t.Fatalf("Expected true, got %v", v)
	}

}

func TestGoReflectCustomIntUnbox(t *testing.T) {
	const SCRIPT = `
	i + 2;
	`

	type CustomInt int
	var i CustomInt = 40

	r := New()
	r.Set("i", i)
	v, err := r.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	if !v.StrictEquals(intToValue(42)) {
		t.Fatalf("Expected int 42, got %v", v)
	}
}

func TestGoReflectPreserveCustomType(t *testing.T) {
	const SCRIPT = `
	i;
	`

	type CustomInt int
	var i CustomInt = 42

	r := New()
	r.Set("i", i)
	v, err := r.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	ve := v.Export()

	if ii, ok := ve.(CustomInt); ok {
		if ii != i {
			t.Fatalf("Wrong value: %v", ii)
		}
	} else {
		t.Fatalf("Wrong type: %v", ve)
	}
}

func TestGoReflectCustomIntValueOf(t *testing.T) {
	const SCRIPT = `
	if (i instanceof Number) {
		i.valueOf();
	} else {
		throw new Error("Value is not a number");
	}
	`

	type CustomInt int
	var i CustomInt = 42

	r := New()
	r.Set("i", i)
	v, err := r.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	if !v.StrictEquals(intToValue(42)) {
		t.Fatalf("Expected int 42, got %v", v)
	}
}

func TestGoReflectEqual(t *testing.T) {
	const SCRIPT = `
	x === y;
	`

	type CustomInt int
	var x CustomInt = 42
	var y CustomInt = 42

	r := New()
	r.Set("x", x)
	r.Set("y", y)
	v, err := r.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	if !v.StrictEquals(valueTrue) {
		t.Fatalf("Expected true, got %v", v)
	}
}

type testGoReflectMethod_O struct {
	field string
	Test  string
}

func (o testGoReflectMethod_O) Method(s string) string {
	return o.field + s
}

func TestGoReflectMethod(t *testing.T) {
	const SCRIPT = `
	o.Method(" 123")
	`

	o := testGoReflectMethod_O{
		field: "test",
	}

	r := New()
	r.Set("o", &o)
	v, err := r.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	if !v.StrictEquals(asciiString("test 123")) {
		t.Fatalf("Expected 'test 123', got %v", v)
	}
}

func (o *testGoReflectMethod_O) Set(s string) {
	o.field = s
}

func (o *testGoReflectMethod_O) Get() string {
	return o.field
}

func TestGoReflectMethodPtr(t *testing.T) {
	const SCRIPT = `
	o.Set("42")
	o.Get()
	`

	o := testGoReflectMethod_O{
		field: "test",
	}

	r := New()
	r.Set("o", &o)
	v, err := r.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	if !v.StrictEquals(asciiString("42")) {
		t.Fatalf("Expected '42', got %v", v)
	}
}

func (b *testBoolS) Method() bool {
	return bool(*b)
}

func TestGoReflectPtrMethodOnNonPtrValue(t *testing.T) {
	var o testGoReflectMethod_O
	o.Get()
	vm := New()
	vm.Set("o", o)
	_, err := vm.RunString(`o.Get()`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = vm.RunString(`o.Method()`)
	if err != nil {
		t.Fatal(err)
	}

	var b testBoolS
	vm.Set("b", b)
	_, err = vm.RunString(`b.Method()`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGoReflectStructField(t *testing.T) {
	type S struct {
		F testGoReflectMethod_O
		B testBoolS
	}
	var s S
	vm := New()
	vm.Set("s", &s)

	const SCRIPT = `
	s.F.Set("Test");
	assert.sameValue(s.F.Method(""), "Test", "1");

	s.B = true;
	assert.sameValue(s.B.Method(), true, "2");

	assert.sameValue(s.B.toString(), "B", "3");
	`

	vm.testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestGoReflectProp(t *testing.T) {
	const SCRIPT = `
	var d1 = Object.getOwnPropertyDescriptor(o, "Get");
	var d2 = Object.getOwnPropertyDescriptor(o, "Test");
	!d1.writable && !d1.configurable && d2.writable && !d2.configurable;
	`

	o := testGoReflectMethod_O{
		field: "test",
	}

	r := New()
	r.Set("o", &o)
	v, err := r.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	if !v.StrictEquals(valueTrue) {
		t.Fatalf("Expected true, got %v", v)
	}
}

func TestGoReflectRedefineFieldSuccess(t *testing.T) {
	const SCRIPT = `
	Object.defineProperty(o, "Test", {value: "AAA"}) === o;
	`

	o := testGoReflectMethod_O{}

	r := New()
	r.Set("o", &o)
	v, err := r.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	if !v.StrictEquals(valueTrue) {
		t.Fatalf("Expected true, got %v", v)
	}

	if o.Test != "AAA" {
		t.Fatalf("Expected 'AAA', got '%s'", o.Test)
	}

}

func TestGoReflectRedefineFieldNonWritable(t *testing.T) {
	const SCRIPT = `
	var thrown = false;
	try {
		Object.defineProperty(o, "Test", {value: "AAA", writable: false});
	} catch (e) {
		if (e instanceof TypeError) {
			thrown = true;
		} else {
			throw e;
		}
	}
	thrown;
	`

	o := testGoReflectMethod_O{Test: "Test"}

	r := New()
	r.Set("o", &o)
	v, err := r.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	if !v.StrictEquals(valueTrue) {
		t.Fatalf("Expected true, got %v", v)
	}

	if o.Test != "Test" {
		t.Fatalf("Expected 'Test', got: '%s'", o.Test)
	}
}

func TestGoReflectRedefineFieldConfigurable(t *testing.T) {
	const SCRIPT = `
	var thrown = false;
	try {
		Object.defineProperty(o, "Test", {value: "AAA", configurable: true});
	} catch (e) {
		if (e instanceof TypeError) {
			thrown = true;
		} else {
			throw e;
		}
	}
	thrown;
	`

	o := testGoReflectMethod_O{Test: "Test"}

	r := New()
	r.Set("o", &o)
	v, err := r.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	if !v.StrictEquals(valueTrue) {
		t.Fatalf("Expected true, got %v", v)
	}

	if o.Test != "Test" {
		t.Fatalf("Expected 'Test', got: '%s'", o.Test)
	}
}

func TestGoReflectRedefineMethod(t *testing.T) {
	const SCRIPT = `
	var thrown = false;
	try {
		Object.defineProperty(o, "Method", {value: "AAA", configurable: true});
	} catch (e) {
		if (e instanceof TypeError) {
			thrown = true;
		} else {
			throw e;
		}
	}
	thrown;
	`

	o := testGoReflectMethod_O{Test: "Test"}

	r := New()
	r.Set("o", &o)
	v, err := r.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	if !v.StrictEquals(valueTrue) {
		t.Fatalf("Expected true, got %v", v)
	}
}

func TestGoReflectEmbeddedStruct(t *testing.T) {
	const SCRIPT = `
	if (o.ParentField2 !== "ParentField2") {
		throw new Error("ParentField2 = " + o.ParentField2);
	}

	if (o.Parent.ParentField2 !== 2) {
		throw new Error("o.Parent.ParentField2 = " + o.Parent.ParentField2);
	}

	if (o.ParentField1 !== 1) {
		throw new Error("o.ParentField1 = " + o.ParentField1);

	}

	if (o.ChildField !== 3) {
		throw new Error("o.ChildField = " + o.ChildField);
	}

	var keys = {};
	for (var k in o) {
		if (keys[k]) {
			throw new Error("Duplicate key: " + k);
		}
		keys[k] = true;
	}

	var expectedKeys = ["ParentField2", "ParentField1", "Parent", "ChildField"];
	for (var i in expectedKeys) {
		if (!keys[expectedKeys[i]]) {
			throw new Error("Missing key in enumeration: " + expectedKeys[i]);
		}
		delete keys[expectedKeys[i]];
	}

	var remainingKeys = Object.keys(keys);
	if (remainingKeys.length > 0) {
		throw new Error("Unexpected keys: " + remainingKeys);
	}

	o.ParentField2 = "ParentField22";
	o.Parent.ParentField2 = 22;
	o.ParentField1 = 11;
	o.ChildField = 33;
	`

	type Parent struct {
		ParentField1 int
		ParentField2 int
	}

	type Child struct {
		ParentField2 string
		Parent
		ChildField int
	}

	vm := New()
	o := Child{
		Parent: Parent{
			ParentField1: 1,
			ParentField2: 2,
		},
		ParentField2: "ParentField2",
		ChildField:   3,
	}
	vm.Set("o", &o)

	_, err := vm.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	if o.ParentField2 != "ParentField22" {
		t.Fatalf("ParentField2 = %q", o.ParentField2)
	}

	if o.Parent.ParentField2 != 22 {
		t.Fatalf("Parent.ParentField2 = %d", o.Parent.ParentField2)
	}

	if o.ParentField1 != 11 {
		t.Fatalf("ParentField1 = %d", o.ParentField1)
	}

	if o.ChildField != 33 {
		t.Fatalf("ChildField = %d", o.ChildField)
	}
}

type jsonTagNamer struct{}

func (jsonTagNamer) FieldName(_ reflect.Type, field reflect.StructField) string {
	if jsonTag := field.Tag.Get("json"); jsonTag != "" {
		return jsonTag
	}
	return field.Name
}

func (jsonTagNamer) MethodName(_ reflect.Type, method reflect.Method) string {
	return method.Name
}

func TestGoReflectCustomNaming(t *testing.T) {

	type testStructWithJsonTags struct {
		A string `json:"b"` // <-- script sees field "A" as property "b"
	}

	o := &testStructWithJsonTags{"Hello world"}
	r := New()
	r.SetFieldNameMapper(&jsonTagNamer{})
	r.Set("fn", func() *testStructWithJsonTags { return o })

	t.Run("get property", func(t *testing.T) {
		v, err := r.RunString(`fn().b`)
		if err != nil {
			t.Fatal(err)
		}
		if !v.StrictEquals(newStringValue(o.A)) {
			t.Fatalf("Expected %q, got %v", o.A, v)
		}
	})

	t.Run("set property", func(t *testing.T) {
		_, err := r.RunString(`fn().b = "Hello universe"`)
		if err != nil {
			t.Fatal(err)
		}
		if o.A != "Hello universe" {
			t.Fatalf("Expected \"Hello universe\", got %q", o.A)
		}
	})

	t.Run("enumerate properties", func(t *testing.T) {
		v, err := r.RunString(`Object.keys(fn())`)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(v.Export(), []interface{}{"b"}) {
			t.Fatalf("Expected [\"b\"], got %v", v.Export())
		}
	})
}

func TestGoReflectCustomObjNaming(t *testing.T) {

	type testStructWithJsonTags struct {
		A string `json:"b"` // <-- script sees field "A" as property "b"
	}

	r := New()
	r.SetFieldNameMapper(&jsonTagNamer{})

	t.Run("Set object in slice", func(t *testing.T) {
		testSlice := &[]testStructWithJsonTags{{"Hello world"}}
		r.Set("testslice", testSlice)
		_, err := r.RunString(`testslice[0] = {b:"setted"}`)
		if err != nil {
			t.Fatal(err)
		}
		if (*testSlice)[0].A != "setted" {
			t.Fatalf("Expected \"setted\", got %q", (*testSlice)[0])
		}
	})

	t.Run("Set object in map", func(t *testing.T) {
		testMap := map[string]testStructWithJsonTags{"key": {"Hello world"}}
		r.Set("testmap", testMap)
		_, err := r.RunString(`testmap["key"] = {b:"setted"}`)
		if err != nil {
			t.Fatal(err)
		}
		if testMap["key"].A != "setted" {
			t.Fatalf("Expected \"setted\", got %q", testMap["key"])
		}
	})

	t.Run("Add object to map", func(t *testing.T) {
		testMap := map[string]testStructWithJsonTags{}
		r.Set("testmap", testMap)
		_, err := r.RunString(`testmap["newkey"] = {b:"setted"}`)
		if err != nil {
			t.Fatal(err)
		}
		if testMap["newkey"].A != "setted" {
			t.Fatalf("Expected \"setted\", got %q", testMap["newkey"])
		}
	})
}

type fieldNameMapper1 struct{}

func (fieldNameMapper1) FieldName(_ reflect.Type, f reflect.StructField) string {
	return strings.ToLower(f.Name)
}

func (fieldNameMapper1) MethodName(_ reflect.Type, m reflect.Method) string {
	return m.Name
}

func TestNonStructAnonFields(t *testing.T) {
	type Test1 struct {
		M bool
	}
	type test3 []int
	type Test4 []int
	type Test2 struct {
		test3
		Test4
		*Test1
	}

	const SCRIPT = `
	JSON.stringify(a);
	a.m && a.test3 === undefined && a.test4.length === 2
	`
	vm := New()
	vm.SetFieldNameMapper(fieldNameMapper1{})
	vm.Set("a", &Test2{Test1: &Test1{M: true}, Test4: []int{1, 2}, test3: nil})
	v, err := vm.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}
	if !v.StrictEquals(valueTrue) {
		t.Fatalf("Unexepected result: %v", v)
	}
}

func TestStructNonAddressable(t *testing.T) {
	type S struct {
		Field int
	}

	const SCRIPT = `
	"use strict";
	
	if (!Object.getOwnPropertyDescriptor(s, "Field").writable) {
		throw new Error("s.Field is non-writable");
	}

	if (!Object.getOwnPropertyDescriptor(s1, "Field").writable) {
		throw new Error("s1.Field is non-writable");
	}

	s1.Field = 42;
	s.Field = 43;
	s;
`

	var s S
	vm := New()
	vm.Set("s", s)
	vm.Set("s1", &s)
	v, err := vm.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}
	exp := v.Export()
	if s1, ok := exp.(S); ok {
		if s1.Field != 43 {
			t.Fatal(s1)
		}
	} else {
		t.Fatalf("Wrong type: %T", exp)
	}
	if s.Field != 42 {
		t.Fatalf("Unexpected s.Field value: %d", s.Field)
	}
}

type testFieldMapper struct {
}

func (testFieldMapper) FieldName(_ reflect.Type, f reflect.StructField) string {
	if tag := f.Tag.Get("js"); tag != "" {
		if tag == "-" {
			return ""
		}
		return tag
	}

	return f.Name
}

func (testFieldMapper) MethodName(_ reflect.Type, m reflect.Method) string {
	return m.Name
}

func TestHidingAnonField(t *testing.T) {
	type InnerType struct {
		AnotherField string
	}

	type OuterType struct {
		InnerType `js:"-"`
		SomeField string
	}

	const SCRIPT = `
	var a = Object.getOwnPropertyNames(o);
	if (a.length !== 2) {
		throw new Error("unexpected length: " + a.length);
	}

	if (a.indexOf("SomeField") === -1) {
		throw new Error("no SomeField");
	}

	if (a.indexOf("AnotherField") === -1) {
		throw new Error("no SomeField");
	}
	`

	var o OuterType

	vm := New()
	vm.SetFieldNameMapper(testFieldMapper{})
	vm.Set("o", &o)

	_, err := vm.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}
}

func TestFieldOverriding(t *testing.T) {
	type InnerType struct {
		AnotherField  string
		AnotherField1 string
	}

	type OuterType struct {
		InnerType     `js:"-"`
		SomeField     string
		AnotherField  string `js:"-"`
		AnotherField1 string
	}

	const SCRIPT = `
	if (o.SomeField !== "SomeField") {
		throw new Error("SomeField");
	}

	if (o.AnotherField !== "AnotherField inner") {
		throw new Error("AnotherField");
	}

	if (o.AnotherField1 !== "AnotherField1 outer") {
		throw new Error("AnotherField1");
	}

	if (o.InnerType) {
		throw new Error("InnerType is present");
	}
	`

	o := OuterType{
		InnerType: InnerType{
			AnotherField:  "AnotherField inner",
			AnotherField1: "AnotherField1 inner",
		},
		SomeField:     "SomeField",
		AnotherField:  "AnotherField outer",
		AnotherField1: "AnotherField1 outer",
	}

	vm := New()
	vm.SetFieldNameMapper(testFieldMapper{})
	vm.Set("o", &o)

	_, err := vm.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDefinePropertyUnexportedJsName(t *testing.T) {
	type T struct {
		Field      int
		unexported int
	}

	vm := New()
	vm.SetFieldNameMapper(fieldNameMapper1{})
	vm.Set("f", &T{unexported: 0})

	_, err := vm.RunString(`
	"use strict";
	Object.defineProperty(f, "field", {value: 42});
	if (f.field !== 42) {
		throw new Error("Unexpected value: " + f.field);
	}
	if (f.hasOwnProperty("unexported")) {
		throw new Error("hasOwnProperty('unexported') is true");
	}
	var thrown;
	try {
		Object.defineProperty(f, "unexported", {value: 1});
	} catch (e) {
		thrown = e;
	}
	if (!(thrown instanceof TypeError)) {
		throw new Error("Unexpected error: ", thrown);
	}
	`)
	if err != nil {
		t.Fatal(err)
	}
}

type fieldNameMapperToLower struct{}

func (fieldNameMapperToLower) FieldName(_ reflect.Type, f reflect.StructField) string {
	return strings.ToLower(f.Name)
}

func (fieldNameMapperToLower) MethodName(_ reflect.Type, m reflect.Method) string {
	return strings.ToLower(m.Name)
}

func TestHasOwnPropertyUnexportedJsName(t *testing.T) {
	vm := New()
	vm.SetFieldNameMapper(fieldNameMapperToLower{})
	vm.Set("f", &testGoReflectMethod_O{})

	_, err := vm.RunString(`
	"use strict";
	if (!f.hasOwnProperty("test")) {
		throw new Error("hasOwnProperty('test') returned false");
	}
	if (!f.hasOwnProperty("method")) {
		throw new Error("hasOwnProperty('method') returned false");
	}
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func BenchmarkGoReflectGet(b *testing.B) {
	type parent struct {
		field, Test1, Test2, Test3, Test4, Test5, Test string
	}

	type child struct {
		parent
		Test6 string
	}

	b.StopTimer()
	vm := New()

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		v := vm.ToValue(child{parent: parent{Test: "Test", field: ""}}).(*Object)
		v.Get("Test")
	}
}

func TestNestedStructSet(t *testing.T) {
	type B struct {
		Field int
	}
	type A struct {
		B B
	}

	const SCRIPT = `
	'use strict';
	a.B.Field++;
	if (a1.B.Field != 1) {
		throw new Error("a1.B.Field = " + a1.B.Field);
	}
	var d = Object.getOwnPropertyDescriptor(a1.B, "Field");
	if (!d.writable) {
		throw new Error("a1.B is not writable");
	}
	a1.B.Field = 42;
	a1;
	`
	a := A{
		B: B{
			Field: 1,
		},
	}
	vm := New()
	vm.Set("a", &a)
	vm.Set("a1", a)
	v, err := vm.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}
	exp := v.Export()
	if v, ok := exp.(A); ok {
		if v.B.Field != 42 {
			t.Fatal(v)
		}
	} else {
		t.Fatalf("Wrong type: %T", exp)
	}

	if v := a.B.Field; v != 2 {
		t.Fatalf("Unexpected a.B.Field: %d", v)
	}
}

func TestStructNonAddressableAnonStruct(t *testing.T) {

	type C struct {
		Z int64
		X string
	}

	type B struct {
		C
		Y string
	}

	type A struct {
		B B
	}

	a := A{
		B: B{
			C: C{
				Z: 1,
				X: "X2",
			},
			Y: "Y3",
		},
	}
	const SCRIPT = `
	"use strict";
	var s = JSON.stringify(a);
	s;
`

	vm := New()
	vm.Set("a", &a)
	v, err := vm.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}

	expected := `{"B":{"C":{"Z":1,"X":"X2"},"Z":1,"X":"X2","Y":"Y3"}}`
	if expected != v.String() {
		t.Fatalf("Expected '%s', got '%s'", expected, v.String())
	}

}

func TestTagFieldNameMapperInvalidId(t *testing.T) {
	vm := New()
	vm.SetFieldNameMapper(TagFieldNameMapper("json", true))
	type S struct {
		Field int `json:"-"`
	}
	vm.Set("s", S{Field: 42})
	res, err := vm.RunString(`s.hasOwnProperty("field") || s.hasOwnProperty("Field")`)
	if err != nil {
		t.Fatal(err)
	}
	if res != valueFalse {
		t.Fatalf("Unexpected result: %v", res)
	}
}

func TestPrimitivePtr(t *testing.T) {
	vm := New()
	s := "test"
	vm.Set("s", &s)
	res, err := vm.RunString(`s instanceof String && s == "test"`) // note non-strict equality
	if err != nil {
		t.Fatal(err)
	}
	if v := res.ToBoolean(); !v {
		t.Fatalf("value: %#v", res)
	}
	s = "test1"
	res, err = vm.RunString(`s == "test1"`)
	if err != nil {
		t.Fatal(err)
	}
	if v := res.ToBoolean(); !v {
		t.Fatalf("value: %#v", res)
	}
}

func TestStringer(t *testing.T) {
	vm := New()
	vm.Set("e", errors.New("test"))
	res, err := vm.RunString("e.toString()")
	if err != nil {
		t.Fatal(err)
	}
	if v := res.Export(); v != "test" {
		t.Fatalf("v: %v", v)
	}
}

func ExampleTagFieldNameMapper() {
	vm := New()
	vm.SetFieldNameMapper(TagFieldNameMapper("json", true))
	type S struct {
		Field int `json:"field"`
	}
	vm.Set("s", S{Field: 42})
	res, _ := vm.RunString(`s.field`)
	fmt.Println(res.Export())
	// Output: 42
}

func ExampleUncapFieldNameMapper() {
	vm := New()
	s := testGoReflectMethod_O{
		Test: "passed",
	}
	vm.SetFieldNameMapper(UncapFieldNameMapper())
	vm.Set("s", s)
	res, _ := vm.RunString(`s.test + " and " + s.method("passed too")`)
	fmt.Println(res.Export())
	// Output: passed and passed too
}

func TestGoReflectWithProto(t *testing.T) {
	type S struct {
		Field int
	}
	var s S
	vm := New()
	vm.Set("s", &s)
	vm.testScriptWithTestLib(`
	(function() {
	'use strict';
	var proto = {
		Field: "protoField",
		test: 42
	};
	var test1Holder;
	Object.defineProperty(proto, "test1", {
		set: function(v) {
			test1Holder = v;
		},
		get: function() {
			return test1Holder;
		}
	});
	Object.setPrototypeOf(s, proto);
	assert.sameValue(s.Field, 0, "s.Field");
	s.Field = 2;
	assert.sameValue(s.Field, 2, "s.Field");
	assert.sameValue(s.test, 42, "s.test");
	assert.throws(TypeError, function() {
		Object.defineProperty(s, "test", {value: 43});
	});
	test1Holder = 1;
	assert.sameValue(s.test1, 1, "s.test1");
	s.test1 = 2;
	assert.sameValue(test1Holder, 2, "test1Holder");
	})();
	`, _undefined, t)
}

func TestGoReflectSymbols(t *testing.T) {
	type S struct {
		Field int
	}
	var s S
	vm := New()
	vm.Set("s", &s)
	_, err := vm.RunString(`
	'use strict';
	var sym = Symbol(66);
	s[sym] = "Test";
	if (s[sym] !== "Test") {
		throw new Error("s[sym]=" + s[sym]);
	}
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGoReflectSymbolEqualityQuirk(t *testing.T) {
	type Field struct {
	}
	type S struct {
		Field *Field
	}
	var s = S{
		Field: &Field{},
	}
	vm := New()
	vm.Set("s", &s)
	res, err := vm.RunString(`
	var sym = Symbol(66);
	var field1 = s.Field;
	field1[sym] = true;
	var field2 = s.Field;
	// Because a wrapper is created every time the property is accessed
	// field1 and field2 will be different instances of the wrapper.
	// Symbol properties only exist in the wrapper, they cannot be placed into the original Go value,
	// hence the following:
	field1 === field2 && field1[sym] === true && field2[sym] === undefined;
	`)
	if err != nil {
		t.Fatal(err)
	}
	if res != valueTrue {
		t.Fatal(res)
	}
}

func TestGoObj__Proto__(t *testing.T) {
	type S struct {
		Field int
	}
	vm := New()
	vm.Set("s", S{})
	vm.Set("m", map[string]interface{}{})
	vm.Set("mr", map[int]string{})
	vm.Set("a", []interface{}{})
	vm.Set("ar", []string{})
	_, err := vm.RunString(`
	function f(s, expectedCtor, prefix) {
		if (s.__proto__ !== expectedCtor.prototype) {
			throw new Error(prefix + ": __proto__: " + s.__proto__);
		}
		s.__proto__ = null;
		if (s.__proto__ !== undefined) { // as there is no longer a prototype, there is no longer the __proto__ property
			throw new Error(prefix + ": __proto__ is not undefined: " + s.__proto__);
		}
		var proto = Object.getPrototypeOf(s);
		if (proto !== null) {
			throw new Error(prefix + ": proto is not null: " + proto);
		}
	}
	f(s, Object, "struct");
	f(m, Object, "simple map");
	f(mr, Object, "reflect map");
	f(a, Array, "slice");
	f(ar, Array, "reflect slice");
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGoReflectUnicodeProps(t *testing.T) {
	type S struct {
		Тест string
	}
	vm := New()
	var s S
	vm.Set("s", &s)
	_, err := vm.RunString(`
	if (!s.hasOwnProperty("Тест")) {
		throw new Error("hasOwnProperty");
	}
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGoReflectPreserveType(t *testing.T) {
	vm := New()
	var expect = time.Duration(math.MaxInt64)
	vm.Set(`make`, func() time.Duration {
		return expect
	})
	vm.Set(`handle`, func(d time.Duration) {
		if d.String() != expect.String() {
			t.Fatal(`expect`, expect, `, but get`, d)
		}
	})
	_, e := vm.RunString(`
	var d=make()
	handle(d)
	`)
	if e != nil {
		t.Fatal(e)
	}
}

func TestGoReflectCopyOnWrite(t *testing.T) {
	type Inner struct {
		Field int
	}
	type S struct {
		I Inner
	}
	var s S
	s.I.Field = 1

	vm := New()
	vm.Set("s", &s)
	_, err := vm.RunString(`
		if (s.I.Field !== 1) {
			throw new Error("s.I.Field: " + s.I.Field);
		}

		let tmp = s.I; // tmp becomes a reference to s.I
		if (tmp.Field !== 1) {
			throw new Error("tmp.Field: " + tmp.Field);
		}

		s.I.Field = 2;
		if (s.I.Field !== 2) {
			throw new Error("s.I.Field (1): " + s.I.Field);
		}
		if (tmp.Field !== 2) {
			throw new Error("tmp.Field (1): " + tmp.Field);
		}

		s.I = {Field: 3}; // at this point tmp is changed to a copy
		if (s.I.Field !== 3) {
			throw new Error("s.I.Field (2): " + s.I.Field);
		}
		if (tmp.Field !== 2) {
			throw new Error("tmp.Field (2): " + tmp.Field);
		}
	`)

	if err != nil {
		t.Fatal(err)
	}
}

func TestReflectSetReflectValue(t *testing.T) {
	o := []testGoReflectMethod_O{{}}
	vm := New()
	vm.Set("o", o)
	_, err := vm.RunString(`
		const t = o[0];
		t.Set("a");
		o[0] = {};
		o[0].Set("b");
		if (t.Get() !== "a") {
			throw new Error();
		}
	`)

	if err != nil {
		t.Fatal(err)
	}
}

func TestReflectOverwriteReflectMap(t *testing.T) {
	vm := New()
	type S struct {
		M map[int]interface{}
	}
	var s S
	s.M = map[int]interface{}{
		0: true,
	}
	vm.Set("s", &s)
	_, err := vm.RunString(`
	s.M = {1: false};
	`)
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := s.M[0]; exists {
		t.Fatal(s)
	}
}

type testBoolS bool

func (testBoolS) String() string {
	return "B"
}

type testIntS int

func (testIntS) String() string {
	return "I"
}

type testStringS string

func (testStringS) String() string {
	return "S"
}

func TestGoReflectToPrimitive(t *testing.T) {
	vm := New()

	f := func(expr string, expected Value, t *testing.T) {
		v, err := vm.RunString(expr)
		if err != nil {
			t.Fatal(err)
		}
		if IsNaN(expected) {
			if IsNaN(v) {
				return
			}
		} else {
			if v.StrictEquals(expected) {
				return
			}
		}
		t.Fatalf("%s: expected: %v, actual: %v", expr, expected, v)
	}

	t.Run("Not Stringers", func(t *testing.T) {
		type Bool bool
		var b Bool = true

		t.Run("Bool", func(t *testing.T) {
			vm.Set("b", b)
			f("+b", intToValue(1), t)
			f("`${b}`", asciiString("true"), t)
			f("b.toString()", asciiString("true"), t)
			f("b.valueOf()", valueTrue, t)
		})

		t.Run("*Bool", func(t *testing.T) {
			vm.Set("b", &b)
			f("+b", intToValue(1), t)
			f("`${b}`", asciiString("true"), t)
			f("b.toString()", asciiString("true"), t)
			f("b.valueOf()", valueTrue, t)
		})

		type Int int
		var i Int = 1

		t.Run("Int", func(t *testing.T) {
			vm.Set("i", i)
			f("+i", intToValue(1), t)
			f("`${i}`", asciiString("1"), t)
			f("i.toString()", asciiString("1"), t)
			f("i.valueOf()", intToValue(1), t)
		})

		t.Run("*Int", func(t *testing.T) {
			vm.Set("i", &i)
			f("+i", intToValue(1), t)
			f("`${i}`", asciiString("1"), t)
			f("i.toString()", asciiString("1"), t)
			f("i.valueOf()", intToValue(1), t)
		})

		type Uint uint
		var ui Uint = 1

		t.Run("Uint", func(t *testing.T) {
			vm.Set("ui", ui)
			f("+ui", intToValue(1), t)
			f("`${ui}`", asciiString("1"), t)
			f("ui.toString()", asciiString("1"), t)
			f("ui.valueOf()", intToValue(1), t)
		})

		t.Run("*Uint", func(t *testing.T) {
			vm.Set("ui", &i)
			f("+ui", intToValue(1), t)
			f("`${ui}`", asciiString("1"), t)
			f("ui.toString()", asciiString("1"), t)
			f("ui.valueOf()", intToValue(1), t)
		})

		type Float float64
		var fl Float = 1.1

		t.Run("Float", func(t *testing.T) {
			vm.Set("fl", fl)
			f("+fl", floatToValue(1.1), t)
			f("`${fl}`", asciiString("1.1"), t)
			f("fl.toString()", asciiString("1.1"), t)
			f("fl.valueOf()", floatToValue(1.1), t)
		})

		t.Run("*Float", func(t *testing.T) {
			vm.Set("fl", &fl)
			f("+fl", floatToValue(1.1), t)
			f("`${fl}`", asciiString("1.1"), t)
			f("fl.toString()", asciiString("1.1"), t)
			f("fl.valueOf()", floatToValue(1.1), t)
		})

		fl = Float(math.Inf(1))
		t.Run("FloatInf", func(t *testing.T) {
			vm.Set("fl", fl)
			f("+fl", _positiveInf, t)
			f("fl.toString()", asciiString("Infinity"), t)
		})

		type Empty struct{}

		var e Empty
		t.Run("Empty", func(t *testing.T) {
			vm.Set("e", &e)
			f("+e", _NaN, t)
			f("`${e}`", asciiString("[object Object]"), t)
			f("e.toString()", asciiString("[object Object]"), t)
			f("e.valueOf()", vm.ToValue(&e), t)
		})
	})

	t.Run("Stringers", func(t *testing.T) {
		var b testBoolS = true
		t.Run("Bool", func(t *testing.T) {
			vm.Set("b", b)
			f("`${b}`", asciiString("B"), t)
			f("b.toString()", asciiString("B"), t)
			f("b.valueOf()", valueTrue, t)
			f("+b", intToValue(1), t)
		})

		t.Run("*Bool", func(t *testing.T) {
			vm.Set("b", &b)
			f("`${b}`", asciiString("B"), t)
			f("b.toString()", asciiString("B"), t)
			f("b.valueOf()", valueTrue, t)
			f("+b", intToValue(1), t)
		})

		var i testIntS = 1
		t.Run("Int", func(t *testing.T) {
			vm.Set("i", i)
			f("`${i}`", asciiString("I"), t)
			f("i.toString()", asciiString("I"), t)
			f("i.valueOf()", intToValue(1), t)
			f("+i", intToValue(1), t)
		})

		t.Run("*Int", func(t *testing.T) {
			vm.Set("i", &i)
			f("`${i}`", asciiString("I"), t)
			f("i.toString()", asciiString("I"), t)
			f("i.valueOf()", intToValue(1), t)
			f("+i", intToValue(1), t)
		})

		var s testStringS
		t.Run("String", func(t *testing.T) {
			vm.Set("s", s)
			f("`${s}`", asciiString("S"), t)
			f("s.toString()", asciiString("S"), t)
			f("s.valueOf()", asciiString("S"), t)
			f("+s", _NaN, t)
		})

		t.Run("*String", func(t *testing.T) {
			vm.Set("s", &s)
			f("`${s}`", asciiString("S"), t)
			f("s.toString()", asciiString("S"), t)
			f("s.valueOf()", asciiString("S"), t)
			f("+s", _NaN, t)
		})
	})
}

type testGoReflectFuncRt struct {
}

func (*testGoReflectFuncRt) M(call FunctionCall, r *Runtime) Value {
	if r == nil {
		panic(typeError("Runtime is nil"))
	}
	return call.Argument(0)
}

func (*testGoReflectFuncRt) C(call ConstructorCall, r *Runtime) *Object {
	if r == nil {
		panic(typeError("Runtime is nil in constructor"))
	}
	call.This.Set("r", call.Argument(0))
	return nil
}

func TestGoReflectFuncWithRuntime(t *testing.T) {
	vm := New()
	var s testGoReflectFuncRt
	vm.Set("s", &s)
	res, err := vm.RunString("s.M(true)")
	if err != nil {
		t.Fatal(err)
	}
	if res != valueTrue {
		t.Fatal(res)
	}

	res, err = vm.RunString("new s.C(true).r")
	if err != nil {
		t.Fatal(err)
	}
	if res != valueTrue {
		t.Fatal(res)
	}
}

func TestGoReflectDefaultToString(t *testing.T) {
	var s testStringS
	vm := New()
	v := vm.ToValue(s).(*Object)
	v.Delete("toString")
	v.Delete("valueOf")
	vm.Set("s", v)
	_, err := vm.RunString(`
		class S {
			toString() {
				return "X";
			}
		}

		if (s.toString() !== "S") {
			throw new Error(s.toString());
		}
		if (("" + s) !== "S") {
			throw new Error("" + s);
		}

		Object.setPrototypeOf(s, S.prototype);
		if (s.toString() !== "X") {
			throw new Error(s.toString());
		}
		if (("" + s) !== "X") {
			throw new Error("" + s);
		}
	`)
	if err != nil {
		t.Fatal(err)
	}
}
