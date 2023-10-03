package goja

import (
	"fmt"
	"io/fs"
	"sync"
	"testing"
	"testing/fstest"
)

func TestSimpleModule(t *testing.T) {
	t.Parallel()
	type cacheElement struct {
		m   ModuleRecord
		err error
	}
	type testCase struct {
		fs fs.FS
		a  string
		b  string
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
        export function s(){
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
			globalThis.p();
globalThis.s = b();
`,
			b: `globalThis.p(); export default () => {globalThis.p(); return 5 };`,
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
		"dynamic import": {
			a: `
			globalThis.p();
import("dep.js").then((imported) => {
			globalThis.p()
	globalThis.s = imported.default();
});`,
			b: `export default function() {globalThis.p(); return 5;}`,
		},
		"dynamic import error": {
			a: ` do {
  import('dep.js').catch(error => {
			if (error.name == "SyntaxError") {
globalThis.s = 5;
			}
  });
} while (false);
`,
			b: `
import { x } from "0-fixture.js";
			`,
			fs: &fstest.MapFS{
				"0-fixture.js": &fstest.MapFile{
					Data: []byte(`
					export * from "1-fixture.js";
					export * from "2-fixture.js"; 
				`),
				},
				"1-fixture.js": &fstest.MapFile{
					Data: []byte(`export var x`),
				},
				"2-fixture.js": &fstest.MapFile{
					Data: []byte(`export var x`),
				},
			},
		},
	}
	for name, cases := range testCases {
		cases := cases
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			mu := sync.Mutex{}
			cache := make(map[string]cacheElement)
			var hostResolveImportedModule func(referencingScriptOrModule interface{}, specifier string) (ModuleRecord, error)
			hostResolveImportedModule = func(referencingScriptOrModule interface{}, specifier string) (ModuleRecord, error) {
				mu.Lock()
				defer mu.Unlock()
				k, ok := cache[specifier]
				if ok {
					return k.m, k.err
				}
				var src string
				switch specifier {
				case "a.js":
					src = cases.a
				case "dep.js":
					src = cases.b
				default:
					b, err := fs.ReadFile(cases.fs, specifier)
					if err != nil {
						panic(specifier)
					}
					src = string(b)
				}
				p, err := ParseModule(specifier, src, hostResolveImportedModule)
				if err != nil {
					cache[specifier] = cacheElement{err: err}
					return nil, err
				}
				cache[specifier] = cacheElement{m: p}
				return p, nil
			}

			linked := make(map[ModuleRecord]error)
			linkMu := new(sync.Mutex)
			link := func(m ModuleRecord) error {
				linkMu.Lock()
				defer linkMu.Unlock()
				if err, ok := linked[m]; ok {
					return err
				}
				err := m.Link()
				linked[m] = err
				return err
			}

			m, err := hostResolveImportedModule(nil, "a.js")
			if err != nil {
				t.Fatalf("got error %s", err)
			}
			p := m.(*SourceTextModuleRecord)

			err = link(p)
			if err != nil {
				t.Fatalf("got error %s", err)
			}

			for i := 0; i < 10; i++ {
				i := i
				m := m
				t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
					t.Parallel()
					var err error
					vm := New()
					vm.Set("p", vm.ToValue(func() {
						// fmt.Println("p called")
					}))
					vm.Set("l", func(v Value) {
						fmt.Printf("%+v\n", v)
						fmt.Println("l called")
						fmt.Printf("iter stack ; %+v\n", vm.vm.iterStack)
					})
					if err != nil {
						t.Fatalf("got error %s", err)
					}
					eventLoopQueue := make(chan func(), 2) // the most basic and likely buggy event loop
					vm.SetImportModuleDynamically(func(referencingScriptOrModule interface{}, specifierValue Value, pcap interface{}) {
						specifier := specifierValue.String()

						eventLoopQueue <- func() {
							ex := vm.runWrapped(func() {
								m, err := hostResolveImportedModule(referencingScriptOrModule, specifier)
								vm.FinishLoadingImportModule(referencingScriptOrModule, specifierValue, pcap, m, err)
							})
							if ex != nil {
								vm.FinishLoadingImportModule(referencingScriptOrModule, specifierValue, pcap, nil, ex)
							}
						}
					})
					var promise *Promise
					eventLoopQueue <- func() {
						promise = m.Evaluate(vm)
					}
				outer:
					for {
						select {
						case fn := <-eventLoopQueue:
							fn()
						default:
							break outer
						}
					}
					if promise.state != PromiseStateFulfilled {
						t.Fatalf("got %+v", promise.Result().Export())
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
