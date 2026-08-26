package goja

import (
	"bytes"
	stdhex "encoding/hex"
	"fmt"
	"testing"
)

func TestUint16ArrayObject(t *testing.T) {
	vm := New()
	buf := vm._newArrayBuffer(vm.global.ArrayBufferPrototype, nil)
	buf.data = make([]byte, 16)
	if nativeEndian == littleEndian {
		buf.data[2] = 0xFE
		buf.data[3] = 0xCA
	} else {
		buf.data[2] = 0xCA
		buf.data[3] = 0xFE
	}
	a := vm.newUint16ArrayObject(buf, 1, 1, nil)
	v := a.getIdx(valueInt(0), nil)
	if v != valueInt(0xCAFE) {
		t.Fatalf("v: %v", v)
	}
}

func TestArrayBufferGoWrapper(t *testing.T) {
	vm := New()
	data := []byte{0xAA, 0xBB}
	buf := vm.NewArrayBuffer(data)
	vm.Set("buf", buf)
	_, err := vm.RunString(`
	var a = new Uint8Array(buf);
	if (a.length !== 2 || a[0] !== 0xAA || a[1] !== 0xBB) {
		throw new Error(a);
	}
	`)
	if err != nil {
		t.Fatal(err)
	}
	ret, err := vm.RunString(`
	var b = Uint8Array.of(0xCC, 0xDD);
	b.buffer;
	`)
	if err != nil {
		t.Fatal(err)
	}
	buf1 := ret.Export().(ArrayBuffer)
	data1 := buf1.Bytes()
	if len(data1) != 2 || data1[0] != 0xCC || data1[1] != 0xDD {
		t.Fatal(data1)
	}
	if buf1.Detached() {
		t.Fatal("buf1.Detached() returned true")
	}
	if !buf1.Detach() {
		t.Fatal("buf1.Detach() returned false")
	}
	if !buf1.Detached() {
		t.Fatal("buf1.Detached() returned false")
	}
	_, err = vm.RunString(`
	if (b[0] !== undefined) {
		throw new Error("b[0] !== undefined");
	}
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestTypedArrayIdx(t *testing.T) {
	const SCRIPT = `
	var a = new Uint8Array(1);

	// 32-bit integer overflow, should not panic on 32-bit architectures
	if (a[4294967297] !== undefined) {
		throw new Error("4294967297");
	}

	// Canonical non-integer
	a["Infinity"] = 8;
	if (a["Infinity"] !== undefined) {
		throw new Error("Infinity");
	}
	a["NaN"] = 1;
	if (a["NaN"] !== undefined) {
		throw new Error("NaN");
	}

	// Non-canonical integer
	a["00"] = "00";
	if (a["00"] !== "00") {
		throw new Error("00");
	}

	// Non-canonical non-integer
	a["1e-3"] = "1e-3";
	if (a["1e-3"] !== "1e-3") {
		throw new Error("1e-3");
	}
	if (a["0.001"] !== undefined) {
		throw new Error("0.001");
	}

	// Negative zero
	a["-0"] = 88;
	if (a["-0"] !== undefined) {
		throw new Error("-0");
	}

	if (a[0] !== 0) {
		throw new Error("0");
	}

	a["9007199254740992"] = 1;
	if (a["9007199254740992"] !== undefined) {
		throw new Error("9007199254740992");
	}
	a["-9007199254740992"] = 1;
	if (a["-9007199254740992"] !== undefined) {
		throw new Error("-9007199254740992");
	}

	// Safe integer overflow, not canonical (Number("9007199254740993") === 9007199254740992)
	a["9007199254740993"] = 1;
	if (a["9007199254740993"] !== 1) {
		throw new Error("9007199254740993");
	}
	a["-9007199254740993"] = 1;
	if (a["-9007199254740993"] !== 1) {
		throw new Error("-9007199254740993");
	}

	// Safe integer overflow, canonical Number("9007199254740994") == 9007199254740994
	a["9007199254740994"] = 1;
	if (a["9007199254740994"] !== undefined) {
		throw new Error("9007199254740994");
	}
	a["-9007199254740994"] = 1;
	if (a["-9007199254740994"] !== undefined) {
		throw new Error("-9007199254740994");
	}
	`

	testScript(SCRIPT, _undefined, t)
}

func TestTypedArraySetDetachedBuffer(t *testing.T) {
	const SCRIPT = `
	let sample = new Uint8Array([42]);
	$DETACHBUFFER(sample.buffer);
	sample[0] = 1;

	assert.sameValue(sample[0], undefined, 'sample[0] = 1 is undefined');
	sample['1.1'] = 1;
	assert.sameValue(sample['1.1'], undefined, 'sample[\'1.1\'] = 1 is undefined');
	sample['-0'] = 1;
	assert.sameValue(sample['-0'], undefined, 'sample[\'-0\'] = 1 is undefined');
	sample['-1'] = 1;
	assert.sameValue(sample['-1'], undefined, 'sample[\'-1\'] = 1 is undefined');
	sample['1'] = 1;
	assert.sameValue(sample['1'], undefined, 'sample[\'1\'] = 1 is undefined');
	sample['2'] = 1;
	assert.sameValue(sample['2'], undefined, 'sample[\'2\'] = 1 is undefined');	
	`
	vm := New()
	vm.Set("$DETACHBUFFER", func(buf *ArrayBuffer) {
		buf.Detach()
	})
	vm.testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestTypedArrayDefinePropDetachedBuffer(t *testing.T) {
	const SCRIPT = `
	var desc = {
	  value: 0,
	  configurable: false,
	  enumerable: true,
	  writable: true
	};
	
	var obj = {
	  valueOf: function() {
		throw new Error("valueOf() was called");
	  }
	};
	let sample = new Uint8Array(42);
	$DETACHBUFFER(sample.buffer);
	
	assert.sameValue(
	Reflect.defineProperty(sample, "0", desc),
	false,
	'Reflect.defineProperty(sample, "0", {value: 0, configurable: false, enumerable: true, writable: true} ) must return false'
	);
	
	assert.sameValue(
	Reflect.defineProperty(sample, "-1", desc),
	false,
	'Reflect.defineProperty(sample, "-1", {value: 0, configurable: false, enumerable: true, writable: true} ) must return false'
	);
	
	assert.sameValue(
	Reflect.defineProperty(sample, "1.1", desc),
	false,
	'Reflect.defineProperty(sample, "1.1", {value: 0, configurable: false, enumerable: true, writable: true} ) must return false'
	);
	
	assert.sameValue(
	Reflect.defineProperty(sample, "-0", desc),
	false,
	'Reflect.defineProperty(sample, "-0", {value: 0, configurable: false, enumerable: true, writable: true} ) must return false'
	);
	
	assert.sameValue(
	Reflect.defineProperty(sample, "2", {
	  configurable: true,
	  enumerable: true,
	  writable: true,
	  value: obj
	}),
	false,
	'Reflect.defineProperty(sample, "2", {configurable: true, enumerable: true, writable: true, value: obj}) must return false'
	);
	
	assert.sameValue(
	Reflect.defineProperty(sample, "3", {
	  configurable: false,
	  enumerable: false,
	  writable: true,
	  value: obj
	}),
	false,
	'Reflect.defineProperty(sample, "3", {configurable: false, enumerable: false, writable: true, value: obj}) must return false'
	);
	
	assert.sameValue(
	Reflect.defineProperty(sample, "4", {
	  writable: false,
	  configurable: false,
	  enumerable: true,
	  value: obj
	}),
	false,
	'Reflect.defineProperty("new TA(42)", "4", {writable: false, configurable: false, enumerable: true, value: obj}) must return false'
	);
	
	assert.sameValue(
	Reflect.defineProperty(sample, "42", desc),
	false,
	'Reflect.defineProperty(sample, "42", {value: 0, configurable: false, enumerable: true, writable: true} ) must return false'
	);
	
	assert.sameValue(
	Reflect.defineProperty(sample, "43", desc),
	false,
	'Reflect.defineProperty(sample, "43", {value: 0, configurable: false, enumerable: true, writable: true} ) must return false'
	);
	
	assert.sameValue(
	Reflect.defineProperty(sample, "5", {
	  get: function() {}
	}),
	false,
	'Reflect.defineProperty(sample, "5", {get: function() {}}) must return false'
	);
	
	assert.sameValue(
	Reflect.defineProperty(sample, "6", {
	  configurable: false,
	  enumerable: true,
	  writable: true
	}),
	false,
	'Reflect.defineProperty(sample, "6", {configurable: false, enumerable: true, writable: true}) must return false'
	);
	`
	vm := New()
	vm.Set("$DETACHBUFFER", func(buf *ArrayBuffer) {
		buf.Detach()
	})
	vm.testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestTypedArrayDefineProperty(t *testing.T) {
	const SCRIPT = `
	var a = new Uint8Array(1);

	assert.throws(TypeError, function() {
		Object.defineProperty(a, "1", {value: 1});
	});
	assert.sameValue(Reflect.defineProperty(a, "1", {value: 1}), false, "1");

	assert.throws(TypeError, function() {
		Object.defineProperty(a, "Infinity", {value: 8});
	});
	assert.sameValue(Reflect.defineProperty(a, "Infinity", {value: 8}), false, "Infinity");

	Object.defineProperty(a, "test", {value: "passed"});
	assert.sameValue(a.test, "passed", "string property");

	assert.throws(TypeError, function() {
		Object.defineProperty(a, "0", {value: 1, writable: false});
	}, "define non-writable");

	assert.throws(TypeError, function() {
		Object.defineProperty(a, "0", {get() { return 1; }});
	}, "define accessor");

	var sample = new Uint8Array([42, 42]);

	assert.sameValue(
	Reflect.defineProperty(sample, "0", {
	  value: 8,
	  configurable: true,
	  enumerable: true,
	  writable: true
	}),
	true
	);

	assert.sameValue(sample[0], 8, "property value was set");
	let descriptor0 = Object.getOwnPropertyDescriptor(sample, "0");
	assert.sameValue(descriptor0.value, 8);
	assert.sameValue(descriptor0.configurable, true, "configurable");
	assert.sameValue(descriptor0.enumerable, true);
	assert.sameValue(descriptor0.writable, true);
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestTypedArrayGetInvalidIndex(t *testing.T) {
	const SCRIPT = `
	var TypedArray = Object.getPrototypeOf(Int8Array);
	var proto = TypedArray.prototype;
	Object.defineProperty(proto, "1", {
		get: function() {
			throw new Error("OrdinaryGet was called!");
		}
	});
	var a = new Uint8Array(1);
	assert.sameValue(a[1], undefined);
	assert.sameValue(a["1"], undefined);
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestExportArrayBufferToBytes(t *testing.T) {
	vm := New()
	bb := []byte("test")
	ab := vm.NewArrayBuffer(bb)
	var b []byte
	err := vm.ExportTo(vm.ToValue(ab), &b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, bb) {
		t.Fatal("Not equal")
	}

	err = vm.ExportTo(vm.ToValue(123), &b)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTypedArrayExport(t *testing.T) {
	vm := New()

	t.Run("uint8", func(t *testing.T) {
		v, err := vm.RunString("new Uint8Array([1, 2])")
		if err != nil {
			t.Fatal(err)
		}
		if a, ok := v.Export().([]uint8); ok {
			if len(a) != 2 || a[0] != 1 || a[1] != 2 {
				t.Fatal(a)
			}
		} else {
			t.Fatal("Wrong export type")
		}
		_, err = vm.RunString(`{
		let a = new Uint8Array([1, 2]);
		if (a[0] !== 1 || a[1] !== 2) {
			throw new Error(a);
		}
		}`)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("uint8-slice", func(t *testing.T) {
		v, err := vm.RunString(`{
			const buf = new Uint8Array([1, 2]).buffer;
			new Uint8Array(buf, 1, 1);
		}`)
		if err != nil {
			t.Fatal(err)
		}
		if a, ok := v.Export().([]uint8); ok {
			if len(a) != 1 || a[0] != 2 {
				t.Fatal(a)
			}
		} else {
			t.Fatal("Wrong export type")
		}
		_, err = vm.RunString(`{
		let a = new Uint8Array([1, 2]);
		if (a[0] !== 1 || a[1] !== 2) {
			throw new Error(a);
		}
		}`)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("int8", func(t *testing.T) {
		v, err := vm.RunString("new Int8Array([1, -2])")
		if err != nil {
			t.Fatal(err)
		}
		if a, ok := v.Export().([]int8); ok {
			if len(a) != 2 || a[0] != 1 || a[1] != -2 {
				t.Fatal(a)
			}
		} else {
			t.Fatal("Wrong export type")
		}
	})

	t.Run("uint16", func(t *testing.T) {
		v, err := vm.RunString("new Uint16Array([1, 63000])")
		if err != nil {
			t.Fatal(err)
		}
		if a, ok := v.Export().([]uint16); ok {
			if len(a) != 2 || a[0] != 1 || a[1] != 63000 {
				t.Fatal(a)
			}
		} else {
			t.Fatal("Wrong export type")
		}
	})

	t.Run("int16", func(t *testing.T) {
		v, err := vm.RunString("new Int16Array([1, -31000])")
		if err != nil {
			t.Fatal(err)
		}
		if a, ok := v.Export().([]int16); ok {
			if len(a) != 2 || a[0] != 1 || a[1] != -31000 {
				t.Fatal(a)
			}
		} else {
			t.Fatal("Wrong export type")
		}
	})

	t.Run("uint32", func(t *testing.T) {
		v, err := vm.RunString("new Uint32Array([1, 123456])")
		if err != nil {
			t.Fatal(err)
		}
		if a, ok := v.Export().([]uint32); ok {
			if len(a) != 2 || a[0] != 1 || a[1] != 123456 {
				t.Fatal(a)
			}
		} else {
			t.Fatal("Wrong export type")
		}
	})

	t.Run("int32", func(t *testing.T) {
		v, err := vm.RunString("new Int32Array([1, -123456])")
		if err != nil {
			t.Fatal(err)
		}
		if a, ok := v.Export().([]int32); ok {
			if len(a) != 2 || a[0] != 1 || a[1] != -123456 {
				t.Fatal(a)
			}
		} else {
			t.Fatal("Wrong export type")
		}
	})

	t.Run("float32", func(t *testing.T) {
		v, err := vm.RunString("new Float32Array([1, -1.23456])")
		if err != nil {
			t.Fatal(err)
		}
		if a, ok := v.Export().([]float32); ok {
			if len(a) != 2 || a[0] != 1 || a[1] != -1.23456 {
				t.Fatal(a)
			}
		} else {
			t.Fatal("Wrong export type")
		}
	})

	t.Run("float64", func(t *testing.T) {
		v, err := vm.RunString("new Float64Array([1, -1.23456789])")
		if err != nil {
			t.Fatal(err)
		}
		if a, ok := v.Export().([]float64); ok {
			if len(a) != 2 || a[0] != 1 || a[1] != -1.23456789 {
				t.Fatal(a)
			}
		} else {
			t.Fatal("Wrong export type")
		}
	})

	t.Run("bigint64", func(t *testing.T) {
		v, err := vm.RunString("new BigInt64Array([18446744073709551617n, 2n])")
		if err != nil {
			t.Fatal(err)
		}
		if a, ok := v.Export().([]int64); ok {
			if len(a) != 2 || a[0] != 1 || a[1] != 2 {
				t.Fatal(a)
			}
		} else {
			t.Fatal("Wrong export type")
		}
	})

	t.Run("biguint64", func(t *testing.T) {
		v, err := vm.RunString("new BigUint64Array([18446744073709551617n, 2n])")
		if err != nil {
			t.Fatal(err)
		}
		if a, ok := v.Export().([]uint64); ok {
			if len(a) != 2 || a[0] != 1 || a[1] != 2 {
				t.Fatal(a)
			}
		} else {
			t.Fatal("Wrong export type")
		}
	})

}

func TestUint8ArrayFromHex(t *testing.T) {
	vm := New()

	// valid-hex string
	retH, err := vm.RunString(`
	Uint8Array.fromHex("0123456789ABcdEf");
	`)
	if err != nil {
		t.Fatal(err)
	}
	bufStr := stdhex.EncodeToString(retH.Export().([]byte)) // means Uint8Array
	if bufStr != "0123456789abcdef" {
		t.Fatal(bufStr)
	}

	data := []byte{0xAA, 0xBB}
	buf := vm.NewArrayBuffer(data)
	vm.Set("buf", buf)
	_, err = vm.RunString(`
	var a = new Uint8Array(buf);
	var b = Uint8Array.fromHex("AaBb");

	if (a.length !== 2 || b.length !== 2) {
		throw new Error(b);
	}
	for (let i = 0; i < a.length; i++) {
		if (a[i] !== b[i]) {
		throw new Error(b);
		}
	}
	`)
	if err != nil {
		t.Fatal(err)
	}

	ret, err := vm.RunString(`
	var b = Uint8Array.fromHex("ccDD");
	b.buffer;
	`)
	if err != nil {
		t.Fatal(err)
	}
	buf1 := ret.Export().(ArrayBuffer)
	data1 := buf1.Bytes()
	if len(data1) != 2 || data1[0] != 0xCC || data1[1] != 0xDD {
		t.Fatal(data1)
	}
	if buf1.Detached() {
		t.Fatal("buf1.Detached() returned true")
	}
	if !buf1.Detach() {
		t.Fatal("buf1.Detach() returned false")
	}
	if !buf1.Detached() {
		t.Fatal("buf1.Detached() returned false")
	}
	_, err = vm.RunString(`
	if (b[0] !== undefined) {
		throw new Error("b[0] !== undefined");
	}
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestUint8ArrayToHex(t *testing.T) {
	vm := New()

	// valid-hex string
	retH, err := vm.RunString(`
	var arr = Uint8Array.fromHex("0123456789ABcdEf");
	arr.toHex();
	`)
	if err != nil {
		t.Fatal(err)
	}
	bufStr := retH.Export().(string) // means Hex string
	if bufStr != "0123456789abcdef" {
		t.Fatal(bufStr)
	}
}

func TestUint8ArraySetFromHex(t *testing.T) {
	testCases := []struct {
		name     string
		script   string
		expected string
	}{
		{
			name: "valid-hex-string",
			script: `
			var arr = Uint8Array.fromHex("0123456789ABcdEf");
			arr.setFromHex("0123456789ABcdEf");
			arr.toHex();
			`,
			expected: "0123456789abcdef",
		},
		{
			// length[Uint8Array] < length(setFromHex)
			name: "array-shorter-than-input",
			script: `
			var arr = Uint8Array.fromHex("01234567");
			arr.setFromHex("0123456789ABcdEf");
			arr.toHex();
			`,
			expected: "01234567",
		},
		{
			// length[Uint8Array] > length(setFromHex)
			name: "array-longer-than-input",
			script: `
			var arr = Uint8Array.fromHex("0123456789ABcdEf");
			arr.setFromHex("AABBCCDD");
			arr.toHex();
			`,
			expected: "aabbccdd89abcdef",
		},
		{
			// length[Uint8Array] > length(setFromHex)
			name: "fresh-array-longer-than-input",
			script: `
			var arr = new Uint8Array(5);
			arr.setFromHex("AABBCCDD");
			arr.toHex();
			`,
			expected: "aabbccdd00",
		},
		{
			// offset length[Uint8Array] > length(setFromHex)
			name: "subarray-longer-than-input",
			script: `
			var arr = new Uint8Array(8);
			arr.subarray(3).setFromHex("cafed00d");
			arr.toHex();
			`,
			expected: "000000cafed00d00",
		},
		{
			// offset + length[Uint8Array] < length(setFromHex)
			name: "subarray-shorter-than-input",
			script: `
			var arr = new Uint8Array(5);
			arr.subarray(3).setFromHex("cafed00d");
			arr.toHex();
			`,
			expected: "000000cafe",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vm := New()
			v, err := vm.RunString(tc.script)
			if err != nil {
				t.Fatal(err)
			}
			if result := v.String(); result != tc.expected {
				t.Fatalf("Expected '%s' but got '%s'", tc.expected, result)
			}
		})
	}

	vm := New()
	t.Run("read", func(t *testing.T) {
		ret, err := vm.RunString(`
		var arr = new Uint8Array(16);
		arr.setFromHex("cafed00d").read;
		`)
		if err != nil {
			t.Fatal(err)
		}
		read := ret.Export().(int64)
		if read != 8 {
			t.Fatal(read)
		}
	})
	t.Run("written", func(t *testing.T) {
		ret, err := vm.RunString(`
		var arr = new Uint8Array(16);
		arr.setFromHex("cafed00d").written;
		`)
		if err != nil {
			t.Fatal(err)
		}
		written := ret.Export().(int64)
		if written != 4 {
			t.Fatal(written)
		}
	})
}

// Whatever was decoded before an error must still be written into the
// destination: the spec performs SetUint8ArrayBytes before the throw.
func TestUint8ArraySetFromHexPartialWrite(t *testing.T) {
	testCases := []struct {
		name     string
		script   string
		expected string
	}{
		{
			name: "invalid-character-mid-string",
			script: `
			var arr = new Uint8Array(4);
			try {
				arr.setFromHex("aabbZZcc");
			} catch (e) {
				if (!(e instanceof SyntaxError)) {
					throw e;
				}
			}
			arr.toHex();
			`,
			expected: "aabb0000", // writes "aabb" and then throws on "ZZ"
		},
		{
			name: "invalid-character-mid-string-subarray",
			script: `
			var arr = new Uint8Array(6);
			try {
				arr.subarray(2).setFromHex("aabbZZcc");
			} catch (e) {
				if (!(e instanceof SyntaxError)) {
					throw e;
				}
			}
			arr.toHex();
			`,
			expected: "0000aabb0000", // writes "____aabb" and then throws on "ZZ"
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vm := New()
			v, err := vm.RunString(tc.script)
			if err != nil {
				t.Fatal(err)
			}
			if result := v.String(); result != tc.expected {
				t.Fatalf("Expected '%s' but got '%s'", tc.expected, result)
			}
		})
	}
}

// assertThrows wraps a script (%s) so that it fails unless the script throws
// an instance of the given error constructor (%[2]s).
const assertThrows = `
var ok = false;
try {
	%s;
} catch (e) {
	ok = e instanceof %s;
}
if (!ok) {
	throw new Error("Expected a %[2]s");
}
`

func TestInvalidUint8ArraySetFromHex(t *testing.T) {
	testCases := []struct {
		name      string
		script    string
		errorName string
	}{
		{
			name:      "non-hex-character",
			script:    `Uint8Array.fromHex("01234567").setFromHex("aabZ")`,
			errorName: "SyntaxError",
		},
		{
			name:      "odd-length",
			script:    `Uint8Array.fromHex("01234567").setFromHex("aab")`,
			errorName: "SyntaxError",
		},
		{
			name:      "contain-whitespace",
			script:    `Uint8Array.fromHex("01234567").setFromHex("aa  bb")`,
			errorName: "SyntaxError",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vm := New()
			_, err := vm.RunString(fmt.Sprintf(assertThrows, tc.script, tc.errorName))
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestInvalidUint8ArrayFromHex(t *testing.T) {
	testCases := []struct {
		name      string
		script    string
		errorName string
	}{
		{
			name:      "non-hex-character",
			script:    `Uint8Array.fromHex("aabZ")`,
			errorName: "SyntaxError",
		},
		{
			name:      "odd-length",
			script:    `Uint8Array.fromHex("aab")`,
			errorName: "SyntaxError",
		},
		{
			name:      "contain-whitespace",
			script:    `Uint8Array.fromHex("aa  bb")`,
			errorName: "SyntaxError",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vm := New()
			_, err := vm.RunString(fmt.Sprintf(assertThrows, tc.script, tc.errorName))
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestUint8ArrayFromBase64(t *testing.T) {
	testCases := []struct {
		name     string
		script   string
		expected string
	}{
		// ---------loose---------
		{
			name:     "loose-base64-missing-padding",
			script:   `Uint8Array.fromBase64("aGVsbG8");`,
			expected: "hello",
		},
		{
			name:     "loose-base64-with-padding",
			script:   `Uint8Array.fromBase64("aGVsbG8=");`,
			expected: "hello",
		},
		{
			name:     "loose-base64-with-whitespace",
			script:   `Uint8Array.fromBase64(" aGVs\tb\nG8\r\n = ");`,
			expected: "hello",
		},
		{
			name:     "loose-base64-empty",
			script:   `Uint8Array.fromBase64("");`,
			expected: "",
		},
		{
			name:     "loose-base64-urllike",
			script:   `Uint8Array.fromBase64("aGVsbG8+ISE/", { alphabet: "base64"});`,
			expected: "hello>!!?",
		},
		// ---------strict---------
		{
			name:     "strict-base64-missing-padding",
			script:   `Uint8Array.fromBase64("aGVsbG8x", { lastChunkHandling: "strict" });`,
			expected: "hello1",
		},
		{
			name:     "strict-base64-with-padding",
			script:   `Uint8Array.fromBase64("aGVsbG8=", { lastChunkHandling: "strict" });`,
			expected: "hello",
		},
		{
			name:     "strict-base64-with-whitespace",
			script:   `Uint8Array.fromBase64(" aGVs\tb\nG8\r\nx ", { lastChunkHandling: "strict" });`,
			expected: "hello1",
		},
		{
			name:     "strict-base64-empty",
			script:   `Uint8Array.fromBase64("", { lastChunkHandling: "strict" });`,
			expected: "",
		},
		// ---------stop-before-partial---------
		{
			name:     "partial-base64-missing-padding",
			script:   `Uint8Array.fromBase64("aGVs", { lastChunkHandling: "stop-before-partial" });`,
			expected: "hel",
		},
		{
			name:     "partial-base64-missing-padding-after-partial",
			script:   `Uint8Array.fromBase64("aGVsbG", { lastChunkHandling: "stop-before-partial" });`,
			expected: "hel",
		},
		{
			name:     "partial-base64-with-padding",
			script:   `Uint8Array.fromBase64("aGVsbG8=", { lastChunkHandling: "stop-before-partial" });`,
			expected: "hello",
		},
		{
			name:     "partial-base64-with-whitespace",
			script:   `Uint8Array.fromBase64(" aGVs\tb\nG8\r\n = ", { lastChunkHandling: "stop-before-partial" });`,
			expected: "hello",
		},
		{
			name:     "partial-base64-empty",
			script:   `Uint8Array.fromBase64("", { lastChunkHandling: "stop-before-partial" });`,
			expected: "",
		},

		// ---------base64url----------
		{
			name:     "loose-base64url-missing-padding",
			script:   `Uint8Array.fromBase64("aGVsbG8-ISE_", { alphabet: "base64url"});`,
			expected: "hello>!!?",
		},
		{
			name:     "loose-base64url-with-padding",
			script:   `Uint8Array.fromBase64("aGVsbG8-ISE_YQ==", { alphabet: "base64url"});`,
			expected: "hello>!!?a",
		},
		{
			name:     "loose-base64url-with-whitespace",
			script:   `Uint8Array.fromBase64(" aGVs\tbG\n8-ISE_Y\r\nQ= = ", { alphabet: "base64url"});`,
			expected: "hello>!!?a",
		},
		{
			name:     "loose-base64url-empty",
			script:   `Uint8Array.fromBase64("", { alphabet: "base64url"});`,
			expected: "",
		},
		{
			name:     "strict-base64url-missing-padding",
			script:   `Uint8Array.fromBase64("aGVsbG8-ISE_", { lastChunkHandling: "strict", alphabet: "base64url" });`,
			expected: "hello>!!?",
		},
		{
			name:     "partial-base64url-with-padding",
			script:   `Uint8Array.fromBase64("aGVsbG8-ISE_YQ==", { lastChunkHandling: "stop-before-partial", alphabet: "base64url" });`,
			expected: "hello>!!?a",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vm := New()
			v, err := vm.RunString(tc.script)
			if err != nil {
				t.Fatal(err)
			}
			result := string(v.Export().([]byte)) // means Uint8Array
			if result != tc.expected {
				t.Fatalf("Expected '%s' but got '%s'", tc.expected, result)
			}
		})
	}
}

func TestInvalidUint8ArrayFromBase64(t *testing.T) {
	testCases := []struct {
		name      string
		script    string
		errorName string
	}{
		// ---------SyntaxError---------
		{
			name:      "non-base64-character",
			script:    `Uint8Array.fromBase64("ab#c")`,
			errorName: "SyntaxError",
		},
		{
			// a trailing chunk of length 1 is invalid even in loose mode
			name:      "loose-last-chunk-is-one-length-character",
			script:    `Uint8Array.fromBase64("abcde")`,
			errorName: "SyntaxError",
		},
		{
			// a trailing chunk of length 1 is invalid even in stop-before-partial mode
			name:      "partial-last-chunk-is-one-length-character",
			script:    `Uint8Array.fromBase64("aGVs bG8x P=", { lastChunkHandling: "stop-before-partial" });`,
			errorName: "SyntaxError",
		},
		{
			name:      "padding-only",
			script:    `Uint8Array.fromBase64("=")`,
			errorName: "SyntaxError",
		},
		{
			name:      "padding-after-single-character",
			script:    `Uint8Array.fromBase64("A=")`,
			errorName: "SyntaxError",
		},
		{
			name:      "padding-inside-chunk",
			script:    `Uint8Array.fromBase64("AB=C")`,
			errorName: "SyntaxError",
		},
		{
			name:      "padding-after-full-chunk",
			script:    `Uint8Array.fromBase64("ABCD=")`,
			errorName: "SyntaxError",
		},
		{
			name:      "character-after-padding",
			script:    `Uint8Array.fromBase64("AB==C")`,
			errorName: "SyntaxError",
		},
		{
			name:      "strict-extra-bits",
			script:    `Uint8Array.fromBase64("QR==", { lastChunkHandling: "strict" })`,
			errorName: "SyntaxError",
		},
		{
			name:      "strict-missing-padding",
			script:    `Uint8Array.fromBase64("aGVsbG8", { lastChunkHandling: "strict" });`,
			errorName: "SyntaxError",
		},
		// ---------SyntaxError with alphabet option---------
		{
			name:      "base64url-but-base64-alphabet",
			script:    `Uint8Array.fromBase64("aGVsbG8-ISE_YQ==", { lastChunkHandling: "stop-before-partial", alphabet: "base64" });`,
			errorName: "SyntaxError",
		},
		{
			name:      "base64-but-base64url-alphabet",
			script:    `Uint8Array.fromBase64("aGVsbG8+ISE/", { alphabet: "base64url"});`,
			errorName: "SyntaxError",
		},
		// ---------TypeError---------
		{
			// the argument must be a String, there is no ToString coercion
			name:      "non-string-input",
			script:    `Uint8Array.fromBase64(1234)`,
			errorName: "TypeError",
		},
		{
			name:      "options-null",
			script:    `Uint8Array.fromBase64("aGVsbG8=", null)`,
			errorName: "TypeError",
		},
		{
			name:      "options-not-an-object",
			script:    `Uint8Array.fromBase64("aGVsbG8=", "loose")`,
			errorName: "TypeError",
		},
		{
			name:      "alphabet-unknown",
			script:    `Uint8Array.fromBase64("aGVsbG8=", { alphabet: "base16" })`,
			errorName: "TypeError",
		},
		{
			name:      "alphabet-null",
			script:    `Uint8Array.fromBase64("aGVsbG8=", { alphabet: null })`,
			errorName: "TypeError",
		},
		{
			name:      "last-chunk-handling-unknown",
			script:    `Uint8Array.fromBase64("aGVsbG8=", { lastChunkHandling: "foo" })`,
			errorName: "TypeError",
		},
		{
			name:      "last-chunk-handling-null",
			script:    `Uint8Array.fromBase64("aGVsbG8=", { lastChunkHandling: null })`,
			errorName: "TypeError",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vm := New()
			_, err := vm.RunString(fmt.Sprintf(assertThrows, tc.script, tc.errorName))
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestUint8ArrayToBase64(t *testing.T) {
	testCases := []struct {
		name     string
		script   string
		expected string
	}{
		// ---------base64 (default)---------
		{
			name:     "no-padding-needed",
			script:   `Uint8Array.fromBase64("aGVs").toBase64();`,
			expected: "aGVs",
		},
		{
			name:     "one-padding-character",
			script:   `Uint8Array.fromBase64("aGVsbG8gd29ybGQ=").toBase64();`,
			expected: "aGVsbG8gd29ybGQ=",
		},
		{
			name:     "two-padding-characters",
			script:   `Uint8Array.fromBase64("aGVsbA==").toBase64();`,
			expected: "aGVsbA==",
		},
		{
			name:     "empty",
			script:   `new Uint8Array(0).toBase64();`,
			expected: "",
		},
		{
			// only the bytes of the subarray view are encoded
			name:     "subarray",
			script:   `Uint8Array.fromHex("00aabb00").subarray(1, 3).toBase64();`,
			expected: "qrs=",
		},
		// ---------alphabet---------
		{
			name:     "base64-alphabet-explicit",
			script:   `Uint8Array.fromHex("fbefbeffffff").toBase64({ alphabet: "base64" });`,
			expected: "++++////",
		},
		{
			name:     "base64url-alphabet",
			script:   `Uint8Array.fromHex("fbefbeffffff").toBase64({ alphabet: "base64url" });`,
			expected: "----____",
		},
		{
			name:     "base64url-alphabet-with-padding",
			script:   `Uint8Array.fromHex("fbef").toBase64({ alphabet: "base64url" });`,
			expected: "--8=",
		},
		// ---------omitPadding---------
		{
			name:     "omit-padding",
			script:   `Uint8Array.fromBase64("aGVsbG8=").toBase64({ omitPadding: true });`,
			expected: "aGVsbG8",
		},
		{
			name:     "omit-padding-false",
			script:   `Uint8Array.fromBase64("aGVsbG8=").toBase64({ omitPadding: false });`,
			expected: "aGVsbG8=",
		},
		{
			// omitPadding is coerced with ToBoolean: any truthy value omits the padding
			name:     "omit-padding-truthy-string",
			script:   `Uint8Array.fromBase64("aGVsbG8=").toBase64({ omitPadding: "false" });`,
			expected: "aGVsbG8",
		},
		{
			name:     "omit-padding-base64url",
			script:   `Uint8Array.fromHex("fbef").toBase64({ alphabet: "base64url", omitPadding: true });`,
			expected: "--8",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vm := New()
			v, err := vm.RunString(tc.script)
			if err != nil {
				t.Fatal(err)
			}
			if result := v.String(); result != tc.expected {
				t.Fatalf("Expected '%s' but got '%s'", tc.expected, result)
			}
		})
	}
}

func TestInvalidUint8ArrayToBase64(t *testing.T) {
	testCases := []struct {
		name      string
		script    string
		errorName string
	}{
		{
			name:      "options-null",
			script:    `new Uint8Array(4).toBase64(null)`,
			errorName: "TypeError",
		},
		{
			name:      "options-not-an-object",
			script:    `new Uint8Array(4).toBase64("base64")`,
			errorName: "TypeError",
		},
		{
			name:      "alphabet-unknown",
			script:    `new Uint8Array(4).toBase64({ alphabet: "hex" })`,
			errorName: "TypeError",
		},
		{
			name:      "alphabet-null",
			script:    `new Uint8Array(4).toBase64({ alphabet: null })`,
			errorName: "TypeError",
		},
		{
			// a String wrapper object is not a String primitive
			name:      "alphabet-string-object",
			script:    `new Uint8Array(4).toBase64({ alphabet: new String("base64") })`,
			errorName: "TypeError",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vm := New()
			_, err := vm.RunString(fmt.Sprintf(assertThrows, tc.script, tc.errorName))
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestUint8ArraySetFromBase64(t *testing.T) {
	testCases := []struct {
		name     string
		script   string
		expected string
	}{
		{
			name: "same-length",
			script: `
			var arr = new Uint8Array(4);
			arr.setFromBase64("aGVsbA==");
			arr.toHex();
			`,
			expected: "68656c6c",
		},
		{
			// only up to the target size is decoded, the rest of the input is not read
			name: "array-shorter-than-input",
			script: `
			var arr = new Uint8Array(3);
			arr.setFromBase64("aGVsbG8=");
			arr.toHex();
			`,
			expected: "68656c",
		},
		{
			// the bytes beyond the decoded length keep their previous content
			name: "array-longer-than-input",
			script: `
			var arr = Uint8Array.fromHex("ffffffffffff");
			arr.setFromBase64("aGVs");
			arr.toHex();
			`,
			expected: "68656cffffff",
		},
		{
			name: "fresh-array-longer-than-input",
			script: `
			var arr = new Uint8Array(5);
			arr.setFromBase64("aGVs");
			arr.toHex();
			`,
			expected: "68656c0000",
		},
		{
			// only the bytes of the subarray view are written
			name: "subarray",
			script: `
			var arr = new Uint8Array(8);
			arr.subarray(3).setFromBase64("aGVs");
			arr.toHex();
			`,
			expected: "00000068656c0000",
		},
		{
			name: "whitespace",
			script: `
			var arr = new Uint8Array(4);
			arr.setFromBase64(" aGVs\tbA==\n");
			arr.toHex();
			`,
			expected: "68656c6c",
		},
		{
			// the partial last chunk "bG8" is not decoded
			name: "stop-before-partial",
			script: `
			var arr = new Uint8Array(6);
			arr.setFromBase64("aGVsbG8", { lastChunkHandling: "stop-before-partial" });
			arr.toHex();
			`,
			expected: "68656c000000",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vm := New()
			v, err := vm.RunString(tc.script)
			if err != nil {
				t.Fatal(err)
			}
			if result := v.String(); result != tc.expected {
				t.Fatalf("Expected '%s' but got '%s'", tc.expected, result)
			}
		})
	}

	vm := New()
	t.Run("read", func(t *testing.T) {
		ret, err := vm.RunString(`
		var arr = new Uint8Array(16);
		arr.setFromBase64("aGVsbA==").read;
		`)
		if err != nil {
			t.Fatal(err)
		}
		read := ret.Export().(int64)
		if read != 8 {
			t.Fatal(read)
		}
	})
	t.Run("written", func(t *testing.T) {
		ret, err := vm.RunString(`
		var arr = new Uint8Array(16);
		arr.setFromBase64("aGVsbA==").written;
		`)
		if err != nil {
			t.Fatal(err)
		}
		written := ret.Export().(int64)
		if written != 4 {
			t.Fatal(written)
		}
	})
	t.Run("read-stops-at-target-size", func(t *testing.T) {
		// the target holds 3 bytes: only the first full chunk (4 characters) is read
		ret, err := vm.RunString(`
		var arr = new Uint8Array(3);
		arr.setFromBase64("aGVsbG8x").read;
		`)
		if err != nil {
			t.Fatal(err)
		}
		read := ret.Export().(int64)
		if read != 4 {
			t.Fatal(read)
		}
	})
}

// Whatever was decoded before an error must still be written into the
// destination: the spec performs SetUint8ArrayBytes before the throw.
func TestUint8ArraySetFromBase64PartialWrite(t *testing.T) {
	testCases := []struct {
		name     string
		script   string
		expected string
	}{
		{
			name: "invalid-character-mid-string",
			script: `
			var arr = new Uint8Array(6);
			try {
				arr.setFromBase64("aGVs#nvalid");
			} catch (e) {
				if (!(e instanceof SyntaxError)) {
					throw e;
				}
			}
			arr.toHex();
			`,
			expected: "68656c000000",
		},
		{
			name: "invalid-character-mid-string-subarray",
			script: `
			var arr = new Uint8Array(8);
			try {
				arr.subarray(2).setFromBase64("aGVs#nvalid");
			} catch (e) {
				if (!(e instanceof SyntaxError)) {
					throw e;
				}
			}
			arr.toHex();
			`,
			expected: "000068656c000000",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vm := New()
			v, err := vm.RunString(tc.script)
			if err != nil {
				t.Fatal(err)
			}
			if result := v.String(); result != tc.expected {
				t.Fatalf("Expected '%s' but got '%s'", tc.expected, result)
			}
		})
	}
}

func TestInvalidUint8ArraySetFromBase64(t *testing.T) {
	testCases := []struct {
		name      string
		script    string
		errorName string
	}{
		// ---------SyntaxError---------
		{
			name:      "non-base64-character",
			script:    `new Uint8Array(8).setFromBase64("ab#c")`,
			errorName: "SyntaxError",
		},
		{
			// a trailing chunk of length 1 is invalid even in loose mode
			name:      "single-extra-character",
			script:    `new Uint8Array(8).setFromBase64("abcde")`,
			errorName: "SyntaxError",
		},
		{
			name:      "strict-missing-padding",
			script:    `new Uint8Array(8).setFromBase64("aGVsbG8", { lastChunkHandling: "strict" })`,
			errorName: "SyntaxError",
		},
		// ---------TypeError---------
		{
			// the argument must be a String, there is no ToString coercion
			name:      "non-string-input",
			script:    `new Uint8Array(8).setFromBase64(1234)`,
			errorName: "TypeError",
		},
		{
			name:      "options-null",
			script:    `new Uint8Array(8).setFromBase64("aGVs", null)`,
			errorName: "TypeError",
		},
		{
			name:      "alphabet-unknown",
			script:    `new Uint8Array(8).setFromBase64("aGVs", { alphabet: "base16" })`,
			errorName: "TypeError",
		},
		{
			name:      "last-chunk-handling-unknown",
			script:    `new Uint8Array(8).setFromBase64("aGVs", { lastChunkHandling: "foo" })`,
			errorName: "TypeError",
		},
		{
			// the receiver must be a Uint8Array
			name:      "receiver-not-uint8array",
			script:    `Uint8Array.prototype.setFromBase64.call(new Int8Array(8), "aGVs")`,
			errorName: "TypeError",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vm := New()
			_, err := vm.RunString(fmt.Sprintf(assertThrows, tc.script, tc.errorName))
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

// The base64/hex methods (23.3.1 and 23.3.2) are additional properties of the
// Uint8Array constructor and Uint8Array.prototype only:
// they must not exist on any other TypedArray.
func TestBase64HexWithoutUint8Array(t *testing.T) {
	methods := []struct {
		name   string
		script string // %s is replaced with a TypedArray constructor name
	}{
		{"fromHex", `%s.fromHex("aabb")`},
		{"setFromHex", `new %s(8).setFromHex("aabb")`},
		{"toHex", `new %s(8).toHex()`},
		{"fromBase64", `%s.fromBase64("aGVsbG8=")`},
		{"setFromBase64", `new %s(8).setFromBase64("aGVsbG8=")`},
		{"toBase64", `new %s(8).toBase64()`},
	}
	typedArrays := []string{
		"Int8Array", "Uint8ClampedArray",
		"Int16Array", "Uint16Array",
		"Int32Array", "Uint32Array",
		"Float32Array", "Float64Array",
		"BigInt64Array", "BigUint64Array",
	}

	for _, m := range methods {
		t.Run(m.name, func(t *testing.T) {
			for _, ta := range typedArrays {
				t.Run(ta, func(t *testing.T) {
					vm := New()
					script := fmt.Sprintf(m.script, ta)
					_, err := vm.RunString(fmt.Sprintf(assertThrows, script, "TypeError"))
					if err != nil {
						t.Fatal(err)
					}
				})
			}
		})
	}
}
