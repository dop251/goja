package parser

import (
	"fmt"
	"testing"
)

// A keyword is an IdentifierName after a member-access dot. Its token spelling
// must not suppress ASI after the completed member expression.
func TestKeywordMemberASI(t *testing.T) {
	for _, word := range []string{"new", "const", "class", "function", "for", "if", "else", "return", "delete", "typeof", "instanceof", "in", "await", "yield", "true", "null"} {
		for _, separator := range []string{"\n", "\r\n", "\u2028", "/*\n*/"} {
			for _, dot := range []string{".", "?."} {
				source := fmt.Sprintf("const value = obj%s%s%sconst next = 1", dot, word, separator)
				if _, err := ParseFile(nil, "keyword-member.js", source, 0); err != nil {
					t.Errorf("%q: %v", source, err)
				}
			}
		}
	}
}

func TestKeywordMemberStillRejectsMissingSeparator(t *testing.T) {
	for _, source := range []string{"const value = obj.new const next = 1", "const value = obj.class let next = 1"} {
		if _, err := ParseFile(nil, "keyword-member.js", source, 0); err == nil {
			t.Errorf("unexpectedly accepted %q", source)
		}
	}
}

func TestKeywordMemberContinuations(t *testing.T) {
	for _, continuation := range []string{"(1)", "[0]", "+ 1", "/ 2", ".property"} {
		source := "const value = obj.new\n" + continuation
		program, err := ParseFile(nil, "keyword-member.js", source, 0)
		if err != nil {
			t.Errorf("%q: %v", source, err)
			continue
		}
		if len(program.Body) != 1 {
			t.Errorf("continuation split into %d statements: %q", len(program.Body), source)
		}
	}
	program, err := ParseFile(nil, "keyword-member.js", "obj.new\n++count", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Body) != 2 {
		t.Fatalf("newline update must be separate statement: %d", len(program.Body))
	}
}
