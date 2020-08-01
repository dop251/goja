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

	if (!/\uD800/g.test(str)) {
		throw new Error("Test 1 failed");
	}
	if (!/\uD800/.test(str)) {
		throw new Error("Test 2 failed");
	}
	if (!/𐀀/.test(str)) {
		throw new Error("Test 3 failed");
	}

	var re = /\uD800/;

	var res = str.replace(re, "X");
	if (res.length !== 2 || res[0] !== "X" || res[1] !== "\uDC00") {
		throw new Error("Test 4 failed");
	}

	res = str.split(re);
	if (res.length !== 2 || res[0] !== "" || res[1] !== "\uDC00") {
		throw new Error("Test 5 failed");
	}

	var res = "a\uD800\uDC00b".split(/\uD800/g);
	if (res.length !== 2 || res[0] !== "a" || res[1] !== "\uDC00b") {
		throw new Error("Test 6 failed");
	}
	`

	testScript1(SCRIPT, _undefined, t)
}

func TestRegexpUnicode(t *testing.T) {
	const SCRIPT = `

	var re = /\uD800/u;
	if (re.test("\uD800\uDC00")) {
		throw new Error("Test 1 failed");
	}

	re = /\uFFFD/u;
	if (re.test("\uD800\uDC00")) {
		throw new Error("Test 2 failed");
	}

	re = /\uD800\uDC00/u;
	if (!re.test("\uD800\uDC00")) {
		throw new Error("Test 3 failed");
	}

	re = /\uD800/u;
	if (!re.test("\uD800")) {
		throw new Error("Test 4 failed");
	}

	var res = 'aaa'.match(/^a/g);
	if (res.length !== 1 || res[0] !== 'a') {
		throw new Error("Test 5 failed");
	}

	re = /(?=)a/; // a hack to use regexp2
	if (re.exec('\ud83d\ude02a').index !== 2) {
		throw new Error("Test 6 failed");
	}

	if (/./.exec('\ud83d\ude02')[0] !== '\ud83d') {
		throw new Error("Test 7 failed");
	}

	var res = "a\uD800\uDC00b".split(/\uD800/gu);
	if (res.length !== 1 || res[0] !== "a\uD800\uDC00b") {
		throw new Error("Test 8 failed");
	}

	/*re = RegExp("\\p{L}", "u");
	if (!re.test("A")) {
		throw new Error("Test 9 failed");
	}*/
	`

	testScript1(SCRIPT, _undefined, t)
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

func TestRegexp2Unicode(t *testing.T) {
	r, err := compileRegexp2(`\p{L}`, false, false)
	if err != nil {
		t.Fatal(err)
	}
	res := r.findSubmatchIndex(newStringValue("A"), 0, true)
	if res == nil {
		t.Fatal("no match")
	}
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
