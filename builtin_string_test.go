package goja

import "testing"

func TestSubstr(t *testing.T) {
	const SCRIPT = `
assert.sameValue('abc'.substr(0, false), '', 'start: 0, length: false');
assert.sameValue('abc'.substr(1, false), '', 'start: 1, length: false');
assert.sameValue('abc'.substr(2, false), '', 'start: 2, length: false');
assert.sameValue('abc'.substr(3, false), '', 'start: 3, length: false');

assert.sameValue('abc'.substr(0, NaN), '', 'start: 0, length: NaN');
assert.sameValue('abc'.substr(1, NaN), '', 'start: 1, length: NaN');
assert.sameValue('abc'.substr(2, NaN), '', 'start: 2, length: NaN');
assert.sameValue('abc'.substr(3, NaN), '', 'start: 3, length: NaN');

assert.sameValue('abc'.substr(0, ''), '', 'start: 0, length: ""');
assert.sameValue('abc'.substr(1, ''), '', 'start: 1, length: ""');
assert.sameValue('abc'.substr(2, ''), '', 'start: 2, length: ""');
assert.sameValue('abc'.substr(3, ''), '', 'start: 3, length: ""');

assert.sameValue('abc'.substr(0, null), '', 'start: 0, length: null');
assert.sameValue('abc'.substr(1, null), '', 'start: 1, length: null');
assert.sameValue('abc'.substr(2, null), '', 'start: 2, length: null');
assert.sameValue('abc'.substr(3, null), '', 'start: 3, length: null');

assert.sameValue('abc'.substr(0, -1), '', '0, -1');
assert.sameValue('abc'.substr(0, -2), '', '0, -2');
assert.sameValue('abc'.substr(0, -3), '', '0, -3');
assert.sameValue('abc'.substr(0, -4), '', '0, -4');

assert.sameValue('abc'.substr(1, -1), '', '1, -1');
assert.sameValue('abc'.substr(1, -2), '', '1, -2');
assert.sameValue('abc'.substr(1, -3), '', '1, -3');
assert.sameValue('abc'.substr(1, -4), '', '1, -4');

assert.sameValue('abc'.substr(2, -1), '', '2, -1');
assert.sameValue('abc'.substr(2, -2), '', '2, -2');
assert.sameValue('abc'.substr(2, -3), '', '2, -3');
assert.sameValue('abc'.substr(2, -4), '', '2, -4');

assert.sameValue('abc'.substr(3, -1), '', '3, -1');
assert.sameValue('abc'.substr(3, -2), '', '3, -2');
assert.sameValue('abc'.substr(3, -3), '', '3, -3');
assert.sameValue('abc'.substr(3, -4), '', '3, -4');

assert.sameValue('abc'.substr(0, 1), 'a', '0, 1');
assert.sameValue('abc'.substr(0, 2), 'ab', '0, 1');
assert.sameValue('abc'.substr(0, 3), 'abc', '0, 1');
assert.sameValue('abc'.substr(0, 4), 'abc', '0, 1');

assert.sameValue('abc'.substr(1, 1), 'b', '1, 1');
assert.sameValue('abc'.substr(1, 2), 'bc', '1, 1');
assert.sameValue('abc'.substr(1, 3), 'bc', '1, 1');
assert.sameValue('abc'.substr(1, 4), 'bc', '1, 1');

assert.sameValue('abc'.substr(2, 1), 'c', '2, 1');
assert.sameValue('abc'.substr(2, 2), 'c', '2, 1');
assert.sameValue('abc'.substr(2, 3), 'c', '2, 1');
assert.sameValue('abc'.substr(2, 4), 'c', '2, 1');

assert.sameValue('abc'.substr(3, 1), '', '3, 1');
assert.sameValue('abc'.substr(3, 2), '', '3, 1');
assert.sameValue('abc'.substr(3, 3), '', '3, 1');
assert.sameValue('abc'.substr(3, 4), '', '3, 1');

assert.sameValue('abc'.substr(0), 'abc', 'start: 0, length: unspecified');
assert.sameValue('abc'.substr(1), 'bc', 'start: 1, length: unspecified');
assert.sameValue('abc'.substr(2), 'c', 'start: 2, length: unspecified');
assert.sameValue('abc'.substr(3), '', 'start: 3, length: unspecified');

assert.sameValue(
  'abc'.substr(0, undefined), 'abc', 'start: 0, length: undefined'
);
assert.sameValue(
  'abc'.substr(1, undefined), 'bc', 'start: 1, length: undefined'
);
assert.sameValue(
  'abc'.substr(2, undefined), 'c', 'start: 2, length: undefined'
);
assert.sameValue(
  'abc'.substr(3, undefined), '', 'start: 3, length: undefined'
);

assert.sameValue('A—', String.fromCharCode(65, 0x2014));

	`

	testScript1(TESTLIB+SCRIPT, _undefined, t)
}

func TestStringMatchSym(t *testing.T) {
	const SCRIPT = `
function Prefix(p) {
	this.p = p;
}

Prefix.prototype[Symbol.match] = function(s) {
	return s.substring(0, this.p.length) === this.p;
}

var prefix1 = new Prefix("abc");
var prefix2 = new Prefix("def");

"abc123".match(prefix1) === true && "abc123".match(prefix2) === false &&
"def123".match(prefix1) === false && "def123".match(prefix2) === true;
`
	testScript1(SCRIPT, valueTrue, t)
}

func TestStringMatchAllSym(t *testing.T) {
	const SCRIPT = `
function Prefix(p) {
	this.p = p;
}

Prefix.prototype[Symbol.matchAll] = function(s) {
	return s.substring(0, this.p.length) === this.p;
}

var prefix1 = new Prefix("abc");
var prefix2 = new Prefix("def");

"abc123".matchAll(prefix1) === true && "abc123".matchAll(prefix2) === false &&
"def123".matchAll(prefix1) === false && "def123".matchAll(prefix2) === true;
`
	testScript1(SCRIPT, valueTrue, t)
}

func TestGenericSplitter(t *testing.T) {
	const SCRIPT = `
function MyRegexp(pattern, flags) {
	if (pattern instanceof MyRegexp) {
		pattern = pattern.wrapped;
	}
	this.wrapped = new RegExp(pattern, flags);
}

MyRegexp.prototype.exec = function() {
	return this.wrapped.exec.apply(this.wrapped, arguments);
}

Object.defineProperty(MyRegexp.prototype, "lastIndex", {
	get: function() {
		return this.wrapped.lastIndex;
	},
	set: function(v) {
		this.wrapped.lastIndex = v;
	}
});

Object.defineProperty(MyRegexp.prototype, "flags", {
	get: function() {
		return this.wrapped.flags;
	}
});

MyRegexp[Symbol.species] = MyRegexp;
MyRegexp.prototype[Symbol.split] = RegExp.prototype[Symbol.split];

var r = new MyRegexp(/ /);
var res = "a b c".split(r);
res.length === 3 && res[0] === "a" && res[1] === "b" && res[2] === "c";
`
	testScript1(SCRIPT, valueTrue, t)
}

func TestStringIterSurrPair(t *testing.T) {
	const SCRIPT = `
var lo = '\uD834';
var hi = '\uDF06';
var pair = lo + hi;
var string = 'a' + pair + 'b' + lo + pair + hi + lo;
var iterator = string[Symbol.iterator]();
var result;

result = iterator.next();
if (result.value !== 'a') {
	throw new Error("at 0: " + result.value);
}
result = iterator.next();
if (result.value !== pair) {
	throw new Error("at 1: " + result.value);
}

`
	testScript1(SCRIPT, _undefined, t)
}

func TestValueStringBuilder(t *testing.T) {
	t.Run("substringASCII", func(t *testing.T) {
		t.Parallel()
		var sb valueStringBuilder
		str := newStringValue("a\U00010000b")
		sb.WriteSubstring(str, 0, 1)
		res := sb.String()
		if res != asciiString("a") {
			t.Fatal(res)
		}
	})

	t.Run("substringASCIIPure", func(t *testing.T) {
		t.Parallel()
		var sb valueStringBuilder
		str := newStringValue("ab")
		sb.WriteSubstring(str, 0, 1)
		res := sb.String()
		if res != asciiString("a") {
			t.Fatal(res)
		}
	})

	t.Run("substringUnicode", func(t *testing.T) {
		t.Parallel()
		var sb valueStringBuilder
		str := newStringValue("a\U00010000b")
		sb.WriteSubstring(str, 1, 3)
		res := sb.String()
		if !res.SameAs(unicodeStringFromRunes([]rune{0x10000})) {
			t.Fatal(res)
		}
	})

	t.Run("substringASCIIUnicode", func(t *testing.T) {
		t.Parallel()
		var sb valueStringBuilder
		str := newStringValue("a\U00010000b")
		sb.WriteSubstring(str, 0, 2)
		res := sb.String()
		if !res.SameAs(unicodeStringFromRunes([]rune{'a', 0xD800})) {
			t.Fatal(res)
		}
	})

	t.Run("substringUnicodeASCII", func(t *testing.T) {
		t.Parallel()
		var sb valueStringBuilder
		str := newStringValue("a\U00010000b")
		sb.WriteSubstring(str, 2, 4)
		res := sb.String()
		if !res.SameAs(unicodeStringFromRunes([]rune{0xDC00, 'b'})) {
			t.Fatal(res)
		}
	})
}
