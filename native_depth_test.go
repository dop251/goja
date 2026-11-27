package goja

import (
	"errors"
	"runtime"
	"testing"
)

// Native functions that call each other recurse in Go without pushing VM frames. Each of these used to overflow the goroutine stack, which is fatal; now they throw a RangeError the script catches.
var nativeRecursionCases = map[string]string{
	"toLocaleString":   `const o = {}; o.toString = Object.prototype.toLocaleString; o.toLocaleString()`,
	"valueOf":          `const o = {}; o.valueOf = Object.prototype.toLocaleString; o.toString = Object.prototype.toLocaleString; +o`,
	"regexpTest":       `const r = /x/; r.exec = RegExp.prototype.test; r.test("x")`,
	"dateToJSON":       `const o = {}; o.toISOString = Date.prototype.toJSON; o.toISOString()`,
	"getter":           `const o = {}; Object.defineProperty(o, "toString", { get: Object.prototype.toLocaleString }); String(o)`,
	"setter":           `const o = {}; Object.defineProperty(o, "x", { set: Reflect.set.bind(null, o, "x", 1, o) }); o.x = 1`,
	"symbolMatch":      `const r = /x/; Object.defineProperty(r, "exec", { value: RegExp.prototype[Symbol.match] }); r.test("x")`,
	"symbolReplace":    `const r = /x/; Object.defineProperty(r, "exec", { value: RegExp.prototype[Symbol.replace] }); "x".replace(r, "y")`,
	"reflectApply":     `let a = [() => 1, null, []]; for (let i = 0; i < 1e5; i++) a = [Reflect.apply, null, a]; Reflect.apply(...a)`,
	"reflectConstruct": `let C = function () {}; for (let i = 0; i < 1e5; i++) C = C.bind(null); Reflect.construct(C, [])`,
	"boundCall":        `let f = () => 1; for (let i = 0; i < 1e5; i++) f = f.bind(null); f()`,
	"boundNew":         `let C = function () {}; for (let i = 0; i < 1e5; i++) C = C.bind(null); new C()`,
	"proxyCall":        `let f = function () {}; for (let i = 0; i < 1e5; i++) f = new Proxy(f, {}); f()`,
	"proxyNew":         `let C = function () {}; for (let i = 0; i < 1e5; i++) C = new Proxy(C, {}); new C()`,
}

func TestNativeRecursionThrowsRangeError(t *testing.T) {
	for name, src := range nativeRecursionCases {
		t.Run(name, func(t *testing.T) {
			r := New()
			r.SetMaxCallStackSize(1000)
			v, err := r.RunString(`let caught = "none"; try { ` + src + ` } catch (e) { caught = e instanceof RangeError ? "RangeError" : String(e); } caught`)
			if err != nil {
				t.Fatal(err)
			}
			if v.String() != "RangeError" {
				t.Fatalf("caught %s, want RangeError", v)
			}
			if d := r.vm.nativeCallDepth; d != 0 {
				t.Fatalf("native call depth %d after the call, want 0", d)
			}
			// The whole limit is available again.
			v, err = r.RunString(`function f(n) { return n ? f(n - 1) + 1 : 0; } f(900)`)
			if err != nil || v.ToInteger() != 900 {
				t.Fatalf("recursion after the RangeError: %v, %v", v, err)
			}
		})
	}
}

func TestNativeRecursionFromGo(t *testing.T) {
	r := New()
	r.SetMaxCallStackSize(1000)
	v, err := r.RunString(`const o = {}; o.toString = Object.prototype.toLocaleString; o.toLocaleString = Object.prototype.toLocaleString; o`)
	if err != nil {
		t.Fatal(err)
	}
	f, _ := AssertFunction(r.Get("Object").ToObject(r).Get("prototype").ToObject(r).Get("toLocaleString"))
	_, err = f(v)
	var ex *Exception
	if !errors.As(err, &ex) || !ex.Value().ToObject(r).Get("constructor").SameAs(r.Get("RangeError")) {
		t.Fatalf("error %v, want a RangeError", err)
	}
	if d := r.vm.nativeCallDepth; d != 0 {
		t.Fatalf("native call depth %d after the call, want 0", d)
	}
}

// JS frames and native calls count together: recursion alternating between them stops within the limit, with the StackOverflowError a JS frame past it throws or the RangeError a native call past it throws.
func TestNativeCallsCountWithJSFrames(t *testing.T) {
	r := New()
	r.SetMaxCallStackSize(1000)
	_, err := r.RunString(`var levels = 0; function depth(n) { levels++; return Reflect.apply(depth, null, [n + 1]); } depth(0)`)
	var overflow *StackOverflowError
	var ex *Exception
	if !errors.As(err, &overflow) && !(errors.As(err, &ex) && ex.Value().ToObject(r).Get("constructor").SameAs(r.Get("RangeError"))) {
		t.Fatalf("error %v, want a stack overflow", err)
	}
	// Each level is at least a JS frame and a native Reflect.apply call.
	if n := r.Get("levels").ToInteger(); n < 200 || n > 500 {
		t.Fatalf("recursed %d levels, want at most 500", n)
	}
	if d := r.vm.nativeCallDepth; d != 0 {
		t.Fatalf("native call depth %d after the call, want 0", d)
	}
}

func TestBoundChainNameMemory(t *testing.T) {
	r := New()
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	v, err := r.RunString(`let f = function g() {}; for (let i = 0; i < 2e4; i++) f = f.bind(null); f.name.length`)
	runtime.ReadMemStats(&after)
	if err != nil {
		t.Fatal(err)
	}
	if n := v.ToInteger(); n != 6*20000+1 {
		t.Fatalf("name length %d, want %d", n, 6*20000+1)
	}
	// Flattening each name as it is made allocates 6 * n^2 / 2 bytes, 1.2 GB here.
	if used := after.TotalAlloc - before.TotalAlloc; used > 200<<20 {
		t.Fatalf("binding allocated %d MB", used>>20)
	}
	v, err = r.RunString(`f.name.startsWith("bound bound ") && f.name.endsWith(" g")`)
	if err != nil || !v.ToBoolean() {
		t.Fatalf("name %v, %v", v, err)
	}
}

func TestExportDepthLimit(t *testing.T) {
	r := New()
	r.SetMaxCallStackSize(1000)
	for name, src := range map[string]string{
		"object": `(() => { let a = {}; for (let i = 0; i < 1e5; i++) a = { a }; return a; })()`,
		"array":  `(() => { let a = []; for (let i = 0; i < 1e5; i++) a = [a]; return a; })()`,
		"getter": `(() => { let a = {}; for (let i = 0; i < 1e5; i++) a = { a }; let n = 0; return { get a() { return n++ ? a : 1; } }; })()`,
		"map":    `(() => { let a = new Map(); for (let i = 0; i < 1e5; i++) a = new Map([[1, a]]); return a; })()`,
	} {
		t.Run(name, func(t *testing.T) {
			v, err := r.RunString(src)
			if err != nil {
				t.Fatal(err)
			}
			if name == "getter" {
				// The first export sees 1, the second the deep value.
				_ = v.Export()
			}
			func() {
				defer func() {
					ex, ok := recover().(*Exception)
					if !ok || !ex.Value().ToObject(r).Get("constructor").SameAs(r.Get("RangeError")) {
						t.Fatalf("Export panicked with %v, want a RangeError *Exception", ex)
					}
				}()
				v.Export()
				t.Fatal("Export returned")
			}()
			var out any
			if err := r.ExportTo(v, &out); err == nil {
				t.Fatal("ExportTo returned no error")
			}
			var m map[string]any
			if name == "object" {
				if err := r.ExportTo(v, &m); err == nil {
					t.Fatal("ExportTo into a map returned no error")
				}
			}
		})
	}
	// A value within the limit exports in full.
	v, err := r.RunString(`let b = {}; for (let i = 0; i < 900; i++) b = { b }; b`)
	if err != nil {
		t.Fatal(err)
	}
	depth := 0
	for m, ok := v.Export().(map[string]any); ok; m, ok = m["b"].(map[string]any) {
		depth++
	}
	if depth != 901 {
		t.Fatalf("exported %d levels, want 901", depth)
	}
	// Inside a call the RangeError is thrown into the script.
	r.Set("exportIt", func(call FunctionCall) Value { call.Argument(0).Export(); return _undefined })
	v, err = r.RunString(`let c = {}; for (let i = 0; i < 1e5; i++) c = { c }; let got; try { exportIt(c); got = "none"; } catch (e) { got = e instanceof RangeError; } got`)
	if err != nil || v != valueTrue {
		t.Fatalf("got %v, %v; want true", v, err)
	}
}
