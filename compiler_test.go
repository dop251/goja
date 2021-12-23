package goja

import (
	"io/ioutil"
	"sync"
	"testing"
)

const TESTLIB = `
function $ERROR(message) {
	throw new Error(message);
}

function Test262Error() {
}

function assert(mustBeTrue, message) {
    if (mustBeTrue === true) {
        return;
    }

    if (message === undefined) {
        message = 'Expected true but got ' + String(mustBeTrue);
    }
    $ERROR(message);
}

assert._isSameValue = function (a, b) {
    if (a === b) {
        // Handle +/-0 vs. -/+0
        return a !== 0 || 1 / a === 1 / b;
    }

    // Handle NaN vs. NaN
    return a !== a && b !== b;
};

assert.sameValue = function (actual, expected, message) {
    if (assert._isSameValue(actual, expected)) {
        return;
    }

    if (message === undefined) {
        message = '';
    } else {
        message += ' ';
    }

    message += 'Expected SameValue(«' + String(actual) + '», «' + String(expected) + '») to be true';

    $ERROR(message);
};

assert.throws = function (expectedErrorConstructor, func, message) {
  if (typeof func !== "function") {
    $ERROR('assert.throws requires two arguments: the error constructor ' +
      'and a function to run');
    return;
  }
  if (message === undefined) {
    message = '';
  } else {
    message += ' ';
  }

  try {
    func();
  } catch (thrown) {
    if (typeof thrown !== 'object' || thrown === null) {
      message += 'Thrown value was not an object!';
      $ERROR(message);
    } else if (thrown.constructor !== expectedErrorConstructor) {
      message += 'Expected a ' + expectedErrorConstructor.name + ' but got a ' + thrown.constructor.name;
      $ERROR(message);
    }
    return;
  }

  message += 'Expected a ' + expectedErrorConstructor.name + ' to be thrown but no exception was thrown at all';
  $ERROR(message);
};

function compareArray(a, b) {
  if (b.length !== a.length) {
    return false;
  }

  for (var i = 0; i < a.length; i++) {
    if (b[i] !== a[i]) {
      return false;
    }
  }
  return true;
}
`

const TESTLIBX = `
	function looksNative(fn) {
		return /native code/.test(Function.prototype.toString.call(fn));
	}

	function deepEqual(a, b) {
		if (typeof a === "object") {
			if (typeof b === "object") {
				if (a === b) {
					return true;
				}
				if (Reflect.getPrototypeOf(a) !== Reflect.getPrototypeOf(b)) {
					return false;
				}
				var keysA = Object.keys(a);
				var keysB = Object.keys(b);
				if (keysA.length !== keysB.length) {
					return false;
				}
				if (!compareArray(keysA.sort(), keysB.sort())) {
					return false;
				}
				for (var i = 0; i < keysA.length; i++) {
					var key = keysA[i];
					if (!deepEqual(a[key], b[key])) {
						return false;
					}
				}
				return true;
			} else {
				return false;
			}
		}
		return assert._isSameValue(a, b);
	}
`

var (
	// The reason it's implemented this way rather than just as _testLib = MustCompile(...)
	// is because when you try to debug the compiler and set a breakpoint it gets triggered during the
	// initialisation which is annoying.
	_testLib, _testLibX       *Program
	testLibOnce, testLibXOnce sync.Once
)

func testLib() *Program {
	testLibOnce.Do(func() {
		_testLib = MustCompile("testlib.js", TESTLIB, false)
	})
	return _testLib
}

func testLibX() *Program {
	testLibXOnce.Do(func() {
		_testLibX = MustCompile("testlibx.js", TESTLIBX, false)
	})
	return _testLibX
}

func (r *Runtime) testPrg(p *Program, expectedResult Value, t *testing.T) {
	vm := r.vm
	vm.prg = p
	vm.pc = 0
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

func (r *Runtime) testScriptWithTestLib(script string, expectedResult Value, t *testing.T) {
	_, err := r.RunProgram(testLib())
	if err != nil {
		t.Fatal(err)
	}

	r.testScript(script, expectedResult, t)
}

func (r *Runtime) testScriptWithTestLibX(script string, expectedResult Value, t *testing.T) {
	_, err := r.RunProgram(testLib())
	if err != nil {
		t.Fatal(err)
	}

	_, err = r.RunProgram(testLibX())
	if err != nil {
		t.Fatal(err)
	}

	r.testScript(script, expectedResult, t)
}

func (r *Runtime) testScript(script string, expectedResult Value, t *testing.T) {
	r.testPrg(MustCompile("test.js", script, false), expectedResult, t)
}

func testScript(script string, expectedResult Value, t *testing.T) {
	New().testScript(script, expectedResult, t)
}

func testScriptWithTestLib(script string, expectedResult Value, t *testing.T) {
	New().testScriptWithTestLib(script, expectedResult, t)
}

func testScriptWithTestLibX(script string, expectedResult Value, t *testing.T) {
	New().testScriptWithTestLibX(script, expectedResult, t)
}

func TestEmptyProgram(t *testing.T) {
	const SCRIPT = `
	`

	testScript(SCRIPT, _undefined, t)
}

func TestResultEmptyBlock(t *testing.T) {
	const SCRIPT = `
	undefined;
	{}
	`
	testScript(SCRIPT, _undefined, t)
}

func TestResultVarDecl(t *testing.T) {
	const SCRIPT = `
	7; var x = 1;
	`
	testScript(SCRIPT, valueInt(7), t)
}

func TestResultLexDecl(t *testing.T) {
	const SCRIPT = `
	7; {let x = 1};
	`
	testScript(SCRIPT, valueInt(7), t)
}

func TestResultLexDeclBreak(t *testing.T) {
	const SCRIPT = `
	L:{ 7; {let x = 1; break L;}};
	`
	testScript(SCRIPT, valueInt(7), t)
}

func TestResultLexDeclNested(t *testing.T) {
	const SCRIPT = `
	7; {let x = (function() { return eval("8; {let y = 9}")})()};
	`
	testScript(SCRIPT, valueInt(7), t)
}

func TestErrorProto(t *testing.T) {
	const SCRIPT = `
	var e = new TypeError();
	e.name;
	`

	testScript(SCRIPT, asciiString("TypeError"), t)
}

func TestThis1(t *testing.T) {
	const SCRIPT = `
	function independent() {
		return this.prop;
	}
	var o = {};
	o.b = {g: independent, prop: 42};

	o.b.g();
	`
	testScript(SCRIPT, intToValue(42), t)
}

func TestThis2(t *testing.T) {
	const SCRIPT = `
var o = {
  prop: 37,
  f: function() {
    return this.prop;
  }
};

o.f();
`

	testScript(SCRIPT, intToValue(37), t)
}

func TestThisStrict(t *testing.T) {
	const SCRIPT = `
	"use strict";

	Object.defineProperty(Object.prototype, "x", { get: function () { return this; } });

	(5).x === 5;
	`

	testScript(SCRIPT, valueTrue, t)
}

func TestThisNoStrict(t *testing.T) {
	const SCRIPT = `
	Object.defineProperty(Object.prototype, "x", { get: function () { return this; } });

	(5).x == 5;
	`

	testScript(SCRIPT, valueTrue, t)
}

func TestNestedFuncVarResolution(t *testing.T) {
	const SCRIPT = `
	(function outer() {
		var v = 42;
		function inner() {
			return v;
		}
		return inner();
	})();
`
	testScript(SCRIPT, valueInt(42), t)
}

func TestNestedFuncVarResolution1(t *testing.T) {
	const SCRIPT = `
	function outer(argOuter) {
		var called = 0;
	  var inner = function(argInner) {
		if (arguments.length !== 1) {
			throw new Error();
		}
		called++;
		if (argOuter !== 1) {
			throw new Error("argOuter");
		}
		if (argInner !== 2) {
			throw new Error("argInner");
		}
	  };
		inner(2);
	}
	outer(1);
	`
	testScript(SCRIPT, _undefined, t)
}

func TestCallFewerArgs(t *testing.T) {
	const SCRIPT = `
function A(a, b, c) {
	return String(a) + " " + String(b) + " " + String(c);
}

A(1, 2);
`
	testScript(SCRIPT, asciiString("1 2 undefined"), t)
}

func TestCallFewerArgsClosureNoArgs(t *testing.T) {
	const SCRIPT = `
	var x;
	function A(a, b, c) {
		var y = a;
		x = function() { return " " + y };
		return String(a) + " " + String(b) + " " + String(c);
	}

	A(1, 2) + x();
`
	testScript(SCRIPT, asciiString("1 2 undefined 1"), t)
}

func TestCallFewerArgsClosureArgs(t *testing.T) {
	const SCRIPT = `
	var x;
	function A(a, b, c) {
		var y = b;
		x = function() { return " " + a + " " + y };
		return String(a) + " " + String(b) + " " + String(c);
	}

	A(1, 2) + x();
`
	testScript(SCRIPT, asciiString("1 2 undefined 1 2"), t)
}

func TestCallMoreArgs(t *testing.T) {
	const SCRIPT = `
function A(a, b) {
	var c = 4;
	return a - b + c;
}

A(1, 2, 3);
`
	testScript(SCRIPT, intToValue(3), t)
}

func TestCallMoreArgsDynamic(t *testing.T) {
	const SCRIPT = `
function A(a, b) {
	var c = 4;
	if (false) {
		eval("");
	}
	return a - b + c;
}

A(1, 2, 3);
`
	testScript(SCRIPT, intToValue(3), t)
}

func TestCallLessArgsDynamic(t *testing.T) {
	const SCRIPT = `
function A(a, b, c) {
	// Make it stashful
	function B() {
		return a;
	}
	return String(a) + " " + String(b) + " " + String(c);
}

A(1, 2);
`
	testScript(SCRIPT, asciiString("1 2 undefined"), t)
}

func TestCallLessArgsDynamicLocalVar(t *testing.T) {
	const SCRIPT = `
	function f(param) {
		var a = 42;
		if (false) {
			eval("");
		}
		return a;
	}
	f();
`

	testScript(SCRIPT, intToValue(42), t)
}

/*
func TestFib(t *testing.T) {
	testScript(TEST_FIB, valueInt(9227465), t)
}
*/

func TestNativeCall(t *testing.T) {
	const SCRIPT = `
	var o = Object(1);
	Object.defineProperty(o, "test", {value: 42});
	o.test;
	`
	testScript(SCRIPT, intToValue(42), t)
}

func TestJSCall(t *testing.T) {
	const SCRIPT = `
	function getter() {
		return this.x;
	}
	var o = Object(1);
	o.x = 42;
	Object.defineProperty(o, "test", {get: getter});
	o.test;
	`
	testScript(SCRIPT, intToValue(42), t)

}

func TestLoop1(t *testing.T) {
	const SCRIPT = `
	function A() {
    		var x = 1;
    		for (var i = 0; i < 1; i++) {
        		var x = 2;
    		}
    		return x;
	}

	A();
	`
	testScript(SCRIPT, intToValue(2), t)
}

func TestLoopBreak(t *testing.T) {
	const SCRIPT = `
	function A() {
    		var x = 1;
    		for (var i = 0; i < 1; i++) {
        		break;
        		var x = 2;
    		}
    		return x;
	}

	A();
	`
	testScript(SCRIPT, intToValue(1), t)
}

func TestForLoopOptionalExpr(t *testing.T) {
	const SCRIPT = `
	function A() {
    		var x = 1;
    		for (;;) {
        		break;
        		var x = 2;
    		}
    		return x;
	}

	A();
	`
	testScript(SCRIPT, intToValue(1), t)
}

func TestBlockBreak(t *testing.T) {
	const SCRIPT = `
	var rv = 0;
	B1: {
		rv = 1;
		B2: {
			rv = 2;
			break B1;
		}
		rv = 3;
	}
	rv;
	`
	testScript(SCRIPT, intToValue(2), t)

}

func TestTry(t *testing.T) {
	const SCRIPT = `
	function A() {
		var x = 1;
		try {
			x = 2;
		} catch(e) {
			x = 3;
		} finally {
			x = 4;
		}
		return x;
	}

	A();
	`
	testScript(SCRIPT, intToValue(4), t)
}

func TestTryOptionalCatchBinding(t *testing.T) {
	const SCRIPT = `
	try {
		throw null;
	} catch {
	}
	`
	testScript(SCRIPT, _undefined, t)
}

func TestTryCatch(t *testing.T) {
	const SCRIPT = `
	function A() {
		var x;
		try {
			throw 4;
		} catch(e) {
			x = e;
		}
		return x;
	}

	A();
	`
	testScript(SCRIPT, intToValue(4), t)
}

func TestTryCatchDirectEval(t *testing.T) {
	const SCRIPT = `
	function A() {
		var x;
		try {
			throw 4;
		} catch(e) {
			eval("x = e");
		}
		return x;
	}

	A();
	`
	testScript(SCRIPT, intToValue(4), t)
}

func TestTryExceptionInCatch(t *testing.T) {
	const SCRIPT = `
	function A() {
		var x;
		try {
			throw 4;
		} catch(e) {
			throw 5;
		}
		return x;
	}

	var rv;
	try {
		A();
	} catch (e) {
		rv = e;
	}
	rv;
	`
	testScript(SCRIPT, intToValue(5), t)
}

func TestTryContinueInCatch(t *testing.T) {
	const SCRIPT = `
	var c3 = 0, fin3 = 0;
	while (c3 < 2) {
		try {
			throw "ex1";
		} catch(er1) {
			c3 += 1;
			continue;
		} finally {
			fin3 = 1;
		}
		fin3 = 0;
	}

	fin3;
	`
	testScript(SCRIPT, intToValue(1), t)
}

func TestContinueInWith(t *testing.T) {
	const SCRIPT = `
	var x;
	var o = {x: 0};
	for (var i = 0; i < 2; i++) {
		with(o) {
			x = i;
			if (i === 0) {
				continue;
			}
		}
		break;
	}
	x;
	`
	testScript(SCRIPT, _undefined, t)
}

func TestTryContinueInFinally(t *testing.T) {
	const SCRIPT = `
	var c3 = 0, fin3 = 0;
	while (c3 < 2) {
  		try {
    			throw "ex1";
  		} catch(er1) {
    			c3 += 1;
  		} finally {
    			fin3 = 1;
    			continue;
  		}
  		fin3 = 0;
	}

	fin3;
	`
	testScript(SCRIPT, intToValue(1), t)
}

func TestTryBreakFinallyContinue(t *testing.T) {
	const SCRIPT = `
	for (var i = 0; i < 3; i++) {
	  try {
		break;
	  } finally {
		continue;
	  }
	}
	`
	testScript(SCRIPT, _undefined, t)
}

func TestTryBreakFinallyContinueWithResult(t *testing.T) {
	const SCRIPT = `
	for (var i = 0; i < 3; i++) {
	  try {
		true;
		break;
	  } finally {
		continue;
	  }
	}
	`
	testScript(SCRIPT, _undefined, t)
}

func TestTryBreakFinallyContinueWithResult1(t *testing.T) {
	const SCRIPT = `
	for (var i = 0; i < 3; i++) {
	  try {
		true;
		break;
	  } finally {
		var x = 1;
		continue;
	  }
	}
	`
	testScript(SCRIPT, _undefined, t)
}

func TestTryBreakFinallyContinueWithResultNested(t *testing.T) {
	const SCRIPT = `
LOOP:
	for (var i = 0; i < 3; i++) {
	  try {
		if (true) {
			false; break;
		}
	  } finally {
		if (true) {
			true; continue;
		}
	  }
	}
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestTryBreakOuterFinallyContinue(t *testing.T) {
	const SCRIPT = `
	let iCount = 0, jCount = 0;
	OUTER: for (let i = 0; i < 1; i++) {
		iCount++;
		for (let j = 0; j < 2; j++) {
			jCount++;
			try {
				break OUTER;
			} finally {
				continue;
			}
		}
	}
	""+iCount+jCount;
	`
	testScript(SCRIPT, asciiString("12"), t)
}

func TestTryIllegalContinueWithFinallyOverride(t *testing.T) {
	const SCRIPT = `
	L: {
		while (Math.random() > 0.5) {
			try {
				continue L;
			} finally {
				break;
			}
		}
	}
	`
	_, err := Compile("", SCRIPT, false)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTryIllegalContinueWithFinallyOverrideNoLabel(t *testing.T) {
	const SCRIPT = `
	L: {
		try {
			continue;
		} finally {
			break L;
		}
	}
	`
	_, err := Compile("", SCRIPT, false)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTryIllegalContinueWithFinallyOverrideDummy(t *testing.T) {
	const SCRIPT = `
	L: {
		while (false) {
			try {
				continue L;
			} finally {
				break;
			}
		}
	}
	`
	_, err := Compile("", SCRIPT, false)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTryNoResult(t *testing.T) {
	const SCRIPT = `
	true;
    L:
    try {
        break L;
    } finally {
    }
	`
	testScript(SCRIPT, _undefined, t)
}

func TestCatchLexicalEnv(t *testing.T) {
	const SCRIPT = `
	function F() {
		try {
			throw 1;
		} catch (e) {
			var x = e;
		}
		return x;
	}

	F();
	`
	testScript(SCRIPT, intToValue(1), t)
}

func TestThrowType(t *testing.T) {
	const SCRIPT = `
	function Exception(message) {
		this.message = message;
	}


	function A() {
		try {
			throw new Exception("boo!");
		} catch(e) {
			return e;
		}
	}
	var thrown = A();
	thrown !== null && typeof thrown === "object" && thrown.constructor === Exception;
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestThrowConstructorName(t *testing.T) {
	const SCRIPT = `
	function Exception(message) {
		this.message = message;
	}


	function A() {
		try {
			throw new Exception("boo!");
		} catch(e) {
			return e;
		}
	}
	A().constructor.name;
	`

	testScript(SCRIPT, asciiString("Exception"), t)
}

func TestThrowNativeConstructorName(t *testing.T) {
	const SCRIPT = `


	function A() {
		try {
			throw new TypeError();
		} catch(e) {
			return e;
		}
	}
	A().constructor.name;
	`

	testScript(SCRIPT, asciiString("TypeError"), t)
}

func TestEmptyTryNoCatch(t *testing.T) {
	const SCRIPT = `
	var called = false;
	try {
	} finally {
		called = true;
	}
	called;
	`

	testScript(SCRIPT, valueTrue, t)
}

func TestTryReturnFromCatch(t *testing.T) {
	const SCRIPT = `
	function f(o) {
		var x = 42;

		function innerf(o) {
			try {
				throw o;
			} catch (e) {
				return x;
			}
		}

		return innerf(o);
	}
	f({});
	`

	testScript(SCRIPT, valueInt(42), t)
}

func TestTryCompletionResult(t *testing.T) {
	const SCRIPT = `
	99; do { -99; try { 39 } catch (e) { -1 } finally { break; -2 }; } while (false);
	`

	testScript(SCRIPT, _undefined, t)
}

func TestIfElse(t *testing.T) {
	const SCRIPT = `
	var rv;
	if (rv === undefined) {
		rv = "passed";
	} else {
		rv = "failed";
	}
	rv;
	`

	testScript(SCRIPT, asciiString("passed"), t)
}

func TestIfElseRetVal(t *testing.T) {
	const SCRIPT = `
	var x;
	if (x === undefined) {
		"passed";
	} else {
		"failed";
	}
	`

	testScript(SCRIPT, asciiString("passed"), t)
}

func TestWhileReturnValue(t *testing.T) {
	const SCRIPT = `
	var x = 0;
	while(true) {
		x = 1;
		break;
	}
	`
	testScript(SCRIPT, intToValue(1), t)
}

func TestIfElseLabel(t *testing.T) {
	const SCRIPT = `
	var x = 0;
	abc: if (true) {
		x = 1;
		break abc;
	}
	`
	testScript(SCRIPT, intToValue(1), t)
}

func TestIfMultipleLabels(t *testing.T) {
	const SCRIPT = `
	var x = 0;
	xyz:abc: if (true) {
		break xyz;
	}
	`
	testScript(SCRIPT, _undefined, t)
}

func TestBreakOutOfTry(t *testing.T) {
	const SCRIPT = `
	function A() {
		var x = 1;
		B: {
			try {
				x = 2;
			} catch(e) {
				x = 3;
			} finally {
				break B;
				x = 4;
			}
		}
		return x;
	}

	A();
	`
	testScript(SCRIPT, intToValue(2), t)
}

func TestReturnOutOfTryNested(t *testing.T) {
	const SCRIPT = `
	function A() {
		function nested() {
			try {
				return 1;
			} catch(e) {
				return 2;
			}
		}
		return nested();
	}

	A();
	`
	testScript(SCRIPT, intToValue(1), t)
}

func TestContinueLoop(t *testing.T) {
	const SCRIPT = `
	function A() {
		var r = 0;
		for (var i = 0; i < 5; i++) {
			if (i > 1) {
				continue;
			}
			r++;
		}
		return r;
	}

	A();
	`
	testScript(SCRIPT, intToValue(2), t)
}

func TestContinueOutOfTry(t *testing.T) {
	const SCRIPT = `
	function A() {
		var r = 0;
		for (var i = 0; i < 5; i++) {
			try {
				if (i > 1) {
					continue;
				}
			} catch(e) {
				return 99;
			}
			r++;
		}
		return r;
	}

	A();
	`
	testScript(SCRIPT, intToValue(2), t)
}

func TestThisInCatch(t *testing.T) {
	const SCRIPT = `
	function O() {
		try {
			f();
		} catch (e) {
			this.value = e.toString();
		}
	}

	function f() {
		throw "ex";
	}

	var o = new O();
	o.value;
	`
	testScript(SCRIPT, asciiString("ex"), t)
}

func TestNestedTry(t *testing.T) {
	const SCRIPT = `
	var ex;
	try {
  		throw "ex1";
	} catch (er1) {
  		try {
    			throw "ex2";
  		} catch (er1) {
			ex = er1;
		}
	}
	ex;
	`
	testScript(SCRIPT, asciiString("ex2"), t)
}

func TestNestedTryInStashlessFunc(t *testing.T) {
	const SCRIPT = `
	function f() {
		var ex1, ex2;
		try {
			throw "ex1";
		} catch (er1) {
			try {
				throw "ex2";
			} catch (er1) {
				ex2 = er1;
			}
			ex1 = er1;
		}
		return ex1 == "ex1" && ex2 == "ex2";
	}
	f();
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestEvalLexicalDecl(t *testing.T) {
	const SCRIPT = `
	eval("let x = true; x;");
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestEvalInCatchInStashlessFunc(t *testing.T) {
	const SCRIPT = `
	function f() {
		var ex;
		try {
			throw "ex1";
		} catch (er1) {
			eval("ex = er1");
		}
		return ex;
	}
	f();
	`
	testScript(SCRIPT, asciiString("ex1"), t)
}

func TestCatchClosureInStashlessFunc(t *testing.T) {
	const SCRIPT = `
	function f() {
		var ex;
		try {
			throw "ex1";
		} catch (er1) {
			return function() {
				return er1;
			}
		}
	}
	f()();
	`
	testScript(SCRIPT, asciiString("ex1"), t)
}

func TestCatchVarNotUsedInStashlessFunc(t *testing.T) {
	const SCRIPT = `
	function f() {
		var ex;
		try {
			throw "ex1";
		} catch (er1) {
			ex = "ok";
		}
		return ex;
	}
	f();
	`
	testScript(SCRIPT, asciiString("ok"), t)
}

func TestNew(t *testing.T) {
	const SCRIPT = `
	function O() {
		this.x = 42;
	}

	new O().x;
	`

	testScript(SCRIPT, intToValue(42), t)
}

func TestStringConstructor(t *testing.T) {
	const SCRIPT = `
	function F() {
		return String(33) + " " + String("cows");
	}

	F();
	`
	testScript(SCRIPT, asciiString("33 cows"), t)
}

func TestError(t *testing.T) {
	const SCRIPT = `
	function F() {
		return new Error("test");
	}

	var e = F();
	e.message == "test" && e.name == "Error";
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestTypeError(t *testing.T) {
	const SCRIPT = `
	function F() {
		return new TypeError("test");
	}

	var e = F();
	e.message == "test" && e.name == "TypeError";
	`

	testScript(SCRIPT, valueTrue, t)
}

func TestToString(t *testing.T) {
	const SCRIPT = `
	var o = {x: 42};
	o.toString = function() {
		return String(this.x);
	}

	var o1 = {};
	o.toString() + " ### " + o1.toString();
	`
	testScript(SCRIPT, asciiString("42 ### [object Object]"), t)
}

func TestEvalOrder(t *testing.T) {
	const SCRIPT = `
	var o = {f: function() {return 42}, x: 0};
	var trace = "";

	function F1() {
	    trace += "First!";
	    return o;
	}

	function F2() {
	    trace += "Second!";
	    return "f";
	}

	function F3() {
	    trace += "Third!";
	}

	var rv = F1()[F2()](F3());
	rv += trace;
	rv;
	`

	testScript(SCRIPT, asciiString("42First!Second!Third!"), t)
}

func TestPostfixIncBracket(t *testing.T) {
	const SCRIPT = `
	var o = {x: 42};
	var trace = "";

	function F1() {
	    trace += "First!";
	    return o;
	}

	function F2() {
	    trace += "Second!";
	    return "x";
	}


	var rv = F1()[F2()]++;
	rv + trace + o.x;
	`
	testScript(SCRIPT, asciiString("42First!Second!43"), t)
}

func TestPostfixIncDot(t *testing.T) {
	const SCRIPT = `
	var o = {x: 42};
	var trace = "";

	function F1() {
	    trace += "First!";
	    return o;
	}

	var rv = F1().x++;
	rv + trace + o.x;
	`
	testScript(SCRIPT, asciiString("42First!43"), t)
}

func TestPrefixIncBracket(t *testing.T) {
	const SCRIPT = `
	var o = {x: 42};
	var trace = "";

	function F1() {
	    trace += "First!";
	    return o;
	}

	function F2() {
	    trace += "Second!";
	    return "x";
	}


	var rv = ++F1()[F2()];
	rv + trace + o.x;
	`
	testScript(SCRIPT, asciiString("43First!Second!43"), t)
}

func TestPrefixIncDot(t *testing.T) {
	const SCRIPT = `
	var o = {x: 42};
	var trace = "";

	function F1() {
	    trace += "First!";
	    return o;
	}

	var rv = ++F1().x;
	rv + trace + o.x;
	`
	testScript(SCRIPT, asciiString("43First!43"), t)
}

func TestPostDecObj(t *testing.T) {
	const SCRIPT = `
	var object = {valueOf: function() {return 1}};
	var y = object--;
	var ok = false;
	if (y === 1) {
		ok = true;
	}
	ok;
	`

	testScript(SCRIPT, valueTrue, t)
}

func TestPropAcc1(t *testing.T) {
	const SCRIPT = `
	1..toString()
	`

	testScript(SCRIPT, asciiString("1"), t)
}

func TestEvalDirect(t *testing.T) {
	const SCRIPT = `
	var rv = false;
    function foo(){ rv = true; }

    var o = { };
    function f() {
        try {
	        eval("o.bar( foo() );");
		} catch (e) {
		}
    }
    f();
	rv;
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestEvalRet(t *testing.T) {
	const SCRIPT = `
	eval("for (var i = 0; i < 3; i++) {i}")
	`

	testScript(SCRIPT, valueInt(2), t)
}

func TestEvalFunctionDecl(t *testing.T) {
	const SCRIPT = `
	eval("function F() {}")
	`

	testScript(SCRIPT, _undefined, t)
}

func TestEvalFunctionExpr(t *testing.T) {
	const SCRIPT = `
	eval("(function F() {return 42;})")()
	`

	testScript(SCRIPT, intToValue(42), t)
}

func TestEvalDirectScope(t *testing.T) {
	const SCRIPT = `
	var __10_4_2_1_3 = "str";
	function testcase() {
		var __10_4_2_1_3 = "str1";
		try {
			throw "error";
		} catch (e) {
			var __10_4_2_1_3 = "str2";
			return eval("__10_4_2_1_3");
		}
	}
	testcase();
	`

	testScript(SCRIPT, asciiString("str2"), t)
}

func TestEvalDirectScope1(t *testing.T) {
	const SCRIPT = `
	'use strict';
	var __10_4_2_1_5 = "str";
	function testcase() {
				var __10_4_2_1_5 = "str1";
				var r = eval("\
							  var __10_4_2_1_5 = \'str2\'; \
							  eval(\"\'str2\' === __10_4_2_1_5\")\
							");
				return r;
		}
	testcase();
	`

	testScript(SCRIPT, valueTrue, t)
}

func TestEvalDirectCreateBinding(t *testing.T) {
	const SCRIPT = `
	function f() {
		eval("var x = true");
		return x;
	}
	var res = f();
	var thrown = false;
	try {
		x;
	} catch(e) {
		if (e instanceof ReferenceError) {
			thrown = true;
		} else {
			throw e;
		}
	}
	res && thrown;
	`

	testScript(SCRIPT, valueTrue, t)
}

func TestEvalDirectCreateBinding1(t *testing.T) {
	const SCRIPT = `
	function f() {
		eval("let x = 1; var y = 2; function f1() {return x};");
		assert.throws(ReferenceError, function() { x });
		return ""+y+f1();
	}
	f();
	`

	testScriptWithTestLib(SCRIPT, asciiString("21"), t)
}

func TestEvalDirectCreateBinding3(t *testing.T) {
	const SCRIPT = `
	function f() {
		let x;
		try {
			eval("var y=1, x=2");
		} catch(e) {}
		return y;
	}
	assert.throws(ReferenceError, f);
	`

	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestEvalGlobalStrict(t *testing.T) {
	const SCRIPT = `
	'use strict';
	var evalStr =
	'for (var x in this) {\n'+
	'  if ( x === \'Math\' ) {\n'+
	'  }\n'+
	'}\n';

	eval(evalStr);
	`

	testScript(SCRIPT, _undefined, t)
}

func TestEvalEmptyStrict(t *testing.T) {
	const SCRIPT = `
	'use strict';
	eval("");
	`

	testScript(SCRIPT, _undefined, t)
}

func TestEvalFuncDecl(t *testing.T) {
	const SCRIPT = `
	'use strict';
	var funcA = eval("function __funcA(__arg){return __arg;}; __funcA");
	typeof funcA;
	`

	testScript(SCRIPT, asciiString("function"), t)
}

func TestGetAfterSet(t *testing.T) {
	const SCRIPT = `
	function f() {
		var x = 1;
		return x;
	}
	`

	testScript(SCRIPT, _undefined, t)
}

func TestForLoopRet(t *testing.T) {
	const SCRIPT = `
	for (var i = 0; i < 20; i++) { if (i > 2) {break;} else { i }}
	`

	testScript(SCRIPT, _undefined, t)
}

func TestForLoopRet1(t *testing.T) {
	const SCRIPT = `
	for (var i = 0; i < 20; i++) { if (i > 2) {42;; {L:{break;}}} else { i }}
	`

	testScript(SCRIPT, intToValue(42), t)
}

func TestForInLoopRet(t *testing.T) {
	const SCRIPT = `
	var o = [1, 2, 3, 4];
	for (var i in o) { if (i > 2) {break;} else { i }}
	`

	testScript(SCRIPT, _undefined, t)
}

func TestForInLoopRet1(t *testing.T) {
	const SCRIPT = `
	var o = {};
	o.x = 1;
	o.y = 2;
	for (var i in o) {
		true;
	}

	`

	testScript(SCRIPT, valueTrue, t)
}

func TestDoWhileLoopRet(t *testing.T) {
	const SCRIPT = `
	var i = 0;
	do {
		if (i > 2) {
			break;
		} else {
			i;
		}
	} while (i++ < 20);
	`

	testScript(SCRIPT, _undefined, t)
}

func TestDoWhileContinueRet(t *testing.T) {
	const SCRIPT = `
	var i = 0;
	do {
		if (i > 2) {
			true;
			continue;
		} else {
			i;
		}
	} while (i++ < 20);
	`

	testScript(SCRIPT, valueTrue, t)
}

func TestWhileLoopRet(t *testing.T) {
	const SCRIPT = `
	var i; while (i < 20) { if (i > 2) {break;} else { i++ }}
	`

	testScript(SCRIPT, _undefined, t)
}

func TestLoopRet1(t *testing.T) {
	const SCRIPT = `
	for (var i = 0; i < 20; i++) { }
	`

	testScript(SCRIPT, _undefined, t)
}

func TestInstanceof(t *testing.T) {
	const SCRIPT = `
	var rv;
	try {
		true();
	} catch (e) {
		rv = e instanceof TypeError;
	}
	rv;
	`

	testScript(SCRIPT, valueTrue, t)
}

func TestStrictAssign(t *testing.T) {
	const SCRIPT = `
	'use strict';
	var rv;
	var called = false;
	function F() {
		called = true;
		return 1;
	}
	try {
		x = F();
	} catch (e) {
		rv = e instanceof ReferenceError;
	}
	rv + " " + called;
	`

	testScript(SCRIPT, asciiString("true true"), t)
}

func TestStrictScope(t *testing.T) {
	const SCRIPT = `
	var rv;
	var called = false;
	function F() {
		'use strict';
		x = 1;
	}
	try {
		F();
	} catch (e) {
		rv = e instanceof ReferenceError;
	}
	x = 1;
	rv + " " + x;
	`

	testScript(SCRIPT, asciiString("true 1"), t)
}

func TestStringObj(t *testing.T) {
	const SCRIPT = `
	var s = new String("test");
	s[0] + s[2] + s[1];
	`

	testScript(SCRIPT, asciiString("tse"), t)
}

func TestStringPrimitive(t *testing.T) {
	const SCRIPT = `
	var s = "test";
	s[0] + s[2] + s[1];
	`

	testScript(SCRIPT, asciiString("tse"), t)
}

func TestCallGlobalObject(t *testing.T) {
	const SCRIPT = `
	var rv;
	try {
		this();
	} catch (e) {
		rv = e instanceof TypeError
	}
	rv;
	`

	testScript(SCRIPT, valueTrue, t)
}

func TestFuncLength(t *testing.T) {
	const SCRIPT = `
	function F(x, y) {

	}
	F.length
	`

	testScript(SCRIPT, intToValue(2), t)
}

func TestNativeFuncLength(t *testing.T) {
	const SCRIPT = `
	eval.length + Object.defineProperty.length + String.length
	`

	testScript(SCRIPT, intToValue(5), t)
}

func TestArguments(t *testing.T) {
	const SCRIPT = `
	function F() {
		return arguments.length + " " + arguments[1];
	}

	F(1,2,3)
	`

	testScript(SCRIPT, asciiString("3 2"), t)
}

func TestArgumentsPut(t *testing.T) {
	const SCRIPT = `
	function F(x, y) {
		arguments[0] -= arguments[1];
		return x;
	}

	F(5, 2)
	`

	testScript(SCRIPT, intToValue(3), t)
}

func TestArgumentsPutStrict(t *testing.T) {
	const SCRIPT = `
	function F(x, y) {
		'use strict';
		arguments[0] -= arguments[1];
		return x;
	}

	F(5, 2)
	`

	testScript(SCRIPT, intToValue(5), t)
}

func TestArgumentsExtra(t *testing.T) {
	const SCRIPT = `
	function F(x, y) {
		return arguments[2];
	}

	F(1, 2, 42)
	`

	testScript(SCRIPT, intToValue(42), t)
}

func TestArgumentsExist(t *testing.T) {
	const SCRIPT = `
	function F(x, arguments) {
		return arguments;
	}

	F(1, 42)
	`

	testScript(SCRIPT, intToValue(42), t)
}

func TestArgumentsDelete(t *testing.T) {
	const SCRIPT = `
	function f(x) {
		delete arguments[0];
		arguments[0] = 42;
		return x;
	}
	f(1)
	`

	testScript(SCRIPT, intToValue(1), t)
}

func TestArgumentsInEval(t *testing.T) {
	const SCRIPT = `
	function f() {
		return eval("arguments");
	}
	f(1)[0];
	`

	testScript(SCRIPT, intToValue(1), t)
}

func TestArgumentsRedeclareInEval(t *testing.T) {
	const SCRIPT = `
	assert.sameValue("arguments" in this, false, "No global 'arguments' binding");

	function f(p = eval("var arguments = 'param'"), arguments) {}
	assert.throws(SyntaxError, f);

	assert.sameValue("arguments" in this, false, "No global 'arguments' binding");
	`

	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestArgumentsRedeclareArrow(t *testing.T) {
	const SCRIPT = `
	const oldArguments = globalThis.arguments;
	let count = 0;
	const f = (p = eval("var arguments = 'param'"), q = () => arguments) => {
	  var arguments = "local";
	  assert.sameValue(arguments, "local", "arguments");
	  assert.sameValue(q(), "param", "q");
	  count++;
	}
	f();
	assert.sameValue(count, 1);
	assert.sameValue(globalThis.arguments, oldArguments, "globalThis.arguments unchanged");
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestEvalParamWithDef(t *testing.T) {
	const SCRIPT = `
	function f(param = 0) {
		eval("var param = 1");
		return param;
	}
	f();
	`

	testScript(SCRIPT, valueInt(1), t)
}

func TestArgumentsRedefinedAsLetDyn(t *testing.T) {
	const SCRIPT = `
	function f() {
		let arguments;
		eval(""); // force dynamic scope
		return arguments;
	}
	
	f(1,2);
	`

	testScript(SCRIPT, _undefined, t)
}

func TestWith(t *testing.T) {
	const SCRIPT = `
	var b = 1;
	var o = {a: 41};
	with(o) {
		a += b;
	}
	o.a;

	`

	testScript(SCRIPT, intToValue(42), t)
}

func TestWithInFunc(t *testing.T) {
	const SCRIPT = `
	function F() {
		var b = 1;
		var c = 0;
		var o = {a: 40, c: 1};
		with(o) {
			a += b + c;
		}
		return o.a;
	}

	F();
	`

	testScript(SCRIPT, intToValue(42), t)
}

func TestAssignNonExtendable(t *testing.T) {
	const SCRIPT = `
	'use strict';

	function F() {
    		this.x = 1;
	}

	var o = new F();
	Object.preventExtensions(o);
	o.x = 42;
	o.x;
	`

	testScript(SCRIPT, intToValue(42), t)
}

func TestAssignNonExtendable1(t *testing.T) {
	const SCRIPT = `
	'use strict';

	function F() {
	}

	var o = new F();
	var rv;

	Object.preventExtensions(o);
	try {
		o.x = 42;
	} catch (e) {
		rv = e.constructor === TypeError;
	}

	rv += " " + o.x;
	rv;
	`

	testScript(SCRIPT, asciiString("true undefined"), t)
}

func TestAssignStrict(t *testing.T) {
	const SCRIPT = `
	'use strict';

	try {
		eval("eval = 42");
	} catch(e) {
		var rv = e instanceof SyntaxError
	}
	rv;
	`

	testScript(SCRIPT, valueTrue, t)
}

func TestIllegalArgmentName(t *testing.T) {
	const SCRIPT = `
	'use strict';

	try {
		eval("function F(eval) {}");
	} catch (e) {
		var rv = e instanceof SyntaxError
	}
	rv;
	`

	testScript(SCRIPT, valueTrue, t)
}

func TestFunction(t *testing.T) {
	const SCRIPT = `

	var f0 = Function("");
	var f1 = Function("return ' one'");
	var f2 = Function("arg", "return ' ' + arg");
	f0() + f1() + f2("two");
	`

	testScript(SCRIPT, asciiString("undefined one two"), t)
}

func TestFunction1(t *testing.T) {
	const SCRIPT = `

	var f = function f1(count) {
		if (count == 0) {
			return true;
		}
		return f1(count-1);
	}

	f(1);
	`

	testScript(SCRIPT, valueTrue, t)
}

func TestFunction2(t *testing.T) {
	const SCRIPT = `
	var trace = "";
	function f(count) {
    		trace += "f("+count+")";
    		if (count == 0) {
        		return;
    		}
    		return f(count-1);
	}

	function f1() {
    		trace += "f1";
	}

	var f2 = f;
	f = f1;
	f2(1);
	trace;

	`

	testScript(SCRIPT, asciiString("f(1)f1"), t)
}

func TestFunctionToString(t *testing.T) {
	const SCRIPT = `

	Function("arg1", "arg2", "return 42").toString();
	`

	testScript(SCRIPT, asciiString("function anonymous(arg1,arg2\n) {\nreturn 42\n}"), t)
}

func TestObjectLiteral(t *testing.T) {
	const SCRIPT = `
	var getterCalled = false;
	var setterCalled = false;

	var o = {get x() {getterCalled = true}, set x() {setterCalled = true}};

	o.x;
	o.x = 42;

	getterCalled && setterCalled;
	`

	testScript(SCRIPT, valueTrue, t)
}

func TestConst(t *testing.T) {
	const SCRIPT = `

	var v1 = true && true;
	var v2 = 1/(-1 * 0);
	var v3 = 1 == 2 || v1;
	var v4 = true && false
	v1 === true && v2 === -Infinity && v3 === v1 && v4 === false;
	`

	testScript(SCRIPT, valueTrue, t)
}

func TestConstWhile(t *testing.T) {
	const SCRIPT = `
	var c = 0;
	while (2 + 2 === 4) {
		if (++c > 9) {
			break;
		}
	}
	c === 10;
	`

	testScript(SCRIPT, valueTrue, t)
}

func TestConstWhileThrow(t *testing.T) {
	const SCRIPT = `
	var thrown = false;
	try {
		while ('s' in true) {
			break;
		}
	} catch (e) {
		thrown = e instanceof TypeError
	}
	thrown;
	`

	testScript(SCRIPT, valueTrue, t)
}

func TestDupParams(t *testing.T) {
	const SCRIPT = `
	function F(x, y, x) {
		return x;
	}

	F(1, 2);
	`

	testScript(SCRIPT, _undefined, t)
}

func TestUseUnsuppliedParam(t *testing.T) {
	const SCRIPT = `
	function getMessage(message) {
		if (message === undefined) {
			message = '';
		}
		message += " 123 456";
		return message;
	}

	getMessage();
	`

	testScript(SCRIPT, asciiString(" 123 456"), t)
}

func TestForInLetWithInitializer(t *testing.T) {
	const SCRIPT = `for (let x = 3 in {}) { }`
	_, err := Compile("", SCRIPT, false)
	if err == nil {
		t.Fatal("Expected error")
	}
}

func TestForInLoop(t *testing.T) {
	const SCRIPT = `
	function Proto() {}
	Proto.prototype.x = 42;
	var o = new Proto();
	o.y = 44;
	o.x = 45;
	var hasX = false;
	var hasY = false;

	for (var i in o) {
    		switch(i) {
    		case "x":
        		if (hasX) {
            			throw new Error("Already has X");
        		}
        		hasX = true;
        		break;
    		case "y":
        		if (hasY) {
            			throw new Error("Already has Y");
        		}
        		hasY = true;
        		break;
    		}
	}

	hasX && hasY;
	`

	testScript(SCRIPT, valueTrue, t)
}

func TestWhileLoopResult(t *testing.T) {
	const SCRIPT = `
	while(false);

	`

	testScript(SCRIPT, _undefined, t)
}

func TestEmptySwitch(t *testing.T) {
	const SCRIPT = `
	switch(1){}
	`

	testScript(SCRIPT, _undefined, t)
}

func TestEmptyDoWhile(t *testing.T) {
	const SCRIPT = `
	do {} while(false)
	`

	testScript(SCRIPT, _undefined, t)
}

func TestSwitch(t *testing.T) {
	const SCRIPT = `
	function F(x) {
		var i = 0;
		switch (x) {
		case 0:
			i++;
		case 1:
			i++;
		default:
			i++;
		case 2:
			i++;
			break;
		case 3:
			i++;
		}
		return i;
	}

	F(0) + F(1) + F(2) + F(4);

	`

	testScript(SCRIPT, intToValue(10), t)
}

func TestSwitchDefFirst(t *testing.T) {
	const SCRIPT = `
	function F(x) {
		var i = 0;
		switch (x) {
		default:
			i++;
		case 0:
			i++;
		case 1:
			i++;
		case 2:
			i++;
			break;
		case 3:
			i++;
		}
		return i;
	}

	F(0) + F(1) + F(2) + F(4);

	`

	testScript(SCRIPT, intToValue(10), t)
}

func TestSwitchResult(t *testing.T) {
	const SCRIPT = `
	var x = 2;

	switch (x) {
	case 0:
		"zero";
	case 1:
		"one";
	case 2:
		"two";
		break;
	case 3:
		"three";
	default:
		"default";
	}
	`

	testScript(SCRIPT, asciiString("two"), t)
}

func TestSwitchResult1(t *testing.T) {
	const SCRIPT = `
	var x = 0;
	switch (x) { case 0: "two"; case 1: break}
	`

	testScript(SCRIPT, asciiString("two"), t)
}

func TestSwitchResult2(t *testing.T) {
	const SCRIPT = `
	6; switch ("a") { case "a": 7; case "b": }
	`

	testScript(SCRIPT, valueInt(7), t)
}

func TestSwitchResultJumpIntoEmptyEval(t *testing.T) {
	const SCRIPT = `
	function t(x) {
		return eval("switch(x) { case 1: 2; break; case 2: let x = 1; case 3: x+2; break; case 4: default: 9}");
	}
	""+t(2)+t();
	`

	testScript(SCRIPT, asciiString("39"), t)
}

func TestSwitchResultJumpIntoEmpty(t *testing.T) {
	const SCRIPT = `
	switch(2) { case 1: 2; break; case 2: let x = 1; case 3: x+2; case 4: {let y = 2}; break; default: 9};
	`

	testScript(SCRIPT, valueInt(3), t)
}

func TestSwitchLexical(t *testing.T) {
	const SCRIPT = `
	switch (true) { case true: let x = 1; }
	`

	testScript(SCRIPT, _undefined, t)
}

func TestSwitchBreakOuter(t *testing.T) {
	const SCRIPT = `
	LOOP:
	for (let i = 0; i < 10; i++) {
		switch (i) {
		case 0:
			continue;
		case 1:
			let x = 1;
			continue;
		case 2:
			try {
				x++;
			} catch (e) {
				if (e instanceof ReferenceError) {
					break LOOP;
				}
			}
			throw new Error("Exception was not thrown");
		}
	}
	`

	testScript(SCRIPT, _undefined, t)
}

func TestIfBreakResult(t *testing.T) {
	const SCRIPT = `
	L: {if (true) {42;} break L;}
	`

	testScript(SCRIPT, intToValue(42), t)
}

func TestSwitchNoMatch(t *testing.T) {
	const SCRIPT = `
	var result;
	var x;
	switch (x) {
	case 0:
		result = "2";
		break;
	}

	result;

	`

	testScript(SCRIPT, _undefined, t)
}

func TestSwitchNoMatchNoDefault(t *testing.T) {
	const SCRIPT = `
		switch (1) {
		case 0:
		}
	`

	testScript(SCRIPT, _undefined, t)
}

func TestSwitchNoMatchNoDefaultNoResult(t *testing.T) {
	const SCRIPT = `
		switch (1) {
		case 0:
		}
		42;
	`

	testScript(SCRIPT, intToValue(42), t)
}

func TestSwitchNoMatchNoDefaultNoResultMatch(t *testing.T) {
	const SCRIPT = `
		switch (1) {
		case 1:
		}
		42;
	`

	testScript(SCRIPT, intToValue(42), t)
}

func TestEmptySwitchNoResult(t *testing.T) {
	const SCRIPT = `
		switch (1) {}
		42;
	`

	testScript(SCRIPT, intToValue(42), t)
}

func TestGetOwnPropertyNames(t *testing.T) {
	const SCRIPT = `
	var o = {
		prop1: 42,
		prop2: "test"
	}

	var hasProp1 = false;
	var hasProp2 = false;

	var names = Object.getOwnPropertyNames(o);
	for (var i in names) {
		var p = names[i];
		switch(p) {
		case "prop1":
			hasProp1 = true;
			break;
		case "prop2":
			hasProp2 = true;
			break;
		}
	}

	hasProp1 && hasProp2;
	`

	testScript(SCRIPT, valueTrue, t)
}

func TestArrayLiteral(t *testing.T) {
	const SCRIPT = `

	var f1Called = false;
	var f2Called = false;
	var f3Called = false;
	var errorThrown = false;

	function F1() {
		f1Called = true;
	}

	function F2() {
		f2Called = true;
	}

	function F3() {
		f3Called = true;
	}


	try {
		var a = [F1(), x(F3()), F2()];
	} catch(e) {
		if (e instanceof ReferenceError) {
			errorThrown = true;
		} else {
			throw e;
		}
	}

	f1Called && !f2Called && f3Called && errorThrown && a === undefined;
	`

	testScript(SCRIPT, valueTrue, t)
}

func TestJumpOutOfReturn(t *testing.T) {
	const SCRIPT = `
	function f() {
		var a;
		if (a == 0) {
			return true;
		}
	}

	f();
	`

	testScript(SCRIPT, _undefined, t)
}

func TestSwitchJumpOutOfReturn(t *testing.T) {
	const SCRIPT = `
	function f(x) {
		switch(x) {
		case 0:
			break;
		default:
			return x;
		}
	}

	f(0);
	`

	testScript(SCRIPT, _undefined, t)
}

func TestSetToReadOnlyPropertyStrictBracket(t *testing.T) {
	const SCRIPT = `
	'use strict';

	var o = {};
	var thrown = false;
	Object.defineProperty(o, "test", {value: 42, configurable: true});
	try {
		o["test"] = 43;
	} catch (e) {
		thrown = e instanceof TypeError;
	}

	thrown;
	`

	testScript(SCRIPT, valueTrue, t)
}

func TestSetToReadOnlyPropertyStrictDot(t *testing.T) {
	const SCRIPT = `
	'use strict';

	var o = {};
	var thrown = false;
	Object.defineProperty(o, "test", {value: 42, configurable: true});
	try {
		o.test = 43;
	} catch (e) {
		thrown = e instanceof TypeError;
	}

	thrown;
	`

	testScript(SCRIPT, valueTrue, t)
}

func TestDeleteNonConfigurablePropertyStrictBracket(t *testing.T) {
	const SCRIPT = `
	'use strict';

	var o = {};
	var thrown = false;
	Object.defineProperty(o, "test", {value: 42});
	try {
		delete o["test"];
	} catch (e) {
		thrown = e instanceof TypeError;
	}

	thrown;
	`

	testScript(SCRIPT, valueTrue, t)
}

func TestDeleteNonConfigurablePropertyStrictDot(t *testing.T) {
	const SCRIPT = `
	'use strict';

	var o = {};
	var thrown = false;
	Object.defineProperty(o, "test", {value: 42});
	try {
		delete o.test;
	} catch (e) {
		thrown = e instanceof TypeError;
	}

	thrown;
	`

	testScript(SCRIPT, valueTrue, t)
}

func TestCompound1(t *testing.T) {
	const SCRIPT = `
	var x = 0;
  	var scope = {x: 1};
    	var f;
  	with (scope) {
    		f = function() {
        		x *= (delete scope.x, 2);
    		}
  	}
	f();

	scope.x === 2 && x === 0;

	`

	testScript(SCRIPT, valueTrue, t)
}

func TestCompound2(t *testing.T) {
	const SCRIPT = `

var x;
x = "x";
x ^= "1";

	`
	testScript(SCRIPT, intToValue(1), t)
}

func TestDeleteArguments(t *testing.T) {
	defer func() {
		if _, ok := recover().(*CompilerSyntaxError); !ok {
			t.Fatal("Expected syntax error")
		}
	}()
	const SCRIPT = `
	'use strict';

	function f() {
		delete arguments;
	}

	`
	testScript(SCRIPT, _undefined, t)
}

func TestReturnUndefined(t *testing.T) {
	const SCRIPT = `
	function f() {
    		return x;
	}

	var thrown = false;
	try {
		f();
	} catch (e) {
		thrown = e instanceof ReferenceError;
	}

	thrown;
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestForBreak(t *testing.T) {
	const SCRIPT = `
	var supreme, count;
	supreme = 5;
	var __evaluated =  eval("for(count=0;;) {if (count===supreme)break;else count++; }");
    	if (__evaluated !== void 0) {
        	throw new Error('#1: __evaluated === 4. Actual:  __evaluated ==='+ __evaluated  );
    	}

	`
	testScript(SCRIPT, _undefined, t)
}

func TestLargeNumberLiteral(t *testing.T) {
	const SCRIPT = `
	var x = 0x800000000000000000000;
	x.toString();
	`
	testScript(SCRIPT, asciiString("9.671406556917033e+24"), t)
}

func TestIncDelete(t *testing.T) {
	const SCRIPT = `
	var o = {x: 1};
	o.x += (delete o.x, 1);
	o.x;
	`
	testScript(SCRIPT, intToValue(2), t)
}

func TestCompoundAssignRefError(t *testing.T) {
	const SCRIPT = `
	var thrown = false;
	try {
		a *= 1;
	} catch (e) {
		if (e instanceof ReferenceError) {
			thrown = true;
		} else {
			throw e;
		}
	}
	thrown;
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestObjectLiteral__Proto__(t *testing.T) {
	const SCRIPT = `
	var o = {
		__proto__: null,
		test: 42
	}

	Object.getPrototypeOf(o);
	`

	testScript(SCRIPT, _null, t)
}

func TestEmptyCodeError(t *testing.T) {
	if _, err := New().RunString(`i`); err == nil {
		t.Fatal("Expected an error")
	} else {
		if e := err.Error(); e != "ReferenceError: i is not defined at <eval>:1:1(0)" {
			t.Fatalf("Unexpected error: '%s'", e)
		}
	}
}

func TestForOfArray(t *testing.T) {
	const SCRIPT = `
	var array = [0, 'a', true, false, null, /* hole */, undefined, NaN];
	var i = 0;
	
	for (var value of array) {
	  assert.sameValue(value, array[i], 'element at index ' + i);
	  i++;
	}
	
	assert.sameValue(i, 8, 'Visits all elements');
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestForOfReturn(t *testing.T) {
	const SCRIPT = `
	var callCount = 0;
	var iterationCount = 0;
	var iterable = {};
	var x = {
	  set attr(_) {
		throw new Test262Error();
	  }
	};
	
	iterable[Symbol.iterator] = function() {
	  return {
		next: function() {
		  return { done: false, value: 0 };
		},
		return: function() {
		  callCount += 1;
		}
	  }
	};
	
	assert.throws(Test262Error, function() {
	  for (x.attr of iterable) {
		iterationCount += 1;
	  }
	});
	
	assert.sameValue(iterationCount, 0, 'The loop body is not evaluated');
	assert.sameValue(callCount, 1, 'Iterator is closed');
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestForOfReturn1(t *testing.T) {
	const SCRIPT = `
	var iterable = {};
	var iterationCount = 0;

	iterable[Symbol.iterator] = function() {
	  return {
		next: function() {
		  return { done: false, value: null };
		},
		get return() {
		  throw new Test262Error();
		}
	  };
	};

	assert.throws(Test262Error, function() {
	  for (var x of iterable) {
		iterationCount += 1;
		break;
	  }
	});

	assert.sameValue(iterationCount, 1, 'The loop body is evaluated');
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestForOfLet(t *testing.T) {
	const SCRIPT = `
	var iterCount = 0;
	function f() {}
	for (var let of [23]) {
		f(let);
		if (let != 23) {
			throw new Error("");
		}
		iterCount += 1;
	}

	iterCount;
`
	testScript(SCRIPT, valueInt(1), t)
}

func TestForOfLetLet(t *testing.T) {
	const SCRIPT = `
	for (let let of [23]) {
	}
`
	_, err := Compile("", SCRIPT, false)
	if err == nil {
		t.Fatal("Expected error")
	}
}

func TestForHeadLet(t *testing.T) {
	const SCRIPT = `
	for (let = 0; let < 2; let++);
`
	testScript(SCRIPT, _undefined, t)
}

func TestLhsLet(t *testing.T) {
	const SCRIPT = `
	let = 1;
	let;
	`
	testScript(SCRIPT, valueInt(1), t)
}

func TestLetPostfixASI(t *testing.T) {
	const SCRIPT = `
	let
	++
	`
	_, err := Compile("", SCRIPT, false)
	if err == nil {
		t.Fatal("Expected error")
	}
}

func TestIteratorReturnNormal(t *testing.T) {
	const SCRIPT = `
	var iterable = {};
	var iterationCount = 0;

	iterable[Symbol.iterator] = function() {
	  return {
		next: function() {
		  return { done: ++iterationCount > 2, value: null };
		},
		get return() {
		  throw new Test262Error();
		}
	  };
	};

	for (var x of iterable) {
	}
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestIteratorReturnErrorNested(t *testing.T) {
	const SCRIPT = `
	var returnCalled = {};
	function iter(id) {
		return function() {
			var count = 0;
			return {
				next: function () {
					return {
						value: null,
						done: ++count > 2
					};
				},
				return: function () {
					returnCalled[id] = true;
					throw new Error(id);
				}
			};
		}
	}
	var iterable1 = {};
	iterable1[Symbol.iterator] = iter("1");
	var iterable2 = {};
	iterable2[Symbol.iterator] = iter("2");

	try {
		for (var i of iterable1) {
			for (var j of iterable2) {
				break;
			}
		}
		throw new Error("no exception was thrown");
	} catch (e) {
		if (e.message !== "2") {
			throw e;
		}
	}
	if (!returnCalled["1"]) {
		throw new Error("no return 1");
	}
	if (!returnCalled["2"]) {
		throw new Error("no return 2");
	}
	`
	testScript(SCRIPT, _undefined, t)
}

func TestReturnFromForInLoop(t *testing.T) {
	const SCRIPT = `
	(function f() {
		for (var i in {a: 1}) {
			return true;
		}
	})();
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestReturnFromForOfLoop(t *testing.T) {
	const SCRIPT = `
	(function f() {
		for (var i of [1]) {
			return true;
		}
	})();
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestIfStackLeaks(t *testing.T) {
	const SCRIPT = `
	var t = 0;
	if (t === 0) {
		t;
	}
	`
	testScript(SCRIPT, _positiveZero, t)
}

func TestWithCallee(t *testing.T) {
	const SCRIPT = `
	function O() {
		var that = this;
		this.m = function() {
			return this === that;
		}
	}
	with(new O()) {
		m();
	}
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestWithScope(t *testing.T) {
	const SCRIPT = `
	function f(o) {
		var x = 42;

		function innerf(o) {
			with (o) {
				return x;
			}
		}

		return innerf(o);
	}
	f({});
	`
	testScript(SCRIPT, valueInt(42), t)
}

func TestEvalCallee(t *testing.T) {
	const SCRIPT = `
	(function () {
		'use strict';
		var v = function() {
			return this === undefined;
		};
		return eval('v()');
	})();
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestEvalBindingDeleteVar(t *testing.T) {
	const SCRIPT = `
	(function () {
		eval("var x = 1");
		return x === 1 && delete x;
	})();
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestEvalBindingDeleteFunc(t *testing.T) {
	const SCRIPT = `
	(function () {
		eval("function x(){}");
		return typeof x === "function" && delete x;
	})();
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestDeleteGlobalLexical(t *testing.T) {
	const SCRIPT = `
	let x;
	delete x;
	`
	testScript(SCRIPT, valueFalse, t)
}

func TestDeleteGlobalEval(t *testing.T) {
	const SCRIPT = `
	eval("var x");
	delete x;
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestGlobalVarNames(t *testing.T) {
	vm := New()
	_, err := vm.RunString("(0,eval)('var x')")
	if err != nil {
		t.Fatal(err)
	}
	_, err = vm.RunString("let x")
	if err == nil {
		t.Fatal("Expected error")
	}
}

func TestTryResultEmpty(t *testing.T) {
	const SCRIPT = `
	1; try { } finally { }
	`
	testScript(SCRIPT, _undefined, t)
}

func TestTryResultEmptyCatch(t *testing.T) {
	const SCRIPT = `
	1; try { throw null } catch(e) { }
	`
	testScript(SCRIPT, _undefined, t)
}

func TestTryResultEmptyContinueLoop(t *testing.T) {
	const SCRIPT = `
	for (var i = 0; i < 2; i++) { try {throw null;} catch(e) {continue;} 'bad'}
	`
	testScript(SCRIPT, _undefined, t)
}

func TestTryEmptyCatchStackLeak(t *testing.T) {
	const SCRIPT = `
	(function() {
		var f;
		// Make sure the outer function is not stashless.
		(function() {
			f++;
		})();
		try {
			throw new Error();
		} catch(e) {}
	})();
	`
	testScript(SCRIPT, _undefined, t)
}

func TestTryThrowEmptyCatch(t *testing.T) {
	const SCRIPT = `
	try {
		throw new Error();
	}
	catch (e) {}
	`
	testScript(SCRIPT, _undefined, t)
}

func TestFalsyLoopBreak(t *testing.T) {
	const SCRIPT = `
	while(false) {
	  	break;
	}
	for(;false;) {
		break;
	}
	undefined;
	`
	MustCompile("", SCRIPT, false)
}

func TestFalsyLoopBreakWithResult(t *testing.T) {
	const SCRIPT = `
	while(false) {
	  break;
	}
	`
	testScript(SCRIPT, _undefined, t)
}

func TestDummyCompile(t *testing.T) {
	const SCRIPT = `
	'use strict';
	
	for (;false;) {
		eval = 1;
	}
	`

	_, err := Compile("", SCRIPT, false)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDummyCompileForUpdate(t *testing.T) {
	const SCRIPT = `
	'use strict';
	
	for (;false;eval=1) {
	}
	`

	_, err := Compile("", SCRIPT, false)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestObjectLiteralWithNumericKeys(t *testing.T) {
	const SCRIPT = `
	var o = {1e3: true};
	var keys = Object.keys(o);
	var o1 = {get 1e3() {return true;}};
	var keys1 = Object.keys(o1);
	var o2 = {1e21: true};
	var keys2 = Object.keys(o2);
	keys.length === 1 && keys[0] === "1000" && 
	keys1.length === 1 && keys1[0] === "1000" && o1[1e3] === true &&
	keys2.length === 1 && keys2[0] === "1e+21";
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestEscapedObjectPropertyKeys(t *testing.T) {
	const SCRIPT = `
	var obj = {
		w\u0069th: 42
	};
	var obj = {
		with() {42}
	};
	`

	_, err := Compile("", SCRIPT, false)
	if err != nil {
		t.Fatal(err)
	}
}

func TestEscapedKeywords(t *testing.T) {
	const SCRIPT = `r\u0065turn;`
	_, err := Compile("", SCRIPT, false)
	if err == nil {
		t.Fatal("Expected error")
	}
}

func TestEscapedLet(t *testing.T) {
	const SCRIPT = `
this.let = 0;

l\u0065t // ASI
a;

// If the parser treated the previous escaped "let" as a lexical declaration,
// this variable declaration will result an early syntax error.
var a;
`
	_, err := Compile("", SCRIPT, false)
	if err != nil {
		t.Fatal(err)
	}
}

func TestObjectLiteralFuncProps(t *testing.T) {
	const SCRIPT = `
	(function() {
		'use strict';
		var o = {
			eval: function() {return 1;},
			arguments() {return 2;},
			test: function test1() {}
		}
		assert.sameValue(o.eval.name, "eval");
		assert.sameValue(o.arguments.name, "arguments");
		assert.sameValue(o.eval(), 1);
		assert.sameValue(o.arguments(), 2);
		assert.sameValue(o.test.name, "test1");
	})();
	`

	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestFuncName(t *testing.T) {
	const SCRIPT = `
	var method = 1;
	var o = {
		method: function() {
			return method;
		},
		method1: function method() {
			return method;
		}
	}
	o.method() === 1 && o.method1() === o.method1;
	`

	testScript(SCRIPT, valueTrue, t)
}

func TestFuncNameAssign(t *testing.T) {
	const SCRIPT = `
	var f = function() {};
	var f1;
	f1 = function() {};
	let f2 = function() {};

	f.name === "f" && f1.name === "f1" && f2.name === "f2";
	`

	testScript(SCRIPT, valueTrue, t)
}

func TestLexicalDeclGlobal(t *testing.T) {
	const SCRIPT = `
	if (true) {
		let it = "be";
		if (it !== "be") {
			throw new Error(it);
		}
	}
	let thrown = false;
	try {
		it;
	} catch(e) {
		if (e instanceof ReferenceError) {
			thrown = true;
		}
	}
	thrown;
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestLexicalDeclFunction(t *testing.T) {
	const SCRIPT = `
	function f() {
		if (true) {
			let it = "be";
			if (it !== "be") {
				throw new Error(it);
			}
		}
		let thrown = false;
		try {
			it;
		} catch(e) {
			if (e instanceof ReferenceError) {
				thrown = true;
			}
		}
		return thrown;
	}
	f();
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestLexicalDynamicScope(t *testing.T) {
	const SCRIPT = `
	const global = 1;
	function f() {
		const func = global + 1;
		function inner() {
			function assertThrows(fn) {
				let thrown = false;
				try {
					fn();
				} catch (e) {
					if (e instanceof TypeError) {
						thrown = true;
					} else {
						throw e;
					}
				}
				if (!thrown) {
					throw new Error("Did not throw");
				}
			}

			assertThrows(function() {
				func++;
			});
			assertThrows(function() {
				global++;
			});

			assertThrows(function() {
				eval("func++");
			});
			assertThrows(function() {
				eval("global++");
			});

			return eval("func + 1");
		}
		return inner();
	}
	f();
	`
	testScript(SCRIPT, valueInt(3), t)
}

func TestNonStrictLet(t *testing.T) {
	const SCRIPT = `
	var let = 1;
	`

	testScript(SCRIPT, _undefined, t)
}

func TestStrictLet(t *testing.T) {
	const SCRIPT = `
	var let = 1;
	`

	_, err := Compile("", SCRIPT, true)
	if err == nil {
		t.Fatal("Expected an error")
	}
}

func TestLetLet(t *testing.T) {
	const SCRIPT = `
	let let = 1;
	`

	_, err := Compile("", SCRIPT, false)
	if err == nil {
		t.Fatal("Expected an error")
	}
}

func TestLetASI(t *testing.T) {
	const SCRIPT = `
	while (false) let // ASI
	x = 1;
	`

	_, err := Compile("", SCRIPT, false)
	if err != nil {
		t.Fatal(err)
	}
}

func TestLetASI1(t *testing.T) {
	const SCRIPT = `
	let
	x = 1;
	`

	_, err := Compile("", SCRIPT, true)
	if err != nil {
		t.Fatal(err)
	}
}

func TestLetNoASI(t *testing.T) {
	const SCRIPT = `
	function f() {}let
x = 1;
	`

	_, err := Compile("", SCRIPT, true)
	if err != nil {
		t.Fatal(err)
	}
}

func TestLetNoASI1(t *testing.T) {
	const SCRIPT = `
let
let = 1;
	`

	_, err := Compile("", SCRIPT, false)
	if err == nil {
		t.Fatal("Expected error")
	}
}

func TestLetArrayWithNewline(t *testing.T) {
	const SCRIPT = `
    with ({}) let
    [a] = 0;
	`

	_, err := Compile("", SCRIPT, false)
	if err == nil {
		t.Fatal("Expected error")
	}
}

func TestDynamicUninitedVarAccess(t *testing.T) {
	const SCRIPT = `
	function f() {
		var x;
		return eval("x");
	}
	f();
	`
	testScript(SCRIPT, _undefined, t)
}

func TestLexicalForLoopNoClosure(t *testing.T) {
	const SCRIPT = `
	let sum = 0;
	for (let i = 0; i < 3; i++) {
		sum += i;
	}
	sum;
	`
	testScript(SCRIPT, valueInt(3), t)
}

func TestLexicalForLoopClosure(t *testing.T) {
	const SCRIPT = `
	var f = [];
	for (let i = 0; i < 3; i++) {
		f.push(function() {
			return i;
		});
	}
	f.length === 3 && f[0]() === 0 && f[1]() === 1 && f[2]() === 2;
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestLexicalForLoopClosureInNext(t *testing.T) {
	const SCRIPT = `
	const a = [];
	for (let i = 0; i < 5; a.push(function () { return i; }), ++i) { }
	let res = "";
	for (let k = 0; k < 5; ++k) {
		res += ""+a[k]();
	}
	res;
	`
	testScript(SCRIPT, asciiString("12345"), t)
}

func TestVarForLoop(t *testing.T) {
	const SCRIPT = `
	var f = [];
	for (var i = 0, j = 0; i < 3; i++) {
		f.push(function() {
			return i;
		});
	}
	f.length === 3 && f[0]() === 3 && f[1]() === 3 && f[2]() === 3;
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestLexicalForOfLoop(t *testing.T) {
	const SCRIPT = `
	var f = [];
	for (let i of [0, 1, 2]) {
		f.push(function() {
			return i;
		});
	}
	f.length === 3 && f[0]() === 0 && f[1]() === 1 && f[2]() === 2;
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestLexicalForOfLoopContBreak(t *testing.T) {
	const SCRIPT = `
	const f = [];
	for (let i of [0, 1, 2, 3, 4, 5]) {
		if (i % 2) continue;
		f.push(function() {
			return i;
		});
		if (i > 2) break;
	}
	let res = "";
	f.forEach(function(item) {res += item()});
	f.length === 3 && res === "024";
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestVarBlockConflict(t *testing.T) {
	const SCRIPT = `
	let x;
	{
		if (false) {
			var x;
		}
	}
	`
	_, err := Compile("", SCRIPT, false)
	if err == nil {
		t.Fatal("Expected an error")
	}
}

func TestVarBlockConflictEval(t *testing.T) {
	const SCRIPT = `
	assert.throws(SyntaxError, function() {
		let x;
		{
			if (true) {
				eval("var x");
			}
		}
	});
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestVarBlockNoConflict(t *testing.T) {
	const SCRIPT = `
	function f() {
		let x;
		function ff() {
			{
				var x = 3;
			}
		}
		ff();
	}
	f();
	`
	testScript(SCRIPT, _undefined, t)
}

func TestVarBlockNoConflictEval(t *testing.T) {
	const SCRIPT = `
	function f() {
		let x;
		function ff() {
			{
				eval("var x = 3");
			}
		}
		ff();
	}
	f();
	`
	testScript(SCRIPT, _undefined, t)
}

func TestVarDeclCorrectScope(t *testing.T) {
	const SCRIPT = `
	function f() {
		{
			let z;
			eval("var x = 3");
		}
		return x;
	}
	f();
	`
	testScript(SCRIPT, valueInt(3), t)
}

func TestLexicalCatch(t *testing.T) {
	const SCRIPT = `
	try {
		throw null;
	} catch (e) {
		let x = 1;
		function f() {}
		e;
	}
	`
	testScript(SCRIPT, _null, t)
}

func TestArgumentsLexicalDecl(t *testing.T) {
	const SCRIPT = `
	function f1() {
		let arguments;
		return arguments;
	}
	f1(42);
	`
	testScript(SCRIPT, _undefined, t)
}

func TestArgumentsLexicalDeclAssign(t *testing.T) {
	const SCRIPT = `
	function f1() {
		let arguments = arguments;
		return a;
	}
	assert.throws(ReferenceError, function() {
		f1(42);
	});
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestLexicalConstModifyFromEval(t *testing.T) {
	const SCRIPT = `
	const x = 1;
	function f() {
		eval("x = 2");
	}
	assert.throws(TypeError, function() {
		f();
	});
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestLexicalStrictNames(t *testing.T) {
	const SCRIPT = `let eval = 1;`

	_, err := Compile("", SCRIPT, true)
	if err == nil {
		t.Fatal("Expected an error")
	}
}

func TestAssignAfterStackExpand(t *testing.T) {
	// make sure the reference to the variable x does not remain stale after the stack is copied
	const SCRIPT = `
	function f() {
		let sum = 0;
		for (let i = 0; i < arguments.length; i++) {
			sum += arguments[i];
		}
		return sum;
	}
	function testAssignment() {
	  var x = 0;
	  var scope = {};

	  with (scope) {
		x = (scope.x = f(0, 0, 0, 0, 0, 0, 1, 1), 1);
	  }

	  if (scope.x !== 2) {
		throw new Error('#1: scope.x === 2. Actual: ' + (scope.x));
	  }
	  if (x !== 1) {
		throw new Error('#2: x === 1. Actual: ' + (x));
	  }
	}
	testAssignment();
	`
	testScript(SCRIPT, _undefined, t)
}

func TestArgAccessFromDynamicStash(t *testing.T) {
	const SCRIPT = `
	function f(arg) {
		function test() {
			eval("");
			return a;
		}
		return arg;
	}
	f(true);
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestLoadMixedLex(t *testing.T) {
	const SCRIPT = `
	function f() {
		let a = 1;
		{
			function inner() {
				eval("var a = true");
				return a;
			}
			return inner();
		}
	}
	f();
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestObjectLiteralSpread(t *testing.T) {
	const SCRIPT = `
	let src = {prop1: 1};
	Object.defineProperty(src, "prop2", {value: 2, configurable: true});
	Object.defineProperty(src, "prop3", {value: 3, enumerable: true, configurable: true});
	let target = {prop4: 4, ...src};
	assert(deepEqual(target, {prop1: 1, prop3: 3, prop4: 4}));
	`
	testScriptWithTestLibX(SCRIPT, _undefined, t)
}

func TestArrayLiteralSpread(t *testing.T) {
	const SCRIPT = `
	let a1 = [1, 2];
	let a2 = [3, 4];
	let a = [...a1, 0, ...a2, 1];
	assert(compareArray(a, [1, 2, 0, 3, 4, 1]));
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestObjectAssignmentPattern(t *testing.T) {
	const SCRIPT = `
	let a, b, c;
	({a, b, c=3} = {a: 1, b: 2});
	assert.sameValue(a, 1, "a");
	assert.sameValue(b, 2, "b");
	assert.sameValue(c, 3, "c");
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestObjectAssignmentPatternNested(t *testing.T) {
	const SCRIPT = `
	let a, b, c, d;
	({a, b, c: {d} = 3} = {a: 1, b: 2, c: {d: 4}});
	assert.sameValue(a, 1, "a");
	assert.sameValue(b, 2, "b");
	assert.sameValue(c, undefined, "c");
	assert.sameValue(d, 4, "d");
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestObjectAssignmentPatternEvalOrder(t *testing.T) {
	const SCRIPT = `
	let trace = "";
	let target_obj = {};

	function src() {
	    trace += "src(),";
		return {
			get a() {
				trace += "get a,";
				return "a";
			}
		}
	}
	
	function prop1() {
		trace += "prop1(),"
		return {
			toString: function() {
				trace += "prop1-to-string(),";
				return "a";
			}
		}
	}
	
	function prop2() {
		trace += "prop2(),";
		return {
			toString: function() {
				trace += "prop2-to-string(),";
				return "b";
			}
		}
	}
	
	function target() {
		trace += "target(),"
		return target_obj;
	}
	
	let a, b;
	
	({[prop1()]: target().a, [prop2()]: b} = src());
	if (target_obj.a !== "a") {
		throw new Error("target_obj.a="+target_obj.a);
	}
	trace;
	`
	testScript(SCRIPT, asciiString("src(),prop1(),prop1-to-string(),target(),get a,prop2(),prop2-to-string(),"), t)
}

func TestArrayAssignmentPatternEvalOrder(t *testing.T) {
	const SCRIPT = `
	let trace = "";

	let src_arr = {
		[Symbol.iterator]: function() {
			let done = false;
			return {
				next: function() {
					trace += "next,";
					if (!done) {
						done = true;
						return {value: 0};
					}
					return {done: true};
				},
				return: function() {
					trace += "return,";
				}
			}
		}
	}

	function src() {
		trace += "src(),";
		return src_arr;
	}

	let tgt = {
		get a() {
			trace += "get a,";
			return "a";
		},
		get b() {
			trace += "get b,";
			return "b";
		}
	}

	function target() {
		trace += "target(),";
		return tgt;
	}

	function default_a() {
		trace += "default a,";
		return "def_a";
	}

	function default_b() {
		trace += "default b,";
		return "def_b";
	}

	([target().a = default_a(), target().b = default_b()] = src());
	trace;
	`
	testScript(SCRIPT, asciiString("src(),target(),next,target(),next,default b,"), t)
}

func TestObjectAssignPatternRest(t *testing.T) {
	const SCRIPT = `
	let a, b, c, d;
	({a, b, c, ...d} = {a: 1, b: 2, d: 4});
	assert.sameValue(a, 1, "a");
	assert.sameValue(b, 2, "b");
	assert.sameValue(c, undefined, "c");
	assert(deepEqual(d, {d: 4}), "d");
	`
	testScriptWithTestLibX(SCRIPT, _undefined, t)
}

func TestObjectBindPattern(t *testing.T) {
	const SCRIPT = `
	let {a, b, c, ...d} = {a: 1, b: 2, d: 4};
	assert.sameValue(a, 1, "a");
	assert.sameValue(b, 2, "b");
	assert.sameValue(c, undefined, "c");
	assert(deepEqual(d, {d: 4}), "d");

	var { x: y, } = { x: 23 };
	
	assert.sameValue(y, 23);
	
	assert.throws(ReferenceError, function() {
	  x;
	});
	`
	testScriptWithTestLibX(SCRIPT, _undefined, t)
}

func TestObjLiteralShorthandWithInitializer(t *testing.T) {
	const SCRIPT = `
	o = {a=1};
	`
	_, err := Compile("", SCRIPT, false)
	if err == nil {
		t.Fatal("Expected an error")
	}
}

func TestObjLiteralShorthandLetStringLit(t *testing.T) {
	const SCRIPT = `
	o = {"let"};
	`
	_, err := Compile("", SCRIPT, false)
	if err == nil {
		t.Fatal("Expected an error")
	}
}

func TestObjLiteralComputedKeys(t *testing.T) {
	const SCRIPT = `
	let o = {
		get [Symbol.toString]() {
		}
	}
	`
	testScript(SCRIPT, _undefined, t)
}

func TestObjLiteralComputedKeysEvalOrder(t *testing.T) {
	const SCRIPT = `
	let trace = [];
	function key() {
		trace.push("key");
		return {
			toString: function() {
				trace.push("key-toString");
				return "key";
			}
		}
	}
	function val() {
		trace.push("val");
		return "val";
	}
	
	const _ = {
		[key()]: val(),
	}
	
	trace.join(",");
	`
	testScript(SCRIPT, asciiString("key,key-toString,val"), t)
}

func TestArrayAssignPattern(t *testing.T) {
	const SCRIPT = `
	let a, b;
	([a, b] = [1, 2]);
	a === 1 && b === 2;
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestArrayAssignPattern1(t *testing.T) {
	const SCRIPT = `
	let a, b;
	([a = 3, b = 2] = [1]);
	a === 1 && b === 2;
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestArrayAssignPatternLHS(t *testing.T) {
	const SCRIPT = `
	let a = {};
	[ a.b, a['c'] = 2 ] = [1];
	a.b === 1 && a.c === 2;
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestArrayAssignPatternElision(t *testing.T) {
	const SCRIPT = `
	let a, b;
	([a,, b] = [1, 4, 2]);
	a === 1 && b === 2;
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestArrayAssignPatternRestPattern(t *testing.T) {
	const SCRIPT = `
	let a, b, z;
	[ z, ...[a, b] ] = [0, 1, 2];
	z === 0 && a === 1 && b === 2;
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestArrayBindingPattern(t *testing.T) {
	const SCRIPT = `
	let [a, b] = [1, 2];
	a === 1 && b === 2;
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestObjectPatternShorthandInit(t *testing.T) {
	const SCRIPT = `
	[...{ x = 1 }] = [];
	x;
	`
	testScript(SCRIPT, valueInt(1), t)
}

func TestArrayBindingPatternRestPattern(t *testing.T) {
	const SCRIPT = `
	const [a, b, ...[c, d]] = [1, 2, 3, 4];
	a === 1 && b === 2 && c === 3 && d === 4;
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestForVarPattern(t *testing.T) {
	const SCRIPT = `
	var o = {a: 1};
	var trace = "";
	for (var [key, value] of Object.entries(o)) {
		trace += key+":"+value;
	}
	trace;
	`
	testScript(SCRIPT, asciiString("a:1"), t)
}

func TestForLexPattern(t *testing.T) {
	const SCRIPT = `
	var o = {a: 1};
	var trace = "";
	for (const [key, value] of Object.entries(o)) {
		trace += key+":"+value;
	}
	trace;
	`
	testScript(SCRIPT, asciiString("a:1"), t)
}

func TestBindingPatternRestTrailingComma(t *testing.T) {
	const SCRIPT = `
	const [a, b, ...rest,] = [];
	`
	_, err := Compile("", SCRIPT, false)
	if err == nil {
		t.Fatal("Expected an error")
	}
}

func TestAssignPatternRestTrailingComma(t *testing.T) {
	const SCRIPT = `
	([a, b, ...rest,] = []);
	`
	_, err := Compile("", SCRIPT, false)
	if err == nil {
		t.Fatal("Expected an error")
	}
}

func TestFuncParamInitializerSimple(t *testing.T) {
	const SCRIPT = `
	function f(a = 1) {
		return a;
	}
	""+f()+f(2);
	`
	testScript(SCRIPT, asciiString("12"), t)
}

func TestFuncParamObjectPatternSimple(t *testing.T) {
	const SCRIPT = `
	function f({a, b} = {a: 1, b: 2}) {
		return "" + a + b;
	}
	""+f()+" "+f({a: 3, b: 4});
	`
	testScript(SCRIPT, asciiString("12 34"), t)
}

func TestFuncParamRestStackSimple(t *testing.T) {
	const SCRIPT = `
	function f(arg1, ...rest) {
		return rest;
	}
	let ar = f(1, 2, 3);
	ar.join(",");
	`
	testScript(SCRIPT, asciiString("2,3"), t)
}

func TestFuncParamRestStashSimple(t *testing.T) {
	const SCRIPT = `
	function f(arg1, ...rest) {
		eval("true");
		return rest;
	}
	let ar = f(1, 2, 3);
	ar.join(",");
	`
	testScript(SCRIPT, asciiString("2,3"), t)
}

func TestRestArgsNotInStash(t *testing.T) {
	const SCRIPT = `
	function f(...rest) {
		() => rest;
		return rest.length;
	}
	f(1,2);
	`
	testScript(SCRIPT, valueInt(2), t)
}

func TestRestArgsInStash(t *testing.T) {
	const SCRIPT = `
	function f(first, ...rest) {
		() => first;
		() => rest;
		return rest.length;
	}
	f(1,2);
	`
	testScript(SCRIPT, valueInt(1), t)
}

func TestRestArgsInStashFwdRef(t *testing.T) {
	const SCRIPT = `
	function f(first = eval(), ...rest) {
		() => first;
		() => rest;
		return rest.length === 1 && rest[0] === 2;
	}
	f(1,2);
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestFuncParamRestPattern(t *testing.T) {
	const SCRIPT = `
	function f(arg1, ...{0: rest1, 1: rest2}) {
		return ""+arg1+" "+rest1+" "+rest2;
	}
	f(1, 2, 3);
	`
	testScript(SCRIPT, asciiString("1 2 3"), t)
}

func TestFuncParamForwardRef(t *testing.T) {
	const SCRIPT = `
	function f(a = b + 1, b) {
		return ""+a+" "+b;
	}
	f(1, 2);
	`
	testScript(SCRIPT, asciiString("1 2"), t)
}

func TestFuncParamForwardRefMissing(t *testing.T) {
	const SCRIPT = `
	function f(a = b + 1, b) {
		return ""+a+" "+b;
	}
	assert.throws(ReferenceError, function() {
		f();
	});
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestFuncParamInnerRef(t *testing.T) {
	const SCRIPT = `
	function f(a = inner) {
		var inner = 42;
		return a;
	}
	assert.throws(ReferenceError, function() {
		f();
	});
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestFuncParamInnerRefEval(t *testing.T) {
	const SCRIPT = `
	function f(a = eval("inner")) {
		var inner = 42;
		return a;
	}
	assert.throws(ReferenceError, function() {
		f();
	});
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestFuncParamCalleeName(t *testing.T) {
	const SCRIPT = `
	function f(a = f) {
		var f;
		return f;
	}
	typeof f();
	`
	testScript(SCRIPT, asciiString("undefined"), t)
}

func TestFuncParamVarCopy(t *testing.T) {
	const SCRIPT = `
	function f(a = f) {
		var a;
		return a;
	}
	typeof f();
	`
	testScript(SCRIPT, asciiString("function"), t)
}

func TestFuncParamScope(t *testing.T) {
	const SCRIPT = `
	var x = 'outside';
	var probe1, probe2;
	
	function f(
		_ = probe1 = function() { return x; },
		__ = (eval('var x = "inside";'), probe2 = function() { return x; })
	) {
	}
	f();
	probe1()+" "+probe2();
	`
	testScript(SCRIPT, asciiString("inside inside"), t)
}

func TestDefParamsStackPtr(t *testing.T) {
	const SCRIPT = `
	function A() {};
	A.B = function () {};
	function D(message = '') {
	  var C = A.B;
	  C([1,2,3]);
	};
	
	D();
	`
	testScript(SCRIPT, _undefined, t)
}

func TestNestedVariadicCalls(t *testing.T) {
	const SCRIPT = `
	function f() {
		return Array.prototype.join.call(arguments, ",");
	}
	f(...[1], "a", f(...[2]));
	`
	testScript(SCRIPT, asciiString("1,a,2"), t)
}

func TestVariadicNew(t *testing.T) {
	const SCRIPT = `
	function C() {
		this.res = Array.prototype.join.call(arguments, ",");
	}
	var c = new C(...[1], "a", new C(...[2]).res);
	c.res;
	`
	testScript(SCRIPT, asciiString("1,a,2"), t)
}

func TestVariadicUseStackVars(t *testing.T) {
	const SCRIPT = `
	function A(message) { return message; }
	function B(...args){
			return A(...args);
	}
	B("C");
	`
	testScript(SCRIPT, asciiString("C"), t)
}

func TestCatchParamPattern(t *testing.T) {
	const SCRIPT = `
	function f() {
		let x = 3;
		try {
			throw {a: 1, b: 2};
		} catch ({a, b, c = x}) {
			let x = 99;
			return ""+a+" "+b+" "+c;
		}
	}
	f();
	`
	testScript(SCRIPT, asciiString("1 2 3"), t)
}

func TestArrowUseStrict(t *testing.T) {
	// simple parameter list -- ok
	_, err := Compile("", "(a) => {'use strict';}", false)
	if err != nil {
		t.Fatal(err)
	}
	// non-simple parameter list -- syntax error
	_, err = Compile("", "(a=0) => {'use strict';}", false)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestArrowBoxedThis(t *testing.T) {
	const SCRIPT = `
	var context;
	fn = function() {
		return (arg) => { var local; context = this; };
	};
	
	fn()();
	context === this;
	`

	testScript(SCRIPT, valueTrue, t)
}

func TestParameterOverride(t *testing.T) {
	const SCRIPT = `
	function f(arg) {
		var arg = arg || "default"
		return arg
	}
	f()
	`
	testScript(SCRIPT, asciiString("default"), t)
}

func TestEvalInIterScope(t *testing.T) {
	const SCRIPT = `
	for (let a = 0; a < 1; a++) {
		eval("a");
	}
	`

	testScript(SCRIPT, valueInt(0), t)
}

func TestTemplateLiterals(t *testing.T) {
	vm := New()
	_, err := vm.RunString("const a = 1, b = 'b';")
	if err != nil {
		t.Fatal(err)
	}
	f := func(t *testing.T, template, expected string) {
		res, err := vm.RunString(template)
		if err != nil {
			t.Fatal(err)
		}
		if actual := res.Export(); actual != expected {
			t.Fatalf("Expected: %q, actual: %q", expected, actual)
		}
	}
	t.Run("empty", func(t *testing.T) {
		f(t, "``", "")
	})
	t.Run("noSub", func(t *testing.T) {
		f(t, "`test`", "test")
	})
	t.Run("emptyTail", func(t *testing.T) {
		f(t, "`a=${a},b=${b}`", "a=1,b=b")
	})
	t.Run("emptyHead", func(t *testing.T) {
		f(t, "`${a},b=${b}$`", "1,b=b$")
	})
	t.Run("headAndTail", func(t *testing.T) {
		f(t, "`a=${a},b=${b}$`", "a=1,b=b$")
	})
}

func TestTaggedTemplate(t *testing.T) {
	const SCRIPT = `
		let res;
		const o = {
			tmpl() {
				res = this;
				return () => {};
			}
		}
		` +
		"o.tmpl()`test`;" + `
		res === o;
		`

	testScript(SCRIPT, valueTrue, t)
}

func TestDuplicateGlobalFunc(t *testing.T) {
	const SCRIPT = `
	function a(){}
	function b(){ return "b" }
	function c(){ return "c" }
	function a(){}
	b();
	`

	testScript(SCRIPT, asciiString("b"), t)
}

func TestDuplicateFunc(t *testing.T) {
	const SCRIPT = `
	function f() {
		function a(){}
		function b(){ return "b" }
		function c(){ return "c" }
		function a(){}
		return b();
	}
	f();
	`

	testScript(SCRIPT, asciiString("b"), t)
}

func TestSrcLocations(t *testing.T) {
	// Do not reformat, assertions depend on line and column numbers
	const SCRIPT = `
	let i = {
		valueOf() {
			throw new Error();
		}
	};
	try {
		i++;
	} catch(e) {
		assertStack(e, [["test.js", "valueOf", 4, 10],
						["test.js", "", 8, 3]
						]);
	}

	Object.defineProperty(globalThis, "x", {
		get() {
			throw new Error();
		},
		set() {
			throw new Error();
		}
	});

	try {
		x;
	} catch(e) {
		assertStack(e, [["test.js", "get", 17, 10],
						["test.js", "", 25, 3]
						]);
	}

	try {
		x++;
	} catch(e) {
		assertStack(e, [["test.js", "get", 17, 10],
						["test.js", "", 33, 3]
						]);
	}

	try {
		x = 2;
	} catch(e) {
		assertStack(e, [["test.js", "set", 20, 10],
						["test.js", "", 41, 3]
						]);
	}

	try {
		+i;
	} catch(e) {
		assertStack(e, [["test.js", "valueOf", 4, 10],
						["test.js", "", 49, 4]
						]);
	}


	function assertStack(e, expected) {
		const lines = e.stack.split('\n');
		let lnum = 1;
		for (const [file, func, line, col] of expected) {
			const expLine = func === "" ?
				"\tat " + file + ":" + line + ":" + col + "(" :
				"\tat " + func + " (" + file + ":" + line + ":" + col + "(";
			assert.sameValue(lines[lnum].substring(0, expLine.length), expLine, "line " + lnum);
			lnum++;
		}
	}
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestSrcLocationThrowLiteral(t *testing.T) {
	vm := New()
	_, err := vm.RunString(`
	const z = 1;
	throw "";
	`)
	if ex, ok := err.(*Exception); ok {
		pos := ex.stack[0].Position()
		if pos.Line != 3 {
			t.Fatal(pos)
		}
	} else {
		t.Fatal(err)
	}
}

func TestSrcLocation(t *testing.T) {
	prg := MustCompile("test.js", `
f();
var x = 1;
let y = 1;
let [z1, z2] = [0, 0];

var [z3, z4] = [0, 0];
	`, false)
	const (
		varLine     = 3
		letLine     = 4
		dstrLetLine = 5
		dstrVarLine = 7
	)
	linesOfInterest := map[int]string{
		varLine:     "var",
		letLine:     "let",
		dstrLetLine: "destruct let",
		dstrVarLine: "destruct var",
	}
	for i := range prg.code {
		loc := prg.src.Position(prg.sourceOffset(i))
		delete(linesOfInterest, loc.Line)
		if len(linesOfInterest) == 0 {
			break
		}
	}
	for _, v := range linesOfInterest {
		t.Fatalf("no %s line", v)
	}
}

func TestBadObjectKey(t *testing.T) {
	_, err := Compile("", "({!:0})", false)
	if err == nil {
		t.Fatal("expected error")
	}
}

/*
func TestBabel(t *testing.T) {
	src, err := ioutil.ReadFile("babel7.js")
	if err != nil {
		t.Fatal(err)
	}
	vm := New()
	_, err = vm.RunString(string(src))
	if err != nil {
		t.Fatal(err)
	}
	_, err = vm.RunString(`var result = Babel.transform("", {presets: ["es2015"]});`)
	if err != nil {
		t.Fatal(err)
	}
}*/

func BenchmarkCompile(b *testing.B) {
	data, err := ioutil.ReadFile("testdata/S15.10.2.12_A1_T1.js")
	if err != nil {
		b.Fatal(err)
	}

	src := string(data)

	for i := 0; i < b.N; i++ {
		_, err := Compile("test.js", src, false)
		if err != nil {
			b.Fatal(err)
		}
	}
}
