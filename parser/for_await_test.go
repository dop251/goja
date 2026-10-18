package parser

import (
	"testing"

	"github.com/dop251/goja/ast"
)

func TestForAwaitOfSyntax(t *testing.T) {
	for _, src := range []string{
		"async function f() { for await (const x of xs) {} }",
		"async function f() { for await (let [a, {b}] of xs) {} }",
		"async function f() { for await (var {a, b: [c]} of xs) {} }",
		"async function f() { for await (x of xs) {} }",
		"async function f() { for await (o.p of xs) {} }",
		"async function f() { for await ([a, o.b] of xs) {} }",
		"async function f() { for await ({a, b: o[c]} of xs) {} }",
		"async function f() { for await (async of [7]) {} }",
		"var f = async () => { for await (const x of xs) {} }",
		"var f = async x => { for await (const y of x) {} }",
		"var o = { async m() { for await (const x of xs) {} } }",
		"class C { async m() { for await (const x of xs) {} } static async s() { for await (x of xs); } }",
		"async function f() { outer: for await (const x of xs) { for await (const y of x) { continue outer; } } }",
		"async function f() { for await (const x of await xs) { await x; } }",
		"async function f() { for await (async.p of xs) {} }",
		"async function f() { for await (async\nof xs) {} }",
		"async function f() { for await (x of xs) break; }",
	} {
		if _, err := ParseFile(nil, "", src, 0); err != nil {
			t.Errorf("%q: %v", src, err)
		}
	}

	for _, src := range []string{
		// await is not allowed here
		"for await (x of xs) {}",
		"function f() { for await (x of xs) {} }",
		"async function f() { function g() { for await (x of xs) {} } }",
		"async function f() { () => { for await (x of xs) {} } }",
		"async function f() { var o = { m() { for await (x of xs) {} } } }",
		"async function f() { class C { static { for await (x of xs) {} } } }",
		"function* g() { for await (x of xs) {} }",
		// only the of-form is allowed
		"async function f() { for await (x in xs) {} }",
		"async function f() { for await (var x in xs) {} }",
		"async function f() { for await (;;) {} }",
		"async function f() { for await (let i = 0; i < 1; i++) {} }",
		// initializers
		"async function f() { for await (var x = 1 of xs) {} }",
		"async function f() { for await (let x = 1 of xs) {} }",
		"async function f() { for await (const x = 1 of xs) {} }",
		// malformed heads
		"async function f() { for await (x o\\u0066 xs) {} }",
		"async function f() { for await (x + 1 of xs) {} }",
		"async function f() { for await (x of xs, ys) {} }",
		"async function f() { for \\u0061wait (x of xs) {} }",
		// 'async of' is only allowed as a for-await head
		"for (async of [7]) {}",
		"async function f() { for (async of [7]) {} }",
	} {
		if _, err := ParseFile(nil, "", src, 0); err == nil {
			t.Errorf("unexpectedly accepted %q", src)
		}
	}
}

func TestForAwaitOfAST(t *testing.T) {
	program, err := ParseFile(nil, "", "async function f() { for await (x of xs) {} for (y of ys) {} }", 0)
	if err != nil {
		t.Fatal(err)
	}
	body := program.Body[0].(*ast.FunctionDeclaration).Function.Body.List
	if !body[0].(*ast.ForOfStatement).Await {
		t.Error("for await: Await is false")
	}
	if body[1].(*ast.ForOfStatement).Await {
		t.Error("for-of: Await is true")
	}
}
