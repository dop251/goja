package goja

import "testing"

func TestGoReflectArray(t *testing.T) {
	vm := New()
	vm.Set("a", [...]int{1, 2, 3})
	_, err := vm.RunString(`
	if (!Array.isArray(a)) {
		throw new Error("isArray() returned false");
	}
	if (a[0] !== 1 || a[1] !== 2 || a[2] !== 3) {
		throw new Error("Array contents is incorrect");
	}
	if (!a.hasOwnProperty("length")) {
		throw new Error("hasOwnProperty() returned false");
	}
	let desc = Object.getOwnPropertyDescriptor(a, "length");
	if (desc.value !== 3 || desc.writable || desc.enumerable || desc.configurable) {
		throw new Error("incorrect property descriptor: " + JSON.stringify(desc));
	}
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGoReflectArraySort(t *testing.T) {
	vm := New()
	vm.Set("a", [...]int{3, 1, 2})
	v, err := vm.RunString(`
		a.sort();
		if (a[0] !== 1 || a[1] !== 2 || a[2] !== 3) {
			throw new Error(a.toString());
		}
		a;
	`)
	if err != nil {
		t.Fatal(err)
	}
	res := v.Export()
	if a, ok := res.([3]int); ok {
		if a[0] != 1 || a[1] != 2 || a[2] != 3 {
			t.Fatal(a)
		}
	} else {
		t.Fatalf("Wrong type: %T", res)
	}
}
