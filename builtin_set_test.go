package goja

import (
	"fmt"
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
	new Set(o);
	undefined;
	`
	testScript(SCRIPT, _undefined, t)
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
