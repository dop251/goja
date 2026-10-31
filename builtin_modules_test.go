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

func TestModuleNamespace(t *testing.T) {
	vm := New()
	if err := vm.DefineModule("ns.js", `export const b = 1; export function a() {} export class C {}; export {b as z};`); err != nil {
		t.Fatal(err)
	}
	nsv, err := vm.ModuleNamespace("ns.js")
	if err != nil {
		t.Fatal(err)
	}
	ns := nsv.(*Object)
	if got := ns.Keys(); len(got) != 4 || got[0] != "C" || got[1] != "a" || got[2] != "b" || got[3] != "z" {
		t.Fatalf("unexpected namespace keys: %v", got)
	}
	if ns.Prototype() != nil {
		t.Fatal("module namespace must have a null prototype")
	}
	if tag := ns.self.getSym(SymToStringTag, nil); tag == nil || tag.String() != "Module" {
		t.Fatalf("unexpected toStringTag: %v", tag)
	}
	d := ns.self.getOwnPropStr("a")
	if d == nil {
		t.Fatal("missing export property")
	}
}
