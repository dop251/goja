package goja

import (
	"runtime"
	"testing"
)

func TestWeakMapExpiry(t *testing.T) {
	vm := New()
	_, err := vm.RunString(`
	var m = new WeakMap();
	var key = {};
	m.set(key, true);
	if (!m.has(key)) {
		throw new Error("has");
	}
	if (m.get(key) !== true) {
		throw new Error("value does not match");
	}
	key = undefined;
	`)
	if err != nil {
		t.Fatal(err)
	}
	runtime.GC()
	wmo := vm.Get("m").ToObject(vm).self.(*weakMapObject)
	wmo.m.Lock()
	l := len(wmo.m.data)
	wmo.m.Unlock()
	if l > 0 {
		t.Fatal("Object has not been removed")
	}
}
