package goja

import (
	"runtime"
	"testing"
	"time"
	"unsafe"
)

func TestWeakMap(t *testing.T) {
	vm := New()
	_, err := vm.RunString(`
	var m = new WeakMap();
	var m1 = new WeakMap();
	var key = {};
	m.set(key, true);
	m1.set(key, false);
	if (!m.has(key)) {
		throw new Error("has");
	}
	if (m.get(key) !== true) {
		throw new Error("value does not match");
	}
	if (!m1.has(key)) {
		throw new Error("has (m1)");
	}
	if (m1.get(key) !== false) {
		throw new Error("m1 value does not match");
	}
	m.delete(key);
	if (m.has(key)) {
		throw new Error("m still has after delete");
	}
	if (!m1.has(key)) {
		throw new Error("m1 does not have after delete from m");
	}
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWeakMapGetAdderGetIteratorOrder(t *testing.T) {
	const SCRIPT = `
	let getterCalled = 0;

	class M extends WeakMap {
	    get set() {
	        getterCalled++;
	        return null;
	    }
	}

	let getIteratorCalled = 0;

	let iterable = {};
	iterable[Symbol.iterator] = () => {
	    getIteratorCalled++
	    return {
	        next: 1
	    };
	}

	let thrown = false;

	try {
	    new M(iterable);
	} catch (e) {
	    if (e instanceof TypeError) {
	        thrown = true;
	    } else {
	        throw e;
	    }
	}

	thrown && getterCalled === 1 && getIteratorCalled === 0;
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestWeakMapUpdateKey(t *testing.T) {
	const SCRIPT = `
	const m = new WeakMap();
	let key = {};
	m.set(key, 1);
	m.set(key, 2);
	m.get(key);
	`
	testScript(SCRIPT, intToValue(2), t)
}

func TestWeakMapCleanup(t *testing.T) {
	t.Parallel()
	vm := New()
	_, err := vm.RunString(`
		var m = new WeakMap();
		var key = {};
		m.set(key, true);
	`)
	if err != nil {
		t.Fatal(err)
	}
	vm.Set("key", _undefined)
	runtime.GC()
	m, _ := vm.Get("m").(*Object)
	if m == nil {
		t.Fatal("m is not an Object")
	}
	wmo := m.self.(*weakMapObject)
	if wmo == nil {
		t.Fatal("m is not a WeakMap")
	}
	for range 5 {
		wmo.m.Lock()
		if l := len(wmo.m.m); l == 0 {
			wmo.m.Unlock()
			return
		}
		wmo.m.Unlock()
		runtime.GC()
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("m is not empty")
}

func TestWeakMapKeyAddressReuse(t *testing.T) {
	vm := New()
	val, err := vm.RunString("new WeakMap()")
	if err != nil {
		t.Fatal(err)
	}
	wmo := val.(*Object).self.(*weakMapObject)

	keyA := vm.NewObject()
	wmo.m.set(keyA, asciiString("stale"))
	addrA := uintptr(unsafe.Pointer(keyA))

	wmo.m.Lock()
	var unlocked bool
	defer func() {
		if !unlocked {
			wmo.m.Unlock()
		}
	}()

	keyA = nil
	runtime.GC()

	var fresh *Object
	for range 1024 {
		f := vm.NewObject()
		if uintptr(unsafe.Pointer(f)) == addrA {
			fresh = f
			break
		}
	}
	if fresh == nil {
		unlocked = true
		wmo.m.Unlock()
		t.Skip("allocator did not reuse key address")
	}

	if v := wmo.m.m[fresh.getId()]; v != nil {
		unlocked = true
		wmo.m.Unlock()
		t.Fatalf("fresh key at recycled address read stale value %v", v)
	}

	wmo.m.m[fresh.getId()] = asciiString("fresh_value")
	unlocked = true
	wmo.m.Unlock()

	for i := 0; i < 50; i++ {
		runtime.GC()
		time.Sleep(2 * time.Millisecond)
	}

	if v := wmo.m.get(fresh); v == nil || v.String() != "fresh_value" {
		t.Fatalf("live key's entry was prematurely deleted by dead key cleanup: got %v, want fresh_value", v)
	}
}

func BenchmarkWeakMapObjectKeyLookup(b *testing.B) {
	vm := New()
	v, err := vm.RunString("new WeakMap()")
	if err != nil {
		b.Fatal(err)
	}
	wmo := v.(*Object).self.(*weakMapObject)
	keys := make([]*Object, 64)
	for i := range keys {
		keys[i] = vm.NewObject()
		wmo.m.set(keys[i], intToValue(int64(i)))
	}
	var s Value
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, k := range keys {
			s = wmo.m.get(k)
		}
	}
	_ = s
}
