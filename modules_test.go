package goja

import (
	"fmt"
	"testing"
)

func TestSimpleModule(t *testing.T) {
	vm := New()
	type cacheElement struct {
		m   ModuleRecord
		err error
	}
	a := `

  import { b } from "dep.js";

  globalThis.s = b()
    `
	b := `export let b= function() { return 5 };
`
	cache := make(map[string]cacheElement)
	var hostResolveImportedModule func(referencingScriptOrModule interface{}, specifier string) (ModuleRecord, error)
	hostResolveImportedModule = func(referencingScriptOrModule interface{}, specifier string) (ModuleRecord, error) {
		k, ok := cache[specifier]
		if ok {
			return k.m, k.err
		}
		var src string
		switch specifier {
		case "a.js":
			src = a
		case "dep.js":
			src = b
		default:
			panic(specifier)
		}
		p, err := vm.ParseModule(src)
		if err != nil {
			cache[specifier] = cacheElement{err: err}
			return nil, err
		}
		p.compiler = newCompiler()
		p.compiler.hostResolveImportedModule = hostResolveImportedModule
		cache[specifier] = cacheElement{m: p}
		return p, nil
	}

	vm.hostResolveImportedModule = hostResolveImportedModule
	vm.Set("l", func() {
		fmt.Println("l called")
		fmt.Printf("iter stack ; %+v", vm.vm.iterStack)
	})
	m, err := vm.hostResolveImportedModule(nil, "a.js")
	if err != nil {
		t.Fatalf("got error %s", err)
	}
	p := m.(*SourceTextModuleRecord)

	err = p.Link()
	if err != nil {
		t.Fatalf("got error %s", err)
	}

	err = p.Evaluate()
	if err != nil {
		t.Fatalf("got error %s", err)
	}
	v := vm.Get("s")
	if v == nil || v.ToNumber().ToInteger() != 5 {
		t.Fatalf("expected 5 got %s", v)
	}
}
