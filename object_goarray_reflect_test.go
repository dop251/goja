package goja

import (
	"testing"
)

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

func TestGoReflectArrayCopyOnChange(t *testing.T) {
	vm := New()

	v, err := vm.RunString(`
	a => {
		let tmp = a[0];
		if (tmp !== a[0]) {
			throw new Error("tmp !== a[0]");
		}

		a[0] = a[1];
		if (tmp === a[0]) {
			throw new Error("tmp === a[0]");
		}
		if (tmp.Test !== "1") {
			throw new Error("tmp.Test: " + tmp.Test + " (" + typeof tmp.Test + ")");
		}
		if (a[0].Test !== "2") {
			throw new Error("a[0].Test: " + a[0].Test);
		}

		a[0].Test = "3";
		if (a[0].Test !== "3") {
			throw new Error("a[0].Test (1): " + a[0].Test);
		}

		tmp = a[0];
		tmp.Test = "4";
		if (a[0].Test !== "4") {
			throw new Error("a[0].Test (2): " + a[0].Test);
		}

		delete a[0];
		if (a[0] && a[0].Test !== "") {
			throw new Error("a[0].Test (3): " + a[0].Test);
		}
		if (tmp.Test !== "4") {
			throw new Error("tmp.Test (1): " + tmp.Test);
		}

		a[1] = tmp;
		if (a[1].Test !== "4") {
			throw new Error("a[1].Test: " + a[1].Test);
		}

        // grow
		tmp = a[1];
		a.push(null);
		if (a.length !== 3) {
			throw new Error("a.length after push: " + a.length);
		}

		tmp.Test = "5";
		if (a[1].Test !== "5") {
			throw new Error("a[1].Test (1): " + a[1].Test);
		}

		// shrink
		a.length = 1;
		if (a.length !== 1) {
			throw new Error("a.length after shrink: " + a.length);
		}

		if (tmp.Test !== "5") {
			throw new Error("tmp.Test (shrink): " + tmp.Test);
		}
	}
	`)
	if err != nil {
		t.Fatal(err)
	}

	fn, ok := AssertFunction(v)
	if !ok {
		t.Fatal("Not a function")
	}

	t.Run("[]struct", func(t *testing.T) {
		a := []struct {
			Test string
		}{{"1"}, {"2"}}
		_, err := fn(nil, vm.ToValue(a))
		if err != nil {
			t.Fatal(err)
		}
		if a[0].Test != "" {
			t.Fatalf("a[0]: %#v", a[0])
		}

		if a[1].Test != "4" {
			t.Fatalf("a0[1]: %#v", a[1])
		}
	})

	// The copy-on-change mechanism doesn't apply to the types below because the contained values are references.
	// These tests are here for completeness and to prove that the behaviour is consistent.

	t.Run("[]I", func(t *testing.T) {
		type I interface {
			Get() string
		}

		a := []I{&testGoReflectMethod_O{Test: "1"}, &testGoReflectMethod_O{Test: "2"}}

		_, err = fn(nil, vm.ToValue(a))
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("[]interface{}", func(t *testing.T) {
		a := []interface{}{&testGoReflectMethod_O{Test: "1"}, &testGoReflectMethod_O{Test: "2"}}

		_, err = fn(nil, vm.ToValue(a))
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestCopyOnChangeReflectSlice(t *testing.T) {
	vm := New()
	v, err := vm.RunString(`
	s => {
		s.A.push(1);
		if (s.A.length !== 1) {
			throw new Error("s.A.length: " + s.A.length);
		}
		if (s.A[0] !== 1) {
			throw new Error("s.A[0]: " + s.A[0]);
		}
		let tmp = s.A;
		if (tmp !== s.A) {
			throw new Error("tmp !== s.A");
		}
		s.A = [2];
		if (tmp === s.A) {
			throw new Error("tmp === s.A");
		}
		if (tmp[0] !== 1) {
			throw new Error("tmp[0]: " + tmp[0]);
		}
		if (s.A[0] !== 2) {
			throw new Error("s.A[0] (1): " + s.A[0]);
		}
	}
	`)
	if err != nil {
		t.Fatal(err)
	}
	fn, ok := AssertFunction(v)
	if !ok {
		t.Fatal("Not a function")
	}

	t.Run("[]int", func(t *testing.T) {
		type S struct {
			A []int
		}
		var s S
		_, err := fn(nil, vm.ToValue(&s))
		if err != nil {
			t.Fatal(err)
		}
		if len(s.A) != 1 {
			t.Fatal(s)
		}
		if s.A[0] != 2 {
			t.Fatal(s.A)
		}
	})

	t.Run("[]interface{}", func(t *testing.T) {
		type S struct {
			A []interface{}
		}
		var s S
		_, err := fn(nil, vm.ToValue(&s))
		if err != nil {
			t.Fatal(err)
		}
		if len(s.A) != 1 {
			t.Fatal(s)
		}
		if s.A[0] != int64(2) {
			t.Fatal(s.A)
		}
	})
}

func TestCopyOnChangeSort(t *testing.T) {
	a := []struct {
		Test string
	}{{"2"}, {"1"}}

	vm := New()
	vm.Set("a", &a)

	_, err := vm.RunString(`
		let a0 = a[0];
		let a1 = a[1];
		a.sort((a, b) => a.Test.localeCompare(b.Test));
		if (a[0].Test !== "1") {
			throw new Error("a[0]: " + a[0]);
		}
		if (a[1].Test !== "2") {
			throw new Error("a[1]: " + a[1]);
		}
		if (a0 !== a[1]) {
			throw new Error("a0 !== a[1]");
		}
		if (a1 !== a[0]) {
			throw new Error("a1 !== a[0]");
		}
	`)
	if err != nil {
		t.Fatal(err)
	}

	if a[0].Test != "1" || a[1].Test != "2" {
		t.Fatal(a)
	}
}

type testStringerArray [8]byte

func (a testStringerArray) String() string {
	return "X"
}

func TestReflectArrayToString(t *testing.T) {
	vm := New()
	var a testStringerArray
	vm.Set("a", &a)
	res, err := vm.RunString("`${a}`")
	if err != nil {
		t.Fatal(err)
	}
	if exp := res.Export(); exp != "X" {
		t.Fatal(exp)
	}

	var a1 [2]byte
	vm.Set("a", &a1)
	res, err = vm.RunString("`${a}`")
	if err != nil {
		t.Fatal(err)
	}
	if exp := res.Export(); exp != "0,0" {
		t.Fatal(exp)
	}
}
