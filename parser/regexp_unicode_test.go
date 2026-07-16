package parser

import (
	"regexp"
	"testing"
)

// TestUnicodePropertyMatchers verifies the commit that enables \p{...} escapes.
func TestUnicodePropertyMatchers(t *testing.T) {
	t.Run("basic pLd", func(t *testing.T) {
		pattern, err := TransformRegExp(`\p{Letter}`, false, false)
		if err != nil {
			t.Fatalf("TransformRegExp: %v", err)
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			t.Fatalf("compiled pattern is invalid Go regexp: %s — error: %v", pattern, err)
		}
		if !re.MatchString("A") {
			t.Fatal("\\p{Letter} should match 'A'")
		}
	})

	t.Run("basic pd", func(t *testing.T) {
		pattern, err := TransformRegExp(`\p{Nd}`, false, false)
		if err != nil {
			t.Fatalf("TransformRegExp: %v", err)
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			t.Fatalf("compiled pattern is invalid Go regexp: %s — error: %v", pattern, err)
		}
		if !re.MatchString("1") {
			t.Fatal("\\p{Nd} should match '1'")
		}
	})

	t.Run("shortcuts pLl_pLu", func(t *testing.T) {
		pattern, err := TransformRegExp(`\p{Lowercase_Letter}`, false, false)
		if err != nil {
			t.Fatalf("TransformRegExp: %v", err)
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			t.Fatalf("compiled pattern is invalid Go regexp: %s — error: %v", pattern, err)
		}
		if !re.MatchString("a") {
			t.Fatal("\\p{Lowercase_Letter} should match 'a'")
		}

		pattern2, err := TransformRegExp(`\p{Uppercase_Letter}`, false, false)
		if err != nil {
			t.Fatalf("TransformRegExp: %v", err)
		}
		re2, err := regexp.Compile(pattern2)
		if err != nil {
			t.Fatalf("compiled pattern is invalid Go regexp: %s — error: %v", pattern2, err)
		}
		if !re2.MatchString("A") {
			t.Fatal("\\p{Uppercase_Letter} should match 'A'")
		}
	})

	t.Run("combined with other chars", func(t *testing.T) {
		pattern, err := TransformRegExp(`abc\p{Letter}xyz`, false, false)
		if err != nil {
			t.Fatalf("TransformRegExp: %v", err)
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			t.Fatalf("compiled pattern is invalid Go regexp: %s — error: %v", pattern, err)
		}
		if !re.MatchString("abcAxyz") {
			t.Fatal("\\p{Letter} combined with literals should match 'abcAxyz'")
		}
	})

	t.Run("strip backslash on unrecognized escapes like a", func(t *testing.T) {
		// In JS, \a is just "a" — the backslash is dropped.
		pattern, err := TransformRegExp(`\a`, false, false)
		if err != nil {
			t.Fatalf("TransformRegExp: %v", err)
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			t.Fatalf("compiled pattern is invalid Go regexp: %s — error: %v", pattern, err)
		}
		if !re.MatchString("a") {
			t.Fatal("\\a should strip the backslash and match 'a'")
		}
	})
}
