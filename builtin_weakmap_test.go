package goja

import (
	"runtime"
	"strings"
	"testing"
	"time"
	"weak"
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

func TestWeakMapValueRefersKey(t *testing.T) {
	vm := New()
	key := vm.NewObject()

	key.Set("pad", strings.Repeat("x", 200))
	value := vm.NewObject()
	value.Set("owner", key)
	var wm weakMap
	wm.set(key, value)
	if !wm.has(key) {
		t.Fatal("weak map does not have key")
	}
	ref := weak.Make(key)
	key = nil
	value = nil

	for range 5 {
		runtime.GC()
		runtime.GC()
		if ref.Value() == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	runtime.KeepAlive(&wm)
	t.Fatal("the key has not been garbage-collected")
}

func TestWeakMapValueRemovedAfterMapIsGone(t *testing.T) {
	vm := New()
	key := vm.NewObject()
	value := vm.NewObject()
	value.Set("pad", strings.Repeat("x", 200))
	wm := &weakMap{}
	wm.set(key, value)

	ref := weak.Make(value)
	wm = nil
	value = nil

	for range 5 {
		runtime.GC()
		runtime.GC()
		if ref.Value() == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	runtime.KeepAlive(key)
	t.Fatal("the value has not been garbage-collected")
}
