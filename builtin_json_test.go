package goja

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"testing"
	"time"

	"github.com/dop251/goja/unistring"
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

func TestJSONStringifyCircularWrappedGo(t *testing.T) {
	type CircularType struct {
		Self *CircularType
	}
	vm := New()
	v := CircularType{}
	v.Self = &v
	vm.Set("v", &v)
	_, err := vm.RunString("JSON.stringify(v)")
	if err == nil {
		t.Fatal("Expected error")
	}
	if !strings.HasPrefix(err.Error(), "TypeError: Converting circular structure to JSON") {
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

	testScript(SCRIPT, intToValue(10), t)
}

func TestQuoteMalformedSurrogatePair(t *testing.T) {
	testScript(`JSON.stringify("\uD800")`, asciiString(`"\ud800"`), t)
}

func TestEOFWrapping(t *testing.T) {
	vm := New()

	_, err := vm.RunString("JSON.parse('{')")
	if err == nil {
		t.Fatal("Expected error")
	}

	if !strings.Contains(err.Error(), "Unexpected end of JSON input") {
		t.Fatalf("Error doesn't contain human-friendly wrapper: %v", err)
	}
}

type testMarshalJSONErrorStruct struct {
	e error
}

func (s *testMarshalJSONErrorStruct) MarshalJSON() ([]byte, error) {
	return nil, s.e
}

func TestMarshalJSONError(t *testing.T) {
	vm := New()
	v := testMarshalJSONErrorStruct{e: errors.New("test error")}
	vm.Set("v", &v)
	_, err := vm.RunString("JSON.stringify(v)")
	if !errors.Is(err, v.e) {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestJSONParseConformance(t *testing.T) {
	const SCRIPT = `
	function assert(v, msg) { if (!v) { throw new Error(msg); } }
	function assertThrowsSyntax(fn, msg) {
		try {
			fn();
		} catch (e) {
			assert(e instanceof SyntaxError, msg + ": expected SyntaxError, got " + e);
			return;
		}
		throw new Error(msg + ": expected to throw");
	}

	// numbers
	assert(JSON.parse('42') === 42, "int");
	assert(JSON.parse('-1') === -1, "negative int");
	assert(Object.is(JSON.parse('-0'), -0), "-0");
	assert(Object.is(JSON.parse('-0.0'), -0), "-0.0");
	assert(JSON.parse('1e999') === Infinity, "1e999");
	assert(JSON.parse('-1e999') === -Infinity, "-1e999");
	assert(JSON.parse('9007199254740993') === 9007199254740992, "2^53+1 rounds");
	assert(JSON.parse('123456789012345678901234567890') === 1.2345678901234568e+29, "big int to float");
	assert(JSON.parse('1.5') === 1.5, "float");
	assert(JSON.parse('1.0') === 1, "1.0");
	assert(JSON.parse('0.5e2') === 50, "exponent");
	assert(JSON.parse('5E-1') === 0.5, "neg exponent");
	assert(JSON.parse('1e+2') === 100, "plus exponent");

	// strings
	assert(JSON.parse('"abc"') === "abc", "plain string");
	assert(JSON.parse('""') === "", "empty string");
	assert(JSON.parse('"\\u0041"') === "A", "\\u escape");
	assert(JSON.parse('"\\ud800"').charCodeAt(0) === 0xD800, "lone high surrogate preserved");
	assert(JSON.parse('"\\udc00"').charCodeAt(0) === 0xDC00, "lone low surrogate preserved");
	assert(JSON.parse('"\\ud83d\\ude00"') === "\ud83d\ude00", "surrogate pair");
	assert(JSON.parse('"\\"\\\\\\/\\b\\f\\n\\r\\t"') === "\"\\/\b\f\n\r\t", "escapes");
	assert(JSON.parse('"héllo日本"') === "héllo日本", "raw unicode");

	// literals, arrays, objects
	assert(JSON.parse('true') === true, "true");
	assert(JSON.parse('false') === false, "false");
	assert(JSON.parse('null') === null, "null");
	assert(JSON.parse('[]').length === 0, "empty array");
	assert(JSON.parse('[1,[2,[3]]]')[1][1][0] === 3, "nested array");
	assert(Object.keys(JSON.parse('{}')).length === 0, "empty object");
	assert(JSON.parse('{"a":{"b":2}}').a.b === 2, "nested object");
	assert(JSON.parse('{"a":1,"a":2}').a === 2, "duplicate key last wins");
	assert(JSON.parse(' \t\r\n1 \t\r\n' ) === 1, "whitespace");
	var protoObj = JSON.parse('{"__proto__":{"x":1}}');
	assert(Object.getPrototypeOf(protoObj) === Object.prototype, "__proto__ is own property");
	assert(protoObj.__proto__.x === 1, "__proto__ value");
	var uniKeys = JSON.parse('{"ключ":1,"\\u043a\\u043b\\u044e\\u04472":2}');
	assert(uniKeys["ключ"] === 1 && uniKeys["ключ2"] === 2, "unicode keys");

	// syntax errors
	assertThrowsSyntax(function() { JSON.parse('') }, "empty");
	assertThrowsSyntax(function() { JSON.parse('  ') }, "whitespace only");
	assertThrowsSyntax(function() { JSON.parse('{') }, "unterminated object");
	assertThrowsSyntax(function() { JSON.parse('[1,') }, "unterminated array");
	assertThrowsSyntax(function() { JSON.parse('"abc') }, "unterminated string");
	assertThrowsSyntax(function() { JSON.parse('tru') }, "bad literal");
	assertThrowsSyntax(function() { JSON.parse('truex') }, "literal with trailing");
	assertThrowsSyntax(function() { JSON.parse('01') }, "leading zero");
	assertThrowsSyntax(function() { JSON.parse('1.') }, "trailing dot");
	assertThrowsSyntax(function() { JSON.parse('.5') }, "leading dot");
	assertThrowsSyntax(function() { JSON.parse('+1') }, "leading plus");
	assertThrowsSyntax(function() { JSON.parse('-') }, "bare minus");
	assertThrowsSyntax(function() { JSON.parse('1e') }, "bare exponent");
	assertThrowsSyntax(function() { JSON.parse('NaN') }, "NaN");
	assertThrowsSyntax(function() { JSON.parse("'abc'") }, "single quotes");
	assertThrowsSyntax(function() { JSON.parse('[1,]') }, "trailing comma array");
	assertThrowsSyntax(function() { JSON.parse('{"a":1,}') }, "trailing comma object");
	assertThrowsSyntax(function() { JSON.parse('{"a" 1}') }, "missing colon");
	assertThrowsSyntax(function() { JSON.parse('{a:1}') }, "unquoted key");
	assertThrowsSyntax(function() { JSON.parse('1 2') }, "trailing garbage");
	assertThrowsSyntax(function() { JSON.parse('"a\tb"') }, "raw control char");
	assertThrowsSyntax(function() { JSON.parse('"\\uZZZZ"') }, "bad hex escape");
	assertThrowsSyntax(function() { JSON.parse('"\\x41"') }, "bad escape");

	// round-trip
	var obj = {a: [1, -2.5, "x", true, null, {b: "\ud800\u00e9"}], c: -0.125};
	assert(JSON.stringify(JSON.parse(JSON.stringify(obj))) === JSON.stringify(obj), "round trip");
	`
	testScript(SCRIPT, _undefined, t)
}

func TestJSONParseDepthLimit(t *testing.T) {
	vm := New()
	_, err := vm.RunString(`JSON.parse("[".repeat(20000) + "]".repeat(20000))`)
	if err == nil {
		t.Fatal("Expected error")
	}
	if !strings.Contains(err.Error(), "SyntaxError") {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestJSONParsedObjectShapes(t *testing.T) {
	vm := New()
	value, err := vm.RunString(`JSON.parse('[{"a":1,"b":2},{"a":3,"b":4},{"b":5,"a":6}]')`)
	if err != nil {
		t.Fatal(err)
	}
	array := value.(*Object)
	first := array.self.getIdx(0, nil).(*Object)
	second := array.self.getIdx(1, nil).(*Object)
	third := array.self.getIdx(2, nil).(*Object)
	firstImpl, ok := first.self.(*jsonObject)
	if !ok {
		t.Fatalf("expected compact JSON object, got %T", first.self)
	}
	secondImpl, ok := second.self.(*jsonObject)
	if !ok {
		t.Fatalf("expected compact JSON object, got %T", second.self)
	}
	thirdImpl, ok := third.self.(*jsonObject)
	if !ok {
		t.Fatalf("expected compact JSON object, got %T", third.self)
	}
	if firstImpl.shape != secondImpl.shape {
		t.Fatal("objects with the same ordered keys did not share a shape")
	}
	if firstImpl.shape == thirdImpl.shape {
		t.Fatal("objects with different key order unexpectedly shared a shape")
	}

	if err := first.Set("a", 10); err != nil {
		t.Fatal(err)
	}
	if _, ok := first.self.(*jsonObject); !ok {
		t.Fatal("replacing an existing value materialized the object")
	}
	if err := first.Set("c", 20); err != nil {
		t.Fatal(err)
	}
	if _, ok := first.self.(*baseObject); !ok {
		t.Fatalf("adding a property did not materialize the object: %T", first.self)
	}
	if got := first.Get("a").ToInteger(); got != 10 {
		t.Fatalf("existing property was not preserved: %d", got)
	}
}

func TestJSONParsedObjectMutationSemantics(t *testing.T) {
	const script = `
	function assert(value, message) {
		if (!value) throw new Error(message);
	}

	let ordered = JSON.parse('{"z":0,"10":10,"2":2,"a":1,"2":22}');
	assert(Object.keys(ordered).join(",") === "2,10,z,a", "property order");
	assert(ordered[2] === 22, "duplicate key value");
	let descriptor = Object.getOwnPropertyDescriptor(ordered, "a");
	assert(descriptor.writable && descriptor.enumerable && descriptor.configurable, "descriptor defaults");

	ordered.a = 7;
	Object.defineProperty(ordered, "hidden", {value: 8});
	Object.defineProperty(ordered, "computed", {
		get() { return this.a + 1; }, enumerable: true, configurable: true
	});
	delete ordered.z;
	Object.setPrototypeOf(ordered, {inherited: 9});
	assert(ordered.a === 7 && ordered.computed === 8 && ordered.inherited === 9, "materialized reads");
	assert(Object.keys(ordered).join(",") === "2,10,a,computed", "materialized enumeration");
	assert(!Object.prototype.hasOwnProperty.call(ordered, "z"), "materialized delete");

	let revived = JSON.parse('{"keep":1,"drop":2,"nested":{"value":3}}', function(key, value) {
		if (key === "drop") return undefined;
		if (typeof value === "number") return value * 2;
		return value;
	});
	assert(revived.keep === 2 && !("drop" in revived) && revived.nested.value === 6, "reviver");

	let parsedPrototype = JSON.parse('{"x":1}');
	let child = Object.create(parsedPrototype);
	child.x = 2;
	assert(child.x === 2 && parsedPrototype.x === 1, "prototype assignment");
	assert(Object.prototype.hasOwnProperty.call(child, "x"), "receiver property");
	Object.freeze(parsedPrototype);
	assert(Object.isFrozen(parsedPrototype), "freeze");

	let changing = JSON.parse('{"a":{"x":1},"b":2,"c":3}');
	changing.a.toJSON = function(key) {
		assert(key === "a", "toJSON key");
		delete changing.b;
		changing.c = 4;
		changing.d = 5;
		return "A";
	};
	assert(JSON.stringify(changing) === '{"a":"A","c":4}', "stringify mutation snapshot");

	let replacement = JSON.parse('{"a":1,"b":2,"c":3}');
	let replaced = JSON.stringify(replacement, function(key, value) {
		if (key === "a") {
			delete this.b;
			this.c = 9;
			this.d = 10;
		}
		return value;
	});
	assert(replaced === '{"a":1,"c":9}', "replacer mutation snapshot");

	let iterated = JSON.parse('{"a":1,"b":2,"c":3}');
	let seen = [];
	for (let key in iterated) {
		seen.push(key);
		if (key === "a") {
			delete iterated.b;
			iterated.d = 4;
		}
	}
	assert(seen.join(",") === "a,c", "iteration mutation snapshot");

	let spaced = JSON.stringify({a: {}, b: {c: 1}}, null, 2);
	assert(spaced === '{\n  "a": {},\n  "b": {\n    "c": 1\n  }\n}', "empty object indentation");
	`
	testScript(script, _undefined, t)
}

func TestJSONShapeCacheBound(t *testing.T) {
	parser := jsonParser{}
	var first *jsonShape
	for i := range maxJSONShapes {
		pairs := []jsonPair{{key: unistring.String(fmt.Sprintf("key-%d", i)), value: valueTrue}}
		shape := parser.shapeFor(pairs)
		if shape == nil {
			t.Fatalf("shape cache reached its limit early at %d", i)
		}
		if i == 0 {
			first = shape
		}
	}
	if got := parser.shapeFor([]jsonPair{{key: "key-0", value: valueFalse}}); got != first {
		t.Fatal("cached shape was not reused after reaching the limit")
	}
	if got := parser.shapeFor([]jsonPair{{key: "one-too-many", value: valueTrue}}); got != nil {
		t.Fatal("new shape was cached after reaching the limit")
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
	b.ReportAllocs()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		stringify(nil, o)
	}
}

var jsonFixtures = []string{
	"medium.json", "medium-10x.json", "medium-100x.json",
	"unicode.json", "numbers.json", "strings.json",
}

func loadJSONFixture(b *testing.B, name string) string {
	b.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "json", name))
	if err != nil {
		b.Fatal(err)
	}
	return string(data)
}

func BenchmarkJSONParseFixture(b *testing.B) {
	for _, name := range jsonFixtures {
		b.Run(name, func(b *testing.B) {
			data := loadJSONFixture(b, name)
			vm := New()
			parse, _ := AssertFunction(vm.Get("JSON").(*Object).Get("parse"))
			arg := vm.ToValue(data)
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := parse(nil, arg); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// TestJSONParseRetainedMemory reports the heap retained by a parsed fixture.
// Run with -v.
func TestJSONParseRetainedMemory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping memory measurement in short mode")
	}
	for _, name := range jsonFixtures {
		data, err := os.ReadFile(filepath.Join("testdata", "json", name))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				t.Skip("generate JSON fixtures with go run ./testdata/json/gen.go")
			}
			t.Fatal(err)
		}
		vm := New()
		parse, _ := AssertFunction(vm.Get("JSON").(*Object).Get("parse"))
		arg := vm.ToValue(string(data))
		var m1, m2 goruntime.MemStats
		goruntime.GC()
		goruntime.ReadMemStats(&m1)
		v, err := parse(nil, arg)
		if err != nil {
			t.Fatal(err)
		}
		goruntime.GC()
		goruntime.ReadMemStats(&m2)
		t.Logf("%-16s input %7.1f KB, retained %8.1f KB", name, float64(len(data))/1024, float64(m2.HeapAlloc-m1.HeapAlloc)/1024)
		goruntime.KeepAlive(v)
	}
}

func BenchmarkJSONStringifyFixture(b *testing.B) {
	for _, name := range jsonFixtures {
		b.Run(name, func(b *testing.B) {
			data := loadJSONFixture(b, name)
			vm := New()
			jsonObj := vm.Get("JSON").(*Object)
			parse, _ := AssertFunction(jsonObj.Get("parse"))
			stringify, _ := AssertFunction(jsonObj.Get("stringify"))
			parsed, err := parse(nil, vm.ToValue(data))
			if err != nil {
				b.Fatal(err)
			}
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := stringify(nil, parsed); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
