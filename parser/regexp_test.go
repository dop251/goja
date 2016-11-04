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
				is(err, expect)
			}

			test("[", "Unterminated character class")

			test("(", "Unterminated group")

			test("\\(?=)", "Unmatched ')'")

			test(")", "Unmatched ')'")
		}

		{
			// err
			test := func(input, expect string, expectErr interface{}) {
				output, err := TransformRegExp(input)
				is(output, expect)
				is(err, expectErr)
			}

			test(")", "", "Unmatched ')'")

			test("\\0", "\\0", nil)

		}

		{
			// err
			test := func(input string, expect string) {
				result, err := TransformRegExp(input)
				is(err, nil)
				is(result, expect)
				_, err = regexp.Compile(result)
				is(err, nil)
			}

			testErr := func(input string, expectErr string) {
				_, err := TransformRegExp(input)
				is(err, expectErr)
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

			test("(?:[%\\s])", "(?:[%" + WhitespaceChars +"])")

			test("[[]", "[[]")

			test("\\101", "\\x41")

			test("\\51", "\\x29")

			test("\\051", "\\x29")

			test("\\175", "\\x7d")

			test("\\04", "\\x04")

			testErr(`<%([\s\S]+?)%>`, "S in class")

			test(`(.)^`, "([^\\r\\n])^")

			test(`\$`, `\$`)

			test(`[G-b]`, `[G-b]`)

			test(`[G-b\0]`, `[G-b\0]`)
		}
	})
}

func TestTransformRegExp(t *testing.T) {
	tt(t, func() {
		pattern, err := TransformRegExp(`\s+abc\s+`)
		is(err, nil)
		is(pattern, `[` + WhitespaceChars + `]+abc[` + WhitespaceChars +`]+`)
		is(regexp.MustCompile(pattern).MatchString("\t abc def"), true)
	})
}
