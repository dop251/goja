package goja

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
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

func TestJSONMarshalGoDate(t *testing.T) {
	vm := New()
	o := vm.NewObject()
	o.Set("test", time.Unix(86400, 0).UTC())
	b, err := json.Marshal(o)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"test":"1970-01-02T00:00:00Z"}` {
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

func TestJSONParseReviver(t *testing.T) {
	// example from
	// https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/JSON/parse
	const SCRIPT = `
	JSON.parse('{"p": 5}', function(key, value) {
	  return typeof value === 'number'
        ? value * 2 // return value * 2 for numbers
	    : value     // return everything else unchanged
	 })["p"]
	`

	testScript1(SCRIPT, intToValue(10), t)
}

func TestQuoteMalformedSurrogatePair(t *testing.T) {
	testScript1(`JSON.stringify("\uD800")`, asciiString(`"\ud800"`), t)
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
