package goja

import "testing"

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
