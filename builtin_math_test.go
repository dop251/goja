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
