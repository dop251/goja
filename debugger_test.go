package goja

import (
	"testing"

	"github.com/dop251/goja/parser"
)

func TestDebuggerBreakpoint(t *testing.T) {
	const SCRIPT = `
	x = 1;
	y = 2;
	z = 3;
	f = 4;
	`

	r := &Runtime{}
	r.init()
	debugger := r.EnableDebugMode()

	if err := debugger.SetBreakpoint("test.js", 3); err != nil {
		t.Fatal(err)
	} else {
		t.Log("Set breakpoint on line 3")
	}
	if err := debugger.SetBreakpoint("test.js", 4); err != nil {
		t.Fatal(err)
	} else {
		t.Log("Set breakpoint on line 4")
	}
	if err := debugger.SetBreakpoint("test.js", 5); err != nil {
		t.Fatal(err)
	} else {
		t.Log("Set breakpoint on line 5")
	}

	ch := make(chan struct{})
	go func() {
		defer close(ch)
		defer func() {
			if t.Failed() {
				r.Interrupt("failed test")
			}
		}()

		reason, resume := debugger.WaitToActivate()
		if reason != BreakpointActivation {
			t.Fatalf("wrong activation %s", reason)
		} else {
			t.Logf("hit first breakpoint on line %d", debugger.Line())
		}

		if debugger.Line() != 3 {
			t.Fatalf("wrong line: %d", debugger.Line())
		}
		resume()

		if err := debugger.ClearBreakpoint("test.js", 5); err != nil {
			t.Fatal("cannot clear breakpoint on line 5")
		} else {
			t.Log("cleared breakpoint on line 5")
		}

		if breakpoints, err := debugger.Breakpoints(); err != nil {
			t.Fatalf("error while executing %s", err)
		} else {
			t.Logf("breakpoints are now set on lines: %v", breakpoints["test.js"])
		}

		// Go to next, so that the breakpointed line is executed
		debugger.Next()

		reason, resume = debugger.WaitToActivate()
		if reason != BreakpointActivation {
			t.Fatalf("wrong activation %s", reason)
		} else {
			t.Logf("hit second breakpoint on line %d", debugger.Line())
		}

		if debugger.Line() != 4 {
			t.Fatalf("wrong line: %d", debugger.Line())
		}
		resume()

		// Go to next, so that the breakpointed line is executed
		// This line acts as continue, since there are no blockers in the way (no breakpoints)
		debugger.Next()
	}()
	testScript1WithRuntime(SCRIPT, intToValue(4), t, r)
	<-ch // wait for the debugger
}

func TestDebuggerNext(t *testing.T) {
	const SCRIPT = `debugger
	x = 1;
	y = 2;
	z = 3;
	`
	r := &Runtime{}
	r.init()
	debugger := r.EnableDebugMode()

	ch := make(chan struct{})
	go func() {
		defer close(ch)
		defer func() {
			if t.Failed() {
				r.Interrupt("failed test")
			}
		}()
		reason, resume := debugger.WaitToActivate()
		t.Logf("%d\n", debugger.Line())
		if reason != DebuggerStatementActivation {
			t.Fatalf("wrong activation %s", reason)
		}

		if err := debugger.Next(); err != nil {
			t.Fatalf("error while executing %s", err)
		}
		if debugger.PC() != 4 && debugger.Line() != 3 {
			t.Fatalf("wrong line and vm.pc, PC: %d, Line: %d", debugger.PC(), debugger.Line())
		} else {
			src, _ := debugger.List()
			t.Logf("Go to line 3: > %s\n", src[debugger.Line()-1])
		}

		if err := debugger.Next(); err != nil {
			t.Fatalf("error while executing %s", err)
		}
		if debugger.PC() != 6 && debugger.Line() != 4 {
			t.Fatalf("wrong line and vm.pc, PC: %d, Line: %d", debugger.PC(), debugger.Line())
		} else {
			src, _ := debugger.List()
			t.Logf("Go to line 4: > %s\n", src[debugger.Line()-1])
		}
		resume()
	}()
	testScript1WithRuntime(SCRIPT, intToValue(3), t, r)
	<-ch // wait for the debugger
}

func TestDebuggerContinue(t *testing.T) {
	const SCRIPT = `debugger
	x = 1;
	y = 2;
	z = 3;
	debugger;
	f = 4;
	`
	r := &Runtime{}
	r.init()
	debugger := r.EnableDebugMode()

	ch := make(chan struct{})
	go func() {
		defer close(ch)
		defer func() {
			if t.Failed() {
				r.Interrupt("failed test")
			}
		}()
		reason, resume := debugger.WaitToActivate()
		t.Logf("%d\n", debugger.Line())
		if reason != DebuggerStatementActivation {
			t.Fatalf("wrong activation %s", reason)
		} else {
			t.Log("Hit first debugger statement")
		}
		resume()
		reason, resume = debugger.WaitToActivate()
		if reason != DebuggerStatementActivation {
			t.Fatalf("wrong activation %s", reason)
		} else {
			t.Log("Hit second debugger statement")
		}

		if debugger.PC() != 7 && debugger.Line() != 6 {
			t.Fatalf("wrong line and vm.pc, PC: %d, Line: %d", debugger.PC(), debugger.Line())
		} else {
			src, _ := debugger.List()
			t.Logf("Continue to line 6: > %s\n", src[debugger.Line()-1])
		}
		resume()
	}()
	testScript1WithRuntime(SCRIPT, intToValue(4), t, r)
	<-ch // wait for the debugger
}

func TestDebuggerStepIn(t *testing.T) {
	const SCRIPT = `debugger
	function test() {
		a = 1 + 2;
		return a
	}
	test()
	`
	r := &Runtime{}
	r.init()
	debugger := r.EnableDebugMode()

	ch := make(chan struct{})
	go func() {
		defer close(ch)
		defer func() {
			if t.Failed() {
				r.Interrupt("failed test")
			}
		}()
		reason, resume := debugger.WaitToActivate()
		t.Logf("%d\n", debugger.Line())
		if reason != DebuggerStatementActivation {
			t.Fatalf("wrong activation %s", reason)
		}

		if err := debugger.StepIn(); err != nil {
			t.Fatalf("error while executing %s", err)
		}
		if debugger.PC() != 4 && debugger.Line() != 6 {
			t.Fatalf("wrong line and vm.pc, PC: %d, Line: %d", debugger.PC(), debugger.Line())
		} else {
			src, _ := debugger.List()
			t.Logf("Step-in to line 6: > %s\n", src[debugger.Line()-1])
		}

		if err := debugger.StepIn(); err != nil {
			t.Fatalf("error while executing %s", err)
		}
		// Running inside a function returns scoped vm.pc and line number (everything's reset)
		if debugger.PC() != 0 && debugger.Line() != 2 {
			t.Fatalf("wrong line and vm.pc, PC: %d, Line: %d", debugger.PC(), debugger.Line())
		} else {
			src, _ := debugger.List()
			t.Logf("Step-in to line 2 (line 1 of function): > %s\n", src[debugger.Line()])
		}
		resume()
	}()
	testScript1WithRuntime(SCRIPT, intToValue(3), t, r)
	<-ch // wait for the debugger
}

func TestDebuggerExecAndPrint(t *testing.T) {
	const SCRIPT = `
	function test() {
		var a = true;
		debugger;
		return a;
	}
	test()
	`
	r := &Runtime{}
	r.init()
	debugger := r.EnableDebugMode()

	ch := make(chan struct{})
	go func() {
		defer close(ch)
		defer func() {
			if t.Failed() {
				r.Interrupt("failed test")
			}
		}()
		reason, resume := debugger.WaitToActivate()
		t.Logf("%d\n", debugger.Line())
		if reason != DebuggerStatementActivation {
			t.Fatalf("wrong activation %s", reason)
		}
		if v, err := debugger.Exec("a = false"); err != nil {
			t.Fatalf("error while executing %s", err)
		} else if v.ToBoolean() {
			t.Fatalf("wrong returned value %+v", v)
		} else {
			t.Logf("SET a = %s", v)
		}

		if v, err := debugger.Print("a"); err != nil {
			t.Fatalf(" error while executing %s", err)
		} else if v == "true" {
			t.Fatalf("wrong returned value %+v", v)
		} else {
			t.Logf("GET a == %s", v)
		}
		resume()
	}()
	testScript1WithRuntime(SCRIPT, valueFalse, t, r)
	<-ch // wait for the debugger
}

func TestDebuggerList(t *testing.T) {
	const SCRIPT = `debugger
	x = 1;
	`
	r := &Runtime{}
	r.init()
	debugger := r.EnableDebugMode()

	ch := make(chan struct{})
	go func() {
		defer close(ch)
		defer func() {
			if t.Failed() {
				r.Interrupt("failed test")
			}
		}()
		reason, resume := debugger.WaitToActivate()
		t.Logf("%d\n", debugger.Line())
		if reason != DebuggerStatementActivation {
			t.Fatalf("wrong activation %s", reason)
		}

		if err := debugger.Next(); err != nil {
			t.Fatalf("error while executing %s", err)
		}
		if src, err := debugger.List(); err != nil || src[debugger.Line()-1] != "	x = 1;" {
			t.Fatalf("error while executing %s", err)
		} else {
			t.Logf("Current line (%d) contains %s", debugger.Line(), src[debugger.Line()-1])
		}
		resume()
	}()
	testScript1WithRuntime(SCRIPT, intToValue(1), t, r)
	<-ch // wait for the debugger
}

func TestDebuggerSimpleCaseWhereLineIsIncorrectlyReported(t *testing.T) {
	t.Skip() // this is blocking forever
	const SCRIPT = `debugger;
	function test() {
		var a = true;
		debugger;
		return a;
	}
	test()
	`
	r := &Runtime{}
	r.init()
	debugger := r.EnableDebugMode()

	ch := make(chan struct{})
	go func() {
		defer close(ch)
		defer func() {
			if t.Failed() {
				r.Interrupt("failed test")
			}
		}()
		b, c := debugger.WaitToActivate()
		t.Logf("PC: %d, Line: %d", debugger.PC(), debugger.Line())
		if b != DebuggerStatementActivation {
			t.Fatalf("wrong activation: %s", b)
		}
		if debugger.PC() != 2 && debugger.Line() != 1 {
			// debugger should wait on the debugger statement and continue from there
			// yet it executes the debugger statement, which increases program counter (vm.pc) by 1,
			// which causes the debugger to stop at the next executable line
			t.Fatalf("wrong line and vm.pc, PC: %d, Line: %d", debugger.PC(), debugger.Line())
		}
		c()
	}()
	testScript1WithRuntime(SCRIPT, valueTrue, t, r)
	<-ch // wait for the debugger
}

func testScript1WithRuntime(script string, expectedResult Value, t *testing.T, r *Runtime) {
	prg, err := parser.ParseFile(nil, "test.js", script, 0)
	if err != nil {
		t.Fatal(err)
	}

	c := newCompiler(true) // TODO have it as a parameter?
	c.compile(prg, false, false, true)

	vm := r.vm
	vm.prg = c.p
	vm.prg.dumpCode(t.Logf)
	vm.result = _undefined
	vm.debug()
	v := vm.result
	t.Logf("stack size: %d", len(vm.stack))
	t.Logf("stashAllocs: %d", vm.stashAllocs)

	if v == nil && expectedResult != nil || !v.SameAs(expectedResult) {
		t.Fatalf("Result: %+v, expected: %+v", v, expectedResult)
	}

	if vm.sp != 0 {
		t.Fatalf("sp: %d", vm.sp)
	}

	if l := len(vm.iterStack); l > 0 {
		t.Fatalf("iter stack is not empty: %d", l)
	}
}
