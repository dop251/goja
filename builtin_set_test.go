package goja

import (
	"fmt"
	"strings"
	"testing"
)

func TestSetEvilIterator(t *testing.T) {
	const SCRIPT = `
	var o = {};
	o[Symbol.iterator] = function() {
		return {
			next: function() {
				if (!this.flag) {
					this.flag = true;
					return {};
				}
				return {done: true};
			}
		}
	}
	new Set(o).has(undefined);
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestSetEvilIterator1(t *testing.T) {
	const SCRIPT = `
	var o = [];
	o[Symbol.iterator] = function() {
		return {
			next: function() {
				if (!this.flag) {
					this.flag = true;
					return {};
				}
				return {done: true};
			}
		}
	}
	new Set(o).has(undefined);
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestSetEvilIterator2(t *testing.T) {
	const SCRIPT = `
	var o = [];
	o[Symbol.iterator] = Set.prototype[Symbol.iterator];
	assert.throws(TypeError, () => new Set(o));
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestSetEvilStdIterator(t *testing.T) {
	const SCRIPT = `
	const ArrayIteratorPrototype = Object.getPrototypeOf([].values());
	let values = [1, 2, 3, 4];
	ArrayIteratorPrototype.next = function() {
	  let done = values.length === 0;
	  let value = values.pop();
	  return {value, done};
	};
	
	const s = new Set([]);
	assert.sameValue(s.size, 4);
	`

	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestSetEvilStdIterator1(t *testing.T) {
	const SCRIPT = `
	const ArrayIteratorPrototype = Object.getPrototypeOf([].values());
	const SetIteratorPrototype = Object.getPrototypeOf(new Set().values());
	ArrayIteratorPrototype.next = SetIteratorPrototype.next;
	
	assert.throws(TypeError, () => new Set([]));
	`

	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func ExampleRuntime_ExportTo_setToMap() {
	vm := New()
	s, err := vm.RunString(`
	new Set([1, 2, 3])
	`)
	if err != nil {
		panic(err)
	}
	m := make(map[int]struct{})
	err = vm.ExportTo(s, &m)
	if err != nil {
		panic(err)
	}
	fmt.Println(m)
	// Output: map[1:{} 2:{} 3:{}]
}

func ExampleRuntime_ExportTo_setToSlice() {
	vm := New()
	s, err := vm.RunString(`
	new Set([1, 2, 3])
	`)
	if err != nil {
		panic(err)
	}
	var a []int
	err = vm.ExportTo(s, &a)
	if err != nil {
		panic(err)
	}
	fmt.Println(a)
	// Output: [1 2 3]
}

func TestSetExportToSliceCircular(t *testing.T) {
	vm := New()
	s, err := vm.RunString(`
	let s = new Set();
	s.add(s);
	s;
	`)
	if err != nil {
		t.Fatal(err)
	}
	var a []Value
	err = vm.ExportTo(s, &a)
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != 1 {
		t.Fatalf("len: %d", len(a))
	}
	if a[0] != s {
		t.Fatalf("a: %v", a)
	}
}

func TestSetExportToArrayMismatchedLengths(t *testing.T) {
	vm := New()
	s, err := vm.RunString(`
	new Set([1, 2])
	`)
	if err != nil {
		panic(err)
	}
	var s1 [3]int
	err = vm.ExportTo(s, &s1)
	if err == nil {
		t.Fatal("expected error")
	}
	if msg := err.Error(); !strings.Contains(msg, "lengths mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetExportToNilMap(t *testing.T) {
	vm := New()
	var m map[int]interface{}
	res, err := vm.RunString("new Set([1])")
	if err != nil {
		t.Fatal(err)
	}
	err = vm.ExportTo(res, &m)
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 1 {
		t.Fatal(m)
	}
	if _, exists := m[1]; !exists {
		t.Fatal(m)
	}
}

func TestSetExportToNonNilMap(t *testing.T) {
	vm := New()
	m := map[int]interface{}{
		2: true,
	}
	res, err := vm.RunString("new Set([1])")
	if err != nil {
		t.Fatal(err)
	}
	err = vm.ExportTo(res, &m)
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 1 {
		t.Fatal(m)
	}
	if _, exists := m[1]; !exists {
		t.Fatal(m)
	}
}

func TestSetGetAdderGetIteratorOrder(t *testing.T) {
	const SCRIPT = `
	let getterCalled = 0;

	class S extends Set {
	    get add() {
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
	    new S(iterable);
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

func TestSetDifferenceEvilStdIterator(t *testing.T) {
	const SCRIPT = `
	const SetIteratorPrototype = Object.getPrototypeOf(new Set().values());

	SetIteratorPrototype.next = function() {
	  return {done: true};
	}

	var s1 = new Set([1,2,3]);
	var s2 = new Set([2]);
	var s3 = s1.difference(s2);
	s3.size;
	`
	testScript(SCRIPT, intToValue(3), t)
}
