package goja

import (
	"github.com/dop251/goja/parser"
	"io/ioutil"
	"os"
	"testing"
)

func testScript(script string, expectedResult Value, t *testing.T) {
	prg, err := parser.ParseFile(nil, "test.js", script, 0)
	if err != nil {
		t.Fatal(err)
	}

	c := newCompiler()
	c.compile(prg)

	r := &Runtime{}
	r.init()

	vm := r.vm
	vm.prg = c.p
	vm.prg.dumpCode(t.Logf)
	vm.run()
	vm.pop()
	t.Logf("stack size: %d", len(vm.stack))
	t.Logf("stashAllocs: %d", vm.stashAllocs)

	v := vm.r.globalObject.self.getStr("rv")
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
	c.compile(prg)

	r := &Runtime{}
	r.init()

	vm := r.vm
	vm.prg = c.p
	vm.prg.dumpCode(t.Logf)
	vm.run()
	v := vm.pop()
	t.Logf("stack size: %d", len(vm.stack))
	t.Logf("stashAllocs: %d", vm.stashAllocs)

	if v == nil && expectedResult != nil || !v.SameAs(expectedResult) {
		t.Fatalf("Result: %+v, expected: %+v", v, expectedResult)
	}

	if vm.sp != 0 {
		t.Fatalf("sp: %d", vm.sp)
	}
}

func TestEmptyProgram(t *testing.T) {
	const SCRIPT = `
	`

	testScript1(SCRIPT, _undefined, t)
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

func TestCallLessArgs(t *testing.T) {
	const SCRIPT = `
function A(a, b, c) {
	return String(a) + " " + String(b) + " " + String(c);
}

var rv = A(1, 2);
`
	testScript(SCRIPT, asciiString("1 2 undefined"), t)
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
	1..toString() === "1"
	`

	testScript1(SCRIPT, valueTrue, t)
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

func TestLoopRet(t *testing.T) {
	const SCRIPT = `
	for (var i = 0; i < 20; i++) { if (i > 1) {break;} else { i }}
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

func TestForInLoopRet(t *testing.T) {
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

func TestWhileLoopResult(t *testing.T) {
	const SCRIPT = `
	while(false);

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

func TestSwitchNoMatch(t *testing.T) {
	const SCRIPT = `
	var x = 5;
	var result;
	switch (x) {
	case 0:
		result = "2";
		break;
	}

	result;

	`

	testScript1(SCRIPT, _undefined, t)
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

// FIXME
/*
func TestDummyCompile(t *testing.T) {
	const SCRIPT = `
'use strict';

for (;false;) {
    eval = 1;
}

	`
	defer func() {
		if recover() == nil {
			t.Fatal("Expected panic")
		}
	}()

	testScript1(SCRIPT, _undefined, t)
}*/

func BenchmarkCompile(b *testing.B) {
	f, err := os.Open("testdata/S15.10.2.12_A1_T1.js")

	data, err := ioutil.ReadAll(f)
	if err != nil {
		b.Fatal(err)
	}
	f.Close()

	src := string(data)

	for i := 0; i < b.N; i++ {
		_, err := Compile("test.js", src, false)
		if err != nil {
			b.Fatal(err)
		}
	}

}
