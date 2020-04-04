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
	try {
		(b[0]);
		throw new Error("expected TypeError");
	} catch (e) {
		if (!(e instanceof TypeError)) {
			throw e;
		}
	}
	`)
	if err != nil {
		t.Fatal(err)
	}
}
