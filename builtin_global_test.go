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

func TestParseIntLargeDecimal(t *testing.T) {
	const SCRIPT = `
assert.sameValue(parseInt("123456789012345680000"), Number("123456789012345680000"), "20 digits");
assert.sameValue(parseInt("-123456789012345680000"), Number("-123456789012345680000"), "negative");
assert.sameValue(parseInt("123456789012345680000abc"), Number("123456789012345680000"), "trailing garbage");
assert.sameValue(parseInt("9223372036854775808"), 9223372036854775808, "2^63");
assert.sameValue(parseInt("12345678901234567890123"), Number("12345678901234567890123"), "23 digits");
`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestParseIntLargeNonDecimalRadix(t *testing.T) {
	const SCRIPT = `
assert.sameValue(parseInt("c760b46463a0c40d05", 16), 3.677870375570712e+21, "radix 16");
assert.sameValue(parseInt("itaunfap8aa0q", 32), 21809762542452814000, "radix 32");
assert.sameValue(parseInt("-1701324071575532202006324217", 8), -4.5401505851592977e+24, "radix 8");
`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}
