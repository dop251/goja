package goja

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestJSONMarshalObject(t *testing.T) {
	vm := New()
	o := vm.NewObject()
	o.Set("test", 42)
	o.Set("testfunc", vm.Get("Error"))
	b, err := json.Marshal(o)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"test":42}` {
		t.Fatalf("Unexpected value: %s", b)
	}
}

func TestJSONMarshalObjectCircular(t *testing.T) {
	vm := New()
	o := vm.NewObject()
	o.Set("o", o)
	_, err := json.Marshal(o)
	if err == nil {
		t.Fatal("Expected error")
	}
	if !strings.HasSuffix(err.Error(), "Converting circular structure to JSON") {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func BenchmarkJSONStringify(b *testing.B) {
	b.StopTimer()
	vm := New()
	var createObj func(level int) *Object
	createObj = func(level int) *Object {
		o := vm.NewObject()
		o.Set("field1", "test")
		o.Set("field2", 42)
		if level > 0 {
			level--
			o.Set("obj1", createObj(level))
			o.Set("obj2", createObj(level))
		}
		return o
	}

	o := createObj(3)
	json := vm.Get("JSON").(*Object)
	stringify, _ := AssertFunction(json.Get("stringify"))
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		stringify(nil, o)
	}
}
