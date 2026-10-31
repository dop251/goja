package goja

import "testing"

func TestDynamicImport(t *testing.T) {
	vm := New()
	v, err := vm.RunString(`typeof import("x").then`)
	if err != nil {
		t.Fatal(err)
	}
	if v.String() != "function" {
		t.Fatalf("expected a promise, got %v", v)
	}
}
