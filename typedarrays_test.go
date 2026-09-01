package goja

import (
	"bytes"
	stdbase64 "encoding/base64"
	stdhex "encoding/hex"
	"fmt"
	"strings"
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
	const SCRIPT = `
	var arr = Uint8Array.fromHex("0123456789ABcdEf");
	assert.sameValue(arr.toHex(), "0123456789abcdef", "valid-hex string");
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestUint8ArraySetFromHex(t *testing.T) {
	const SCRIPT = `
	var arr = Uint8Array.fromHex("0123456789ABcdEf");
	arr.setFromHex("0123456789ABcdEf");
	assert.sameValue(arr.toHex(), "0123456789abcdef", "valid-hex-string");

	// offset length[Uint8Array] > length(setFromHex)
	var arr2 = new Uint8Array(8);
	arr2.subarray(3).setFromHex("cafed00d");
	assert.sameValue(arr2.toHex(), "000000cafed00d00", "subarray-longer-than-input");
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

// Whatever was decoded before an error must still be written into the
// destination: the spec performs SetUint8ArrayBytes before the throw.
func TestUint8ArraySetFromHexPartialWrite(t *testing.T) {
	const SCRIPT = `
	var arr = new Uint8Array(4);
	assert.throws(SyntaxError, function() {
		arr.setFromHex("aabbZZcc");
	}, "invalid-character-mid-string");
	// writes "aabb" and then throws on "ZZ"
	assert.sameValue(arr.toHex(), "aabb0000", "invalid-character-mid-string");

	var arr2 = new Uint8Array(6);
	assert.throws(SyntaxError, function() {
		arr2.subarray(2).setFromHex("aabbZZcc");
	}, "invalid-character-mid-string-subarray");
	// writes "____aabb" and then throws on "ZZ"
	assert.sameValue(arr2.toHex(), "0000aabb0000", "invalid-character-mid-string-subarray");
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestUint8ArrayFromBase64(t *testing.T) {
	testCases := []struct {
		name     string
		script   string
		expected string
	}{
		// ---------loose---------
		{
			// whitespace around the padding is not covered by test262
			name:     "loose-base64-with-whitespace",
			script:   `Uint8Array.fromBase64(" aGVs\tb\nG8\r\n = ")`,
			expected: "hello",
		},
		// ---------strict---------
		{
			// input is an exact multiple of 4 chars: no padding needed even in strict mode
			name:     "strict-base64-missing-padding",
			script:   `Uint8Array.fromBase64("aGVsbG8x", { lastChunkHandling: "strict" })`,
			expected: "hello1",
		},
		{
			name:     "strict-base64-with-whitespace",
			script:   `Uint8Array.fromBase64(" aGVs\tb\nG8\r\nx ", { lastChunkHandling: "strict" })`,
			expected: "hello1",
		},
		{
			name:     "strict-base64-empty",
			script:   `Uint8Array.fromBase64("", { lastChunkHandling: "strict" })`,
			expected: "",
		},
		// ---------stop-before-partial---------
		{
			name:     "partial-base64-with-whitespace",
			script:   `Uint8Array.fromBase64(" aGVs\tb\nG8\r\n = ", { lastChunkHandling: "stop-before-partial" })`,
			expected: "hello",
		},
		{
			name:     "partial-base64-empty",
			script:   `Uint8Array.fromBase64("", { lastChunkHandling: "stop-before-partial" })`,
			expected: "",
		},

		// ---------base64url----------
		{
			name:     "loose-base64url-with-padding",
			script:   `Uint8Array.fromBase64("aGVsbG8-ISE_YQ==", { alphabet: "base64url"})`,
			expected: "hello>!!?a",
		},
		{
			name:     "loose-base64url-with-whitespace",
			script:   `Uint8Array.fromBase64(" aGVs\tbG\n8-ISE_Y\r\nQ= = ", { alphabet: "base64url"})`,
			expected: "hello>!!?a",
		},
		{
			name:     "loose-base64url-empty",
			script:   `Uint8Array.fromBase64("", { alphabet: "base64url"})`,
			expected: "",
		},
		{
			name:     "strict-base64url-missing-padding",
			script:   `Uint8Array.fromBase64("aGVsbG8-ISE_", { lastChunkHandling: "strict", alphabet: "base64url" })`,
			expected: "hello>!!?",
		},
		{
			name:     "partial-base64url-with-padding",
			script:   `Uint8Array.fromBase64("aGVsbG8-ISE_YQ==", { lastChunkHandling: "stop-before-partial", alphabet: "base64url" })`,
			expected: "hello>!!?a",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expectedHex := stdhex.EncodeToString([]byte(tc.expected))
			script := fmt.Sprintf(`assert.sameValue(%s.toHex(), "%s", "%s");`, tc.script, expectedHex, tc.name)
			testScriptWithTestLib(script, _undefined, t)
		})
	}
}

func TestInvalidUint8ArrayFromBase64(t *testing.T) {
	const SCRIPT = `
	// ---------SyntaxError---------
	assert.throws(SyntaxError, function() {
		Uint8Array.fromBase64("AB=C");
	}, "padding-inside-chunk");
	assert.throws(SyntaxError, function() {
		Uint8Array.fromBase64("AB==C");
	}, "character-after-padding");
	// ---------TypeError---------
	assert.throws(TypeError, function() {
		Uint8Array.fromBase64("aGVsbG8=", null);
	}, "options-null");
	assert.throws(TypeError, function() {
		Uint8Array.fromBase64("aGVsbG8=", "loose");
	}, "options-not-an-object");
	assert.throws(TypeError, function() {
		Uint8Array.fromBase64("aGVsbG8=", { alphabet: null });
	}, "alphabet-null");
	assert.throws(TypeError, function() {
		Uint8Array.fromBase64("aGVsbG8=", { lastChunkHandling: null });
	}, "last-chunk-handling-null");
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestUint8ArrayToBase64(t *testing.T) {
	const SCRIPT = `
	// ---------base64 (default)---------
	var actual = Uint8Array.fromBase64("aGVs").toBase64();
	assert.sameValue(actual, "aGVs", "no-padding-needed");

	var actual = Uint8Array.fromBase64("aGVsbG8gd29ybGQ=").toBase64();
	assert.sameValue(actual, "aGVsbG8gd29ybGQ=", "one-padding-character");

	var actual = Uint8Array.fromBase64("aGVsbA==").toBase64();
	assert.sameValue(actual, "aGVsbA==", "two-padding-characters");

	var actual = new Uint8Array(0).toBase64();
	assert.sameValue(actual, "", "empty");

	// only the bytes of the subarray view are encoded
	var actual = Uint8Array.fromHex("00aabb00").subarray(1, 3).toBase64();
	assert.sameValue(actual, "qrs=", "subarray");

	// ---------alphabet---------
	var actual = Uint8Array.fromHex("fbefbeffffff").toBase64({ alphabet: "base64" });
	assert.sameValue(actual, "++++////", "base64-alphabet-explicit");

	var actual = Uint8Array.fromHex("fbefbeffffff").toBase64({ alphabet: "base64url" });
	assert.sameValue(actual, "----____", "base64url-alphabet");

	var actual = Uint8Array.fromHex("fbef").toBase64({ alphabet: "base64url" });
	assert.sameValue(actual, "--8=", "base64url-alphabet-with-padding");

	// ---------omitPadding---------
	var actual = Uint8Array.fromBase64("aGVsbG8=").toBase64({ omitPadding: true });
	assert.sameValue(actual, "aGVsbG8", "omit-padding");

	var actual = Uint8Array.fromBase64("aGVsbG8=").toBase64({ omitPadding: false });
	assert.sameValue(actual, "aGVsbG8=", "omit-padding-false");

	// omitPadding is coerced with ToBoolean: any truthy value omits the padding
	var actual = Uint8Array.fromBase64("aGVsbG8=").toBase64({ omitPadding: "false" });
	assert.sameValue(actual, "aGVsbG8", "omit-padding-truthy-string");

	var actual = Uint8Array.fromHex("fbef").toBase64({ alphabet: "base64url", omitPadding: true });
	assert.sameValue(actual, "--8", "omit-padding-base64url");
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestInvalidUint8ArrayToBase64(t *testing.T) {
	const SCRIPT = `
	assert.throws(TypeError, function() {
		new Uint8Array(4).toBase64(null);
	}, "options-null");
	assert.throws(TypeError, function() {
		new Uint8Array(4).toBase64("base64");
	}, "options-not-an-object");
	assert.throws(TypeError, function() {
		new Uint8Array(4).toBase64({ alphabet: null });
	}, "alphabet-null");
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestUint8ArraySetFromBase64(t *testing.T) {
	const SCRIPT = `
	var arr = new Uint8Array(4);
	arr.setFromBase64("aGVsbA==");
	assert.sameValue(arr.toHex(), "68656c6c", "same-length");

	// only up to the target size is decoded, the rest of the input is not read
	var arr = new Uint8Array(3);
	arr.setFromBase64("aGVsbG8=");
	assert.sameValue(arr.toHex(), "68656c", "array-shorter-than-input");

	// the bytes beyond the decoded length keep their previous content
	var arr = Uint8Array.fromHex("ffffffffffff");
	arr.setFromBase64("aGVs");
	assert.sameValue(arr.toHex(), "68656cffffff", "array-longer-than-input");

	var arr = new Uint8Array(5);
	arr.setFromBase64("aGVs");
	assert.sameValue(arr.toHex(), "68656c0000", "fresh-array-longer-than-input");

	// only the bytes of the subarray view are written
	var arr = new Uint8Array(8);
	arr.subarray(3).setFromBase64("aGVs");
	assert.sameValue(arr.toHex(), "00000068656c0000", "subarray");

	var arr = new Uint8Array(4);
	arr.setFromBase64(" aGVs\tbA==\n");
	assert.sameValue(arr.toHex(), "68656c6c", "whitespace");

	// the partial last chunk "bG8" is not decoded
	var arr = new Uint8Array(6);
	arr.setFromBase64("aGVsbG8", { lastChunkHandling: "stop-before-partial" });
	assert.sameValue(arr.toHex(), "68656c000000", "stop-before-partial");

	// the target holds 3 bytes: only the first full chunk (4 characters) is read
	var arr = new Uint8Array(3);
	assert.sameValue(arr.setFromBase64("aGVsbG8x").read, 4, "read-stops-at-target-size");
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

// Whatever was decoded before an error must still be written into the
// destination: the spec performs SetUint8ArrayBytes before the throw.
func TestUint8ArraySetFromBase64PartialWrite(t *testing.T) {
	const SCRIPT = `
	var arr = new Uint8Array(6);
	assert.throws(SyntaxError, function() {
		arr.setFromBase64("aGVs#nvalid");
	}, "invalid-character-mid-string");
	assert.sameValue(arr.toHex(), "68656c000000", "invalid-character-mid-string");

	var arr2 = new Uint8Array(8);
	assert.throws(SyntaxError, function() {
		arr2.subarray(2).setFromBase64("aGVs#nvalid");
	}, "invalid-character-mid-string-subarray");
	assert.sameValue(arr2.toHex(), "000068656c000000", "invalid-character-mid-string-subarray");

	var arr3 = new Uint8Array(12);
	assert.throws(SyntaxError, function() {
		arr3.subarray(2).setFromBase64("aGVsBBBB#nvalidá"); // unicode
	}, "invalid-character-mid-string-subarray");
	assert.sameValue(arr3.toHex(), "000068656c04104100000000", "invalid-character-mid-string-subarray-unicode");

	var arr4 = new Uint8Array(12);
	assert.throws(SyntaxError, function() {
		arr4.subarray(2).setFromBase64("aGVsBBB#nvalidá"); // unicode
	}, "invalid-character-mid-string-subarray");
	assert.sameValue(arr4.toHex(), "000068656c00000000000000", "invalid-character-mid-string-subarray-unicode 1");

	var arr5 = new Uint8Array(12);
	assert.throws(SyntaxError, function() {
		arr5.subarray(2).setFromBase64("aGVsBBBĀnvalidá"); // unicode
	}, "invalid-character-mid-string-subarray");
	assert.sameValue(arr5.toHex(), "000068656c00000000000000", "invalid-character-mid-string-subarray-unicode 1");

	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestInvalidUint8ArraySetFromBase64(t *testing.T) {
	const SCRIPT = `
	// ---------SyntaxError---------
	// a trailing chunk of length 1 is invalid even in loose mode
	assert.throws(SyntaxError, function() {
		new Uint8Array(8).setFromBase64("abcde");
	}, "single-extra-character");
	// ---------TypeError---------
	assert.throws(TypeError, function() {
		new Uint8Array(8).setFromBase64("aGVs", null);
	}, "options-null");
	assert.throws(TypeError, function() {
		new Uint8Array(8).setFromBase64("aGVs", { alphabet: "base16" });
	}, "alphabet-unknown");
	assert.throws(TypeError, function() {
		new Uint8Array(8).setFromBase64("aGVs", { lastChunkHandling: "foo" });
	}, "last-chunk-handling-unknown");
	// the receiver must be a Uint8Array
	assert.throws(TypeError, function() {
		Uint8Array.prototype.setFromBase64.call(new Int8Array(8), "aGVs");
	}, "receiver-not-uint8array");
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
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
					script := fmt.Sprintf(`assert.throws(TypeError, function() { %s; });`, fmt.Sprintf(m.script, ta))
					testScriptWithTestLib(script, _undefined, t)
				})
			}
		})
	}
}

// -------------- Uint8Array Base64 Benchmarks

// decoded byte lengths to measure
var uint8ArrayBenchSizes = []int{16, 1 << 10, 64 << 10, 1 << 20}

func hexInput(decodedLen int) String {
	return asciiString(strings.Repeat("a7", decodedLen))
}

func base64Input(decodedLen int) String {
	return asciiString(stdbase64.StdEncoding.EncodeToString([]byte(strings.Repeat("a", decodedLen))))
}

func BenchmarkUint8ArrayTo(b *testing.B) {
	r := New()
	encoders := []struct {
		name string
		f    func(FunctionCall) Value
	}{
		{"hex", r.uint8ArrayProto_toHex},
		{"base64", r.uint8ArrayProto_toBase64},
	}

	for _, encoder := range encoders {
		b.Run(encoder.name, func(b *testing.B) {
			for _, size := range uint8ArrayBenchSizes {
				b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
					call := FunctionCall{
						This: r.newTypedArrayWithData(bytes.Repeat([]byte{0xa7}, size), r.getUint8Array(), r.newUint8ArrayObject, nil).val,
					}
					b.ReportAllocs()
					b.SetBytes(int64(size))
					for b.Loop() {
						encoder.f(call)
					}
				})
			}
		})
	}
}

func BenchmarkUint8ArrayFrom(b *testing.B) {
	r := New()
	decoders := []struct {
		name  string
		input func(int) String
		f     func(FunctionCall) Value
	}{
		{"hex", hexInput, r.uint8Array_fromHex},
		{"base64", base64Input, r.uint8Array_fromBase64},
	}
	for _, decoder := range decoders {
		b.Run(decoder.name, func(b *testing.B) {
			for _, size := range uint8ArrayBenchSizes {
				b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
					in := decoder.input(size)
					call := FunctionCall{
						Arguments: []Value{in},
					}
					b.ReportAllocs()
					b.SetBytes(int64(size))
					for b.Loop() {
						decoder.f(call)
					}
				})
			}
		})
	}
}

func BenchmarkUint8ArraySetFrom(b *testing.B) {
	r := New()
	decoders := []struct {
		name  string
		input func(int) String
		f     func(FunctionCall) Value
	}{
		{"hex", hexInput, r.uint8ArrayProto_setFromHex},
		{"base64", base64Input, r.uint8ArrayProto_setFromBase64},
	}

	for _, decoder := range decoders {
		b.Run(decoder.name, func(b *testing.B) {
			for _, size := range uint8ArrayBenchSizes {
				b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
					in := decoder.input(size)
					call := FunctionCall{
						Arguments: []Value{in},
					}
					// the receiver must be large enough to hold the whole input,
					// otherwise the decoding stops at its length
					call.This = r.newTypedArrayWithData(make([]byte, size), r.getUint8Array(), r.newUint8ArrayObject, nil).val
					b.ReportAllocs()
					b.SetBytes(int64(size))
					for b.Loop() {
						decoder.f(call)
					}
				})
			}
		})
	}
}
