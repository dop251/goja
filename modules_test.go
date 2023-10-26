package goja

import (
	"fmt"
	"sync"
	"testing"
)

func TestSimpleModule(t *testing.T) {
	t.Parallel()
	type cacheElement struct {
		m   ModuleRecord
		err error
	}

	testCases := map[string]map[string]string{
		"function export": {
			"a.js": `
				import { b } from "dep.js";
				globalThis.s = b()
			`,
			"dep.js": `export function b() { return 5 };`,
		},
		"let export": {
			"a.js": `
				import { b } from "dep.js";
				globalThis.s = b()
			`,
			"dep.js": `export let b = function() {return 5 };`,
		},
		"const export": {
			"a.js": `
				import { b } from "dep.js";
				globalThis.s = b()
			`,
			"dep.js": `export const b = function() { return 5 };`,
		},
		"let export with update": {
			"a.js": `
				import { s , b} from "dep.js";
				s()
				globalThis.s = b()
			`,
			"dep.js": `
				export let b = "something";
				export function s(){
					b = function() {
						return 5;
					};
				}`,
		},
		"default export": {
			"a.js": `
				import b from "dep.js";
				globalThis.s = b()
			`,
			"dep.js": `export default function() { return 5 };`,
		},
		"default loop": {
			"a.js": `
				import b from "a.js";
				export default function() {return 5;};
				globalThis.s = b()
			`,
		},
		"default export arrow": {
			"a.js": `
				import b from "dep.js";
				globalThis.s = b();
			`,
			"dep.js": `export default () => {return 5 };`,
		},
		"default export with as": {
			"a.js": `
				import b from "dep.js";
				globalThis.s = b()
			`,
			"dep.js": `
				function f() {return 5;};
				export { f as default };
			`,
		},
		"export usage before evaluation as": {
			"a.js": `
				import  "dep.js";
				export function a() { return 5; }
			`,
			"dep.js": `
				import { a } from "a.js";
				globalThis.s = a();
			`,
		},
		"dynamic import": {
			"a.js": `
				import("dep.js").then((imported) => {
					globalThis.s = imported.default();
				});
			`,
			"dep.js": `export default function() { return 5; }`,
		},
		"dynamic import error": {
			"a.js": `
				do {
					import('dep.js').catch(error => {
						if (error.name == "SyntaxError") {
							globalThis.s = 5;
						}
					});
				} while (false);
			`,
			"dep.js": `import { x } from "0-fixture.js";`,
			"0-fixture.js": `
					export * from "1-fixture.js";
					export * from "2-fixture.js";
			`,
			"1-fixture.js": `export var x`,
			"2-fixture.js": `export var x`,
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

				src := string(cases[specifier])
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
					vm := New()
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
					eventLoopQueue <- func() { promise = m.Evaluate(vm) }

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
						err := promise.Result().Export().(error)
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
