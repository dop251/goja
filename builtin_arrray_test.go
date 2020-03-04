package goja

import "testing"

func TestArrayProtoProp(t *testing.T) {
	const SCRIPT = `
	Object.defineProperty(Array.prototype, '0', {value: 42, configurable: true, writable: false})
	var a = []
	a[0] = 1
	a[0]
	`

	testScript1(SCRIPT, valueInt(42), t)
}

func TestArrayDelete(t *testing.T) {
	const SCRIPT = `
	var a = [1, 2];
	var deleted = delete a[0];
	var undef = a[0] === undefined;
	var len = a.length;

	deleted && undef && len === 2;
	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestArrayDeleteNonexisting(t *testing.T) {
	const SCRIPT = `
	Array.prototype[0] = 42;
	var a = [];
	delete a[0] && a[0] === 42;
	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestArraySetLength(t *testing.T) {
	const SCRIPT = `
	var a = [1, 2];
	var assert0 = a.length == 2;
	a.length = "1";
	a.length = 1.0;
	a.length = 1;
	var assert1 = a.length == 1;
	a.length = 2;
	var assert2 = a.length == 2;
	assert0 && assert1 && assert2 && a[1] === undefined;

	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestArrayReverseNonOptimisable(t *testing.T) {
	const SCRIPT = `
	var a = [];
	Object.defineProperty(a, "0", {get: function() {return 42}, set: function(v) {Object.defineProperty(a, "0", {value: v + 1, writable: true, configurable: true})}, configurable: true})
	a[1] = 43;
	a.reverse();

	a.length === 2 && a[0] === 44 && a[1] === 42;
	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestArrayPushNonOptimisable(t *testing.T) {
	const SCRIPT = `
	Object.defineProperty(Object.prototype, "0", {value: 42});
	var a = [];
	var thrown = false;
	try {
		a.push(1);
	} catch (e) {
		thrown = e instanceof TypeError;
	}
	thrown;
	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestArraySetLengthWithPropItems(t *testing.T) {
	const SCRIPT = `
	var a = [1,2,3,4];
	var thrown = false;

	Object.defineProperty(a, "2", {value: 42, configurable: false, writable: false});
	try {
		Object.defineProperty(a, "length", {value: 0, writable: false});
	} catch (e) {
		thrown = e instanceof TypeError;
	}
	thrown && a.length === 3;
	`

	testScript1(SCRIPT, valueTrue, t)
}
