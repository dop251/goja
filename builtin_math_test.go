package goja

import (
	"testing"
)

func TestMathSumPrecise(t *testing.T) {
	const SCRIPT = `
var tenths = [];
for (var i = 0; i < 10; i++) {
	tenths.push(0.1);
}
assert.sameValue(Math.sumPrecise(tenths), 1, "tenths");
assert.sameValue(Math.sumPrecise([1e16, 1, -1e16]), 1, "cancellation");
assert.sameValue(Math.sumPrecise([Number.MAX_VALUE, Number.MAX_VALUE, -Number.MAX_VALUE]), Number.MAX_VALUE, "intermediate overflow");
assert.sameValue(Math.sumPrecise([Number.MAX_VALUE, Number.MAX_VALUE, -Infinity]), -Infinity, "overflow and -Infinity");
assert.sameValue(Math.sumPrecise([Infinity, Number.MAX_VALUE, -Number.MAX_VALUE, 1]), Infinity, "finite values after Infinity");
assert.sameValue(Math.sumPrecise(new Set([1, 2.5, -0])), 3.5, "Set");
`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestMathUnicodeStringCoercion(t *testing.T) {
	const SCRIPT = `
assert.sameValue(Math.abs("\u00a01"), 1, "NBSP prefix");
assert.sameValue(Math.abs("\ufeff-2.5\u2003"), 2.5, "BOM and em space padding");
assert.sameValue(Math.floor("\u00a03.7"), 3, "floor");
assert.sameValue("\u00a042" | 0, 42, "bitwise or");
assert.sameValue(Math.abs("\u00a0Infinity"), Infinity, "Infinity");
assert.sameValue(Math.abs("\u00a0"), 0, "whitespace only");
assert.sameValue(Math.abs("\u00a0\u0431"), NaN, "non-numeric");
assert.sameValue("\u0431" | 0, 0, "non-numeric bitwise or");
`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}
