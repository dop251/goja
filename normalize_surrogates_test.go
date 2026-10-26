package goja

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf16"
)

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

var normalizeBenchmarkResult Value

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
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				normalizeBenchmarkResult = r.stringproto_normalize(call)
			}
		})
	}
}
