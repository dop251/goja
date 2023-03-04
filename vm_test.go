package goja

import (
	"github.com/dop251/goja/file"
	"github.com/dop251/goja/parser"
	"github.com/dop251/goja/unistring"
	"testing"
)

func TestTaggedTemplateArgExport(t *testing.T) {
	vm := New()
	vm.Set("f", func(v Value) {
		v.Export()
	})
	vm.RunString("f`test`")
}

func TestVM1(t *testing.T) {
	r := &Runtime{}
	r.init()

	vm := r.vm

	vm.prg = &Program{
		src:    file.NewFile("dummy", "", 1),
		values: []Value{valueInt(2), valueInt(3), asciiString("test")},
		code: []instruction{
			&bindGlobal{vars: []unistring.String{"v"}},
			newObject,
			setGlobal("v"),
			loadVal(2),
			loadVal(1),
			loadVal(0),
			add,
			setElem,
			pop,
			loadDynamic("v"),
		},
	}

	vm.run()

	rv := vm.pop()

	if obj, ok := rv.(*Object); ok {
		if v := obj.self.getStr("test", nil).ToInteger(); v != 5 {
			t.Fatalf("Unexpected property value: %v", v)
		}
	} else {
		t.Fatalf("Unexpected result: %v", rv)
	}

}

func TestEvalVar(t *testing.T) {
	const SCRIPT = `
	function test() {
		var a;
		return eval("var a = 'yes'; var z = 'no'; a;") === "yes" && a === "yes";
	}
	test();
	`

	testScript(SCRIPT, valueTrue, t)
}

func TestResolveMixedStack1(t *testing.T) {
	const SCRIPT = `
	function test(arg) {
		var a = 1;
		var scope = {};
		(function() {return arg})(); // move arguments to stash
		with (scope) {
			a++; // resolveMixedStack1 here
			return a + arg;
		}
	}
	test(40);
	`

	testScript(SCRIPT, valueInt(42), t)
}

func TestNewArrayFromIterClosed(t *testing.T) {
	const SCRIPT = `
	const [a, ...other] = [];
	assert.sameValue(a, undefined);
	assert(Array.isArray(other));
	assert.sameValue(other.length, 0);
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func BenchmarkVmNOP2(b *testing.B) {
	prg := []func(*vm){
		//loadVal(0).exec,
		//loadVal(1).exec,
		//add.exec,
		jump(1).exec,
	}

	r := &Runtime{}
	r.init()

	vm := r.vm
	vm.prg = &Program{
		values: []Value{intToValue(2), intToValue(3)},
	}

	for i := 0; i < b.N; i++ {
		vm.pc = 0
		for !vm.halted() {
			prg[vm.pc](vm)
		}
		//vm.sp--
		/*r := vm.pop()
		if r.ToInteger() != 5 {
			b.Fatalf("Unexpected result: %+v", r)
		}
		if vm.sp != 0 {
			b.Fatalf("Unexpected sp: %d", vm.sp)
		}*/
	}
}

func BenchmarkVmNOP(b *testing.B) {
	r := &Runtime{}
	r.init()

	vm := r.vm
	vm.prg = &Program{
		code: []instruction{
			jump(1),
			//jump(1),
		},
	}

	for i := 0; i < b.N; i++ {
		vm.pc = 0
		vm.run()
	}

}

func BenchmarkVm1(b *testing.B) {
	r := &Runtime{}
	r.init()

	vm := r.vm

	//ins1 := loadVal1(0)
	//ins2 := loadVal1(1)

	vm.prg = &Program{
		values: []Value{valueInt(2), valueInt(3)},
		code: []instruction{
			loadVal(0),
			loadVal(1),
			add,
		},
	}

	for i := 0; i < b.N; i++ {
		vm.pc = 0
		vm.run()
		r := vm.pop()
		if r.ToInteger() != 5 {
			b.Fatalf("Unexpected result: %+v", r)
		}
		if vm.sp != 0 {
			b.Fatalf("Unexpected sp: %d", vm.sp)
		}
	}
}

func BenchmarkFib(b *testing.B) {
	const TEST_FIB = `
function fib(n) {
if (n < 2) return n;
return fib(n - 2) + fib(n - 1);
}

fib(35);
`
	b.StopTimer()
	prg, err := parser.ParseFile(nil, "test.js", TEST_FIB, 0)
	if err != nil {
		b.Fatal(err)
	}

	c := newCompiler()
	c.compile(prg, false, true, nil)
	c.p.dumpCode(b.Logf)

	r := &Runtime{}
	r.init()

	vm := r.vm

	var expectedResult Value = valueInt(9227465)

	b.StartTimer()

	vm.prg = c.p
	vm.run()
	v := vm.result

	b.Logf("stack size: %d", len(vm.stack))
	b.Logf("stashAllocs: %d", vm.stashAllocs)

	if !v.SameAs(expectedResult) {
		b.Fatalf("Result: %+v, expected: %+v", v, expectedResult)
	}

}

func BenchmarkEmptyLoop(b *testing.B) {
	const SCRIPT = `
	function f() {
		for (var i = 0; i < 100; i++) {
		}
	}
	f()
	`
	b.StopTimer()
	vm := New()
	prg := MustCompile("test.js", SCRIPT, false)
	// prg.dumpCode(log.Printf)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		vm.RunProgram(prg)
	}
}

func BenchmarkVMAdd(b *testing.B) {
	vm := &vm{}
	vm.stack = append(vm.stack, nil, nil)
	vm.sp = len(vm.stack)

	var v1 Value = valueInt(3)
	var v2 Value = valueInt(5)

	for i := 0; i < b.N; i++ {
		vm.stack[0] = v1
		vm.stack[1] = v2
		add.exec(vm)
		vm.sp++
	}
}

func BenchmarkFuncCall(b *testing.B) {
	const SCRIPT = `
	function f(a, b, c, d) {
	}
	`

	b.StopTimer()

	vm := New()
	prg := MustCompile("test.js", SCRIPT, false)

	vm.RunProgram(prg)
	if f, ok := AssertFunction(vm.Get("f")); ok {
		b.StartTimer()
		for i := 0; i < b.N; i++ {
			f(nil, nil, intToValue(1), intToValue(2), intToValue(3), intToValue(4), intToValue(5), intToValue(6))
		}
	} else {
		b.Fatal("f is not a function")
	}
}

func BenchmarkAssertInt(b *testing.B) {
	v := intToValue(42)
	for i := 0; i < b.N; i++ {
		if i, ok := v.(valueInt); !ok || int64(i) != 42 {
			b.Fatal()
		}
	}
}
