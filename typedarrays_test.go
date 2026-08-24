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
	vm := New()

	// valid-hex string
	retH, err := vm.RunString(`
	var arr = Uint8Array.fromHex("0123456789ABcdEf");
	arr.setFromHex("0123456789ABcdEf");
	arr.toHex();
	`)
	if err != nil {
		t.Fatal(err)
	}
	bufStr := retH.Export().(string) // means Hex string
	if bufStr != "0123456789abcdef" {
		t.Fatal(bufStr)
	}

	// length[Uint8Array] < length(setFromHex)
	retH, err = vm.RunString(`
	var arr = Uint8Array.fromHex("01234567");
	arr.setFromHex("0123456789ABcdEf");
	arr.toHex();
	`)
	if err != nil {
		t.Fatal(err)
	}
	bufStr2 := retH.Export().(string) // means Hex string
	if bufStr2 != "01234567" {
		t.Fatal(bufStr2)
	}

	// length[Uint8Array] > length(setFromHex)
	retH, err = vm.RunString(`
	var arr = Uint8Array.fromHex("0123456789ABcdEf");
	arr.setFromHex("AABBCCDD");
	arr.toHex();
	`)
	if err != nil {
		t.Fatal(err)
	}
	bufStr3 := retH.Export().(string) // means Hex string
	if bufStr3 != "aabbccdd89abcdef" {
		t.Fatal(bufStr3)
	}

	// length[Uint8Array] > length(setFromHex)
	retH, err = vm.RunString(`
	var arr = new Uint8Array(5);
	arr.setFromHex("AABBCCDD");
	arr.toHex();
	`)
	if err != nil {
		t.Fatal(err)
	}
	bufStr4 := retH.Export().(string) // means Hex string
	if bufStr4 != "aabbccdd00" {
		t.Fatal(bufStr4)
	}

	// offset length[Uint8Array] > length(setFromHex)
	retH, err = vm.RunString(`
	var arr = new Uint8Array(8);
	arr.subarray(3).setFromHex("cafed00d");
	arr.toHex();
	`)
	if err != nil {
		t.Fatal(err)
	}
	bufStr5 := retH.Export().(string) // means Hex string
	if bufStr5 != "000000cafed00d00" {
		t.Fatal(bufStr5)
	}

	// offset + length[Uint8Array] < length(setFromHex)
	retH, err = vm.RunString(`
	var arr = new Uint8Array(5);
	arr.subarray(3).setFromHex("cafed00d");
	arr.toHex();
	`)
	if err != nil {
		t.Fatal(err)
	}
	bufStr6 := retH.Export().(string) // means Hex string
	if bufStr6 != "000000cafe" {
		t.Fatal(bufStr6)
	}

	t.Run("read", func(t *testing.T) {
		ret, err := vm.RunString(`
		var arr = new Uint8Array(8);
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
		var arr = new Uint8Array(8);
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

func TestInvalidUint8ArrayFromHex(t *testing.T) {
	vm := New()

	t.Run("non-hex-character", func(t *testing.T) {
		_, err := vm.RunString(`
		var arr = Uint8Array.fromHex("01234567");
		arr.setFromHex("aabZ"); // Invalid hex character
		`)
		if err == nil {
			t.Fatal("Expected error but got none")
		}
	})

	t.Run("odd-length", func(t *testing.T) {
		_, err := vm.RunString(`
		var arr = Uint8Array.fromHex("01234567");
		arr.setFromHex("aab"); // Odd length hex character
		`)
		if err == nil {
			t.Fatal("Expected error but got none")
		}
	})

	t.Run("contain-whitespace", func(t *testing.T) {
		_, err := vm.RunString(`
		var arr = Uint8Array.fromHex("01234567");
		arr.setFromHex("aa  bb"); // Not contain whitespace
		`)
		if err == nil {
			t.Fatal("Expected error but got none")
		}
	})
}

func TestInvalidUint8ArraySetFromHex(t *testing.T) {
	vm := New()

	t.Run("non-hex-character", func(t *testing.T) {
		_, err := vm.RunString(`
		var b = Uint8Array.fromHex("aabZ"); // Invalid hex character
		`)
		if err == nil {
			t.Fatal("Expected error but got none")
		}
	})

	t.Run("odd-length", func(t *testing.T) {
		_, err := vm.RunString(`
		var b = Uint8Array.fromHex("aab"); // Odd length hex character
		`)
		if err == nil {
			t.Fatal("Expected error but got none")
		}
	})

	t.Run("contain-whitespace", func(t *testing.T) {
		_, err := vm.RunString(`
		var b = Uint8Array.fromHex("aa  bb"); // Not contain whitespace
		`)
		if err == nil {
			t.Fatal("Expected error but got none")
		}
	})
}

func TestFromHexWithoutUint8Array(t *testing.T) {
	vm := New()
	t.Run("int8", func(t *testing.T) {
		_, err := vm.RunString(`Int8Array.fromHex("aabb");`)
		if err == nil {
			t.Fatal("Int8Array must not have fromHex method")
		}
	})
	t.Run("uint16", func(t *testing.T) {
		_, err := vm.RunString(`Uint16Array.fromHex("aabb");`)
		if err == nil {
			t.Fatal("Uint16Array must not have fromHex method")
		}
	})
}

func TestSetFromHexWithoutUint8Array(t *testing.T) {
	vm := New()
	t.Run("int8", func(t *testing.T) {
		_, err := vm.RunString(`
		var int8Array = new Int8Array(8);;
		int8Array.setFromHex("cafed00d");
		`)
		if err == nil {
			t.Fatal("Int8Array must not have setFromHex method")
		}
	})
	t.Run("uint16", func(t *testing.T) {
		_, err := vm.RunString(`
		var uint16Array = new Uint16Array(16);;
		uint16Array.setFromHex("cafed00dcafed00d");
		`)
		if err == nil {
			t.Fatal("Uint16Array must not have setFromHex method")
		}
	})
}

func TestToFromHexWithoutUint8Array(t *testing.T) {
	vm := New()
	t.Run("int8", func(t *testing.T) {
		_, err := vm.RunString(`
		var int8Array = new Int8Array(8);;
		int8Array.toHex();
		`)
		if err == nil {
			t.Fatal("Int8Array must not have toHex method")
		}
	})
	t.Run("uint16", func(t *testing.T) {
		_, err := vm.RunString(`
		var uint16Array = new Uint16Array(16);;
		uint16Array.toHex();
		`)
		if err == nil {
			t.Fatal("Uint16Array must not have toHex method")
		}
	})
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

func TestUint8ArrayFromBase64Url(t *testing.T) {
	testCases := []struct {
		name     string
		script   string
		expected string
	}{
		// ---------loose---------
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
		// ---------strict---------
		{
			name:     "strict-base64url-missing-padding",
			script:   `Uint8Array.fromBase64("aGVsbG8-ISE_", { lastChunkHandling: "strict", alphabet: "base64url" });`,
			expected: "hello>!!?",
		},
		{
			name:     "strict-base64url-with-padding",
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

func TestUint8ArrayFromBase64LastChunkHandling(t *testing.T) {
	vm := New()

	t.Run("strict-with-padding", func(t *testing.T) {
		retH, err := vm.RunString(`
		Uint8Array.fromBase64("aGVsbG8gd29ybGQ=", { lastChunkHandling: "strict" });
		`)
		if err != nil {
			t.Fatal(err)
		}
		if s := string(retH.Export().([]byte)); s != "hello world" {
			t.Fatal(s)
		}
	})

	t.Run("strict-missing-padding", func(t *testing.T) {
		_, err := vm.RunString(`
		var ok = false;
		try {
			Uint8Array.fromBase64("aGVsbG8", { lastChunkHandling: "strict" });
		} catch (e) {
			ok = e instanceof SyntaxError;
		}
		if (!ok) {
			throw new Error("Expected a SyntaxError");
		}
		`)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("strict-extra-bits", func(t *testing.T) {
		// "QR==": the last character contributes non-zero bits beyond the single decoded byte
		_, err := vm.RunString(`
		var ok = false;
		try {
			Uint8Array.fromBase64("QR==", { lastChunkHandling: "strict" });
		} catch (e) {
			ok = e instanceof SyntaxError;
		}
		if (!ok) {
			throw new Error("Expected a SyntaxError");
		}
		var loose = Uint8Array.fromBase64("QR==");
		if (loose.length !== 1 || loose[0] !== 65) {
			throw new Error("loose should decode QR== to [65]");
		}
		`)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("stop-before-partial", func(t *testing.T) {
		retH, err := vm.RunString(`
		Uint8Array.fromBase64("aGVsbG8", { lastChunkHandling: "stop-before-partial" });
		`)
		if err != nil {
			t.Fatal(err)
		}
		// only the full-length chunk "aGVs" is decoded, the partial chunk "bG8" is ignored
		if s := string(retH.Export().([]byte)); s != "hel" {
			t.Fatal(s)
		}
	})
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
			name:      "single-extra-character",
			script:    `Uint8Array.fromBase64("abcde")`,
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

func TestFromBase64WithoutUint8Array(t *testing.T) {
	vm := New()
	t.Run("int8", func(t *testing.T) {
		_, err := vm.RunString(`Int8Array.fromBase64("aGVsbG8=");`)
		if err == nil {
			t.Fatal("Int8Array must not have fromBase64 method")
		}
	})
	t.Run("uint16", func(t *testing.T) {
		_, err := vm.RunString(`Uint16Array.fromBase64("aGVsbG8=");`)
		if err == nil {
			t.Fatal("Uint16Array must not have fromBase64 method")
		}
	})
}
