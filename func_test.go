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

func TestGeneratorReturnIterCleanup(t *testing.T) {
	const SCRIPT = `
	let iterReturnCalled = false;
	function* g() {
		const iter = {
			next() {
				return { value: 43 };
			},
			return() {
				iterReturnCalled = true;
				return {};
			},
			[Symbol.iterator]() {
				return this;
			}
		}
		try {
			for (const v of iter) {
				yield v;
			}
			yield 'working';
		} finally {
			yield 'cleanup';
		}
	}

	const gen = g();
	const r1 = gen.next();
	assert.sameValue(r1.value, 43);	assert(!r1.done);
	assert(!iterReturnCalled);

	const r2 = gen.return('X');
	assert.sameValue(r2.value, 'cleanup'); assert(!r2.done);
	assert(iterReturnCalled);

	const r3 = gen.next();
	assert.sameValue(r3.value, 'X'); assert(r3.done);
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestGeneratorReturnIterCleanupThrow(t *testing.T) {
	const SCRIPT = `
	class TestError extends Error {
	}

	let iterReturnCalled = false;
	function* g() {
		const iter = {
			next() {
				return { value: 43 };
			},
			return() {
				iterReturnCalled = true;
				throw new TestError('boo!');
			},
			[Symbol.iterator]() {
				return this;
			}
		}
		try {
			for (const v of iter) {
				yield v;
			}
		} finally {
			yield 'cleanup';
		}
	}

	const gen = g();
	const r1 = gen.next();
	assert.sameValue(r1.value, 43);	assert(!r1.done);
	assert(!iterReturnCalled);

	const r2 = gen.return('X');
	assert.sameValue(r2.value, 'cleanup'); assert(!r2.done);
	assert(iterReturnCalled);
	assert.throws(TestError, () => gen.next());
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestGeneratorReturnIterCleanupThrow1(t *testing.T) {
	const SCRIPT = `
	class TestError extends Error {
	}

	let iterReturnCalled = false;
	function* g() {
		const iter = {
			next() {
				return { value: 43 };
			},
			return() {
				iterReturnCalled = true;
				throw new Error('boo!');
			},
			[Symbol.iterator]() {
				return this;
			}
		}
		try {
			for (const v of iter) {
				yield 1;
				throw new TestError();
			}
		} finally {
			yield 2;
		}
	}
	
	const gen = g();
	assert.sameValue(gen.next().value, 1);
	assert.sameValue(gen.next().value, 2);
	assert.sameValue(gen.return('x').value, 'x');
	assert(iterReturnCalled);

	const gen1 = g();
	iterReturnCalled = false;
	assert.sameValue(gen1.next().value, 1);
	assert.sameValue(gen1.next().value, 2);
	assert.throws(TestError, () => gen1.next());
	assert(iterReturnCalled);
`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestGeneratorReturnIterCleanupThrowNoCatch(t *testing.T) {
	const SCRIPT = `
	class TestError extends Error {
	}

	let iterReturnCalled = false;
	function* g() {
		const iter = {
			next() {
				return { value: 43 };
			},
			return() {
				iterReturnCalled = true;
				throw new TestError('boo!');
			},
			[Symbol.iterator]() {
				return this;
			}
		}
		for (const v of iter) {
			yield v;
		}
	}

	const gen = g();
	const r1 = gen.next();
	assert.sameValue(r1.value, 43);	assert(!r1.done);
	assert(!iterReturnCalled);

	assert.throws(TestError, () => gen.return('X'));
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestGeneratorReturnIterCleanupThrowCaught(t *testing.T) {
	const SCRIPT = `
	class TestError extends Error {
	}

	let iterReturnCalled = false;
	let caught;
	function* g() {
		const iter = {
			next() {
				return { value: 43 };
			},
			return() {
				iterReturnCalled = true;
				throw new TestError('boo!');
			},
			[Symbol.iterator]() {
				return this;
			}
		}
		try {
			for (const v of iter) {
				yield v;
			}
		} catch (e) {
			caught = e;
		} finally {
			yield 'cleanup';
		}
	}

	const gen = g();
	const r1 = gen.next();
	assert.sameValue(r1.value, 43);	assert(!r1.done);
	assert(!iterReturnCalled);

	const r2 = gen.return('X');
	assert.sameValue(r2.value, 'cleanup'); assert(!r2.done);
	assert(iterReturnCalled);
	gen.next();
	assert(caught instanceof TestError);
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestGeneratorReturnIterCleanupThrowNested(t *testing.T) {
	const SCRIPT = `
	class TestError extends Error {
	}

	let iterReturnCalled = false;
	function* g() {
		const iter = {
			next() {
				return { value: 43 };
			},
			return() {
				iterReturnCalled = true;
				throw new Error('boo!');
			},
			[Symbol.iterator]() {
				return this;
			}
		}
		try {
			try {
				for (const v of iter) {
					yield v;
				}
			} finally {
				yield 'cleanup inner';
				throw new TestError('test');
			}
		} finally {
			yield 'cleanup outer';
		}
		throw new Error('must not get here');
	}

	const gen = g();
	const r1 = gen.next();
	assert.sameValue(r1.value, 43);	assert(!r1.done);
	assert(!iterReturnCalled);

	const r2 = gen.return('X');
	assert.sameValue(r2.value, 'cleanup inner'); assert(!r2.done);
	assert(iterReturnCalled);
	const r3 = gen.next();
	assert.sameValue(r3.value, 'cleanup outer'); assert(!r3.done);

	assert.throws(TestError, () => gen.next());
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestGeneratorReturn(t *testing.T) {
	const SCRIPT = `
	function* withCleanup() {
	  try {
		yield 'working';
	  } finally {
		yield 'cleanup';  // Should suspend here
	  }
	}
	
	const gen = withCleanup();
	const r1 = gen.next();           // {value: 'working', done: false}
	assert.sameValue(r1.value, 'working');
	assert(!r1.done);
	const r2 = gen.return('X');      // {value: 'cleanup', done: false}  ← should suspend
	assert.sameValue(r2.value, 'cleanup');
	assert(!r2.done);
	const r3 = gen.next();           // {value: 'X', done: true}         ← then complete
	assert.sameValue(r3.value, 'X');
	assert(r3.done);
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

// The following 8 tests are copied from https://github.com/tc39/test262/pull/4939
// If that PR gets merged and tc39 tests are updated to a version that includes it, they can be removed.

func TestGeneratorReturn1(t *testing.T) {
	const SCRIPT = `
	function* nestedCleanup() {
	  try {
		try {
		  yield "work";
		} finally {
		  yield "inner-cleanup";
		}
	  } finally {
		yield "outer-cleanup";
	  }
	}
	
	var gen = nestedCleanup();
	var result;
	
	result = gen.next();
	assert.sameValue(result.value, "work", "r1.value");
	assert.sameValue(result.done, false, "r1.done");
	
	result = gen.return("cancelled");
	assert.sameValue(
	  result.value,
	  "inner-cleanup",
	  "r2.value (inner finally yield)",
	);
	assert.sameValue(result.done, false, "r2.done (suspended at inner finally)");
	
	result = gen.next();
	assert.sameValue(
	  result.value,
	  "outer-cleanup",
	  "r3.value (outer finally yield)",
	);
	assert.sameValue(result.done, false, "r3.done (suspended at outer finally)");
	
	result = gen.next();
	assert.sameValue(result.value, "cancelled", "r4.value (return value)");
	assert.sameValue(result.done, true, "r4.done (completed)");
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestGeneratorReturn2(t *testing.T) {
	const SCRIPT = `
	function* genWithOverride() {
	  try {
		yield "work";
	  } finally {
		return "cleanup-override";
	  }
	}
	
	var gen = genWithOverride();
	gen.next();
	
	var result = gen.return("cancelled");
	
	assert.sameValue(
	  result.value,
	  "cleanup-override",
	  "Finally return overrides generator.return() value",
	);
	assert.sameValue(result.done, true, "Generator is done");
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestGeneratorReturn3(t *testing.T) {
	const SCRIPT = `
	function* genWithSuspendingThrowingCleanup() {
	  try {
		yield "work";
	  } finally {
		yield "cleanup";
		throw new Error("cleanup-failed-after-yield");
	  }
	}
	
	var gen = genWithSuspendingThrowingCleanup();
	var result;
	
	result = gen.next();
	assert.sameValue(result.value, "work", "r1.value");
	
	result = gen.return("cancelled");
	assert.sameValue(result.value, "cleanup", "r2.value (yield in finally)");
	assert.sameValue(result.done, false, "r2.done (suspended at yield)");
	
	var caught;
	try {
	  gen.next();
	} catch (e) {
	  caught = e.message;
	}
	
	assert.sameValue(
	  caught,
	  "cleanup-failed-after-yield",
	  "Exception after yield should propagate on resume",
	);
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestGeneratorReturn4(t *testing.T) {
	const SCRIPT = `
	function* genWithThrowingCleanup() {
	  try {
		yield "work";
	  } finally {
		throw new Error("cleanup-failed");
	  }
	}
	
	var gen = genWithThrowingCleanup();
	gen.next();
	
	var caught;
	try {
	  gen.return("cancelled");
	} catch (e) {
	  caught = e.message;
	}
	
	assert.sameValue(
	  caught,
	  "cleanup-failed",
	  "Exception from finally should propagate",
	);
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestGeneratorReturn5(t *testing.T) {
	const SCRIPT = `
	var cleanupInput;
	
	function* cleanupNeedsAck() {
	  try {
		yield "work";
	  } finally {
		cleanupInput = yield "cleanup-1";
		yield "cleanup-2";
	  }
	}
	
	var gen = cleanupNeedsAck();
	var result;
	
	result = gen.next();
	assert.sameValue(result.value, "work", "r1.value");
	
	result = gen.return("cancelled");
	assert.sameValue(result.value, "cleanup-1", "r2.value");
	assert.sameValue(result.done, false, "r2.done");
	
	result = gen.next("ack");
	assert.sameValue(result.value, "cleanup-2", "r3.value");
	assert.sameValue(result.done, false, "r3.done");
	
	result = gen.next();
	assert.sameValue(result.value, "cancelled", "r4.value");
	assert.sameValue(result.done, true, "r4.done");
	
	assert.sameValue(
	  cleanupInput,
	  "ack",
	  "yield in finally received value from next()",
	);
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestGeneratorReturn6(t *testing.T) {
	const SCRIPT = `
	var cleanupOrder = [];
	
	function* inner() {
	  try {
		yield "inner-work";
		return "inner-done";
	  } finally {
		yield "inner-cleanup";
		cleanupOrder.push("inner");
	  }
	}
	
	function* outer() {
	  try {
		var result = yield* inner();
		return result;
	  } finally {
		yield "outer-cleanup";
		cleanupOrder.push("outer");
	  }
	}
	
	var gen = outer();
	var result;
	
	result = gen.next();
	assert.sameValue(
	  result.value,
	  "inner-work",
	  "First yield from inner generator",
	);
	assert.sameValue(result.done, false, "First result done");
	
	result = gen.return("cancelled");
	assert.sameValue(
	  result.value,
	  "inner-cleanup",
	  "Should yield from inner finally",
	);
	assert.sameValue(result.done, false, "Should suspend at inner finally yield");
	
	result = gen.next();
	assert.sameValue(
	  result.value,
	  "outer-cleanup",
	  "Should yield from outer finally",
	);
	assert.sameValue(result.done, false, "Should suspend at outer finally yield");
	
	result = gen.next();
	assert.sameValue(result.done, true, "Should be done after all finally blocks");
	assert.sameValue(
	  cleanupOrder.join(","),
	  "inner,outer",
	  "Cleanup order should be inner then outer",
	);
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestGeneratorReturn7(t *testing.T) {
	const SCRIPT = `
	function* delegatedCleanup() {
	  yield "cleanup-1";
	  yield "cleanup-2";
	}
	
	function* withYieldStarCleanup() {
	  try {
		yield "work";
	  } finally {
		yield* delegatedCleanup();
	  }
	}
	
	var gen = withYieldStarCleanup();
	var result;
	
	result = gen.next();
	assert.sameValue(result.value, "work", "r1.value");
	
	result = gen.return("cancelled");
	assert.sameValue(result.value, "cleanup-1", "r2.value (first delegated yield)");
	assert.sameValue(result.done, false, "r2.done");
	
	result = gen.next();
	assert.sameValue(
	  result.value,
	  "cleanup-2",
	  "r3.value (second delegated yield)",
	);
	assert.sameValue(result.done, false, "r3.done");
	
	result = gen.next();
	assert.sameValue(result.value, "cancelled", "r4.value (return value)");
	assert.sameValue(result.done, true, "r4.done");
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestGeneratorReturn8(t *testing.T) {
	const SCRIPT = `
	var inFinally = 0;
	var afterYield = 0;
	
	function* g() {
	  try {
		yield "in-try";
	  } finally {
		inFinally += 1;
		yield "in-finally";
		afterYield += 1;
	  }
	}
	
	var iter = g();
	var result;
	
	result = iter.next();
	assert.sameValue(result.value, "in-try", "First result value");
	assert.sameValue(result.done, false, "First result done");
	assert.sameValue(inFinally, 0, "finally not yet entered");
	
	result = iter.return(42);
	assert.sameValue(
	  result.value,
	  "in-finally",
	  "Second result value (yield in finally)",
	);
	assert.sameValue(result.done, false, "Second result done (suspended at yield)");
	assert.sameValue(inFinally, 1, "finally block entered");
	assert.sameValue(afterYield, 0, "code after yield not yet executed");
	
	result = iter.next();
	assert.sameValue(result.value, 42, "Third result value (return value)");
	assert.sameValue(result.done, true, "Third result done (completed)");
	assert.sameValue(afterYield, 1, "code after yield executed");
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}
