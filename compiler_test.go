package goja

import (
	"github.com/dop251/goja/parser"
	"io/ioutil"
	"testing"
)

func testScript(script string, expectedResult Value, t *testing.T) {
	prg, err := parser.ParseFile(nil, "test.js", script, 0)
	if err != nil {
		t.Fatal(err)
	}

	c := newCompiler()
	c.compile(prg, false, false, true)

	r := &Runtime{}
	r.init()

	vm := r.vm
	vm.prg = c.p
	vm.prg.dumpCode(t.Logf)
	vm.run()
	t.Logf("stack size: %d", len(vm.stack))
	t.Logf("stashAllocs: %d", vm.stashAllocs)

	v := vm.r.globalObject.self.getStr("rv", nil)
	if v == nil {
		v = _undefined
	}
	if !v.SameAs(expectedResult) {
		t.Fatalf("Result: %+v, expected: %+v", v, expectedResult)
	}

	if vm.sp != 0 {
		t.Fatalf("sp: %d", vm.sp)
	}
}

func testScript1(script string, expectedResult Value, t *testing.T) {
	prg, err := parser.ParseFile(nil, "test.js", script, 0)
	if err != nil {
		t.Fatal(err)
	}

	c := newCompiler()
	c.compile(prg, false, false, true)

	r := &Runtime{}
	r.init()

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

func TestEmptyProgram(t *testing.T) {
	const SCRIPT = `
	`

	testScript1(SCRIPT, _undefined, t)
}

func TestResultEmptyBlock(t *testing.T) {
	const SCRIPT = `
	undefined;
	{}
	`
	testScript1(SCRIPT, _undefined, t)
}

func TestResultVarDecl(t *testing.T) {
	const SCRIPT = `
	7; var x = 1;
	`
	testScript1(SCRIPT, valueInt(7), t)
}

func TestResultLexDecl(t *testing.T) {
	const SCRIPT = `
	7; {let x = 1};
	`
	testScript1(SCRIPT, valueInt(7), t)
}

func TestResultLexDeclBreak(t *testing.T) {
	const SCRIPT = `
	L:{ 7; {let x = 1; break L;}};
	`
	testScript1(SCRIPT, valueInt(7), t)
}

func TestResultLexDeclNested(t *testing.T) {
	const SCRIPT = `
	7; {let x = (function() { return eval("8; {let y = 9}")})()};
	`
	testScript1(SCRIPT, valueInt(7), t)
}

func TestErrorProto(t *testing.T) {
	const SCRIPT = `
	var e = new TypeError();
	e.name;
	`

	testScript1(SCRIPT, asciiString("TypeError"), t)
}

func TestThis1(t *testing.T) {
	const SCRIPT = `
	function independent() {
		return this.prop;
	}
	var o = {};
	o.b = {g: independent, prop: 42};

	var rv = o.b.g();
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

var rv = o.f();
`

	testScript(SCRIPT, intToValue(37), t)
}

func TestThisStrict(t *testing.T) {
	const SCRIPT = `
	"use strict";

	Object.defineProperty(Object.prototype, "x", { get: function () { return this; } });

	(5).x === 5;
	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestThisNoStrict(t *testing.T) {
	const SCRIPT = `
	Object.defineProperty(Object.prototype, "x", { get: function () { return this; } });

	(5).x == 5;
	`

	testScript1(SCRIPT, valueTrue, t)
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
	testScript1(SCRIPT, valueInt(42), t)
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
	testScript1(SCRIPT, _undefined, t)
}

func TestCallFewerArgs(t *testing.T) {
	const SCRIPT = `
function A(a, b, c) {
	return String(a) + " " + String(b) + " " + String(c);
}

var rv = A(1, 2);
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

	var rv = A(1, 2) + x();
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

	var rv = A(1, 2) + x();
`
	testScript(SCRIPT, asciiString("1 2 undefined 1 2"), t)
}

func TestCallMoreArgs(t *testing.T) {
	const SCRIPT = `
function A(a, b) {
	var c = 4;
	return a - b + c;
}

var rv = A(1, 2, 3);
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

var rv = A(1, 2, 3);
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

var rv = A(1, 2);
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

	testScript1(SCRIPT, intToValue(42), t)
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
	var rv = o.test;
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
	var rv = o.test;
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

	var rv = A();
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

	var rv = A();
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

	var rv = A();
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

	var rv = A();
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
	testScript1(SCRIPT, _undefined, t)
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

	var rv = A();
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

	var rv = A();
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
	testScript1(SCRIPT, intToValue(1), t)
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
	testScript1(SCRIPT, _undefined, t)
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
	testScript1(SCRIPT, intToValue(1), t)
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
	testScript1(SCRIPT, _undefined, t)
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
	testScript1(SCRIPT, valueTrue, t)
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
	testScript1(SCRIPT, valueTrue, t)
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
	testScript1(SCRIPT, valueTrue, t)
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
	testScript1(SCRIPT, asciiString("12"), t)
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
	testScript1(SCRIPT, _undefined, t)
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
	testScript1(SCRIPT, intToValue(1), t)
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
	var rv = thrown !== null && typeof thrown === "object" && thrown.constructor === Exception;
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

	testScript1(SCRIPT, asciiString("Exception"), t)
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

	testScript1(SCRIPT, asciiString("TypeError"), t)
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

	testScript1(SCRIPT, valueTrue, t)
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

	testScript1(SCRIPT, valueInt(42), t)
}

func TestIfElse(t *testing.T) {
	const SCRIPT = `
	var rv;
	if (rv === undefined) {
		rv = "passed";
	} else {
		rv = "failed";
	}
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

	testScript1(SCRIPT, asciiString("passed"), t)
}

func TestWhileReturnValue(t *testing.T) {
	const SCRIPT = `
	var x = 0;
	while(true) {
		x = 1;
		break;
	}
	`
	testScript1(SCRIPT, intToValue(1), t)
}

func TestIfElseLabel(t *testing.T) {
	const SCRIPT = `
	var x = 0;
	abc: if (true) {
		x = 1;
		break abc;
	}
	`
	testScript1(SCRIPT, intToValue(1), t)
}

func TestIfMultipleLabels(t *testing.T) {
	const SCRIPT = `
	var x = 0;
	xyz:abc: if (true) {
		break xyz;
	}
	`
	testScript1(SCRIPT, _undefined, t)
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
	testScript1(SCRIPT, intToValue(2), t)
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
	testScript1(SCRIPT, intToValue(1), t)
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
	testScript1(SCRIPT, intToValue(2), t)
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
	testScript1(SCRIPT, intToValue(2), t)
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
	testScript1(SCRIPT, asciiString("ex"), t)
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
	testScript1(SCRIPT, asciiString("ex2"), t)
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
	testScript1(SCRIPT, valueTrue, t)
}

func TestEvalLexicalDecl(t *testing.T) {
	const SCRIPT = `
	eval("let x = true; x;");
	`
	testScript1(SCRIPT, valueTrue, t)
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
	testScript1(SCRIPT, asciiString("ex1"), t)
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
	testScript1(SCRIPT, asciiString("ex1"), t)
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
	testScript1(SCRIPT, asciiString("ok"), t)
}

func TestNew(t *testing.T) {
	const SCRIPT = `
	function O() {
		this.x = 42;
	}

	new O().x;
	`

	testScript1(SCRIPT, intToValue(42), t)
}

func TestStringConstructor(t *testing.T) {
	const SCRIPT = `
	function F() {
		return String(33) + " " + String("cows");
	}

	F();
	`
	testScript1(SCRIPT, asciiString("33 cows"), t)
}

func TestError(t *testing.T) {
	const SCRIPT = `
	function F() {
		return new Error("test");
	}

	var e = F();
	var rv = e.message == "test" && e.name == "Error";
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

	testScript1(SCRIPT, valueTrue, t)
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
	testScript1(SCRIPT, asciiString("42 ### [object Object]"), t)
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
	rv += trace + o.x;
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
	rv += trace + o.x;
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
	rv += trace + o.x;
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
	rv += trace + o.x;
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

	testScript1(SCRIPT, valueTrue, t)
}

func TestPropAcc1(t *testing.T) {
	const SCRIPT = `
	1..toString()
	`

	testScript1(SCRIPT, asciiString("1"), t)
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
	`
	testScript(SCRIPT, valueTrue, t)
}

func TestEvalRet(t *testing.T) {
	const SCRIPT = `
	eval("for (var i = 0; i < 3; i++) {i}")
	`

	testScript1(SCRIPT, valueInt(2), t)
}

func TestEvalFunctionDecl(t *testing.T) {
	const SCRIPT = `
	eval("function F() {}")
	`

	testScript1(SCRIPT, _undefined, t)
}

func TestEvalFunctionExpr(t *testing.T) {
	const SCRIPT = `
	eval("(function F() {return 42;})")()
	`

	testScript1(SCRIPT, intToValue(42), t)
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

	testScript1(SCRIPT, asciiString("str2"), t)
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

	testScript1(SCRIPT, valueTrue, t)
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

	testScript1(SCRIPT, valueTrue, t)
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

	testScript1(TESTLIB+SCRIPT, asciiString("21"), t)
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

	testScript1(TESTLIB+SCRIPT, _undefined, t)
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

	testScript1(SCRIPT, _undefined, t)
}

func TestEvalEmptyStrict(t *testing.T) {
	const SCRIPT = `
	'use strict';
	eval("");
	`

	testScript1(SCRIPT, _undefined, t)
}

func TestEvalFuncDecl(t *testing.T) {
	const SCRIPT = `
	'use strict';
	var funcA = eval("function __funcA(__arg){return __arg;}; __funcA");
	typeof funcA;
	`

	testScript1(SCRIPT, asciiString("function"), t)
}

func TestGetAfterSet(t *testing.T) {
	const SCRIPT = `
	function f() {
		var x = 1;
		return x;
	}
	`

	testScript1(SCRIPT, _undefined, t)
}

func TestForLoopRet(t *testing.T) {
	const SCRIPT = `
	for (var i = 0; i < 20; i++) { if (i > 2) {break;} else { i }}
	`

	testScript1(SCRIPT, _undefined, t)
}

func TestForLoopRet1(t *testing.T) {
	const SCRIPT = `
	for (var i = 0; i < 20; i++) { if (i > 2) {42;; {L:{break;}}} else { i }}
	`

	testScript1(SCRIPT, intToValue(42), t)
}

func TestForInLoopRet(t *testing.T) {
	const SCRIPT = `
	var o = [1, 2, 3, 4];
	for (var i in o) { if (i > 2) {break;} else { i }}
	`

	testScript1(SCRIPT, _undefined, t)
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

	testScript1(SCRIPT, valueTrue, t)
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

	testScript1(SCRIPT, _undefined, t)
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

	testScript1(SCRIPT, valueTrue, t)
}

func TestWhileLoopRet(t *testing.T) {
	const SCRIPT = `
	var i; while (i < 20) { if (i > 2) {break;} else { i++ }}
	`

	testScript1(SCRIPT, _undefined, t)
}

func TestLoopRet1(t *testing.T) {
	const SCRIPT = `
	for (var i = 0; i < 20; i++) { }
	`

	testScript1(SCRIPT, _undefined, t)
}

func TestInstanceof(t *testing.T) {
	const SCRIPT = `
	var rv;
	try {
		true();
	} catch (e) {
		rv = e instanceof TypeError;
	}
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
	rv += " " + called;
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
	rv += " " + x;
	`

	testScript(SCRIPT, asciiString("true 1"), t)
}

func TestStringObj(t *testing.T) {
	const SCRIPT = `
	var s = new String("test");
	s[0] + s[2] + s[1];
	`

	testScript1(SCRIPT, asciiString("tse"), t)
}

func TestStringPrimitive(t *testing.T) {
	const SCRIPT = `
	var s = "test";
	s[0] + s[2] + s[1];
	`

	testScript1(SCRIPT, asciiString("tse"), t)
}

func TestCallGlobalObject(t *testing.T) {
	const SCRIPT = `
	var rv;
	try {
		this();
	} catch (e) {
		rv = e instanceof TypeError
	}
	`

	testScript(SCRIPT, valueTrue, t)
}

func TestFuncLength(t *testing.T) {
	const SCRIPT = `
	function F(x, y) {

	}
	F.length
	`

	testScript1(SCRIPT, intToValue(2), t)
}

func TestNativeFuncLength(t *testing.T) {
	const SCRIPT = `
	eval.length + Object.defineProperty.length + String.length
	`

	testScript1(SCRIPT, intToValue(5), t)
}

func TestArguments(t *testing.T) {
	const SCRIPT = `
	function F() {
		return arguments.length + " " + arguments[1];
	}

	F(1,2,3)
	`

	testScript1(SCRIPT, asciiString("3 2"), t)
}

func TestArgumentsPut(t *testing.T) {
	const SCRIPT = `
	function F(x, y) {
		arguments[0] -= arguments[1];
		return x;
	}

	F(5, 2)
	`

	testScript1(SCRIPT, intToValue(3), t)
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

	testScript1(SCRIPT, intToValue(5), t)
}

func TestArgumentsExtra(t *testing.T) {
	const SCRIPT = `
	function F(x, y) {
		return arguments[2];
	}

	F(1, 2, 42)
	`

	testScript1(SCRIPT, intToValue(42), t)
}

func TestArgumentsExist(t *testing.T) {
	const SCRIPT = `
	function F(x, arguments) {
		return arguments;
	}

	F(1, 42)
	`

	testScript1(SCRIPT, intToValue(42), t)
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

	testScript1(SCRIPT, intToValue(1), t)
}

func TestArgumentsInEval(t *testing.T) {
	const SCRIPT = `
	function f() {
		return eval("arguments");
	}
	f(1)[0];
	`

	testScript1(SCRIPT, intToValue(1), t)
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

	testScript1(SCRIPT, intToValue(42), t)
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

	testScript1(SCRIPT, intToValue(42), t)
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

	testScript1(SCRIPT, intToValue(42), t)
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

	testScript1(SCRIPT, asciiString("undefined one two"), t)
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

	testScript1(SCRIPT, valueTrue, t)
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

	testScript1(SCRIPT, asciiString("f(1)f1"), t)
}

func TestFunctionToString(t *testing.T) {
	const SCRIPT = `

	Function("arg1", "arg2", "return 42").toString();
	`

	testScript1(SCRIPT, asciiString("function anonymous(arg1,arg2){return 42}"), t)
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

	testScript1(SCRIPT, valueTrue, t)
}

func TestConst(t *testing.T) {
	const SCRIPT = `

	var v1 = true && true;
	var v2 = 1/(-1 * 0);
	var v3 = 1 == 2 || v1;
	var v4 = true && false
	v1 === true && v2 === -Infinity && v3 === v1 && v4 === false;
	`

	testScript1(SCRIPT, valueTrue, t)
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

	testScript1(SCRIPT, valueTrue, t)
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

	testScript1(SCRIPT, valueTrue, t)
}

func TestDupParams(t *testing.T) {
	const SCRIPT = `
	function F(x, y, x) {
		return x;
	}

	F(1, 2);
	`

	testScript1(SCRIPT, _undefined, t)
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

	testScript1(SCRIPT, asciiString(" 123 456"), t)
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

	testScript1(SCRIPT, valueTrue, t)
}

func TestWhileLoopResult(t *testing.T) {
	const SCRIPT = `
	while(false);

	`

	testScript1(SCRIPT, _undefined, t)
}

func TestEmptySwitch(t *testing.T) {
	const SCRIPT = `
	switch(1){}
	`

	testScript1(SCRIPT, _undefined, t)
}

func TestEmptyDoWhile(t *testing.T) {
	const SCRIPT = `
	do {} while(false)
	`

	testScript1(SCRIPT, _undefined, t)
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

	testScript1(SCRIPT, intToValue(10), t)
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

	testScript1(SCRIPT, intToValue(10), t)
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

	testScript1(SCRIPT, asciiString("two"), t)
}

func TestSwitchResult1(t *testing.T) {
	const SCRIPT = `
	var x = 0;
	switch (x) { case 0: "two"; case 1: break}
	`

	testScript1(SCRIPT, asciiString("two"), t)
}

func TestSwitchResult2(t *testing.T) {
	const SCRIPT = `
	6; switch ("a") { case "a": 7; case "b": }
	`

	testScript1(SCRIPT, valueInt(7), t)
}

func TestSwitchResultJumpIntoEmptyEval(t *testing.T) {
	const SCRIPT = `
	function t(x) {
		return eval("switch(x) { case 1: 2; break; case 2: let x = 1; case 3: x+2; break; case 4: default: 9}");
	}
	""+t(2)+t();
	`

	testScript1(SCRIPT, asciiString("39"), t)
}

func TestSwitchResultJumpIntoEmpty(t *testing.T) {
	const SCRIPT = `
	switch(2) { case 1: 2; break; case 2: let x = 1; case 3: x+2; case 4: {let y = 2}; break; default: 9};
	`

	testScript1(SCRIPT, valueInt(3), t)
}

func TestSwitchLexical(t *testing.T) {
	const SCRIPT = `
	switch (true) { case true: let x = 1; }
	`

	testScript1(SCRIPT, _undefined, t)
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

	testScript1(SCRIPT, _undefined, t)
}

func TestIfBreakResult(t *testing.T) {
	const SCRIPT = `
	L: {if (true) {42;} break L;}
	`

	testScript1(SCRIPT, intToValue(42), t)
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

	testScript1(SCRIPT, _undefined, t)
}

func TestSwitchNoMatchNoDefault(t *testing.T) {
	const SCRIPT = `
		switch (1) {
		case 0:
		}
	`

	testScript1(SCRIPT, _undefined, t)
}

func TestSwitchNoMatchNoDefaultNoResult(t *testing.T) {
	const SCRIPT = `
		switch (1) {
		case 0:
		}
		42;
	`

	testScript1(SCRIPT, intToValue(42), t)
}

func TestSwitchNoMatchNoDefaultNoResultMatch(t *testing.T) {
	const SCRIPT = `
		switch (1) {
		case 1:
		}
		42;
	`

	testScript1(SCRIPT, intToValue(42), t)
}

func TestEmptySwitchNoResult(t *testing.T) {
	const SCRIPT = `
		switch (1) {}
		42;
	`

	testScript1(SCRIPT, intToValue(42), t)
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

	testScript1(SCRIPT, valueTrue, t)
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

	testScript1(SCRIPT, valueTrue, t)
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

	testScript1(SCRIPT, _undefined, t)
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

	testScript1(SCRIPT, _undefined, t)
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

	testScript1(SCRIPT, valueTrue, t)
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

	testScript1(SCRIPT, valueTrue, t)
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

	testScript1(SCRIPT, valueTrue, t)
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

	testScript1(SCRIPT, valueTrue, t)
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

	testScript1(SCRIPT, valueTrue, t)
}

func TestCompound2(t *testing.T) {
	const SCRIPT = `

var x;
x = "x";
x ^= "1";

	`
	testScript1(SCRIPT, intToValue(1), t)
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
	testScript1(SCRIPT, _undefined, t)
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
	testScript1(SCRIPT, valueTrue, t)
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
	testScript1(SCRIPT, _undefined, t)
}

func TestLargeNumberLiteral(t *testing.T) {
	const SCRIPT = `
	var x = 0x800000000000000000000;
	x.toString();
	`
	testScript1(SCRIPT, asciiString("9.671406556917033e+24"), t)
}

func TestIncDelete(t *testing.T) {
	const SCRIPT = `
	var o = {x: 1};
	o.x += (delete o.x, 1);
	o.x;
	`
	testScript1(SCRIPT, intToValue(2), t)
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
	testScript1(SCRIPT, valueTrue, t)
}

func TestObjectLiteral__Proto__(t *testing.T) {
	const SCRIPT = `
	var o = {
		__proto__: null,
		test: 42
	}

	Object.getPrototypeOf(o);
	`

	testScript1(SCRIPT, _null, t)
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
	testScript1(TESTLIB+SCRIPT, _undefined, t)
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
	testScript1(TESTLIB+SCRIPT, _undefined, t)
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
	testScript1(TESTLIB+SCRIPT, _undefined, t)
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
	testScript1(SCRIPT, valueInt(1), t)
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
	testScript1(SCRIPT, _undefined, t)
}

func TestLhsLet(t *testing.T) {
	const SCRIPT = `
	let = 1;
	let;
	`
	testScript1(SCRIPT, valueInt(1), t)
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
	testScript1(TESTLIB+SCRIPT, _undefined, t)
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
	testScript1(SCRIPT, _undefined, t)
}

func TestReturnFromForInLoop(t *testing.T) {
	const SCRIPT = `
	(function f() {
		for (var i in {a: 1}) {
			return true;
		}
	})();
	`
	testScript1(SCRIPT, valueTrue, t)
}

func TestReturnFromForOfLoop(t *testing.T) {
	const SCRIPT = `
	(function f() {
		for (var i of [1]) {
			return true;
		}
	})();
	`
	testScript1(SCRIPT, valueTrue, t)
}

func TestIfStackLeaks(t *testing.T) {
	const SCRIPT = `
	var t = 0;
	if (t === 0) {
		t;
	}
	`
	testScript1(SCRIPT, _positiveZero, t)
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
	testScript1(SCRIPT, valueTrue, t)
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
	testScript1(SCRIPT, valueInt(42), t)
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
	testScript1(SCRIPT, valueTrue, t)
}

func TestEvalBindingDeleteVar(t *testing.T) {
	const SCRIPT = `
	(function () {
		eval("var x = 1");
		return x === 1 && delete x;
	})();
	`
	testScript1(SCRIPT, valueTrue, t)
}

func TestEvalBindingDeleteFunc(t *testing.T) {
	const SCRIPT = `
	(function () {
		eval("function x(){}");
		return typeof x === "function" && delete x;
	})();
	`
	testScript1(SCRIPT, valueTrue, t)
}

func TestDeleteGlobalLexical(t *testing.T) {
	const SCRIPT = `
	let x;
	delete x;
	`
	testScript1(SCRIPT, valueFalse, t)
}

func TestDeleteGlobalEval(t *testing.T) {
	const SCRIPT = `
	eval("var x");
	delete x;
	`
	testScript1(SCRIPT, valueTrue, t)
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
	testScript1(SCRIPT, _undefined, t)
}

func TestTryResultEmptyCatch(t *testing.T) {
	const SCRIPT = `
	1; try { throw null } catch(e) { }
	`
	testScript1(SCRIPT, _undefined, t)
}

func TestTryResultEmptyContinueLoop(t *testing.T) {
	const SCRIPT = `
	for (var i = 0; i < 2; i++) { try {throw null;} catch(e) {continue;} 'bad'}
	`
	testScript1(SCRIPT, _undefined, t)
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
	testScript1(SCRIPT, _undefined, t)
}

func TestTryThrowEmptyCatch(t *testing.T) {
	const SCRIPT = `
	try {
		throw new Error();
	}
	catch (e) {}
	`
	testScript1(SCRIPT, _undefined, t)
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
	testScript1(SCRIPT, _undefined, t)
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
	testScript1(SCRIPT, valueTrue, t)
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

	testScript1(TESTLIB+SCRIPT, _undefined, t)
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

	testScript1(SCRIPT, valueTrue, t)
}

func TestFuncNameAssign(t *testing.T) {
	const SCRIPT = `
	var f = function() {};
	var f1;
	f1 = function() {};
	let f2 = function() {};

	f.name === "f" && f1.name === "f1" && f2.name === "f2";
	`

	testScript1(SCRIPT, valueTrue, t)
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
	testScript1(SCRIPT, valueTrue, t)
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
	testScript1(SCRIPT, valueTrue, t)
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
	testScript1(SCRIPT, valueInt(3), t)
}

func TestNonStrictLet(t *testing.T) {
	const SCRIPT = `
	var let = 1;
	`

	testScript1(SCRIPT, _undefined, t)
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
	testScript1(SCRIPT, _undefined, t)
}

func TestLexicalForLoopNoClosure(t *testing.T) {
	const SCRIPT = `
	let sum = 0;
	for (let i = 0; i < 3; i++) {
		sum += i;
	}
	sum;
	`
	testScript1(SCRIPT, valueInt(3), t)
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
	testScript1(SCRIPT, valueTrue, t)
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
	testScript1(SCRIPT, asciiString("12345"), t)
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
	testScript1(SCRIPT, valueTrue, t)
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
	testScript1(SCRIPT, valueTrue, t)
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
	testScript1(SCRIPT, valueTrue, t)
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
	testScript1(TESTLIB+SCRIPT, _undefined, t)
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
	testScript1(SCRIPT, _undefined, t)
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
	testScript1(SCRIPT, _undefined, t)
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
	testScript1(SCRIPT, valueInt(3), t)
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
	testScript1(SCRIPT, _null, t)
}

func TestArgumentsLexicalDecl(t *testing.T) {
	const SCRIPT = `
	function f1() {
		let arguments;
		return arguments;
	}
	f1(42);
	`
	testScript1(SCRIPT, _undefined, t)
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
	testScript1(TESTLIB+SCRIPT, _undefined, t)
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
	testScript1(TESTLIB+SCRIPT, _undefined, t)
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
	testScript1(SCRIPT, _undefined, t)
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
