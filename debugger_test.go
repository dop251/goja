package goja

import (
	"testing"

	"github.com/dop251/goja/parser"
)

func TestDebuggerSimpleCaseWhereExecAndPrintDontWork(t *testing.T) {
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
		b, c := debugger.WaitToActivate()
		t.Logf("%d\n", debugger.Line())
		if b != DebuggerStatementActivation {
			t.Fatalf("Wrong activation %s", b)
		}
		if v, err := debugger.Exec("a = false"); err != nil {
			t.Fatalf("error while executing %s", err)
		} else if v.ToBoolean() { // TODO this is wrong it should be false, but it doesn't work
			t.Fatalf("wrong returned value %+v", v)
		}

		if v, err := debugger.Print("a"); err != nil { // this should work and return false ... but it doesn't
			t.Fatalf(" error while executing %s", err)
		} else if v == "true" { // TODO this is wrong it should be false, but it doesn't work
			t.Fatalf("wrong returned value %+v", v)
		}
		c()
	}()
	testScript1WithRuntime(SCRIPT, valueFalse, t, r) // TODO: this should be valueFalse, but it doesn't work
	<-ch                                             // wait for the debugger
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
		if b != ProgramStartActivation {
			// program should stop at program start activation
			t.Fatalf("Wrong activation: %s", b)
		}
		if debugger.PC() == 3 && debugger.Line() != 1 {
			// debugger should wait on the debugger statement and continue from there
			t.Fatalf("Wrong line and vm.pc, PC: %d, Line: %d", debugger.PC(), debugger.Line())
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
	vm.run()
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
