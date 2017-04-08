package goja

import (
	"reflect"
	"strings"
	"testing"
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

	if s, ok := v.assertString(); ok {
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
}

type TestGoReflectMethod_Struct struct {
}

func (s *TestGoReflectMethod_Struct) M() int {
	return 42
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
	!!Object.defineProperty(o, "Test", {value: "AAA"});
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

func (*jsonTagNamer) FieldName(t reflect.Type, field reflect.StructField) string {
	if jsonTag := field.Tag.Get("json"); jsonTag != "" {
		return jsonTag
	}
	return field.Name
}

func (*jsonTagNamer) MethodName(t reflect.Type, method reflect.Method) string {
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

type fieldNameMapper1 struct{}

func (fieldNameMapper1) FieldName(t reflect.Type, f reflect.StructField) string {
	return strings.ToLower(f.Name)
}

func (fieldNameMapper1) MethodName(t reflect.Type, m reflect.Method) string {
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
	vm.Set("a", &Test2{Test1: &Test1{M: true}, Test4: []int{1, 2}})
	v, err := vm.RunString(SCRIPT)
	if err != nil {
		t.Fatal(err)
	}
	if !v.StrictEquals(valueTrue) {
		t.Fatalf("Unexepected result: %v", v)
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
		v := vm.ToValue(child{parent: parent{Test: "Test"}}).(*Object)
		v.Get("Test")
	}
}
