package goja

import (
	"testing"
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
