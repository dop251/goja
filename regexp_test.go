package goja

import (
	"testing"
)

func TestRegexp1(t *testing.T) {
	const SCRIPT = `
	var r = new RegExp("(['\"])(.*?)\\1");
	var m = r.exec("'test'");
	m !== null && m.length == 3 && m[2] === "test";
	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestRegexp2(t *testing.T) {
	const SCRIPT = `
	var r = new RegExp("(['\"])(.*?)['\"]");
	var m = r.exec("'test'");
	m !== null && m.length == 3 && m[2] === "test";
	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestRegexpLiteral(t *testing.T) {
	const SCRIPT = `
	var r = /(['\"])(.*?)\1/;
	var m = r.exec("'test'");
	m !== null && m.length == 3 && m[2] === "test";
	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestRegexpRe2Unicode(t *testing.T) {
	const SCRIPT = `
	var r = /(тест)/i;
	var m = r.exec("'Тест'");
	m !== null && m.length == 2 && m[1] === "Тест";
	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestRegexpRe2UnicodeTarget(t *testing.T) {
	const SCRIPT = `
	var r = /(['\"])(.*?)['\"]/i;
	var m = r.exec("'Тест'");
	m !== null && m.length == 3 && m[2] === "Тест";
	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestRegexpRegexp2Unicode(t *testing.T) {
	const SCRIPT = `
	var r = /(['\"])(тест)\1/i;
	var m = r.exec("'Тест'");
	m !== null && m.length == 3 && m[2] === "Тест";
	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestRegexpRegexp2UnicodeTarget(t *testing.T) {
	const SCRIPT = `
	var r = /(['\"])(.*?)\1/;
	var m = r.exec("'Тест'");
	m !== null && m.length == 3 && m[2] === "Тест";
	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestRegexpRe2Whitespace(t *testing.T) {
	const SCRIPT = `
	"\u2000\u2001\u2002\u200b".replace(/\s+/g, "") === "\u200b";
	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestRegexpRegexp2Whitespace(t *testing.T) {
	const SCRIPT = `
	"A\u2000\u2001\u2002A\u200b".replace(/(A)\s+\1/g, "") === "\u200b"
	`
	testScript1(SCRIPT, valueTrue, t)
}

func TestEmptyCharClassRe2(t *testing.T) {
	const SCRIPT = `
	/[]/.test("\u0000");
	`

	testScript1(SCRIPT, valueFalse, t)
}

func TestNegatedEmptyCharClassRe2(t *testing.T) {
	const SCRIPT = `
	/[^]/.test("\u0000");
	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestEmptyCharClassRegexp2(t *testing.T) {
	const SCRIPT = `
	/([])\1/.test("\u0000\u0000");
	`

	testScript1(SCRIPT, valueFalse, t)
}

func TestRegexp2Negate(t *testing.T) {
	const SCRIPT = `
	/([\D1])\1/.test("aa");
	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestAlternativeRe2(t *testing.T) {
	const SCRIPT = `
	/()|/.exec("") !== null;
	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestRegexpReplaceGlobal(t *testing.T) {
	const SCRIPT = `
	"QBZPbage\ny_cynprubyqre".replace(/^\s*|\s*$/g, '')
	`

	testScript1(SCRIPT, asciiString("QBZPbage\ny_cynprubyqre"), t)
}

func TestRegexpNumCaptures(t *testing.T) {
	const SCRIPT = `
	"Fubpxjnir Synfu 9.0  e115".replace(/([a-zA-Z]|\s)+/, '')
	`
	testScript1(SCRIPT, asciiString("9.0  e115"), t)
}

func TestRegexpNumCaptures1(t *testing.T) {
	const SCRIPT = `
	"Fubpxjnir Sy\tfu 9.0  e115".replace(/^.*\s+(\S+\s+\S+$)/, '')
	`
	testScript1(SCRIPT, asciiString(""), t)
}

func TestRegexpSInClass(t *testing.T) {
	const SCRIPT = `
	/[\S]/.test("\u2028");
	`
	testScript1(SCRIPT, valueFalse, t)
}

func TestRegexpDotMatchSlashR(t *testing.T) {
	const SCRIPT = `
	/./.test("\r");
	`

	testScript1(SCRIPT, valueFalse, t)
}

func TestRegexpDotMatchSlashRInGroup(t *testing.T) {
	const SCRIPT = `
	/(.)/.test("\r");
	`

	testScript1(SCRIPT, valueFalse, t)
}

func TestRegexpSplitWithBackRef(t *testing.T) {
	const SCRIPT = `
	"a++b+-c".split(/([+-])\1/).join(" $$ ")
	`

	testScript1(SCRIPT, asciiString("a $$ + $$ b+-c"), t)
}

func TestEscapeNonASCII(t *testing.T) {
	const SCRIPT = `
	/\⩓/.test("⩓")
	`

	testScript1(SCRIPT, valueTrue, t)
}

func TestRegexpUTF16(t *testing.T) {
	const SCRIPT = `
	var str = "\uD800\uDC00";

	assert(/\uD800/g.test(str), "#1");
	assert(/\uD800/.test(str), "#2");
	assert(/𐀀/.test(str), "#3");

	var re = /\uD800/;

	assert(compareArray(str.replace(re, "X"), ["X", "\uDC00"]), "#4");
	assert(compareArray(str.split(re), ["", "\uDC00"]), "#5");
	assert(compareArray("a\uD800\uDC00b".split(/\uD800/g), ["a", "\uDC00b"]), "#6");
	assert(compareArray("a\uD800\uDC00b".split(/(?:)/g), ["a", "\uD800", "\uDC00", "b"]), "#7");

	re = /(?=)a/; // a hack to use regexp2
	assert.sameValue(re.exec('\ud83d\ude02a').index, 2, "#8");

	assert.sameValue(/./.exec('\ud83d\ude02')[0], '\ud83d', "#9");

	assert(RegExp("\uD800").test("\uD800"), "#10");

	var cu = 0xD800;
	var xx = "a\\" + String.fromCharCode(cu);
	var pattern = eval("/" + xx + "/");
	assert.sameValue(pattern.source, "a\\\\\\ud800", "Code unit: " + cu.toString(16), "#11");
	assert(pattern.test("a\\\uD800"), "#12");
	`

	testScript1(TESTLIB+SCRIPT, _undefined, t)
}

func TestRegexpUnicode(t *testing.T) {
	const SCRIPT = `

	assert(!/\uD800/u.test("\uD800\uDC00"), "#1");
	assert(!/\uFFFD/u.test("\uD800\uDC00"), "#2");

	assert(/\uD800\uDC00/u.test("\uD800\uDC00"), "#3");

	assert(/\uD800/u.test("\uD800"), "#4");

	assert(compareArray("a\uD800\uDC00b".split(/\uD800/gu), ["a\uD800\uDC00b"]), "#5");

	assert(compareArray("a\uD800\uDC00b".split(/(?:)/gu), ["a", "𐀀", "b"]), "#6");

	var re = eval('/' + /\ud834\udf06/u.source + '/u');
	assert(re.test('\ud834\udf06'), "#9");

	/*re = RegExp("\\p{L}", "u");
	if (!re.test("A")) {
		throw new Error("Test 9 failed");
	}*/
	`

	testScript1(TESTLIB+SCRIPT, _undefined, t)
}

func TestConvertRegexpToUnicode(t *testing.T) {
	if s := convertRegexpToUnicode(`test\uD800\u0C00passed`); s != `test\uD800\u0C00passed` {
		t.Fatal(s)
	}
	if s := convertRegexpToUnicode(`test\uD800\uDC00passed`); s != `test𐀀passed` {
		t.Fatal(s)
	}
	if s := convertRegexpToUnicode(`test\u0023passed`); s != `test\u0023passed` {
		t.Fatal(s)
	}
	if s := convertRegexpToUnicode(`test\u0passed`); s != `test\u0passed` {
		t.Fatal(s)
	}
	if s := convertRegexpToUnicode(`test\uD800passed`); s != `test\uD800passed` {
		t.Fatal(s)
	}
	if s := convertRegexpToUnicode(`test\uD800`); s != `test\uD800` {
		t.Fatal(s)
	}
	if s := convertRegexpToUnicode(`test\uD80`); s != `test\uD80` {
		t.Fatal(s)
	}
	if s := convertRegexpToUnicode(`\\uD800\uDC00passed`); s != `\\uD800\uDC00passed` {
		t.Fatal(s)
	}
	if s := convertRegexpToUnicode(`testpassed`); s != `testpassed` {
		t.Fatal(s)
	}
}

func TestConvertRegexpToUtf16(t *testing.T) {
	if s := convertRegexpToUtf16(`𐀀`); s != `\ud800\udc00` {
		t.Fatal(s)
	}
	if s := convertRegexpToUtf16(`\𐀀`); s != `\\\ud800\udc00` {
		t.Fatal(s)
	}
}

func TestEscapeInvalidUtf16(t *testing.T) {
	if s := escapeInvalidUtf16(asciiString("test")); s != "test" {
		t.Fatal(s)
	}
	if s := escapeInvalidUtf16(newStringValue("test\U00010000")); s != "test\U00010000" {
		t.Fatal(s)
	}
	if s := escapeInvalidUtf16(unicodeStringFromRunes([]rune{'t', 0xD800})); s != "t\\ud800" {
		t.Fatal(s)
	}
	if s := escapeInvalidUtf16(unicodeStringFromRunes([]rune{'t', 0xD800, 'p'})); s != "t\\ud800p" {
		t.Fatal(s)
	}
	if s := escapeInvalidUtf16(unicodeStringFromRunes([]rune{0xD800, 'p'})); s != "\\ud800p" {
		t.Fatal(s)
	}
	if s := escapeInvalidUtf16(unicodeStringFromRunes([]rune{'t', '\\', 0xD800, 'p'})); s != `t\\\ud800p` {
		t.Fatal(s)
	}
}

func TestRegexpAssertion(t *testing.T) {
	const SCRIPT = `
	var res = 'aaa'.match(/^a/g);
	res.length === 1 || res[0] === 'a';
	`
	testScript1(SCRIPT, valueTrue, t)
}

func TestRegexpUnicodeAdvanceStringIndex(t *testing.T) {
	const SCRIPT = `
	// deoptimise RegExp
	var origExec = RegExp.prototype.exec;
	RegExp.prototype.exec = function(s) {
		return origExec.call(this, s);
	};

	var re = /(?:)/gu;
	var str = "a\uD800\uDC00b";
	assert(compareArray(str.split(re), ["a", "𐀀", "b"]), "#1");

	re.lastIndex = 3;
	assert.sameValue(re.exec(str).index, 3, "#2");

	re.lastIndex = 2;
	assert.sameValue(re.exec(str).index, 1, "#3");

	re.lastIndex = 4;
	assert.sameValue(re.exec(str).index, 4, "#4");

	re.lastIndex = 5;
	assert.sameValue(re.exec(str), null, "#5");
	`
	testScript1(TESTLIB+SCRIPT, _undefined, t)
}

func TestRegexpInit(t *testing.T) {
	const SCRIPT = `
	RegExp(".").lastIndex;
	`
	testScript1(SCRIPT, intToValue(0), t)
}

func TestRegexpToString(t *testing.T) {
	const SCRIPT = `
	RegExp.prototype.toString.call({
	source: 'foo',
    flags: 'bar'});
	`
	testScript1(SCRIPT, asciiString("/foo/bar"), t)
}

func TestRegexpEscapeSource(t *testing.T) {
	const SCRIPT = `
	/href="(.+?)(\/.*\/\S+?)\/"/.source;
	`
	testScript1(SCRIPT, asciiString(`href="(.+?)(\/.*\/\S+?)\/"`), t)
}

func BenchmarkRegexpSplitWithBackRef(b *testing.B) {
	const SCRIPT = `
	"aaaaaaaaaaaaaaaaaaaaaaaaa++bbbbbbbbbbbbbbbbbbbbbb+-ccccccccccccccccccccccc".split(/([+-])\1/)
	`
	b.StopTimer()
	prg, err := Compile("test.js", SCRIPT, false)
	if err != nil {
		b.Fatal(err)
	}
	vm := New()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		vm.RunProgram(prg)
	}
}
