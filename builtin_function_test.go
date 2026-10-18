package goja

import (
	"testing"
)

func TestHashbangInFunctionConstructor(t *testing.T) {
	const SCRIPT = `
	assert.throws(SyntaxError, function() {
		new Function("#!")
	});
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestFunctionApplyNullArgArray(t *testing.T) {
	const SCRIPT = `
	assert.sameValue(0, (function() {return arguments.length}).apply(undefined, null))
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestFunctionBindNegativeLength(t *testing.T) {
	const SCRIPT = `
	function target() {}
	Object.defineProperty(target, "length", {value: -5});
	// Zero bound arguments: max(-5 - 0, 0) === 0
	var b0 = target.bind({});
	assert.sameValue(0, b0.length, "zero bound args, integer length");

	// Additional bound arguments: max(-5 - 2, 0) === 0
	var b2 = target.bind({}, 1, 2);
	assert.sameValue(0, b2.length, "two bound args, integer length");

	// Fractional negative length is truncated toward zero before clamping.
	function target2() {}
	Object.defineProperty(target2, "length", {value: -5.7});
	var bf = target2.bind({});
	assert.sameValue(0, bf.length, "zero bound args, float length");
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}
