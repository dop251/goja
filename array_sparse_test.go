package goja

import (
	"testing"
)

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
	vm := New()
	_, err := vm.RunString(`
	var a = [];
	a[20470] = 5; // switch to sparse`)
	if err != nil {
		t.Fatal(err)
	}
	a := vm.Get("a").(*Object)
	if _, ok := a.self.(*sparseArrayObject); !ok {
		t.Fatal("1: array is not sparse")
	}
	_, err = vm.RunString(`
	var cutoffIdx = Math.round(20470 - 20470/8);
	for (var i = a.length - 1; i >= cutoffIdx; i--) {
		a[i] = i;
	}

	// At this point it will have switched to a normal array
	if (a.length != 20471) {
		throw new Error("Invalid length: " + a.length);
	}

	for (var i = 0; i < cutoffIdx; i++) {
		if (a[i] !== undefined) {
			throw new Error("Invalid value at " + i + ": " + a[i]);
		}
	}

	for (var i = cutoffIdx; i < a.length; i++) {
		if (a[i] !== i) {
			throw new Error("Invalid value at " + i + ": " + a[i]);
		}
	}`)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := a.self.(*arrayObject); !ok {
		t.Fatal("2: array is not normal")
	}
	_, err = vm.RunString(`
	// Now try to expand. Should stay a normal array
	a[20471] = 20471;
	if (a.length != 20472) {
		throw new Error("Invalid length: " + a.length);
	}

	for (var i = 0; i < cutoffIdx; i++) {
		if (a[i] !== undefined) {
			throw new Error("Invalid value at " + i + ": " + a[i]);
		}
	}

	for (var i = cutoffIdx; i < a.length; i++) {
		if (a[i] !== i) {
			throw new Error("Invalid value at " + i + ": " + a[i]);
		}
	}`)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := a.self.(*arrayObject); !ok {
		t.Fatal("3: array is not normal")
	}
	_, err = vm.RunString(`
	// Delete enough elements for it to become sparse again.
	var cutoffIdx1 = Math.round(20472 - 20472/10);
	for (var i = cutoffIdx; i < cutoffIdx1; i++) {
		delete a[i];
	}

	// This should switch it back to sparse.
	a[25590] = 25590;
	if (a.length != 25591) {
		throw new Error("Invalid length: " + a.length);
	}

	for (var i = 0; i < cutoffIdx1; i++) {
		if (a[i] !== undefined) {
			throw new Error("Invalid value at " + i + ": " + a[i]);
		}
	}

	for (var i = cutoffIdx1; i < 20472; i++) {
		if (a[i] !== i) {
			throw new Error("Invalid value at " + i + ": " + a[i]);
		}
	}

	for (var i = 20472; i < 25590; i++) {
		if (a[i] !== undefined) {
			throw new Error("Invalid value at " + i + ": " + a[i]);
		}
	}

	if (a[25590] !== 25590) {
		throw new Error("Invalid value at 25590: " + a[25590]);
	}
	`)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := a.self.(*sparseArrayObject); !ok {
		t.Fatal("4: array is not sparse")
	}
}

func TestSparseArrayOwnKeys(t *testing.T) {
	const SCRIPT = `
	var a1 = [];
	a1[500000] = 1;
	var seen = false;
	var count = 0;
	var keys = Object.keys(a1);
	keys.length === 1 && keys[0] === "500000"; 
	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestSparseArrayEnumerate(t *testing.T) {
	const SCRIPT = `
	var a1 = [];
	a1[500000] = 1;
	var seen = false;
	var count = 0;
	for (var i in a1) {
		if (i === "500000") {
			if (seen) {
				throw new Error("seen twice");
			}
			seen = true;
		}
		count++;
	}
	seen && count === 1;
	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestArraySparseMaxLength(t *testing.T) {
	const SCRIPT = `
	var a = [];
	a[4294967294]=1;
	a.length === 4294967295 && a[4294967294] === 1;
	`

	testScript1(SCRIPT, valueTrue, t)
}
