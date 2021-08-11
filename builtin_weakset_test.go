package goja

import (
	"testing"
)

func TestWeakSetBasic(t *testing.T) {
	const SCRIPT = `
	var s = new WeakSet();
	var o = {};
	s.add(o);
	if (!s.has(o)) {
		throw new Error("has");
	}
	s.delete(o);
	if (s.has(o)) {
		throw new Error("still has");
	}
	`
	testScript1(SCRIPT, _undefined, t)
}

func TestWeakSetArraySimple(t *testing.T) {
	const SCRIPT = `
	var o1 = {}, o2 = {}, o3 = {};
	
	var s = new WeakSet([o1, o2, o3]);
	s.has(o1) && s.has(o2) && s.has(o3);
	`
	testScript1(SCRIPT, valueTrue, t)
}

func TestWeakSetArrayGeneric(t *testing.T) {
	const SCRIPT = `
	var o1 = {}, o2 = {}, o3 = {};
	var a = new Array();
	var s;
	var thrown = false;
	a[1] = o2;
	
	try {
		s = new WeakSet(a);
	} catch (e) {
		if (e instanceof TypeError) {
			thrown = true;
		}
	}
	if (!thrown) {
		throw new Error("Case 1 does not throw");
	}
 
	Object.defineProperty(a.__proto__, "0", {value: o1, writable: true, enumerable: true, configurable: true});
	s = new WeakSet(a);
	if (!(s.has(o1) && s.has(o2) && !s.has(o3))) {
		throw new Error("Case 2 failed");
	}

	Object.defineProperty(a, "2", {value: o3, configurable: true});	
	s = new WeakSet(a);
	s.has(o1) && s.has(o2) && s.has(o3);
	`
	testScript1(SCRIPT, valueTrue, t)
}
