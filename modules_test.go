package goja

import (
	"fmt"
	"testing"
)

func TestSimpleModule(t *testing.T) {
	t.Parallel()
	type cacheElement struct {
		m   ModuleRecord
		err error
	}
	type testCase struct {
		a string
		b string
	}

	testCases := map[string]testCase{
		"function export": {
			a: `import { b } from "dep.js";
globalThis.s = b()
`,
			b: `export function b() {globalThis.p(); return 5 };`,
		},
		"let export": {
			a: `import { b } from "dep.js";
globalThis.s = b()
`,
			b: `export let b = function() {globalThis.p(); return 5 };`,
		},
		"const export": {
			a: `import { b } from "dep.js";
globalThis.s = b()
`,
			b: `export const b = function() {globalThis.p(); return 5 };`,
		},
		"let export with update": {
			a: `import { s , b} from "dep.js";
      s()
globalThis.s = b()
`,
			b: `export let b = "something";
        export function s (){
        globalThis.p()
          b = function() {globalThis.p(); return 5 };
        }`,
		},
		"default export": {
			a: `import b from "dep.js";
globalThis.s = b()
`,
			b: `export default function() {globalThis.p(); return 5 };`,
		},
		"default loop": {
			a: `import b from "a.js";
export default function() {return 5;};
globalThis.s = b()
`,
			b: ``,
		},
		"default export arrow": {
			a: `import b from "dep.js";
globalThis.s = b()
`,
			b: `export default () => {globalThis.p(); return 5 };`,
		},
		"default export with as": {
			a: `import b from "dep.js";
globalThis.s = b()
`,
			b: `function f() {return 5;};
      export { f as default };`,
		},
		"export usage before evaluation as": {
			a: `import  "dep.js";
            export function a() {return 5;}
`,
			b: `import { a } from "a.js";
           globalThis.s = a();`,
		},
	}
	for name, cases := range testCases {
		a, b := cases.a, cases.b
		t.Run(name, func(t *testing.T) {
			t.Parallel()
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
				p, err := ParseModule(specifier, src, hostResolveImportedModule)
				if err != nil {
					cache[specifier] = cacheElement{err: err}
					return nil, err
				}
				cache[specifier] = cacheElement{m: p}
				return p, nil
			}

			m, err := hostResolveImportedModule(nil, "a.js")
			if err != nil {
				t.Fatalf("got error %s", err)
			}
			p := m.(*SourceTextModuleRecord)

			err = p.Link()
			if err != nil {
				t.Fatalf("got error %s", err)
			}

			for i := 0; i < 10; i++ {
				i := i
				t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
					t.Parallel()
					var err error
					vm := New()
					vm.Set("p", vm.ToValue(func() {
						// fmt.Println("p called")
					}))
					vm.Set("l", func() {
						fmt.Println("l called")
						fmt.Printf("iter stack ; %+v", vm.vm.iterStack)
					})
					if err != nil {
						t.Fatalf("got error %s", err)
					}
					promise := m.Evaluate(vm)
					if promise.state != PromiseStateFulfilled {
						err = promise.Result().Export().(error)
						t.Fatalf("got error %s", err)
					}
					v := vm.Get("s")
					if v == nil || v.ToNumber().ToInteger() != 5 {
						t.Fatalf("expected 5 got %s", v)
					}
				})
			}
		})
	}
}
