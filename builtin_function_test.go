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
