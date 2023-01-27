package goja

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
)

func TestFuncProto(t *testing.T) {
	const SCRIPT = `
	"use strict";
	function A() {}
	A.__proto__ = Object;
	A.prototype = {};

	function B() {}
	B.__proto__ = Object.create(null);
	var thrown = false;
	try {
		delete B.prototype;
	} catch (e) {
		thrown = e instanceof TypeError;
	}
	thrown;
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestFuncPrototypeRedefine(t *testing.T) {
	const SCRIPT = `
	let thrown = false;
	try {
		Object.defineProperty(function() {}, "prototype", {
			set: function(_value) {},
		});
	} catch (e) {
		if (e instanceof TypeError) {
			thrown = true;
		} else {
			throw e;
		}
	}
	thrown;
	`

	testScript(SCRIPT, valueTrue, t)
}

func TestFuncExport(t *testing.T) {
	vm := New()
	typ := reflect.TypeOf((func(FunctionCall) Value)(nil))

	f := func(expr string, t *testing.T) {
		v, err := vm.RunString(expr)
		if err != nil {
			t.Fatal(err)
		}
		if actualTyp := v.ExportType(); actualTyp != typ {
			t.Fatalf("Invalid export type: %v", actualTyp)
		}
		ev := v.Export()
		if actualTyp := reflect.TypeOf(ev); actualTyp != typ {
			t.Fatalf("Invalid export value: %v", ev)
		}
	}

	t.Run("regular function", func(t *testing.T) {
		f("(function() {})", t)
	})

	t.Run("arrow function", func(t *testing.T) {
		f("(()=>{})", t)
	})

	t.Run("method", func(t *testing.T) {
		f("({m() {}}).m", t)
	})

	t.Run("class", func(t *testing.T) {
		f("(class {})", t)
	})
}

func TestFuncWrapUnwrap(t *testing.T) {
	vm := New()
	f := func(a int, b string) bool {
		return a > 0 && b != ""
	}
	var f1 func(int, string) bool
	v := vm.ToValue(f)
	if et := v.ExportType(); et != reflect.TypeOf(f1) {
		t.Fatal(et)
	}
	err := vm.ExportTo(v, &f1)
	if err != nil {
		t.Fatal(err)
	}
	if !f1(1, "a") {
		t.Fatal("not true")
	}
}

func TestWrappedFunc(t *testing.T) {
	vm := New()
	f := func(a int, b string) bool {
		return a > 0 && b != ""
	}
	vm.Set("f", f)
	const SCRIPT = `
	assert.sameValue(typeof f, "function");
	const s = f.toString()
	assert(s.endsWith("TestWrappedFunc.func1() { [native code] }"), s);
	assert(f(1, "a"));
	assert(!f(0, ""));
	`
	vm.testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestWrappedFuncErrorPassthrough(t *testing.T) {
	vm := New()
	e := errors.New("test")
	f := func(a int) error {
		if a > 0 {
			return e
		}
		return nil
	}

	var f1 func(a int64) error
	err := vm.ExportTo(vm.ToValue(f), &f1)
	if err != nil {
		t.Fatal(err)
	}
	if err := f1(1); err != e {
		t.Fatal(err)
	}
}

func ExampleAssertConstructor() {
	vm := New()
	res, err := vm.RunString(`
		(class C {
			constructor(x) {
				this.x = x;
			}
		})
	`)
	if err != nil {
		panic(err)
	}
	if ctor, ok := AssertConstructor(res); ok {
		obj, err := ctor(nil, vm.ToValue("Test"))
		if err != nil {
			panic(err)
		}
		fmt.Print(obj.Get("x"))
	} else {
		panic("Not a constructor")
	}
	// Output: Test
}

type testAsyncCtx struct {
	group    string
	refCount int
}

type testAsyncContextTracker struct {
	ctx     *testAsyncCtx
	logFunc func(...interface{})
	resumed bool
}

func (s *testAsyncContextTracker) Grab() interface{} {
	ctx := s.ctx
	if ctx != nil {
		s.logFunc("Grab", ctx.group)
		ctx.refCount++
	}
	return ctx
}

func (s *testAsyncContextTracker) Resumed(trackingObj interface{}) {
	s.logFunc("Resumed", trackingObj)
	if s.resumed {
		panic("Nested Resumed() calls")
	}
	s.ctx = trackingObj.(*testAsyncCtx)
	s.resumed = true
}

func (s *testAsyncContextTracker) releaseCtx() {
	s.ctx.refCount--
	if s.ctx.refCount < 0 {
		panic("refCount < 0")
	}
	if s.ctx.refCount == 0 {
		s.logFunc(s.ctx.group, "is finished")
	}
}

func (s *testAsyncContextTracker) Exited() {
	s.logFunc("Exited")
	if s.ctx != nil {
		s.releaseCtx()
		s.ctx = nil
	}
	s.resumed = false
}

func TestAsyncContextTracker(t *testing.T) {
	r := New()
	var tracker testAsyncContextTracker
	tracker.logFunc = t.Log

	group := func(name string, asyncFunc func(FunctionCall) Value) Value {
		prevCtx := tracker.ctx
		defer func() {
			t.Log("Returned", name)
			tracker.releaseCtx()
			tracker.ctx = prevCtx
		}()
		tracker.ctx = &testAsyncCtx{
			group:    name,
			refCount: 1,
		}
		t.Log("Set", name)
		return asyncFunc(FunctionCall{})
	}
	r.SetAsyncContextTracker(&tracker)
	r.Set("group", group)
	r.Set("check", func(expectedGroup, msg string) {
		var groupName string
		if tracker.ctx != nil {
			groupName = tracker.ctx.group
		}
		if groupName != expectedGroup {
			t.Fatalf("Unexpected group (%q), expected %q in %s", groupName, expectedGroup, msg)
		}
		t.Log("In", msg)
	})

	t.Run("", func(t *testing.T) {
		_, err := r.RunString(`
		group("1", async () => {
		  check("1", "line A");
		  await 3;
		  check("1", "line B");
		  group("2", async () => {
		     check("2", "line C");
		     await 4;
		     check("2", "line D");
		 })
		}).then(() => {
            check("", "line E");
		})
		`)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("", func(t *testing.T) {
		_, err := r.RunString(`
		group("some", async () => {
			check("some", "line A");
		    (async () => {
				check("some", "line B");
		        await 1;
				check("some", "line C");
		        await 2;
				check("some", "line D");
		    })();
			check("some", "line E");
		});
	`)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("", func(t *testing.T) {
		_, err := r.RunString(`
	group("Main", async () => {
		check("Main", "0.1");
		await Promise.all([
			group("A", async () => {
				check("A", "1.1");
				await 1;
				check("A", "1.2");
			}),
			(async () => {
				check("Main", "3.1");
			})(),
			group("B", async () => {
				check("B", "2.1");
				await 2;
				check("B", "2.2");
			})
		]);
		check("Main", "0.2");
	});
	`)
		if err != nil {
			t.Fatal(err)
		}
	})
}
