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
