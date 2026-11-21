package goja

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf16"
)

func TestStringTrimPreservesUTF16(t *testing.T) {
	const SCRIPT = `
var values = ['\uD800', '\uDC00', 'a\uD800b\uDC00c', '\uD83D\uDE00', '\uFFFD'];
values.push('a'.repeat(40) + '\uD800' + 'b'.repeat(40));
for (var value of values) {
    assert.sameValue(value.trim(), value, "unchanged trim");
    assert.sameValue((' \t' + value + '\uFEFF\n').trim(), value, "trim");
    assert.sameValue(('\u2000' + value + ' ').trimStart(), value + ' ', "trimStart");
    assert.sameValue((' ' + value + '\u3000').trimEnd(), ' ' + value, "trimEnd");
    assert.sameValue((' ' + value).trimLeft(), value, "trimLeft");
    assert.sameValue((value + ' ').trimRight(), value, "trimRight");
}
assert.sameValue('\u2000\uFEFF\n'.trim(), '', "all whitespace");
assert.sameValue(' \u0085\u200B '.trim(), '\u0085\u200B', "non-whitespace");
`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

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

	testScriptWithTestLib(SCRIPT, _undefined, t)
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
	testScript(SCRIPT, valueTrue, t)
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
	testScript(SCRIPT, valueTrue, t)
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
	testScript(SCRIPT, valueTrue, t)
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
	testScript(SCRIPT, _undefined, t)
}

func TestValueStringBuilder(t *testing.T) {
	t.Run("substringASCII", func(t *testing.T) {
		t.Parallel()
		var sb StringBuilder
		str := newStringValue("a\U00010000b")
		sb.WriteSubstring(str, 0, 1)
		res := sb.String()
		if res != asciiString("a") {
			t.Fatal(res)
		}
	})

	t.Run("substringASCIIPure", func(t *testing.T) {
		t.Parallel()
		var sb StringBuilder
		str := newStringValue("ab")
		sb.WriteSubstring(str, 0, 1)
		res := sb.String()
		if res != asciiString("a") {
			t.Fatal(res)
		}
	})

	t.Run("substringUnicode", func(t *testing.T) {
		t.Parallel()
		var sb StringBuilder
		str := newStringValue("a\U00010000b")
		sb.WriteSubstring(str, 1, 3)
		res := sb.String()
		if !res.SameAs(unicodeStringFromRunes([]rune{0x10000})) {
			t.Fatal(res)
		}
	})

	t.Run("substringASCIIUnicode", func(t *testing.T) {
		t.Parallel()
		var sb StringBuilder
		str := newStringValue("a\U00010000b")
		sb.WriteSubstring(str, 0, 2)
		res := sb.String()
		if !res.SameAs(unicodeStringFromRunes([]rune{'a', 0xD800})) {
			t.Fatal(res)
		}
	})

	t.Run("substringUnicodeASCII", func(t *testing.T) {
		t.Parallel()
		var sb StringBuilder
		str := newStringValue("a\U00010000b")
		sb.WriteSubstring(str, 2, 4)
		res := sb.String()
		if !res.SameAs(unicodeStringFromRunes([]rune{0xDC00, 'b'})) {
			t.Fatal(res)
		}
	})

	t.Run("concatSubstringUnicodeASCII", func(t *testing.T) {
		t.Parallel()
		var sb StringBuilder
		sb.WriteString(newStringValue("юникод"))
		sb.WriteSubstring(asciiString(" ascii"), 0, 6)
		if res := sb.String(); !res.SameAs(newStringValue("юникод ascii")) {
			t.Fatal(res)
		}
	})

	t.Run("concat_ASCII_importedASCII", func(t *testing.T) {
		t.Parallel()
		var sb StringBuilder
		sb.WriteString(asciiString("ascii"))
		sb.WriteString(&importedString{s: " imported_ascii1234567890"})
		s := sb.String()
		if res, ok := s.(asciiString); !ok || res != "ascii imported_ascii1234567890" {
			t.Fatal(s)
		}
	})

	t.Run("concat_ASCII_importedUnicode", func(t *testing.T) {
		t.Parallel()
		var sb StringBuilder
		sb.WriteString(asciiString("ascii"))
		sb.WriteString(&importedString{s: " imported_юникод"})
		s := sb.String()
		if res, ok := s.(unicodeString); !ok || !res.SameAs(newStringValue("ascii imported_юникод")) {
			t.Fatal(s)
		}
	})

}

func TestStringSplit(t *testing.T) {
	const SCRIPT = `
	function test(s, sep, limit, expected) {
		const actual = s.split(sep, limit);
		if (!actual || !compareArray(actual, expected)) {
			throw new Error("s: '" + s + "', sep: '"+ sep +"', limit: "+limit+". actual: " + JSON.stringify(actual) + ", expected: " + JSON.stringify(expected));
		}
	}

	const longUnicode = "ú".repeat(33);
	const longASCII = "a".repeat(33);

	const cases = [
		["", "#", 2, [""]],
		["", "#", undefined, [""]],
		["", "", 2, []],
		["", "ú", undefined, [""]],
		["ú", "ú", undefined, ["", ""]],
		["ú", "ú", -2, ["", ""]],
		["ú", "ú", 1, [""]],
		["ú", "ú", 0, []],
		["a", "ú", undefined, ["a"]],
		["a\u{1F600}b", "", undefined, ["a", "\ud83d", "\ude00", "b"]],
		["a\u{1F600}b", "\ud83d\ude00", undefined, ["a", "b"]],

		[longUnicode, ",", undefined, [longUnicode]],
		[longUnicode, "ú", undefined, Array(34).fill("")],
		[longUnicode, "ú", -2, Array(34).fill("")],
		[longUnicode, "ú", 1, [""]],
		[longUnicode, "ú", 0, []],

		[longASCII, "ú", undefined, [longASCII]],
		[longASCII, "", undefined, Array(longASCII.length).fill("a")],

		["a,aú,úa", ",", undefined, ["a", "aú", "úa"]],
	];

	for (const c of cases) {
		test.apply(null, c);
	}
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestStringReplaceAllEmptySearchUnicode(t *testing.T) {
	const SCRIPT = `
assert.sameValue('aé'.replaceAll('', '-'), '-a-é-');
assert.sameValue('\uD83D\uDE00'.replaceAll('', '-'), '-\uD83D-\uDE00-');
assert.sameValue('é'.repeat(40).replaceAll('', '-'), '-' + 'é-'.repeat(40));
assert.sameValue('aé'.replaceAll('', function(m, pos) { return pos; }), '0a1é2');
var concat = 'é'.repeat(20) + 'x'.repeat(20);
assert.sameValue(concat.replaceAll('', '-'), '-' + 'é-'.repeat(20) + 'x-'.repeat(20));

assert.sameValue('aé'.indexOf('', 2), 2);
assert.sameValue('aé'.indexOf('', 5), 2);
assert.sameValue('aé'.includes('', 2), true);
assert.sameValue('aé'.indexOf('é', 2), -1);
assert.sameValue('aéaé'.replaceAll('é', '-'), 'a-a-');
`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestStringFromCodePoint(t *testing.T) {
	const SCRIPT = `
var one = (function() { var x = -0; x++; return x; })();

assert.sameValue(String.fromCodePoint(-0), "\0", "-0");
assert.sameValue(String.fromCodePoint("-0"), "\0", "string -0");
assert.sameValue(String.fromCodePoint(one), "\x01", "float 1");
assert.sameValue(String.fromCodePoint(0x10FFFF), "\uDBFF\uDFFF", "0x10FFFF");

assert.throws(RangeError, () => { String.fromCodePoint(-1); }, "-1");
assert.throws(RangeError, () => { String.fromCodePoint(0.5); }, "0.5");
assert.throws(RangeError, () => { String.fromCodePoint(-0.5); }, "-0.5");
assert.throws(RangeError, () => { String.fromCodePoint(0x110000); }, "0x110000");
assert.throws(RangeError, () => { String.fromCodePoint(NaN); }, "NaN");
assert.throws(RangeError, () => { String.fromCodePoint(Infinity); }, "Infinity");
assert.throws(RangeError, () => { String.fromCodePoint(-Infinity); }, "-Infinity");
`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestStringWellFormed(t *testing.T) {
	const SCRIPT = `
var lo = '\uD834';
var hi = '\uDF06';
var pair = lo + hi;

assert(''.isWellFormed(), "empty");
assert('abc'.isWellFormed(), "ascii");
assert(('a' + pair + 'b').isWellFormed(), "pair");
assert(!(lo + 'a').isWellFormed(), "lone leading");
assert(!('a' + hi).isWellFormed(), "lone trailing");
assert(!(hi + lo).isWellFormed(), "reversed pair");

assert.sameValue('abc'.toWellFormed(), 'abc', "ascii");
assert.sameValue(('a' + pair).toWellFormed(), 'a' + pair, "pair");
assert.sameValue((lo + 'a' + pair + hi + lo).toWellFormed(), '\uFFFDa' + pair + '\uFFFD\uFFFD', "lone");
assert.sameValue((hi + lo).toWellFormed(), '\uFFFD\uFFFD', "reversed pair");
`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestStringHTMLMethods(t *testing.T) {
	const SCRIPT = `
assert.sameValue('x'.anchor('a"b"'), '<a name="a&quot;b&quot;">x</a>', "anchor");
assert.sameValue('x'.big(), '<big>x</big>', "big");
assert.sameValue('x'.blink(), '<blink>x</blink>', "blink");
assert.sameValue('x'.bold(), '<b>x</b>', "bold");
assert.sameValue('x'.fixed(), '<tt>x</tt>', "fixed");
assert.sameValue('x'.fontcolor('red'), '<font color="red">x</font>', "fontcolor");
assert.sameValue('x'.fontsize(7), '<font size="7">x</font>', "fontsize");
assert.sameValue('x'.italics(), '<i>x</i>', "italics");
assert.sameValue('x'.link('"'), '<a href="&quot;">x</a>', "link");
assert.sameValue('x'.small(), '<small>x</small>', "small");
assert.sameValue('x'.strike(), '<strike>x</strike>', "strike");
assert.sameValue('x'.sub(), '<sub>x</sub>', "sub");
assert.sameValue('x'.sup(), '<sup>x</sup>', "sup");

assert.sameValue('x'.anchor(), '<a name="undefined">x</a>', "no argument");
assert.sameValue('é'.bold(), '<b>é</b>', "unicode this");
assert.sameValue('x'.link('é"é"'), '<a href="é&quot;é&quot;">x</a>', "unicode attribute");
assert.throws(TypeError, () => { 'x'.anchor(Symbol()); }, "symbol attribute");
`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestNormalizeUnpairedSurrogates(t *testing.T) {
	for _, tc := range []struct{ name, source, composed, decomposed string }{
		{"lone-high", `"\ud800"`, "[55296]", "[55296]"},
		{"lone-low", `"\udfff"`, "[57343]", "[57343]"},
		{"separated", `"\ud800e\u0301\udfff"`, "[55296,233,57343]", "[55296,101,769,57343]"},
		{"valid-pair", `"\ud83d\ude42"`, "[55357,56898]", "[55357,56898]"},
		{"combining", `"e\u0301"`, "[233]", "[101,769]"},
		{"replacement", `"\ufffd"`, "[65533]", "[65533]"},
	} {
		for _, form := range []string{"NFC", "NFD", "NFKC", "NFKD"} {
			t.Run(tc.name+"/"+form, func(t *testing.T) {
				src := fmt.Sprintf(`(function(){var s=(%s).normalize(%q);var units=[];for(var i=0;i<s.length;i++)units.push(s.charCodeAt(i));return JSON.stringify(units);})()`, tc.source, form)
				v, err := New().RunString(src)
				if err != nil {
					t.Fatal(err)
				}
				want := tc.composed
				if form == "NFD" || form == "NFKD" {
					want = tc.decomposed
				}
				if v.String() != want {
					t.Errorf("code units=%s want=%s", v.String(), want)
				}
			})
		}
	}
}

func TestNormalizeSurrogatesInConcatenatedStrings(t *testing.T) {
	const SCRIPT = `
	var prefix = "x".repeat(40);
	for (var form of ["NFC", "NFD", "NFKC", "NFKD"]) {
		var suffix = form === "NFD" || form === "NFKD" ? "e\u0301" : "\u00e9";
		assert.sameValue((prefix + "\ud800e\u0301\udfff").normalize(form), prefix + "\ud800" + suffix + "\udfff");
	}
	// Normalize the whole concatenation: this valid pair spans two operands.
	var left = prefix + "\ud835";
	var right = "\udc00e\u0301";
	assert.sameValue((left + right).normalize("NFKC"), prefix + "A\u00e9");
	assert.sameValue("\ud800e\u0301".normalize(), "\ud800\u00e9");
	assert.sameValue("\ud800e\u0301".normalize(undefined), "\ud800\u00e9");
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestNormalizeImportedString(t *testing.T) {
	r := New()
	if err := r.Set("text", "e\u0301"); err != nil {
		t.Fatal(err)
	}
	testScript := `text.normalize() === "\u00e9" && text.normalize("NFD") === "e\u0301"`
	v, err := r.RunString(testScript)
	if err != nil || !v.ToBoolean() {
		t.Fatalf("normalizing an imported Go string: %v, %v", v, err)
	}
}

func BenchmarkNormalize(b *testing.B) {
	for _, tc := range []struct {
		name string
		text String
	}{
		{"ASCII", asciiString("ASCII normalization control")},
		{"Unicode", StringFromUTF16(utf16.Encode([]rune(strings.Repeat("e\u0301", 16))))},
	} {
		b.Run(tc.name, func(b *testing.B) {
			r := New()
			call := FunctionCall{This: tc.text, Arguments: []Value{asciiString("NFC")}}
			b.ReportAllocs()
			for b.Loop() {
				r.stringproto_normalize(call)
			}
		})
	}
}
