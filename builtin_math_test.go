package goja

import (
	"testing"
)

func TestMathSign(t *testing.T) {
	const SCRIPT = `
assert.sameValue(Math.sign(undefined), NaN, "undefined");
assert.sameValue(Math.sign("abc"), NaN, "non-numeric string");
assert.sameValue(Math.sign(null), 0, "null");
assert.sameValue(Math.sign("0"), 0, "string zero");
assert.sameValue(Math.sign("-0"), -0, "string negative zero");
assert.sameValue(Math.sign(new Number(-0)), -0, "Number object");
assert.sameValue(Math.sign(-3), -1, "negative");
assert.sameValue(Math.sign(2), 1, "positive");
assert.sameValue(Math.sign(0), 0, "zero");
assert.sameValue(Math.sign(-0), -0, "negative zero");
assert.sameValue(Math.sign(NaN), NaN, "NaN");
`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

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
