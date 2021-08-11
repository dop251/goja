package goja

import (
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

func TestArrayExportProps(t *testing.T) {
	vm := New()
	arr := vm.NewArray()
	err := arr.DefineDataProperty("0", vm.ToValue(true), FLAG_TRUE, FLAG_FALSE, FLAG_TRUE)
	if err != nil {
		t.Fatal(err)
	}
	actual := arr.Export()
	expected := []interface{}{true}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("Expected: %#v, actual: %#v", expected, actual)
	}
}

func TestArrayCanonicalIndex(t *testing.T) {
	const SCRIPT = `
	var a = [];
	a["00"] = 1;
	a["01"] = 2;
	if (a[0] !== undefined) {
		throw new Error("a[0]");
	}
	`

	testScript1(SCRIPT, _undefined, t)
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

func BenchmarkArraySetEmpty(b *testing.B) {
	r := New()
	r.Get("Array").(*Object).Get("prototype").String() // materialise Array.prototype
	a := r.NewArray(0, 0)
	values := a.self.(*arrayObject).values
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		values[0] = nil
		a.self.setOwnIdx(0, valueTrue, true)
	}
}
