package goja

import (
	"testing"
)

/*
func TestArrayBufferNew(t *testing.T) {
	const SCRIPT = `
	var b = new ArrayBuffer(16);
	b.byteLength;
	`

	testScript1(SCRIPT, intToValue(16), t)
}
*/

func TestArrayBufferSetUint32(t *testing.T) {
	vm := New()
	b := vm._newArrayBuffer(vm.global.ArrayBufferPrototype, nil)
	b.data = make([]byte, 4)
	b.setUint32(0, 0xCAFEBABE, bigEndian)

	i := b.getUint32(0, bigEndian)
	if i != 0xCAFEBABE {
		t.Fatal(i)
	}
	i = b.getUint32(0, littleEndian)
	if i != 0xBEBAFECA {
		t.Fatal(i)
	}

	b.setUint32(0, 0xBEBAFECA, littleEndian)
	i = b.getUint32(0, bigEndian)
	if i != 0xCAFEBABE {
		t.Fatal(i)
	}
}

func TestArrayBufferSetInt32(t *testing.T) {
	vm := New()
	b := vm._newArrayBuffer(vm.global.ArrayBufferPrototype, nil)
	b.data = make([]byte, 4)
	b.setInt32(0, -42, littleEndian)
	if v := b.getInt32(0, littleEndian); v != -42 {
		t.Fatal(v)
	}

	b.setInt32(0, -42, bigEndian)
	if v := b.getInt32(0, bigEndian); v != -42 {
		t.Fatal(v)
	}
}

func TestNewUint8Array(t *testing.T) {
	const SCRIPT = `
	var a = new Uint8Array(1);
	a[0] = 42;
	a.byteLength === 1 && a.length === 1 && a[0] === 42;
	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestNewUint16Array(t *testing.T) {
	const SCRIPT = `
	var a = new Uint16Array(1);
	a[0] = 42;
	a.byteLength === 2 && a.length === 1 && a[0] === 42;
	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestTypedArraysSpeciesConstructor(t *testing.T) {
	const SCRIPT = `
    'use strict';
    function MyArray() {
        var NewTarget = this.__proto__.constructor;
        return Reflect.construct(Uint16Array, arguments, NewTarget);
    }
    MyArray.prototype = Object.create(Uint16Array.prototype, {
        constructor: {
            value: MyArray,
            writable: true,
            configurable: true
        }
    });
    var a = new MyArray(1);
    Object.defineProperty(MyArray, Symbol.species, {value: Uint8Array, configurable: true});
    a[0] = 32767;
    var b = a.filter(function() {
        return true;
    });
	if (a[0] !== 32767) {
		throw new Error("a[0]=" + a[0]); 
	}
	if (!(b instanceof Uint8Array)) {
		throw new Error("b instanceof Uint8Array");
	}
	if (b[0] != 255) {
		throw new Error("b[0]=" + b[0]);
	}
	`

	testScript1(SCRIPT, _undefined, t)
}

func TestTypedArrayFromArrayBuffer(t *testing.T) {
	const SCRIPT = `
	var buf = new ArrayBuffer(2);
	var a16 = new Uint16Array(buf);
	if (!(a16 instanceof Uint16Array)) {
		throw new Error("a16 is not an instance");
	}
	if (a16.buffer !== buf) {
		throw new Error("a16.buffer !== buf");
	}
	if (a16.length !== 1) {
		throw new Error("a16.length=" + a16.length);
	}
	var a8 = new Uint8Array(buf);
	a8.fill(0xAA);
	if (a16[0] !== 0xAAAA) {
		throw new Error("a16[0]=" + a16[0]);
	}
	`

	testScript1(SCRIPT, _undefined, t)
}

func TestTypedArraySetOverlapDifSize(t *testing.T) {
	const SCRIPT = `
	var buf = new ArrayBuffer(4);
	var src = new Uint8Array(buf, 1, 2);
	src[0] = 1;
	src[1] = 2;
	var dst = new Uint16Array(buf);
	dst.set(src);
	if (dst[0] !== 1 || dst[1] !== 2) {
		throw new Error("dst: " + dst.join(","));
	}	
	`
	testScript1(SCRIPT, _undefined, t)
}

func TestTypedArraySetOverlapDifSize2(t *testing.T) {
	const SCRIPT = `
	var buf = new ArrayBuffer(4);
	var src = new Uint8Array(buf, 0, 2);
	src[0] = 1;
	src[1] = 2;
	var dst = new Uint16Array(buf);
	dst.set(src);
	if (dst[0] !== 1 || dst[1] !== 2) {
		throw new Error("dst: " + dst.join(","));
	}	
	`
	testScript1(SCRIPT, _undefined, t)
}

func TestTypedArraySetOverlapDifSize3(t *testing.T) {
	const SCRIPT = `
	var buf = new ArrayBuffer(8);
	var src = new Uint8Array(buf, 2, 4);
	src[0] = 1;
	src[1] = 2;
	src[2] = 3;
	src[3] = 4;
	var dst = new Uint16Array(buf);
	dst.set(src);
	if (dst[0] !== 1 || dst[1] !== 2 || dst[2] !== 3 || dst[3] !== 4) {
		throw new Error("dst: " + dst.join(","));
	}	
	`
	testScript1(SCRIPT, _undefined, t)
}

func TestTypedArraySetOverlapDifSize4(t *testing.T) {
	const SCRIPT = `
	var buf = new ArrayBuffer(10);
	var dst = new Uint8Array(buf, 2, 5);
	var src = new Uint16Array(buf);
	src[0] = 1;
	src[1] = 2;
	src[2] = 3;
	src[3] = 4;
	src[4] = 5;
	dst.set(src);
	if (dst[0] !== 1 || dst[1] !== 2 || dst[2] !== 3 || dst[3] !== 4 || dst[4] !== 5) {
		throw new Error("dst: " + dst.join(","));
	}	
	`
	testScript1(SCRIPT, _undefined, t)
}

func TestTypedArraySetNoOverlapDifSizeForward(t *testing.T) {
	const SCRIPT = `
	var buf = new ArrayBuffer(10);
	var dst = new Uint8Array(buf, 7, 2);
	var src = new Uint16Array(buf, 0, 2);
	src[0] = 1;
	src[1] = 2;
	dst.set(src);
	if (dst[0] !== 1 || dst[1] !== 2 || src[0] !== 1 || src[1] !== 2) {
		throw new Error("dst: " + dst.join(","));
	}	
	`
	testScript1(SCRIPT, _undefined, t)
}

func TestTypedArraySetNoOverlapDifSizeBackward(t *testing.T) {
	const SCRIPT = `
	var buf = new ArrayBuffer(10);
	var dst = new Uint8Array(buf, 0, 2);
	var src = new Uint16Array(buf, 6, 2);
	src[0] = 1;
	src[1] = 2;
	dst.set(src);
	if (dst[0] !== 1 || dst[1] !== 2 || src[0] !== 1 || src[1] !== 2) {
		throw new Error("dst: " + dst.join(","));
	}	
	`
	testScript1(SCRIPT, _undefined, t)
}

func TestTypedArraySetNoOverlapDifSizeDifBuffers(t *testing.T) {
	const SCRIPT = `
	var dstBuf = new ArrayBuffer(1024);
	var dst = new Uint8Array(dstBuf, 0, 2);
	var src = new Uint16Array(2);
	src[0] = 1;
	src[1] = 2;
	dst.set(src);
	if (dst[0] !== 1 || dst[1] !== 2 || src[0] !== 1 || src[1] !== 2) {
		throw new Error("dst: " + dst.join(","));
	}	
	`
	testScript1(SCRIPT, _undefined, t)
}

func TestTypedArraySliceSameType(t *testing.T) {
	const SCRIPT = `
	var src = Uint8Array.of(1,2,3,4);
	var dst = src.slice(1, 3);
	if (dst.length !== 2 || dst[0] !== 2 || dst[1] !== 3) {
		throw new Error("dst: " + dst.join(","));
	}	
	`
	testScript1(SCRIPT, _undefined, t)
}

func TestTypedArraySliceDifType(t *testing.T) {
	const SCRIPT = `
	var src = Uint8Array.of(1,2,3,4);
	Object.defineProperty(Uint8Array, Symbol.species, {value: Uint16Array, configurable: true});
	var dst = src.slice(1, 3);
	if (!(dst instanceof Uint16Array)) {
		throw new Error("wrong dst type: " + dst);
	}
	if (dst.length !== 2 || dst[0] !== 2 || dst[1] !== 3) {
		throw new Error("dst: " + dst.join(","));
	}	
	`
	testScript1(SCRIPT, _undefined, t)
}

func TestTypedArraySortComparatorReturnValueFloats(t *testing.T) {
	const SCRIPT = `
	var a = Float64Array.of(
		5.97,
		9.91,
		4.13,
		9.28,
		3.29
	);
	a.sort( function(a, b) { return a - b; } );
	for (var i = 1; i < a.length; i++) {
		if (a[i] < a[i-1]) {
			throw new Error("Array is not sorted: " + a);
		}
	}
	`
	testScript1(SCRIPT, _undefined, t)
}

func TestTypedArraySortComparatorReturnValueNegZero(t *testing.T) {
	const SCRIPT = `
	var a = new Uint8Array([2, 1]);
	a.sort( function(a, b) { return a > b ? 0 : -0; } );
	for (var i = 1; i < a.length; i++) {
		if (a[i] < a[i-1]) {
			throw new Error("Array is not sorted: " + a);
		}
	}
	`
	testScript1(SCRIPT, _undefined, t)
}

func TestInt32ArrayNegativeIndex(t *testing.T) {
	const SCRIPT = `
	new Int32Array()[-1] === undefined;
	`

	testScript1(SCRIPT, valueTrue, t)
}
