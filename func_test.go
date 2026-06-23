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

// countingAsyncContextTracker counts Grab/Resumed/Exited calls.
type countingAsyncContextTracker struct {
	grabbed  int
	resumed  int
	exited   int
	exitFunc func() // if non-nil, called from Exited() instead of the default
}

func (t *countingAsyncContextTracker) Grab() interface{} {
	t.grabbed++
	return nil
}

func (t *countingAsyncContextTracker) Resumed(_ interface{}) {
	t.resumed++
}

func (t *countingAsyncContextTracker) Exited() {
	t.exited++
	if t.exitFunc != nil {
		t.exitFunc()
	}
}

type exitedEnqueueTracker struct {
	r        *Runtime
	exiting  bool
	enqueued bool
}

func (t *exitedEnqueueTracker) Grab() interface{} {
	return nil
}

func (t *exitedEnqueueTracker) Resumed(interface{}) {
	if t.exiting {
		panic("Resumed called while Exited is still running")
	}
}

func (t *exitedEnqueueTracker) Exited() {
	if t.exiting {
		panic("nested Exited call")
	}
	t.exiting = true
	defer func() {
		t.exiting = false
	}()
	if !t.enqueued {
		t.enqueued = true
		_, _ = t.r.RunString(`Promise.resolve().then(function () { markExitedEnqueued(); })`)
	}
}

type resumedEnqueueTracker struct {
	r        *Runtime
	resuming bool
	enqueued bool
}

func (t *resumedEnqueueTracker) Grab() interface{} {
	return nil
}

func (t *resumedEnqueueTracker) Resumed(interface{}) {
	if t.resuming {
		panic("nested Resumed while Resumed is still running")
	}
	t.resuming = true
	defer func() {
		t.resuming = false
	}()
	if !t.enqueued {
		t.enqueued = true
		_, _ = t.r.RunString(`Promise.resolve().then(function () { markResumedEnqueued(); })`)
	}
}

func (t *resumedEnqueueTracker) Exited() {}

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

func TestPromiseJobEnqueuer(t *testing.T) {
	drainJobs := func(t *testing.T, r *Runtime, captured *[]func()) {
		for len(*captured) > 0 {
			jobs := *captured
			*captured = nil
			for _, job := range jobs {
				if err := r.RunPromiseJob(job); err != nil {
					t.Fatalf("RunPromiseJob: %v", err)
				}
			}
		}
	}

	t.Run("hook-called", func(t *testing.T) {
		r := New()
		var called bool
		r.SetPromiseJobEnqueuer(func(job func()) {
			called = true
		})
		_, err := r.RunString(`Promise.resolve().then(() => {})`)
		if err != nil {
			t.Fatal(err)
		}
		if !called {
			t.Fatal("hook was not called")
		}
	})

	t.Run("FIFO-order", func(t *testing.T) {
		r := New()
		var order []int
		var captured []func()
		r.SetPromiseJobEnqueuer(func(job func()) {
			captured = append(captured, job)
		})
		r.Set("push", func(v int) {
			order = append(order, v)
		})
		_, err := r.RunString(`
			Promise.resolve().then(() => push(1));
			Promise.resolve().then(() => push(2));
			Promise.resolve().then(() => push(3));
		`)
		if err != nil {
			t.Fatal(err)
		}
		drainJobs(t, r, &captured)
		if len(order) != 3 || order[0] != 1 || order[1] != 2 || order[2] != 3 {
			t.Fatalf("expected [1 2 3], got %v", order)
		}
	})

	t.Run("default-unchanged", func(t *testing.T) {
		r := New()
		var ran bool
		r.Set("mark", func() { ran = true })
		_, err := r.RunString(`Promise.resolve().then(() => mark())`)
		if err != nil {
			t.Fatal(err)
		}
		if !ran {
			t.Fatal("default job queue was not drained")
		}
	})

	t.Run("jobs-work-when-invoked-later", func(t *testing.T) {
		r := New()
		var result int
		var captured []func()
		r.SetPromiseJobEnqueuer(func(job func()) {
			captured = append(captured, job)
		})
		r.Set("set", func(v int) { result = v })
		_, err := r.RunString(`Promise.resolve().then(() => set(42))`)
		if err != nil {
			t.Fatal(err)
		}
		if result != 0 {
			t.Fatalf("job ran during RunString, result=%d", result)
		}
		if len(captured) != 1 {
			t.Fatalf("expected 1 captured job, got %d", len(captured))
		}
		drainJobs(t, r, &captured)
		if result != 42 {
			t.Fatalf("expected result=42 after running job, got %d", result)
		}
	})

	t.Run("nil-restores-default", func(t *testing.T) {
		r := New()
		var ran bool
		var captured []func()
		r.SetPromiseJobEnqueuer(func(job func()) {
			captured = append(captured, job)
		})
		r.Set("mark", func() { ran = true })
		_, err := r.RunString(`Promise.resolve().then(() => mark())`)
		if err != nil {
			t.Fatal(err)
		}
		if ran {
			t.Fatal("job ran during RunString with hook set")
		}
		r.SetPromiseJobEnqueuer(nil)
		ran = false
		_, err = r.RunString(`Promise.resolve().then(() => mark())`)
		if err != nil {
			t.Fatal(err)
		}
		if !ran {
			t.Fatal("default behavior not restored after nil")
		}
	})

	t.Run("async-await-compat", func(t *testing.T) {
		r := New()
		var result int
		var captured []func()
		r.SetPromiseJobEnqueuer(func(job func()) {
			captured = append(captured, job)
		})
		r.Set("set", func(v int) { result = v })
		_, err := r.RunString(`
			async function f() {
				var x = await 1;
				return x + 41;
			}
			f().then(v => set(v));
		`)
		if err != nil {
			t.Fatal(err)
		}
		drainJobs(t, r, &captured)
		if result != 42 {
			t.Fatalf("expected result=42, got %d", result)
		}
	})

	t.Run("interrupt-recovery", func(t *testing.T) {
		r := New()
		var reached bool
		var captured []func()
		r.SetPromiseJobEnqueuer(func(job func()) {
			captured = append(captured, job)
		})
		r.Set("mark", func() { reached = true })
		_, err := r.RunString(`Promise.resolve().then(() => mark())`)
		if err != nil {
			t.Fatal(err)
		}
		if len(captured) != 1 {
			t.Fatalf("expected 1 captured job, got %d", len(captured))
		}
		r.Interrupt("test-interrupt")
		err = r.RunPromiseJob(captured[0])
		if err == nil {
			t.Fatal("expected error from interrupted job")
		}
		if _, ok := err.(*InterruptedError); !ok {
			t.Fatalf("expected *InterruptedError, got %T: %v", err, err)
		}
		if reached {
			t.Fatal("job should not have completed")
		}
		_, err = r.RunString(`mark()`)
		if err != nil {
			t.Fatalf("runtime not reusable after interrupt: %v", err)
		}
		if !reached {
			t.Fatal("runtime did not execute after ClearInterrupt")
		}
	})

	t.Run("pending-interrupt-native-handler", func(t *testing.T) {
		r := New()
		var captured []func()
		var reached bool

		r.SetPromiseJobEnqueuer(func(job func()) {
			captured = append(captured, job)
		})
		if err := r.Set("nativeHandler", func() {
			reached = true
		}); err != nil {
			t.Fatal(err)
		}

		_, err := r.RunString(`Promise.resolve().then(nativeHandler)`)
		if err != nil {
			t.Fatal(err)
		}
		if len(captured) != 1 {
			t.Fatalf("expected 1 captured job, got %d", len(captured))
		}

		r.Interrupt("test-interrupt")
		err = r.RunPromiseJob(captured[0])
		if _, ok := err.(*InterruptedError); !ok {
			t.Fatalf("expected RunPromiseJob to return *InterruptedError before running native handler under pending interrupt; got %T (%v), native handler reached=%v", err, err, reached)
		}
		if reached {
			t.Fatal("native handler ran even though the interrupt was already pending")
		}
		_, err = r.RunString(`nativeHandler()`)
		if err != nil {
			t.Fatalf("runtime not reusable after interrupted RunPromiseJob: %v", err)
		}
		if !reached {
			t.Fatal("runtime did not execute after interrupted RunPromiseJob cleanup")
		}
	})

	t.Run("pending-interrupt-nil-handler-propagation", func(t *testing.T) {
		r := New()
		var captured []func()

		r.SetPromiseJobEnqueuer(func(job func()) {
			captured = append(captured, job)
		})

		_, err := r.RunString(`Promise.resolve(1).then()`)
		if err != nil {
			t.Fatal(err)
		}
		if len(captured) != 1 {
			t.Fatalf("expected 1 captured nil-handler propagation job, got %d", len(captured))
		}

		r.Interrupt("test-interrupt")
		err = r.RunPromiseJob(captured[0])
		if _, ok := err.(*InterruptedError); !ok {
			t.Fatalf("expected RunPromiseJob to return *InterruptedError before nil-handler propagation under pending interrupt; got %T (%v)", err, err)
		}
		if _, err := r.RunString(`1 + 1`); err != nil {
			t.Fatalf("runtime not reusable after interrupted nil-handler job: %v", err)
		}
	})

	t.Run("stack-overflow-recovery", func(t *testing.T) {
		r := New()
		r.SetMaxCallStackSize(5)
		var reached bool
		var captured []func()
		r.SetPromiseJobEnqueuer(func(job func()) {
			captured = append(captured, job)
		})
		r.Set("mark", func() { reached = true })
		_, err := r.RunString(`
			Promise.resolve().then(function f() { f(); });
		`)
		if err != nil {
			t.Fatal(err)
		}
		if len(captured) != 1 {
			t.Fatalf("expected 1 captured job, got %d", len(captured))
		}
		err = r.RunPromiseJob(captured[0])
		if err == nil {
			t.Fatal("expected error from stack overflow")
		}
		if _, ok := err.(*StackOverflowError); !ok {
			t.Fatalf("expected *StackOverflowError, got %T: %v", err, err)
		}
		if reached {
			t.Fatal("job should not have completed")
		}
		_, err = r.RunString(`mark()`)
		if err != nil {
			t.Fatalf("runtime not reusable after stack overflow: %v", err)
		}
		if !reached {
			t.Fatal("runtime did not execute after stack overflow recovery")
		}
	})

	t.Run("panic-reraise", func(t *testing.T) {
		r := New()
		sentinel := errors.New("not an uncatchable exception")
		defer func() {
			if x := recover(); x == nil {
				t.Fatal("expected RunPromiseJob to re-panic, but it did not panic")
			} else if x != sentinel {
				t.Fatalf("expected re-panic with sentinel error, got %v (%T)", x, x)
			}
		}()
		_ = r.RunPromiseJob(func() { panic(sentinel) })
		t.Fatal("expected RunPromiseJob to re-panic")
	})

	t.Run("stack-cleared-no-promise", func(t *testing.T) {
		r := New()
		r.SetPromiseJobEnqueuer(func(job func()) {})
		_, err := r.RunString(`1 + 1`)
		if err != nil {
			t.Fatal(err)
		}
		if r.vm.stack != nil {
			t.Fatalf("r.vm.stack should be nil after RunString with hook set, got len=%d", len(r.vm.stack))
		}
		if r.jobQueue != nil {
			t.Fatalf("r.jobQueue should be nil after RunString with hook set, got len=%d", len(r.jobQueue))
		}
	})

	t.Run("stack-cleared-with-promise", func(t *testing.T) {
		r := New()
		var captured []func()
		r.SetPromiseJobEnqueuer(func(job func()) {
			captured = append(captured, job)
		})
		_, err := r.RunString(`Promise.resolve().then(() => {}); 1 + 1`)
		if err != nil {
			t.Fatal(err)
		}
		if r.vm.stack != nil {
			t.Fatalf("r.vm.stack should be nil after RunString with hook set, got len=%d", len(r.vm.stack))
		}
		if r.jobQueue != nil {
			t.Fatalf("r.jobQueue should be nil after RunString with hook set, got len=%d", len(r.jobQueue))
		}
		// Captured jobs must still work when drained.
		drainJobs(t, r, &captured)
	})

	t.Run("stack-cleared-after-runpromisejob", func(t *testing.T) {
		r := New()
		var captured []func()
		r.SetPromiseJobEnqueuer(func(job func()) {
			captured = append(captured, job)
		})
		r.Set("set", func(v int) {})
		_, err := r.RunString(`Promise.resolve().then(() => set(42))`)
		if err != nil {
			t.Fatal(err)
		}
		if len(captured) != 1 {
			t.Fatalf("expected 1 captured job, got %d", len(captured))
		}
		if err := r.RunPromiseJob(captured[0]); err != nil {
			t.Fatalf("RunPromiseJob: %v", err)
		}
		if r.vm.stack != nil {
			t.Fatalf("r.vm.stack should be nil after RunPromiseJob, got len=%d", len(r.vm.stack))
		}
	})

	t.Run("stack-cleared-after-runpromisejob-error", func(t *testing.T) {
		r := New()
		var captured []func()
		r.SetPromiseJobEnqueuer(func(job func()) {
			captured = append(captured, job)
		})
		r.Set("mark", func() {})
		_, err := r.RunString(`Promise.resolve().then(() => mark())`)
		if err != nil {
			t.Fatal(err)
		}
		if len(captured) != 1 {
			t.Fatalf("expected 1 captured job, got %d", len(captured))
		}
		r.Interrupt("test-interrupt")
		err = r.RunPromiseJob(captured[0])
		if err == nil {
			t.Fatal("expected error from interrupted job")
		}
		if r.vm.stack != nil {
			t.Fatalf("r.vm.stack should be nil after RunPromiseJob error, got len=%d", len(r.vm.stack))
		}
		if _, err := r.RunString(`1 + 1`); err != nil {
			t.Fatalf("runtime not reusable after error: %v", err)
		}
	})

	t.Run("stranded-jobs-forwarded", func(t *testing.T) {
		r := New()
		var order []int
		var captured []func()
		r.Set("installHook", func() {
			r.SetPromiseJobEnqueuer(func(job func()) {
				captured = append(captured, job)
			})
		})
		r.Set("push", func(v int) {
			order = append(order, v)
		})
		_, err := r.RunString(`
			Promise.resolve().then(() => push(42));
			installHook();
		`)
		if err != nil {
			t.Fatal(err)
		}
		if len(captured) != 1 {
			t.Fatalf("expected 1 forwarded job, got %d", len(captured))
		}
		if r.jobQueue != nil {
			t.Fatalf("r.jobQueue should be nil, got len=%d", len(r.jobQueue))
		}
		drainJobs(t, r, &captured)
		if len(order) != 1 || order[0] != 42 {
			t.Fatalf("expected order=[42], got %v", order)
		}
	})

	t.Run("synchronous-hook", func(t *testing.T) {
		r := New()
		var result int
		r.Set("set", func(v int) { result = v })
		r.SetPromiseJobEnqueuer(func(job func()) {
			r.RunPromiseJob(job)
		})
		_, err := r.RunString(`Promise.resolve(42).then(v => set(v))`)
		if err != nil {
			t.Fatalf("RunString: %v", err)
		}
		if result != 42 {
			t.Fatalf("expected result=42, got %d", result)
		}
	})

	t.Run("reentrant-go-callback", func(t *testing.T) {
		r := New()
		var captured []func()
		r.SetPromiseJobEnqueuer(func(job func()) {
			captured = append(captured, job)
		})
		_, err := r.RunString(`Promise.resolve().then(() => {})`)
		if err != nil {
			t.Fatal(err)
		}
		if len(captured) != 1 {
			t.Fatalf("expected 1 captured job, got %d", len(captured))
		}

		var jobRan bool
		r.Set("runJob", func() {
			if err := r.RunPromiseJob(captured[0]); err != nil {
				t.Fatalf("RunPromiseJob from callback: %v", err)
			}
			jobRan = true
		})
		_, err = r.RunString(`runJob()`)
		if err != nil {
			t.Fatalf("RunString(runJob()): %v", err)
		}
		if !jobRan {
			t.Fatal("job did not run")
		}
		if _, err := r.RunString(`1 + 1`); err != nil {
			t.Fatalf("runtime not reusable: %v", err)
		}
	})

	t.Run("synchronous-hook-interrupt", func(t *testing.T) {
		r := New()
		var reached bool
		r.Set("mark", func() { reached = true })
		r.SetPromiseJobEnqueuer(func(job func()) {
			r.Interrupt("test-interrupt")
			err := r.RunPromiseJob(job)
			if _, ok := err.(*InterruptedError); !ok {
				t.Fatalf("expected *InterruptedError, got %T: %v", err, err)
			}
		})
		_, err := r.RunString(`Promise.resolve().then(() => mark())`)
		if _, ok := err.(*InterruptedError); !ok {
			t.Fatalf("expected *InterruptedError from RunString, got %T: %v", err, err)
		}
		if reached {
			t.Fatal("job should not have completed")
		}
		if _, err := r.RunString(`mark()`); err != nil {
			t.Fatalf("runtime not reusable after interrupt: %v", err)
		}
		if !reached {
			t.Fatal("mark() did not run after recovery")
		}
	})

	t.Run("leave-abrupt-forwards-stranded-jobs", func(t *testing.T) {
		r := New()
		var order []int
		var captured []func()

		r.Set("installHook", func() {
			r.SetPromiseJobEnqueuer(func(job func()) {
				captured = append(captured, job)
			})
		})
		r.Set("push", func(v int) { order = append(order, v) })
		r.Set("boom", func() { r.Interrupt("test-abrupt") })

		_, err := r.RunString(`
			Promise.resolve().then(() => push(42));
			installHook();
			boom();
		`)
		if _, ok := err.(*InterruptedError); !ok {
			t.Fatalf("expected *InterruptedError, got %T: %v", err, err)
		}
		if len(captured) != 1 {
			t.Fatalf("expected 1 forwarded stranded job, got %d", len(captured))
		}
		if r.jobQueue != nil {
			t.Fatalf("r.jobQueue should be nil after leaveAbrupt, got len=%d", len(r.jobQueue))
		}
		if r.vm.stack != nil {
			t.Fatalf("r.vm.stack should be nil after leaveAbrupt, got len=%d", len(r.vm.stack))
		}
		if err := r.RunPromiseJob(captured[0]); err != nil {
			t.Fatalf("RunPromiseJob on forwarded stranded job: %v", err)
		}
		if len(order) != 1 || order[0] != 42 {
			t.Fatalf("expected order=[42], got %v", order)
		}
	})

	t.Run("leave-abrupt-clears-stack", func(t *testing.T) {
		r := New()
		r.Set("boom", func() { r.Interrupt("test-abrupt") })
		_, err := r.RunString(`var x = { a: 1, b: 2, c: 3 }; boom();`)
		if err == nil {
			t.Fatal("expected InterruptedError")
		}
		if r.vm.stack != nil {
			t.Fatalf("r.vm.stack should be nil after interrupt, got len=%d", len(r.vm.stack))
		}
	})

	t.Run("tracker-exited-on-interrupt", func(t *testing.T) {
		r := New()

		var tracker testAsyncContextTracker
		tracker.logFunc = t.Log
		tracker.ctx = &testAsyncCtx{group: "g", refCount: 0}
		r.SetAsyncContextTracker(&tracker)

		var captured []func()
		r.SetPromiseJobEnqueuer(func(job func()) {
			captured = append(captured, job)
		})
		r.Set("boom", func() { r.Interrupt("test-interrupt") })

		_, err := r.RunString(`Promise.resolve().then(() => { boom(); })`)
		if err != nil {
			t.Fatal(err)
		}
		if len(captured) != 1 {
			t.Fatalf("expected 1 captured job, got %d", len(captured))
		}
		err = r.RunPromiseJob(captured[0])
		if _, ok := err.(*InterruptedError); !ok {
			t.Fatalf("expected *InterruptedError, got %T: %v", err, err)
		}
		if tracker.resumed {
			t.Fatal("tracker.resumed should be false; Exited() was not called")
		}
		if _, err := r.RunString(`1 + 1`); err != nil {
			t.Fatalf("runtime not reusable after interrupt: %v", err)
		}
	})

	// Reentrant RunPromiseJob with StackOverflowError. After handleThrow
	// truncates callStack and the defer calls clearStack, the outer script
	// can continue normally.
	t.Run("reentrant-stack-overflow", func(t *testing.T) {
		r := New()
		r.SetMaxCallStackSize(10)
		var hookErr error
		r.SetPromiseJobEnqueuer(func(job func()) {
			hookErr = r.RunPromiseJob(job)
		})
		_, err := r.RunString(`Promise.resolve().then(function f() { f(); })`)
		if err != nil {
			t.Fatalf("outer RunString should succeed after reentrant stack overflow recovery, got: %v", err)
		}
		if hookErr == nil {
			t.Fatal("expected error from reentrant RunPromiseJob")
		}
		if _, ok := hookErr.(*StackOverflowError); !ok {
			t.Fatalf("expected *StackOverflowError from RunPromiseJob, got %T: %v", hookErr, hookErr)
		}
		if _, err := r.RunString(`1 + 1`); err != nil {
			t.Fatalf("runtime not reusable after stack overflow: %v", err)
		}
	})

	// RunPromiseJob called via AssertFunction (runWrapped, not RunProgram).
	t.Run("reentrant-from-assertfunction", func(t *testing.T) {
		r := New()
		var order []int
		var captured []func()
		r.SetPromiseJobEnqueuer(func(job func()) {
			captured = append(captured, job)
		})
		r.Set("push", func(v int) { order = append(order, v) })
		r.Set("runJob", func() {
			if err := r.RunPromiseJob(captured[0]); err != nil {
				t.Fatalf("RunPromiseJob from callback: %v", err)
			}
		})
		// Compile a JS function expression that creates a promise
		// reaction and then calls runJob() to drain it reentrantly.
		p, err := Compile("", `(function() { Promise.resolve().then(function() { push(42) }); runJob(); })`, false)
		if err != nil {
			t.Fatal(err)
		}
		v, err := r.RunProgram(p)
		if err != nil {
			t.Fatal(err)
		}
		fn, ok := AssertFunction(v)
		if !ok {
			t.Fatal("expected function")
		}
		// Call via AssertFunction (uses runWrapped, not RunProgram).
		_, err = fn(nil)
		if err != nil {
			t.Fatalf("AssertFunction call: %v", err)
		}
		if len(order) != 1 || order[0] != 42 {
			t.Fatalf("expected order=[42], got %v", order)
		}
		if _, err := r.RunString(`1 + 1`); err != nil {
			t.Fatalf("runtime not reusable: %v", err)
		}
	})

	// Clearing the hook mid-job: subsequent jobs go to the internal queue
	// and are drained by leave() on the next Run*() call.
	t.Run("hook-replaced-mid-turn", func(t *testing.T) {
		r := New()
		var secondRan bool
		var captured []func()
		r.SetPromiseJobEnqueuer(func(job func()) {
			captured = append(captured, job)
		})
		r.Set("mark", func() { secondRan = true })
		r.Set("clearHook", func() { r.SetPromiseJobEnqueuer(nil) })
		_, err := r.RunString(`Promise.resolve().then(function() { clearHook(); Promise.resolve().then(function() { mark() }); })`)
		if err != nil {
			t.Fatal(err)
		}
		if len(captured) != 1 {
			t.Fatalf("expected 1 captured job, got %d", len(captured))
		}
		// Run the first job: it clears the hook and enqueues a second
		// reaction to the internal queue.
		if err := r.RunPromiseJob(captured[0]); err != nil {
			t.Fatalf("RunPromiseJob: %v", err)
		}
		if secondRan {
			t.Fatal("second reaction should not have run yet")
		}
		if r.jobQueue == nil || len(r.jobQueue) == 0 {
			t.Fatal("expected second reaction in internal queue")
		}
		// leave() should drain the internal queue on the next Run*() call.
		if _, err := r.RunString(`1 + 1`); err != nil {
			t.Fatalf("RunString: %v", err)
		}
		if !secondRan {
			t.Fatal("second reaction was not drained by leave()")
		}
		if r.jobQueue != nil {
			t.Fatalf("r.jobQueue should be nil after leave(), got len=%d", len(r.jobQueue))
		}
	})

	// Deeply chained reactions drained through a synchronous hook.
	t.Run("deep-chain-synchronous-drain", func(t *testing.T) {
		r := New()
		var result int
		r.Set("set", func(v int) { result = v })
		r.SetPromiseJobEnqueuer(func(job func()) {
			if err := r.RunPromiseJob(job); err != nil {
				t.Fatalf("RunPromiseJob in deep chain: %v", err)
			}
		})
		const chainLen = 100
		_, err := r.RunString(fmt.Sprintf(`
			var p = Promise.resolve(0);
			for (var i = 0; i < %d; i++) {
				p = p.then(function(v) { return v + 1; });
			}
			p.then(function(v) { set(v); });
		`, chainLen))
		if err != nil {
			t.Fatalf("RunString: %v", err)
		}
		if result != chainLen {
			t.Fatalf("expected result=%d, got %d", chainLen, result)
		}
		if _, err := r.RunString(`1 + 1`); err != nil {
			t.Fatalf("runtime not reusable: %v", err)
		}
	})

	// Reaction on a settled promise must not execute during .then().
	t.Run("turn-ordering-preserved", func(t *testing.T) {
		r := New()
		var log []int
		var captured []func()
		r.SetPromiseJobEnqueuer(func(job func()) {
			captured = append(captured, job)
		})
		r.Set("push", func(v int) { log = append(log, v) })
		_, err := r.RunString(`
			var p = Promise.resolve(42);
			p.then(() => push(1));
			push(0);
		`)
		if err != nil {
			t.Fatal(err)
		}
		if len(log) != 1 || log[0] != 0 {
			t.Fatalf("expected log=[0] during RunString, got %v", log)
		}
		drainJobs(t, r, &captured)
		if len(log) != 2 || log[0] != 0 || log[1] != 1 {
			t.Fatalf("expected log=[0 1] after drain, got %v", log)
		}
	})

	// Multiple stranded jobs forwarded by leave() must arrive in FIFO order.
	t.Run("fifo-stranded-jobs-forwarded", func(t *testing.T) {
		r := New()
		var order []int
		var captured []func()
		r.Set("installHook", func() {
			r.SetPromiseJobEnqueuer(func(job func()) {
				captured = append(captured, job)
			})
		})
		r.Set("push", func(v int) { order = append(order, v) })
		_, err := r.RunString(`
			Promise.resolve().then(() => push(1));
			Promise.resolve().then(() => push(2));
			Promise.resolve().then(() => push(3));
			installHook();
		`)
		if err != nil {
			t.Fatal(err)
		}
		if len(captured) != 3 {
			t.Fatalf("expected 3 forwarded jobs, got %d", len(captured))
		}
		drainJobs(t, r, &captured)
		if len(order) != 3 || order[0] != 1 || order[1] != 2 || order[2] != 3 {
			t.Fatalf("expected order=[1 2 3], got %v", order)
		}
	})

	// Installing the hook mid-turn: jobs enqueued after the hook is set
	// are delivered directly during the turn, before older jobs forwarded
	// from the internal queue at turn exit. This violates global FIFO and
	// is documented on PromiseJobEnqueuer.
	t.Run("hook-installed-mid-turn-ordering", func(t *testing.T) {
		r := New()
		var order []int
		var captured []func()
		r.Set("push", func(v int) { order = append(order, v) })
		r.Set("installHook", func() {
			r.SetPromiseJobEnqueuer(func(job func()) {
				captured = append(captured, job)
			})
		})
		_, err := r.RunString(`
			Promise.resolve().then(() => push(1));
			Promise.resolve().then(() => push(2));
			installHook();
			Promise.resolve().then(() => push(3));
			Promise.resolve().then(() => push(4));
		`)
		if err != nil {
			t.Fatal(err)
		}
		// Jobs 1,2 were in the internal queue before the hook was set;
		// jobs 3,4 were delivered directly. leave() forwards [1,2] after
		// [3,4], so captured is [3,4,1,2].
		if len(captured) != 4 {
			t.Fatalf("expected 4 captured jobs, got %d", len(captured))
		}
		drainJobs(t, r, &captured)
		if len(order) != 4 || order[0] != 3 || order[1] != 4 || order[2] != 1 || order[3] != 2 {
			t.Fatalf("expected order=[3 4 1 2] (non-FIFO: direct jobs before forwarded), got %v", order)
		}
		if r.jobQueue != nil {
			t.Fatalf("r.jobQueue should be nil after leave(), got len=%d", len(r.jobQueue))
		}
		if _, err := r.RunString(`1 + 1`); err != nil {
			t.Fatalf("runtime not reusable: %v", err)
		}
	})

	// RunPromiseJob forwards internally queued jobs to the hook on success.
	t.Run("runpromisejob-forwards-to-hook", func(t *testing.T) {
		r := New()
		var secondRan bool
		var captured []func()
		r.SetPromiseJobEnqueuer(func(job func()) {
			captured = append(captured, job)
		})
		r.Set("mark", func() { secondRan = true })
		r.Set("clearHook", func() { r.SetPromiseJobEnqueuer(nil) })
		r.Set("restoreHook", func() {
			r.SetPromiseJobEnqueuer(func(job func()) {
				captured = append(captured, job)
			})
		})
		_, err := r.RunString(`Promise.resolve().then(function() { clearHook(); Promise.resolve().then(function() { mark() }); restoreHook(); })`)
		if err != nil {
			t.Fatal(err)
		}
		if len(captured) != 1 {
			t.Fatalf("expected 1 captured job, got %d", len(captured))
		}
		// Run the first job: clears hook, enqueues a second reaction
		// to the internal queue, restores the hook. RunPromiseJob
		// should forward the queued job to the hook.
		if err := r.RunPromiseJob(captured[0]); err != nil {
			t.Fatalf("RunPromiseJob: %v", err)
		}
		if len(captured) != 2 {
			t.Fatalf("expected 2 captured jobs after RunPromiseJob, got %d", len(captured))
		}
		if err := r.RunPromiseJob(captured[1]); err != nil {
			t.Fatalf("RunPromiseJob for forwarded job: %v", err)
		}
		if !secondRan {
			t.Fatal("second reaction was not forwarded to hook")
		}
		if r.jobQueue != nil {
			t.Fatalf("r.jobQueue should be nil, got len=%d", len(r.jobQueue))
		}
	})

	t.Run("runpromisejob-hook-cleared-mid-batch-preserves-fifo", func(t *testing.T) {
		r := New()
		var order []int
		var captured []func()

		r.Set("push", func(v int) { order = append(order, v) })
		r.Set("clearHook", func() { r.SetPromiseJobEnqueuer(nil) })
		r.Set("restoreSyncHook", func() {
			r.SetPromiseJobEnqueuer(func(job func()) {
				if err := r.RunPromiseJob(job); err != nil {
					t.Fatalf("RunPromiseJob from sync hook: %v", err)
				}
			})
		})
		r.SetPromiseJobEnqueuer(func(job func()) {
			captured = append(captured, job)
		})

		_, err := r.RunString(`
			Promise.resolve().then(function() {
				clearHook();
				Promise.resolve().then(function() {
					push(1);
					clearHook();
					Promise.resolve().then(function() { push(3); });
				});
				Promise.resolve().then(function() { push(2); });
				restoreSyncHook();
			});
		`)
		if err != nil {
			t.Fatal(err)
		}
		if len(captured) != 1 {
			t.Fatalf("expected 1 captured job, got %d", len(captured))
		}

		if err := r.RunPromiseJob(captured[0]); err != nil {
			t.Fatalf("RunPromiseJob: %v", err)
		}
		if len(order) != 1 || order[0] != 1 {
			t.Fatalf("expected only first forwarded job to have run immediately, got order=%v", order)
		}
		if _, err := r.RunString(`1 + 1`); err != nil {
			t.Fatalf("RunString drain: %v", err)
		}
		if len(order) != 3 || order[0] != 1 || order[1] != 2 || order[2] != 3 {
			t.Fatalf("expected FIFO order [1 2 3], got %v", order)
		}
	})

	// A sync hook that clears itself during leave() forwarding: jobs
	// enqueued after clearing must be drained internally, not forwarded
	// to the removed hook.
	t.Run("hook-cleared-during-leave-forwarding", func(t *testing.T) {
		r := New()
		var order []int
		var hookCallCount int
		r.Set("push", func(v int) { order = append(order, v) })
		r.Set("installSyncHook", func() {
			r.SetPromiseJobEnqueuer(func(job func()) {
				hookCallCount++
				r.RunPromiseJob(job)
			})
		})
		r.Set("clearHook", func() { r.SetPromiseJobEnqueuer(nil) })
		_, err := r.RunString(`
			Promise.resolve().then(function() {
				push(1);
				clearHook();
				Promise.resolve().then(function() { push(2); });
			});
			installSyncHook();
		`)
		if err != nil {
			t.Fatal(err)
		}
		// Only A was forwarded to the sync hook; B was drained internally
		// after the hook was cleared.
		if hookCallCount != 1 {
			t.Fatalf("expected hookCallCount=1 (only A forwarded, B drained internally), got %d", hookCallCount)
		}
		if len(order) != 2 || order[0] != 1 || order[1] != 2 {
			t.Fatalf("expected order=[1 2], got %v", order)
		}
		if r.jobQueue != nil {
			t.Fatalf("r.jobQueue should be nil, got len=%d", len(r.jobQueue))
		}
		if _, err := r.RunString(`1 + 1`); err != nil {
			t.Fatalf("runtime not reusable: %v", err)
		}
	})

	// Exited() must be called before capability.resolve/reject so the
	// tracker does not span thenable assimilation. When the handler
	// returns a thenable, resolve calls its 'then' method.
	t.Run("tracker-exited-before-resolution", func(t *testing.T) {
		r := New()
		var tracker testAsyncContextTracker
		tracker.logFunc = t.Log
		tracker.ctx = &testAsyncCtx{group: "g", refCount: 0}
		r.SetAsyncContextTracker(&tracker)

		var thenCalledWhileResumed bool
		r.SetPromiseJobEnqueuer(func(job func()) {
			r.RunPromiseJob(job)
		})

		// The thenable's 'then' is called during assimilation inside
		// capability.resolve, after the handler returns.
		r.Set("makeThenable", func() Value {
			thenFn := r.ToValue(func(call FunctionCall) Value {
				if tracker.resumed {
					thenCalledWhileResumed = true
				}
				return _undefined
			})
			obj := r.NewObject()
			obj.Set("then", thenFn)
			return obj
		})

		_, err := r.RunString(`Promise.resolve().then(function() { return makeThenable(); })`)
		if err != nil {
			t.Fatal(err)
		}
		if thenCalledWhileResumed {
			t.Fatal("tracker was still resumed when 'then' was called during thenable assimilation; Exited() was not called before capability.resolve()")
		}
		if _, err := r.RunString(`1 + 1`); err != nil {
			t.Fatalf("runtime not reusable: %v", err)
		}
	})

	// Sync hook + tracker + chained pending reactions: resolving a
	// promise triggers a downstream reaction reentrantly via the sync
	// hook. Exited() must have been called before resolve so the
	// nested Resumed() does not panic.
	t.Run("sync-hook-tracker-chained-reactions", func(t *testing.T) {
		r := New()
		var tracker testAsyncContextTracker
		tracker.logFunc = t.Log
		tracker.ctx = &testAsyncCtx{group: "g", refCount: 0}
		r.SetAsyncContextTracker(&tracker)

		var chainCompleted bool
		r.Set("mark", func() { chainCompleted = true })
		r.SetPromiseJobEnqueuer(func(job func()) {
			r.RunPromiseJob(job)
		})

		// Chain a reaction on the derived promise, then resolve p.
		// The sync hook runs J1; J1's resolve triggers J2 reentrantly.
		_, err := r.RunString(`
			var resolveP;
			var p = new Promise(function(resolve) { resolveP = resolve; });
			var derived = p.then(function() {});
			derived.then(function() { mark(); });
			resolveP();
		`)
		if err != nil {
			t.Fatalf("RunString: %v", err)
		}
		if !chainCompleted {
			t.Fatal("promise chain did not complete; tracker.Resumed() may have panicked with 'Nested Resumed() calls'")
		}
		if _, err := r.RunString(`1 + 1`); err != nil {
			t.Fatalf("runtime not reusable: %v", err)
		}
	})

	// Two jobs in the same initial batch: the first clears the hook,
	// the second must not be forwarded to the removed hook.
	t.Run("same-batch-stale-hook", func(t *testing.T) {
		r := New()
		var order []int
		var hookCallCount int
		r.Set("push", func(v int) { order = append(order, v) })
		r.Set("installSyncHook", func() {
			r.SetPromiseJobEnqueuer(func(job func()) {
				hookCallCount++
				r.RunPromiseJob(job)
			})
		})
		r.Set("clearHook", func() { r.SetPromiseJobEnqueuer(nil) })
		_, err := r.RunString(`
			Promise.resolve().then(function() { clearHook(); push(1); });
			Promise.resolve().then(function() { push(2); });
			installSyncHook();
		`)
		if err != nil {
			t.Fatal(err)
		}
		if hookCallCount != 1 {
			t.Fatalf("expected hookCallCount=1, got %d", hookCallCount)
		}
		if len(order) != 2 || order[0] != 1 || order[1] != 2 {
			t.Fatalf("expected order=[1 2], got %v", order)
		}
		if r.jobQueue != nil {
			t.Fatalf("r.jobQueue should be nil, got len=%d", len(r.jobQueue))
		}
		if _, err := r.RunString(`1 + 1`); err != nil {
			t.Fatalf("runtime not reusable: %v", err)
		}
	})

	// leaveAbrupt with a sync hook: the interrupt flag must stay active
	// for all forwarded jobs. Nested leaveAbrupt() calls (from
	// RunPromiseJob's error path) must not clear the interrupt.
	t.Run("leaveabrupt-interrupt-not-cleared", func(t *testing.T) {
		r := New()
		var order []int
		var hookCallCount int
		r.Set("push", func(v int) { order = append(order, v) })
		r.Set("installSyncHookAndInterrupt", func() {
			r.SetPromiseJobEnqueuer(func(job func()) {
				hookCallCount++
				_ = r.RunPromiseJob(job)
			})
			r.Interrupt("test-abrupt")
		})
		// Both jobs are in r.jobQueue before the hook is set. The hook
		// is then installed and the interrupt triggered.
		_, err := r.RunString(`
			Promise.resolve().then(function() { push(1); });
			Promise.resolve().then(function() { push(2); });
			installSyncHookAndInterrupt();
		`)
		if err == nil {
			t.Fatal("expected InterruptedError from RunString, got nil")
		}
		if hookCallCount != 2 {
			t.Fatalf("expected hookCallCount=2, got %d", hookCallCount)
		}
		// No jobs should have run, both interrupted.
		if len(order) != 0 {
			t.Fatalf("expected order=[], got %v", order)
		}
		if _, err := r.RunString(`1 + 1`); err != nil {
			t.Fatalf("runtime not reusable: %v", err)
		}
	})

	// Standalone RunPromiseJob with a native handler calling RunString:
	// the sentinel context ensures the nested RunProgram is treated as
	// recursive so leave() does not drain jobs while the outer tracker
	// is still resumed.
	t.Run("standalone-runpromisejob-native-reentry", func(t *testing.T) {
		r := New()
		var tracker testAsyncContextTracker
		tracker.logFunc = t.Log
		tracker.ctx = &testAsyncCtx{group: "g", refCount: 0}
		r.SetAsyncContextTracker(&tracker)

		var captured []func()
		r.SetPromiseJobEnqueuer(func(job func()) {
			captured = append(captured, job)
		})

		// nativeHandler clears the hook and calls RunString which
		// creates a new promise reaction.
		r.Set("nativeHandler", func() {
			r.SetPromiseJobEnqueuer(nil)
			_, _ = r.RunString(`Promise.resolve().then(function() {})`)
		})

		// Capture the reaction job.
		_, err := r.RunString(`Promise.resolve().then(nativeHandler)`)
		if err != nil {
			t.Fatal(err)
		}
		if len(captured) != 1 {
			t.Fatalf("expected 1 captured job, got %d", len(captured))
		}

		// Execute the captured job standalone (callStack empty).
		err = r.RunPromiseJob(captured[0])
		if err != nil {
			t.Fatalf("RunPromiseJob: %v", err)
		}

		// Drain jobs that accumulated in r.jobQueue during the nested
		// RunString (hook was cleared).
		if _, err := r.RunString(`1 + 1`); err != nil {
			t.Fatalf("runtime not reusable: %v", err)
		}
	})

	// The tracker used at execution time must be the one captured at
	// scheduling time, not the current runtime tracker. Tracker B is
	// poisoned (resumed=true) so a reaction that wrongly uses B panics.
	t.Run("tracker-identity-captured", func(t *testing.T) {
		r := New()

		// Tracker A, active when the reaction is created.
		var trackerA testAsyncContextTracker
		trackerA.logFunc = t.Log
		trackerA.ctx = &testAsyncCtx{group: "A", refCount: 0}
		r.SetAsyncContextTracker(&trackerA)

		var captured []func()
		r.SetPromiseJobEnqueuer(func(job func()) {
			captured = append(captured, job)
		})

		_, err := r.RunString(`Promise.resolve().then(function() {})`)
		if err != nil {
			t.Fatal(err)
		}
		if len(captured) != 1 {
			t.Fatalf("expected 1 captured job, got %d", len(captured))
		}

		// Replace with tracker B, poisoned so Resumed panics if used.
		var trackerB testAsyncContextTracker
		trackerB.logFunc = t.Log
		trackerB.ctx = &testAsyncCtx{group: "B", refCount: 0}
		trackerB.resumed = true
		r.SetAsyncContextTracker(&trackerB)

		// Should use trackerA (captured at scheduling time), not trackerB.
		err = r.RunPromiseJob(captured[0])
		if err != nil {
			t.Fatalf("RunPromiseJob: %v", err)
		}

		if _, err := r.RunString(`1 + 1`); err != nil {
			t.Fatalf("runtime not reusable: %v", err)
		}
	})

	// Hook replacement during same-batch forwarding: job A replaces
	// the hook, job B must go to the new hook, not the stale one.
	t.Run("hook-replacement-during-forwarding", func(t *testing.T) {
		r := New()
		var order []int
		var hookACount, hookBCount int
		r.Set("push", func(v int) { order = append(order, v) })
		r.Set("installHookA", func() {
			r.SetPromiseJobEnqueuer(func(job func()) {
				hookACount++
				r.RunPromiseJob(job)
			})
		})
		r.Set("replaceWithHookB", func() {
			r.SetPromiseJobEnqueuer(func(job func()) {
				hookBCount++
				r.RunPromiseJob(job)
			})
		})
		_, err := r.RunString(`
			Promise.resolve().then(function() { replaceWithHookB(); push(1); });
			Promise.resolve().then(function() { push(2); });
			installHookA();
		`)
		if err != nil {
			t.Fatal(err)
		}
		if hookACount != 1 {
			t.Fatalf("expected hookACount=1, got %d", hookACount)
		}
		if hookBCount != 1 {
			t.Fatalf("expected hookBCount=1, got %d", hookBCount)
		}
		if len(order) != 2 || order[0] != 1 || order[1] != 2 {
			t.Fatalf("expected order=[1 2], got %v", order)
		}
		if _, err := r.RunString(`1 + 1`); err != nil {
			t.Fatalf("runtime not reusable: %v", err)
		}
	})

	// leaveAbrupt must clear vm.prg, vm.sb, vm.stack, and r.jobQueue.
	t.Run("leaveabrupt-clears-prg-sb", func(t *testing.T) {
		r := New()
		r.Set("boom", func() { r.Interrupt("test-abrupt") })
		_, err := r.RunString(`
			var x = 1 + 2;
			boom();
		`)
		if err == nil {
			t.Fatal("expected InterruptedError, got nil")
		}
		if r.vm.prg != nil {
			t.Fatalf("vm.prg should be nil after leaveAbrupt, got %v", r.vm.prg)
		}
		if r.vm.sb != -1 {
			t.Fatalf("vm.sb should be -1 after leaveAbrupt, got %d", r.vm.sb)
		}
		if r.vm.stack != nil {
			t.Fatal("vm.stack should be nil after leaveAbrupt")
		}
		if r.jobQueue != nil {
			t.Fatalf("r.jobQueue should be nil after leaveAbrupt, got len=%d", len(r.jobQueue))
		}
		if _, err := r.RunString(`1 + 1`); err != nil {
			t.Fatalf("runtime not reusable: %v", err)
		}
	})

	// If the hook panics during RunPromiseJob's forwarding loop, the
	// sentinel context must still be popped from r.vm.callStack.
	t.Run("hook-panic-forwarding-no-sentinel-leak", func(t *testing.T) {
		r := New()
		var captured []func()
		r.SetPromiseJobEnqueuer(func(job func()) {
			captured = append(captured, job)
		})
		sentinel := errors.New("hook panic during forwarding")
		r.Set("setup", func() {
			// Clear the hook so the reaction job goes to r.jobQueue.
			r.SetPromiseJobEnqueuer(nil)
			_, _ = r.RunString(`Promise.resolve().then(function() {})`)
			// Install a panicking hook for the forwarding loop to call.
			r.SetPromiseJobEnqueuer(func(job func()) { panic(sentinel) })
		})
		_, err := r.RunString(`Promise.resolve().then(setup)`)
		if err != nil {
			t.Fatal(err)
		}
		if len(captured) != 1 {
			t.Fatalf("expected 1 captured job, got %d", len(captured))
		}

		// Run the captured job: setup() enqueues a reaction to
		// r.jobQueue and installs the panicking hook. The forwarding
		// loop calls the hook, causing a panic.
		func() {
			defer func() {
				if x := recover(); x == nil {
					t.Fatal("expected RunPromiseJob to re-panic from hook panic during forwarding")
				} else if x != sentinel {
					t.Fatalf("expected re-panic with sentinel, got %v (%T)", x, x)
				}
			}()
			_ = r.RunPromiseJob(captured[0])
		}()

		// No sentinel leak.
		if len(r.vm.callStack) != 0 {
			t.Fatalf("callStack should be empty, got len=%d", len(r.vm.callStack))
		}
		// Runtime must remain reusable.
		if _, err := r.RunString(`1 + 1`); err != nil {
			t.Fatalf("runtime not reusable after hook panic: %v", err)
		}
	})

	// If Exited() panics, it must not be called a second time by the
	// defer safety net.
	t.Run("tracker-exited-panic-no-double-call", func(t *testing.T) {
		r := New()
		var tracker countingAsyncContextTracker
		exitedOnce := false
		tracker.exitFunc = func() {
			if !exitedOnce {
				exitedOnce = true
				panic("Exited() panic on first call")
			}
			panic("Exited() called twice")
		}
		r.SetAsyncContextTracker(&tracker)
		r.SetPromiseJobEnqueuer(func(job func()) {
			r.RunPromiseJob(job)
		})

		defer func() {
			if x := recover(); x == nil {
				t.Fatal("expected Exited() panic to propagate")
			}
			if tracker.exited != 1 {
				t.Fatalf("Exited() should be called once, got %d", tracker.exited)
			}
		}()

		_, err := r.RunString(`Promise.resolve().then(function() {})`)
		if err != nil {
			t.Logf("RunString returned err: %v", err)
		}
	})

	// Nil-handler propagation reactions must call Resumed and Exited.
	t.Run("nil-handler-tracker-invariant", func(t *testing.T) {
		r := New()
		var tracker countingAsyncContextTracker
		r.SetAsyncContextTracker(&tracker)

		// Promise.resolve(1).then() with no fulfill handler.
		_, err := r.RunString(`Promise.resolve(1).then()`)
		if err != nil {
			t.Fatal(err)
		}

		if tracker.grabbed != 1 {
			t.Fatalf("expected Grab=1, got %d", tracker.grabbed)
		}
		if tracker.resumed != 1 {
			t.Fatalf("expected Resumed=1, got %d", tracker.resumed)
		}
		if tracker.exited != 1 {
			t.Fatalf("expected Exited=1, got %d", tracker.exited)
		}
		if _, err := r.RunString(`1 + 1`); err != nil {
			t.Fatalf("runtime not reusable: %v", err)
		}
	})

	// A sync hook with a tracker: a reaction enqueued during the handler
	// must be buffered until Exited() returns, not delivered to the hook
	// while the tracker is still resumed.
	t.Run("sync-hook-tracker-handler-enqueued-reaction", func(t *testing.T) {
		r := New()
		var tracker testAsyncContextTracker
		tracker.logFunc = t.Log
		tracker.ctx = &testAsyncCtx{group: "g", refCount: 0}
		r.SetAsyncContextTracker(&tracker)

		var innerRan bool
		r.Set("markInner", func() { innerRan = true })
		r.SetPromiseJobEnqueuer(func(job func()) {
			if err := r.RunPromiseJob(job); err != nil {
				t.Fatalf("RunPromiseJob: %v", err)
			}
		})

		_, err := r.RunString(`
			Promise.resolve().then(function () {
				Promise.resolve().then(function () { markInner(); });
			});
		`)
		if err != nil {
			t.Fatalf("RunString: %v", err)
		}
		if !innerRan {
			t.Fatal("inner reaction did not run")
		}
		if _, err := r.RunString(`1 + 1`); err != nil {
			t.Fatalf("runtime not reusable: %v", err)
		}
	})

	t.Run("tracker-resumed-enqueue-buffers-until-resumed-returns", func(t *testing.T) {
		r := New()
		tracker := &resumedEnqueueTracker{r: r}
		r.SetAsyncContextTracker(tracker)

		var resumedEnqueuedJobRan bool
		r.Set("markResumedEnqueued", func() {
			resumedEnqueuedJobRan = true
		})
		r.SetPromiseJobEnqueuer(func(job func()) {
			if err := r.RunPromiseJob(job); err != nil {
				t.Fatalf("RunPromiseJob: %v", err)
			}
		})

		defer func() {
			if x := recover(); x != nil {
				t.Fatalf("expected Resumed() to enqueue promise work without nested Resumed; got panic: %v", x)
			}
		}()

		_, err := r.RunString(`Promise.resolve().then(function () {})`)
		if err != nil {
			t.Fatalf("RunString: %v", err)
		}
		if !tracker.enqueued {
			t.Fatal("tracker.Resumed() did not enqueue the nested promise job")
		}
		if !resumedEnqueuedJobRan {
			t.Fatal("promise job enqueued by Resumed() did not run after Resumed() returned")
		}
		if _, err := r.RunString(`1 + 1`); err != nil {
			t.Fatalf("runtime not reusable: %v", err)
		}
	})

	t.Run("tracker-resumed-enqueued-job-precedes-downstream-reaction", func(t *testing.T) {
		r := New()
		tracker := &resumedEnqueueTracker{r: r}
		r.SetAsyncContextTracker(tracker)

		var order []int
		r.Set("markResumedEnqueued", func() { order = append(order, 1) })
		r.Set("push", func(v int) { order = append(order, v) })
		r.SetPromiseJobEnqueuer(func(job func()) {
			if err := r.RunPromiseJob(job); err != nil {
				t.Fatalf("RunPromiseJob: %v", err)
			}
		})

		_, err := r.RunString(`Promise.resolve().then(function () {}).then(function () { push(2); })`)
		if err != nil {
			t.Fatalf("RunString: %v", err)
		}
		if len(order) != 2 || order[0] != 1 || order[1] != 2 {
			t.Fatalf("expected tracker-enqueued job to precede downstream reaction [1 2], got %v", order)
		}
	})

	t.Run("tracker-exited-enqueue-buffers-until-exited-returns", func(t *testing.T) {
		r := New()
		tracker := &exitedEnqueueTracker{r: r}
		r.SetAsyncContextTracker(tracker)

		var exitedEnqueuedJobRan bool
		r.Set("markExitedEnqueued", func() {
			exitedEnqueuedJobRan = true
		})
		r.SetPromiseJobEnqueuer(func(job func()) {
			if err := r.RunPromiseJob(job); err != nil {
				t.Fatalf("RunPromiseJob: %v", err)
			}
		})

		defer func() {
			if x := recover(); x != nil {
				t.Fatalf("expected Exited() to enqueue promise work without nested Resumed; got panic: %v", x)
			}
		}()

		_, err := r.RunString(`Promise.resolve().then(function () {})`)
		if err != nil {
			t.Fatalf("RunString: %v", err)
		}
		if !tracker.enqueued {
			t.Fatal("tracker.Exited() did not enqueue the nested promise job")
		}
		if !exitedEnqueuedJobRan {
			t.Fatal("promise job enqueued by Exited() did not run after Exited() returned")
		}
		if _, err := r.RunString(`1 + 1`); err != nil {
			t.Fatalf("runtime not reusable: %v", err)
		}
	})

	t.Run("tracker-exited-enqueued-job-precedes-downstream-reaction", func(t *testing.T) {
		r := New()
		tracker := &exitedEnqueueTracker{r: r}
		r.SetAsyncContextTracker(tracker)

		var order []int
		r.Set("markExitedEnqueued", func() { order = append(order, 1) })
		r.Set("push", func(v int) { order = append(order, v) })
		r.SetPromiseJobEnqueuer(func(job func()) {
			if err := r.RunPromiseJob(job); err != nil {
				t.Fatalf("RunPromiseJob: %v", err)
			}
		})

		_, err := r.RunString(`Promise.resolve().then(function () {}).then(function () { push(2); })`)
		if err != nil {
			t.Fatalf("RunString: %v", err)
		}
		if len(order) != 2 || order[0] != 1 || order[1] != 2 {
			t.Fatalf("expected tracker-enqueued job to precede downstream reaction [1 2], got %v", order)
		}
	})

	// A hook that panics during abrupt shutdown must not escape
	// leaveAbrupt as a raw panic; RunProgram still returns the
	// original InterruptedError and the runtime stays reusable.
	t.Run("leaveabrupt-hook-panic-state-consistency", func(t *testing.T) {
		r := New()
		sentinel := errors.New("hook bug")
		r.Set("mark", func() {})
		r.Set("installPanickingHookAndInterrupt", func() {
			r.SetPromiseJobEnqueuer(func(job func()) { panic(sentinel) })
			r.Interrupt("trigger-abrupt")
		})

		_, err := r.RunString(`
			Promise.resolve().then(function() { mark(); });
			installPanickingHookAndInterrupt();
		`)
		if err == nil {
			t.Fatal("expected InterruptedError from RunString, got nil")
		}
		if _, ok := err.(*InterruptedError); !ok {
			t.Fatalf("expected *InterruptedError, got %T: %v", err, err)
		}

		if r.vm.stack != nil {
			t.Fatal("vm.stack should be nil after leaveAbrupt hook panic")
		}
		if r.vm.prg != nil {
			t.Fatal("vm.prg should be nil after leaveAbrupt hook panic")
		}
		if r.vm.sb != -1 {
			t.Fatalf("vm.sb should be -1, got %d", r.vm.sb)
		}
		if _, err := r.RunString(`1 + 1`); err != nil {
			t.Fatalf("runtime not reusable after hook panic: %v", err)
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
