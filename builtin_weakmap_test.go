package goja

import (
	"runtime"
	"testing"
	"time"
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
