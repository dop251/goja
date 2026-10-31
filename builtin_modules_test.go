package goja

import "testing"

func TestModuleRegistryAndLinking(t *testing.T) {
	vm := New()
	if err := vm.DefineModule("a.js", `import {x} from "b.js"; export const y = 1; export default 2;`); err != nil {
		t.Fatal(err)
	}
	if err := vm.DefineModule("b.js", `export const x = 1;`); err != nil {
		t.Fatal(err)
	}
	if err := vm.LinkModule("a.js"); err != nil {
		t.Fatal(err)
	}
	if err := vm.DefineModule("dup.js", `export {a}; export {a};`); err == nil {
		t.Fatal("expected a duplicate export error")
	}
	if err := vm.LinkModule("missing.js"); err == nil {
		t.Fatal("expected a missing module error")
	}
	if err := vm.DefineModule("c1.js", `import "c2.js"; export let a = 1;`); err != nil {
		t.Fatal(err)
	}
	if err := vm.DefineModule("c2.js", `import "c1.js"; export let b = 2;`); err != nil {
		t.Fatal(err)
	}
	if err := vm.LinkModule("c1.js"); err != nil {
		t.Fatalf("cyclic import should link: %v", err)
	}
}
