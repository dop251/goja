package goja

import "testing"

func TestSparseArraySetLengthWithPropItems(t *testing.T) {
	const SCRIPT = `
	var a = [1,2,3,4];
	a[100000] = 5;
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

func TestSparseArraySwitch(t *testing.T) {
	const SCRIPT = `
	var a = [];
	a[20470] = 5; // switch to sparse
	for (var i = a.length - 1; i >= 0; i--) {
		a[i] = i; // switch to normal at some point
	}

	if (a.length != 20471) {
		throw new Error("Invalid length: " + a.length);
	}

	for (var i = 0; i < a.length; i++) {
		if (a[i] !== i) {
			throw new Error("Invalid value at " + i + ": " + a[i]);
		}
	}
	`

	testScript1(SCRIPT, _undefined, t)
}
