package goja

import "testing"

func TestNumberPrototypeRadixCoercion(t *testing.T) {
	cases := []struct{ name, script string }{
		{"invalid_low", `assert.throws(RangeError, function() { Number.prototype.toString(1); });`},
		{"invalid_high", `assert.throws(RangeError, function() { Number.prototype.toString(37); });`},
		{"invalid_nan", `assert.throws(RangeError, function() { Number.prototype.toString(NaN); });`},
		{"invalid_infinity", `assert.throws(RangeError, function() { Number.prototype.toString(Infinity); });`},
		{"symbol", `assert.throws(TypeError, function() { Number.prototype.toString(Symbol()); });`},
		{"bigint", `assert.throws(TypeError, function() { Number.prototype.toString(2n); });`},
		{"coercion", `
var calls = 0;
assert.sameValue(Number.prototype.toString({valueOf: function() { calls++; return 2; }}), "0");
assert.sameValue(calls, 1);`},
		{"throws", `
var sentinel = {}, caught;
try { Number.prototype.toString({valueOf: function() { throw sentinel; }}); }
catch (e) { caught = e; }
assert.sameValue(caught, sentinel);`},
		{"defaults", `
assert.sameValue(Number.prototype.toString(), "0");
assert.sameValue(Number.prototype.toString(undefined), "0");`},
		{"bases", `
for (var radix = 2; radix <= 36; radix++) {
    assert.sameValue(Number.prototype.toString(radix), "0");
    assert.sameValue((new Number(0)).toString(radix), "0");
}`},
		{"coercible_radices", `
assert.sameValue(Number.prototype.toString("16"), "0");
assert.sameValue(Number.prototype.toString(2.9), "0");`},
		{"ordinary_receivers", `
assert.sameValue((255).toString(16), "ff");
assert.sameValue((new Number(-255)).toString(16), "-ff");
assert.throws(RangeError, function() { (0).toString(1); });
var calls = 0;
assert.sameValue((0).toString({valueOf: function() { calls++; return 2; }}), "0");
assert.sameValue(calls, 1);`},
		{"receiver_before_radix", `
var calls = 0;
assert.throws(TypeError, function() {
    Number.prototype.toString.call({}, {valueOf: function() { calls++; return 2; }});
});
assert.sameValue(calls, 0);`},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) { testScriptWithTestLib(tt.script, _undefined, t) })
	}
}
