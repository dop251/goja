package goja

import (
	"runtime"
	"strings"
	"sync"
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

func TestWeakMapGetOrInsert(t *testing.T) {
	const SCRIPT = `
	const m = new WeakMap();
	const key = {};
	assert.sameValue(m.getOrInsert(key, 1), 1);
	assert.sameValue(m.getOrInsert(key, 2), 1);
	assert.sameValue(m.getOrInsertComputed(key, function() { throw new Error("should not be called"); }), 1);
	const key1 = {};
	assert.sameValue(m.getOrInsertComputed(key1, function(k) {
		assert.sameValue(k, key1);
		m.set(k, 1);
		return 2;
	}), 2);
	assert.sameValue(m.get(key1), 2);
	assert.throws(TypeError, function() { m.getOrInsert(1, 1); });
	assert.throws(TypeError, function() { m.getOrInsertComputed(1, function() {}); });
	assert.throws(TypeError, function() { m.getOrInsertComputed(key, null); });
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestWeakMapValueRefersKey(t *testing.T) {
	vm := New()
	key := vm.NewObject()

	key.Set("pad", strings.Repeat("x", 200))
	value := vm.NewObject()
	value.Set("owner", key)
	var wm weakMap
	wm.set(key.getWeakMapRefs(true), value)
	if !wm.has(key.getWeakMapRefs(false)) {
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
	wm.set(key.getWeakMapRefs(true), value)

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

func TestWeakMapSymbolKeys(t *testing.T) {
	const SCRIPT = `
	var m = new WeakMap();
	var s = new WeakSet();
	var sym = Symbol("key");
	m.set(sym, 1);
	m.set(Symbol.iterator, 2);
	s.add(sym);
	assert.sameValue(m.get(sym), 1, "unique symbol");
	assert.sameValue(m.get(Symbol.iterator), 2, "well-known symbol");
	assert(s.has(sym), "WeakSet has symbol");
	assert(m.delete(sym), "delete");
	assert(!m.has(sym), "has after delete");
	assert.sameValue(m.getOrInsert(sym, 3), 3, "getOrInsert");
	assert.sameValue(m.getOrInsertComputed(sym, function() { throw new Error("should not be called"); }), 3, "getOrInsertComputed existing");
	var sym1 = Symbol();
	assert.sameValue(m.getOrInsertComputed(sym1, function(k) { assert.sameValue(k, sym1); return 4; }), 4, "getOrInsertComputed");
	assert.sameValue(m.get(sym1), 4, "get after getOrInsertComputed");

	var registered = Symbol.for("key");
	assert.throws(TypeError, function() { m.set(registered, 1); }, "WeakMap.prototype.set");
	assert.throws(TypeError, function() { s.add(registered); }, "WeakSet.prototype.add");
	assert.throws(TypeError, function() { m.getOrInsert(registered, 1); }, "WeakMap.prototype.getOrInsert");
	assert.throws(TypeError, function() { m.getOrInsertComputed(registered, function() {}); }, "WeakMap.prototype.getOrInsertComputed");
	assert.throws(TypeError, function() { new WeakMap([[registered, 1]]); }, "WeakMap constructor");
	assert.throws(TypeError, function() { new WeakSet([registered]); }, "WeakSet constructor");
	assert.sameValue(m.get(registered), undefined, "WeakMap.prototype.get");
	assert(!m.has(registered), "WeakMap.prototype.has");
	assert(!s.delete(registered), "WeakSet.prototype.delete");

	assert.throws(TypeError, function() { new WeakMap([[]]); }, "missing key");
	assert.throws(TypeError, function() { new WeakSet([,]); }, "hole");
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestWeakMapSymbolKeyGarbageCollected(t *testing.T) {
	vm := New()
	key := NewSymbol("key")
	value := vm.NewObject()
	value.Set("owner", key)
	var wm weakMap
	wm.set(key.getWeakMapRefs(true), value)
	if !wm.has(key.getWeakMapRefs(false)) {
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

func TestWeakMapWellKnownSymbolKeyConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			vm := New()
			_, err := vm.RunString(`
			for (var i = 0; i < 100; i++) {
				var m = new WeakMap();
				m.set(Symbol.iterator, i);
				if (m.get(Symbol.iterator) !== i) {
					throw new Error("value does not match");
				}
			}
			`)
			if err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
}
