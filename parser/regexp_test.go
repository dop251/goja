package parser

import (
	"regexp"
	"testing"
)

func TestRegExp(t *testing.T) {
	tt(t, func() {
		{
			// err
			test := func(input string, expect interface{}) {
				_, err := TransformRegExp(input)
				_, incompat := err.(RegexpErrorIncompatible)
				is(incompat, false)
				is(err, expect)
			}

			test("[", "Unterminated character class")

			test("(", "Unterminated group")

			test("\\(?=)", "Unmatched ')'")

			test(")", "Unmatched ')'")
			test("0:(?)", "Invalid group")
			test("(?)", "Invalid group")
			test("(?U)", "Invalid group")
			test("(?)|(?i)", "Invalid group")
			test("(?P<w>)(?P<w>)(?P<D>)", "Invalid group")
		}

		{
			// incompatible
			test := func(input string, expectErr interface{}) {
				_, err := TransformRegExp(input)
				_, incompat := err.(RegexpErrorIncompatible)
				is(incompat, true)
				is(err, expectErr)
			}

			test(`<%([\s\S]+?)%>`, "S in class")

			test("(?<=y)x", "re2: Invalid (?<) <lookbehind>")

			test(`(?!test)`, "re2: Invalid (?!) <lookahead>")

			test(`\1`, "re2: Invalid \\1 <backreference>")

			test(`\8`, "re2: Invalid \\8 <backreference>")

		}

		{
			// err
			test := func(input string, expect string) {
				result, err := TransformRegExp(input)
				is(err, nil)
				_, incompat := err.(RegexpErrorIncompatible)
				is(incompat, false)
				is(result, expect)
				_, err = regexp.Compile(result)
				is(err, nil)
			}

			test("", "")

			test("abc", "abc")

			test(`\abc`, `abc`)

			test(`\a\b\c`, `a\bc`)

			test(`\x`, `x`)

			test(`\c`, `c`)

			test(`\cA`, `\x01`)

			test(`\cz`, `\x1a`)

			test(`\ca`, `\x01`)

			test(`\cj`, `\x0a`)

			test(`\ck`, `\x0b`)

			test(`\+`, `\+`)

			test(`[\b]`, `[\x08]`)

			test(`\u0z01\x\undefined`, `u0z01xundefined`)

			test(`\\|'|\r|\n|\t|\u2028|\u2029`, `\\|'|\r|\n|\t|\x{2028}|\x{2029}`)

			test("]", "]")

			test("}", "}")

			test("%", "%")

			test("(%)", "(%)")

			test("(?:[%\\s])", "(?:[%"+WhitespaceChars+"])")

			test("[[]", "[[]")

			test("\\101", "\\x41")

			test("\\51", "\\x29")

			test("\\051", "\\x29")

			test("\\175", "\\x7d")

			test("\\0", "\\0")

			test("\\04", "\\x04")

			test(`(.)^`, "([^\\r\\n])^")

			test(`\$`, `\$`)

			test(`[G-b]`, `[G-b]`)

			test(`[G-b\0]`, `[G-b\0]`)

			test(`\k`, `k`)

			test(`\x20`, `\x20`)

			test(`😊`, `😊`)

			test(`^.*`, `^[^\r\n]*`)

			test(`(\n)`, `(\n)`)

			test(`(a(bc))`, `(a(bc))`)

			test(`[]`, "[^\u0000-\U0001FFFF]")

			test(`[^]`, "[\u0000-\U0001FFFF]")

			test(`\s+`, "["+WhitespaceChars+"]+")

			test(`\S+`, "[^"+WhitespaceChars+"]+")

		}
	})
}

func TestTransformRegExp(t *testing.T) {
	tt(t, func() {
		pattern, err := TransformRegExp(`\s+abc\s+`)
		is(err, nil)
		_, incompat := err.(RegexpErrorIncompatible)
		is(incompat, false)
		is(pattern, `[`+WhitespaceChars+`]+abc[`+WhitespaceChars+`]+`)
		is(regexp.MustCompile(pattern).MatchString("\t abc def"), true)
	})
}

func BenchmarkTransformRegExp(b *testing.B) {
	f := func(reStr string, b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = TransformRegExp(reStr)
		}
	}

	b.Run("Re", func(b *testing.B) {
		f(`^(([^<>()\[\]\\.,;:\s@"]+(\.[^<>()\[\]\\.,;:\s@"]+)*)|(".+"))@((\[[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}])|(([a-zA-Z\-0-9]+\.)+[a-zA-Z]{2,}))$`, b)
	})

	b.Run("Re2-1", func(b *testing.B) {
		f(`(?=)^(([^<>()\[\]\\.,;:\s@"]+(\.[^<>()\[\]\\.,;:\s@"]+)*)|(".+"))@((\[[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}])|(([a-zA-Z\-0-9]+\.)+[a-zA-Z]{2,}))$`, b)
	})

	b.Run("Re2-1", func(b *testing.B) {
		f(`^(([^<>()\[\]\\.,;:\s@"]+(\.[^<>()\[\]\\.,;:\s@"]+)*)|(".+"))@((\[[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}])|(([a-zA-Z\-0-9]+\.)+[a-zA-Z]{2,}))$(?=)`, b)
	})
}
