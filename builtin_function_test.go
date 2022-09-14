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
