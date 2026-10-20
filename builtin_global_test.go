package goja

import (
	"testing"
)

func TestEncodeURI(t *testing.T) {
	const SCRIPT = `
	encodeURI('тест')
	`

	testScript(SCRIPT, asciiString("%D1%82%D0%B5%D1%81%D1%82"), t)
}

func TestDecodeURI(t *testing.T) {
	const SCRIPT = `
	decodeURI("http://ru.wikipedia.org/wiki/%d0%ae%D0%bd%D0%B8%D0%BA%D0%BE%D0%B4")
	`

	testScript(SCRIPT, newStringValue("http://ru.wikipedia.org/wiki/Юникод"), t)
}

func TestParseIntNegativeZero(t *testing.T) {
	const SCRIPT = `
assert.sameValue(parseInt("-0"), -0, "-0");
assert.sameValue(parseInt(-0.5), -0, "-0.5");

assert.sameValue(parseInt("+0"), 0, "+0");
assert.sameValue(parseInt("-1"), -1, "-1");
assert.sameValue(parseInt("-x"), NaN, "-x");
`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}
