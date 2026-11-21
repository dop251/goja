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

func TestParseIntLarge(t *testing.T) {
	const SCRIPT = `
	// Radix 10 must be the mathematical value rounded exactly once.
	assert.sameValue(parseInt("123456789012345680000"), 123456789012345680000);
	assert.sameValue(parseInt("-123456789012345680000"), -123456789012345680000);
	assert.sameValue(parseInt("123456789012345680000xyz"), 123456789012345680000);
	assert.sameValue(parseInt("9223372036854775808"), 9223372036854775808);
	assert.sameValue(parseInt("123456789012345678901234567890"), 1.2345678901234568e29);

	// Radices 2, 4, 8, 16 and 32 must be exact too.
	assert.sameValue(parseInt("c760b46463a0c40d05", 16), 3.677870375570712e21);
	assert.sameValue(parseInt("C760B46463A0C40D05", 16), 3.677870375570712e21);
	assert.sameValue(parseInt("itaunfap8aa0q", 32), 21809762542452814000);
	assert.sameValue(parseInt("ITAUNFAP8AA0Q", 32), 21809762542452814000);
	assert.sameValue(parseInt("-1701324071575532202006324217", 8), -4.5401505851592977e24);
	assert.sameValue(parseInt("10000000000000000000000000000000000000000000000000000000000000000", 2), 18446744073709552000);
	assert.sameValue(parseInt("377777777777777777777777777", 8), 1.2089258196146292e24);
	assert.sameValue(parseInt("333333333333333333333333333333333", 4), 73786976294838210000);
	assert.sameValue(parseInt("1230123012301230123012301230123012301230123012301230123012301230", 4), 1.4411959069592688e38);

	// Non-power-of-two radices may be approximated, but must not be off by a factor.
	assert.sameValue(parseInt("11112220022122120101211020120210210211221", 3), 18446744073709552000);

	// Overflow becomes Infinity.
	assert.sameValue(parseInt("9".repeat(400)), Infinity);
	assert.sameValue(parseInt("-" + "9".repeat(400)), -Infinity);
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}
