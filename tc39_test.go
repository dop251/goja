package goja

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"gopkg.in/yaml.v2"
)

const (
	tc39BASE = "testdata/test262"
)

var (
	invalidFormatError = errors.New("Invalid file format")

	ignorableTestError = newSymbol(stringEmpty)
)

var (
	skipPrefixes prefixList

	skipList = map[string]bool{

		// out-of-date (https://github.com/tc39/test262/issues/3407)
		"test/language/expressions/prefix-increment/S11.4.4_A6_T3.js":        true,
		"test/language/expressions/prefix-increment/S11.4.4_A6_T2.js":        true,
		"test/language/expressions/prefix-increment/S11.4.4_A6_T1.js":        true,
		"test/language/expressions/prefix-decrement/S11.4.5_A6_T3.js":        true,
		"test/language/expressions/prefix-decrement/S11.4.5_A6_T2.js":        true,
		"test/language/expressions/prefix-decrement/S11.4.5_A6_T1.js":        true,
		"test/language/expressions/postfix-increment/S11.3.1_A6_T3.js":       true,
		"test/language/expressions/postfix-increment/S11.3.1_A6_T2.js":       true,
		"test/language/expressions/postfix-increment/S11.3.1_A6_T1.js":       true,
		"test/language/expressions/postfix-decrement/S11.3.2_A6_T3.js":       true,
		"test/language/expressions/postfix-decrement/S11.3.2_A6_T2.js":       true,
		"test/language/expressions/postfix-decrement/S11.3.2_A6_T1.js":       true,
		"test/language/expressions/compound-assignment/S11.13.2_A7.1_T4.js":  true,
		"test/language/expressions/compound-assignment/S11.13.2_A7.1_T2.js":  true,
		"test/language/expressions/compound-assignment/S11.13.2_A7.1_T1.js":  true,
		"test/language/expressions/compound-assignment/S11.13.2_A7.11_T4.js": true,
		"test/language/expressions/compound-assignment/S11.13.2_A7.11_T2.js": true,
		"test/language/expressions/compound-assignment/S11.13.2_A7.11_T1.js": true,
		"test/language/expressions/compound-assignment/S11.13.2_A7.10_T4.js": true,
		"test/language/expressions/compound-assignment/S11.13.2_A7.10_T2.js": true,
		"test/language/expressions/compound-assignment/S11.13.2_A7.10_T1.js": true,
		"test/language/expressions/compound-assignment/S11.13.2_A7.9_T4.js":  true,
		"test/language/expressions/compound-assignment/S11.13.2_A7.9_T2.js":  true,
		"test/language/expressions/compound-assignment/S11.13.2_A7.9_T1.js":  true,
		"test/language/expressions/compound-assignment/S11.13.2_A7.8_T4.js":  true,
		"test/language/expressions/compound-assignment/S11.13.2_A7.8_T2.js":  true,
		"test/language/expressions/compound-assignment/S11.13.2_A7.8_T1.js":  true,
		"test/language/expressions/compound-assignment/S11.13.2_A7.7_T4.js":  true,
		"test/language/expressions/compound-assignment/S11.13.2_A7.7_T2.js":  true,
		"test/language/expressions/compound-assignment/S11.13.2_A7.7_T1.js":  true,
		"test/language/expressions/compound-assignment/S11.13.2_A7.6_T4.js":  true,
		"test/language/expressions/compound-assignment/S11.13.2_A7.6_T2.js":  true,
		"test/language/expressions/compound-assignment/S11.13.2_A7.6_T1.js":  true,
		"test/language/expressions/compound-assignment/S11.13.2_A7.5_T4.js":  true,
		"test/language/expressions/compound-assignment/S11.13.2_A7.5_T2.js":  true,
		"test/language/expressions/compound-assignment/S11.13.2_A7.5_T1.js":  true,
		"test/language/expressions/compound-assignment/S11.13.2_A7.4_T4.js":  true,
		"test/language/expressions/compound-assignment/S11.13.2_A7.4_T2.js":  true,
		"test/language/expressions/compound-assignment/S11.13.2_A7.4_T1.js":  true,
		"test/language/expressions/compound-assignment/S11.13.2_A7.3_T4.js":  true,
		"test/language/expressions/compound-assignment/S11.13.2_A7.3_T2.js":  true,
		"test/language/expressions/compound-assignment/S11.13.2_A7.3_T1.js":  true,
		"test/language/expressions/compound-assignment/S11.13.2_A7.2_T4.js":  true,
		"test/language/expressions/compound-assignment/S11.13.2_A7.2_T2.js":  true,
		"test/language/expressions/compound-assignment/S11.13.2_A7.2_T1.js":  true,
		"test/language/expressions/assignment/S11.13.1_A7_T3.js":             true,

		// timezone
		"test/built-ins/Date/prototype/toISOString/15.9.5.43-0-8.js":  true,
		"test/built-ins/Date/prototype/toISOString/15.9.5.43-0-9.js":  true,
		"test/built-ins/Date/prototype/toISOString/15.9.5.43-0-10.js": true,

		// floating point date calculations
		"test/built-ins/Date/UTC/fp-evaluation-order.js": true,

		// quantifier integer limit in regexp
		"test/built-ins/RegExp/quantifier-integer-limit.js": true,

		// GetFunctionRealm
		"test/built-ins/Function/internals/Construct/base-ctor-revoked-proxy.js": true,

		// Uses deprecated __lookupGetter__/__lookupSetter__
		"test/language/expressions/class/elements/private-getter-is-not-a-own-property.js": true,
		"test/language/expressions/class/elements/private-setter-is-not-a-own-property.js": true,
		"test/language/statements/class/elements/private-setter-is-not-a-own-property.js":  true,
		"test/language/statements/class/elements/private-getter-is-not-a-own-property.js":  true,

		// restricted unicode regexp syntax
		"test/built-ins/RegExp/unicode_restricted_quantifiable_assertion.js":         true,
		"test/built-ins/RegExp/unicode_restricted_octal_escape.js":                   true,
		"test/built-ins/RegExp/unicode_restricted_incomple_quantifier.js":            true,
		"test/built-ins/RegExp/unicode_restricted_incomplete_quantifier.js":          true,
		"test/built-ins/RegExp/unicode_restricted_identity_escape_x.js":              true,
		"test/built-ins/RegExp/unicode_restricted_identity_escape_u.js":              true,
		"test/built-ins/RegExp/unicode_restricted_identity_escape_c.js":              true,
		"test/built-ins/RegExp/unicode_restricted_identity_escape_alpha.js":          true,
		"test/built-ins/RegExp/unicode_restricted_identity_escape.js":                true,
		"test/built-ins/RegExp/unicode_restricted_brackets.js":                       true,
		"test/built-ins/RegExp/unicode_restricted_character_class_escape.js":         true,
		"test/annexB/built-ins/RegExp/prototype/compile/pattern-string-invalid-u.js": true,

		// Because goja parser works in UTF-8 it is not possible to pass strings containing invalid UTF-16 code points.
		// This is mitigated by escaping them as \uXXXX, however because of this the RegExp source becomes
		// `\uXXXX` instead of `<the actual UTF-16 code point of XXXX>`.
		// The resulting RegExp will work exactly the same, but it causes these two tests to fail.
		"test/annexB/built-ins/RegExp/RegExp-leading-escape-BMP.js":  true,
		"test/annexB/built-ins/RegExp/RegExp-trailing-escape-BMP.js": true,
		"test/language/literals/regexp/S7.8.5_A1.4_T2.js":            true,
		"test/language/literals/regexp/S7.8.5_A1.1_T2.js":            true,
		"test/language/literals/regexp/S7.8.5_A2.1_T2.js":            true,
		"test/language/literals/regexp/S7.8.5_A2.4_T2.js":            true,

		// async generator
		"test/language/expressions/optional-chaining/member-expression.js":                                                                            true,
		"test/language/expressions/class/elements/same-line-async-method-rs-static-async-generator-method-privatename-identifier-alt.js":              true,
		"test/language/expressions/class/elements/same-line-async-method-rs-static-async-generator-method-privatename-identifier.js":                  true,
		"test/language/destructuring/binding/syntax/destructuring-object-parameters-function-arguments-length.js":                                     true,
		"test/language/destructuring/binding/syntax/destructuring-array-parameters-function-arguments-length.js":                                      true,
		"test/language/comments/hashbang/function-constructor.js":                                                                                     true,
		"test/language/statements/class/elements/after-same-line-static-async-method-rs-static-async-generator-method-privatename-identifier.js":      true,
		"test/language/statements/class/elements/after-same-line-static-async-method-rs-static-async-generator-method-privatename-identifier-alt.js":  true,
		"test/language/statements/class/elements/same-line-async-method-rs-static-async-generator-method-privatename-identifier.js":                   true,
		"test/language/statements/class/elements/same-line-async-method-rs-static-async-generator-method-privatename-identifier-alt.js":               true,
		"test/language/expressions/class/elements/after-same-line-static-async-method-rs-static-async-generator-method-privatename-identifier-alt.js": true,
		"test/language/expressions/class/elements/after-same-line-static-async-method-rs-static-async-generator-method-privatename-identifier.js":     true,
		"test/built-ins/Object/seal/seal-asyncgeneratorfunction.js":                                                                                   true,
		"test/language/statements/switch/scope-lex-async-generator.js":                                                                                true,
		"test/language/statements/class/elements/private-async-generator-method-name.js":                                                              true,
		"test/language/expressions/class/elements/private-async-generator-method-name.js":                                                             true,
		"test/language/expressions/async-generator/name.js":                                                                                           true,
		"test/language/statements/class/elements/same-line-gen-rs-static-async-generator-method-privatename-identifier.js":                            true,
		"test/language/statements/class/elements/same-line-gen-rs-static-async-generator-method-privatename-identifier-alt.js":                        true,
		"test/language/statements/class/elements/new-sc-line-gen-rs-static-async-generator-method-privatename-identifier.js":                          true,
		"test/language/statements/class/elements/new-sc-line-gen-rs-static-async-generator-method-privatename-identifier-alt.js":                      true,
		"test/language/statements/class/elements/after-same-line-static-gen-rs-static-async-generator-method-privatename-identifier.js":               true,
		"test/language/statements/class/elements/after-same-line-static-gen-rs-static-async-generator-method-privatename-identifier-alt.js":           true,
		"test/language/statements/class/elements/after-same-line-gen-rs-static-async-generator-method-privatename-identifier.js":                      true,
		"test/language/statements/class/elements/after-same-line-gen-rs-static-async-generator-method-privatename-identifier-alt.js":                  true,
		"test/language/expressions/class/elements/same-line-gen-rs-static-async-generator-method-privatename-identifier.js":                           true,
		"test/language/expressions/class/elements/same-line-gen-rs-static-async-generator-method-privatename-identifier-alt.js":                       true,
		"test/language/expressions/class/elements/new-sc-line-gen-rs-static-async-generator-method-privatename-identifier.js":                         true,
		"test/language/expressions/class/elements/new-sc-line-gen-rs-static-async-generator-method-privatename-identifier-alt.js":                     true,
		"test/language/expressions/class/elements/after-same-line-static-gen-rs-static-async-generator-method-privatename-identifier.js":              true,
		"test/language/expressions/class/elements/after-same-line-static-gen-rs-static-async-generator-method-privatename-identifier-alt.js":          true,
		"test/language/expressions/class/elements/after-same-line-gen-rs-static-async-generator-method-privatename-identifier.js":                     true,
		"test/language/expressions/class/elements/after-same-line-gen-rs-static-async-generator-method-privatename-identifier-alt.js":                 true,
		"test/built-ins/GeneratorFunction/is-a-constructor.js":                                                                                        true,

		// async iterator
		"test/language/expressions/optional-chaining/iteration-statement-for-await-of.js": true,

		// legacy number literals
		"test/language/literals/numeric/non-octal-decimal-integer.js": true,
		"test/language/literals/string/S7.8.4_A4.3_T2.js":             true,
		"test/language/literals/string/S7.8.4_A4.3_T1.js":             true,

		// integer separators
		"test/language/expressions/object/cpn-obj-lit-computed-property-name-from-integer-separators.js":                  true,
		"test/language/expressions/class/cpn-class-expr-accessors-computed-property-name-from-integer-separators.js":      true,
		"test/language/statements/class/cpn-class-decl-fields-computed-property-name-from-integer-separators.js":          true,
		"test/language/statements/class/cpn-class-decl-computed-property-name-from-integer-separators.js":                 true,
		"test/language/statements/class/cpn-class-decl-accessors-computed-property-name-from-integer-separators.js":       true,
		"test/language/statements/class/cpn-class-decl-fields-methods-computed-property-name-from-integer-separators.js":  true,
		"test/language/expressions/class/cpn-class-expr-fields-computed-property-name-from-integer-separators.js":         true,
		"test/language/expressions/class/cpn-class-expr-computed-property-name-from-integer-separators.js":                true,
		"test/language/expressions/class/cpn-class-expr-fields-methods-computed-property-name-from-integer-separators.js": true,

		// BigInt
		"test/built-ins/Object/seal/seal-biguint64array.js": true,
		"test/built-ins/Object/seal/seal-bigint64array.js":  true,

		// Regexp
		"test/language/literals/regexp/invalid-range-negative-lookbehind.js":    true,
		"test/language/literals/regexp/invalid-range-lookbehind.js":             true,
		"test/language/literals/regexp/invalid-optional-negative-lookbehind.js": true,
		"test/language/literals/regexp/invalid-optional-lookbehind.js":          true,

		// FIXME bugs

		// Left-hand side as a CoverParenthesizedExpression
		"test/language/expressions/assignment/fn-name-lhs-cover.js": true,

		// Character \ missing from character class [\c]
		"test/annexB/built-ins/RegExp/RegExp-invalid-control-escape-character-class.js": true,
		"test/annexB/built-ins/RegExp/RegExp-control-escape-russian-letter.js":          true,

		// Skip due to regexp named groups
		"test/built-ins/String/prototype/replaceAll/searchValue-replacer-RegExp-call.js": true,
	}

	featuresBlackList = []string{
		"async-iteration",
		"Symbol.asyncIterator",
		"BigInt",
		"resizable-arraybuffer",
		"regexp-named-groups",
		"regexp-dotall",
		"regexp-unicode-property-escapes",
		"regexp-match-indices",
		"legacy-regexp",
		"tail-call-optimization",
		"Temporal",
		"import-assertions",
		"dynamic-import",
		"logical-assignment-operators",
		"import.meta",
		"Atomics",
		"Atomics.waitAsync",
		"FinalizationRegistry",
		"WeakRef",
		"numeric-separator-literal",
		"__getter__",
		"__setter__",
		"ShadowRealm",
		"SharedArrayBuffer",
		"error-cause",
		"decorators",
		"regexp-v-flag",
	}
)

func init() {

	skip := func(prefixes ...string) {
		for _, prefix := range prefixes {
			skipPrefixes.Add(prefix)
		}
	}

	skip(
		// Go 1.16 only supports unicode 13
		"test/language/identifiers/start-unicode-14.",
		"test/language/identifiers/part-unicode-14.",

		// generators and async generators (harness/hidden-constructors.js)
		"test/built-ins/Async",

		// async generators
		"test/language/statements/class/elements/wrapped-in-sc-rs-static-async-generator-",
		"test/language/statements/class/elements/same-line-method-rs-static-async-generator-",
		"test/language/statements/class/elements/regular-definitions-rs-static-async-generator-",
		"test/language/statements/class/elements/private-static-async-generator-",
		"test/language/statements/class/elements/new-sc-line-method-rs-static-async-generator-",
		"test/language/statements/class/elements/multiple-stacked-definitions-rs-static-async-generator-",
		"test/language/statements/class/elements/new-no-sc-line-method-rs-static-async-generator-",
		"test/language/statements/class/elements/multiple-definitions-rs-static-async-generator-",
		"test/language/statements/class/elements/after-same-line-static-method-rs-static-async-generator-",
		"test/language/statements/class/elements/after-same-line-method-rs-static-async-generator-",
		"test/language/statements/class/elements/after-same-line-static-method-rs-static-async-generator-",

		"test/language/expressions/class/elements/wrapped-in-sc-rs-static-async-generator-",
		"test/language/expressions/class/elements/same-line-method-rs-static-async-generator-",
		"test/language/expressions/class/elements/regular-definitions-rs-static-async-generator-",
		"test/language/expressions/class/elements/private-static-async-generator-",
		"test/language/expressions/class/elements/new-sc-line-method-rs-static-async-generator-",
		"test/language/expressions/class/elements/multiple-stacked-definitions-rs-static-async-generator-",
		"test/language/expressions/class/elements/new-no-sc-line-method-rs-static-async-generator-",
		"test/language/expressions/class/elements/multiple-definitions-rs-static-async-generator-",
		"test/language/expressions/class/elements/after-same-line-static-method-rs-static-async-generator-",
		"test/language/expressions/class/elements/after-same-line-method-rs-static-async-generator-",
		"test/language/expressions/class/elements/after-same-line-static-method-rs-static-async-generator-",

		"test/language/eval-code/direct/async-gen-",

		// BigInt
		"test/built-ins/TypedArrayConstructors/BigUint64Array/",
		"test/built-ins/TypedArrayConstructors/BigInt64Array/",

		// restricted unicode regexp syntax
		"test/language/literals/regexp/u-",

		// legacy octal escape in strings in strict mode
		"test/language/literals/string/legacy-octal-",
		"test/language/literals/string/legacy-non-octal-",

		// modules
		"test/language/export/",
		"test/language/import/",
		"test/language/module-code/",
	)

}

type tc39Test struct {
	name string
	f    func(t *testing.T)
}

type tc39BenchmarkItem struct {
	name     string
	duration time.Duration
}

type tc39BenchmarkData []tc39BenchmarkItem

type tc39TestCtx struct {
	base         string
	t            *testing.T
	prgCache     map[string]*Program
	prgCacheLock sync.Mutex
	enableBench  bool
	benchmark    tc39BenchmarkData
	benchLock    sync.Mutex
	sabStub      *Program
	//lint:ignore U1000 Only used with race
	testQueue []tc39Test
}

type TC39MetaNegative struct {
	Phase, Type string
}

type tc39Meta struct {
	Negative TC39MetaNegative
	Includes []string
	Flags    []string
	Features []string
	Es5id    string
	Es6id    string
	Esid     string
}

type prefixList struct {
	prefixes map[int]map[string]struct{}
}

func (pl *prefixList) Add(prefix string) {
	l := pl.prefixes[len(prefix)]
	if l == nil {
		l = make(map[string]struct{})
		if pl.prefixes == nil {
			pl.prefixes = make(map[int]map[string]struct{})
		}
		pl.prefixes[len(prefix)] = l
	}
	l[prefix] = struct{}{}
}

func (pl *prefixList) Match(s string) bool {
	for l, prefixes := range pl.prefixes {
		if len(s) >= l {
			if _, exists := prefixes[s[:l]]; exists {
				return true
			}
		}
	}
	return false
}

func (m *tc39Meta) hasFlag(flag string) bool {
	for _, f := range m.Flags {
		if f == flag {
			return true
		}
	}
	return false
}

func parseTC39File(name string) (*tc39Meta, string, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return nil, "", err
	}

	str := string(b)
	metaStart := strings.Index(str, "/*---")
	if metaStart == -1 {
		return nil, "", invalidFormatError
	} else {
		metaStart += 5
	}
	metaEnd := strings.Index(str, "---*/")
	if metaEnd == -1 || metaEnd <= metaStart {
		return nil, "", invalidFormatError
	}

	var meta tc39Meta
	err = yaml.Unmarshal([]byte(str[metaStart:metaEnd]), &meta)
	if err != nil {
		return nil, "", err
	}

	if meta.Negative.Type != "" && meta.Negative.Phase == "" {
		return nil, "", errors.New("negative type is set, but phase isn't")
	}

	return &meta, str, nil
}

func (*tc39TestCtx) detachArrayBuffer(call FunctionCall) Value {
	if obj, ok := call.Argument(0).(*Object); ok {
		if buf, ok := obj.self.(*arrayBufferObject); ok {
			buf.detach()
			return _undefined
		}
	}
	panic(typeError("detachArrayBuffer() is called with incompatible argument"))
}

func (*tc39TestCtx) throwIgnorableTestError(FunctionCall) Value {
	panic(ignorableTestError)
}

func (ctx *tc39TestCtx) runTC39Test(name, src string, meta *tc39Meta, t testing.TB) {
	defer func() {
		if x := recover(); x != nil {
			panic(fmt.Sprintf("panic while running %s: %v", name, x))
		}
	}()
	vm := New()
	_262 := vm.NewObject()
	_262.Set("detachArrayBuffer", ctx.detachArrayBuffer)
	_262.Set("createRealm", ctx.throwIgnorableTestError)
	_262.Set("evalScript", func(call FunctionCall) Value {
		script := call.Argument(0).String()
		result, err := vm.RunString(script)
		if err != nil {
			panic(err)
		}
		return result
	})
	vm.Set("$262", _262)
	vm.Set("IgnorableTestError", ignorableTestError)
	vm.RunProgram(ctx.sabStub)
	var out []string
	async := meta.hasFlag("async")
	if async {
		err := ctx.runFile(ctx.base, path.Join("harness", "doneprintHandle.js"), vm)
		if err != nil {
			t.Fatal(err)
		}
		vm.Set("print", func(msg string) {
			out = append(out, msg)
		})
	} else {
		vm.Set("print", t.Log)
	}

	err, early := ctx.runTC39Script(name, src, meta.Includes, vm)

	if err != nil {
		if meta.Negative.Type == "" {
			if err, ok := err.(*Exception); ok {
				if err.Value() == ignorableTestError {
					t.Skip("Test threw IgnorableTestError")
				}
			}
			t.Fatalf("%s: %v", name, err)
		} else {
			if (meta.Negative.Phase == "early" || meta.Negative.Phase == "parse") && !early || meta.Negative.Phase == "runtime" && early {
				t.Fatalf("%s: error %v happened at the wrong phase (expected %s)", name, err, meta.Negative.Phase)
			}
			var errType string

			switch err := err.(type) {
			case *Exception:
				if o, ok := err.Value().(*Object); ok {
					if c := o.Get("constructor"); c != nil {
						if c, ok := c.(*Object); ok {
							errType = c.Get("name").String()
						} else {
							t.Fatalf("%s: error constructor is not an object (%v)", name, o)
						}
					} else {
						t.Fatalf("%s: error does not have a constructor (%v)", name, o)
					}
				} else {
					t.Fatalf("%s: error is not an object (%v)", name, err.Value())
				}
			case *CompilerSyntaxError:
				errType = "SyntaxError"
			case *CompilerReferenceError:
				errType = "ReferenceError"
			default:
				t.Fatalf("%s: error is not a JS error: %v", name, err)
			}

			if errType != meta.Negative.Type {
				vm.vm.prg.dumpCode(t.Logf)
				t.Fatalf("%s: unexpected error type (%s), expected (%s)", name, errType, meta.Negative.Type)
			}
		}
	} else {
		if meta.Negative.Type != "" {
			vm.vm.prg.dumpCode(t.Logf)
			t.Fatalf("%s: Expected error: %v", name, err)
		}
	}

	if vm.vm.sp != 0 {
		t.Fatalf("sp: %d", vm.vm.sp)
	}

	if l := len(vm.vm.iterStack); l > 0 {
		t.Fatalf("iter stack is not empty: %d", l)
	}
	if async {
		complete := false
		for _, line := range out {
			if strings.HasPrefix(line, "Test262:AsyncTestFailure:") {
				t.Fatal(line)
			} else if line == "Test262:AsyncTestComplete" {
				complete = true
			}
		}
		if !complete {
			for _, line := range out {
				t.Log(line)
			}
			t.Fatal("Test262:AsyncTestComplete was not printed")
		}
	}
}

func (ctx *tc39TestCtx) runTC39File(name string, t testing.TB) {
	if skipList[name] {
		t.Skip("Excluded")
	}
	if skipPrefixes.Match(name) {
		t.Skip("Excluded")
	}
	p := path.Join(ctx.base, name)
	meta, src, err := parseTC39File(p)
	if err != nil {
		//t.Fatalf("Could not parse %s: %v", name, err)
		t.Errorf("Could not parse %s: %v", name, err)
		return
	}
	if meta.hasFlag("module") {
		t.Skip("module")
	}
	if meta.Es5id == "" {
		for _, feature := range meta.Features {
			for _, bl := range featuresBlackList {
				if feature == bl {
					t.Skip("Blacklisted feature")
				}
			}
		}
	}

	var startTime time.Time
	if ctx.enableBench {
		startTime = time.Now()
	}

	hasRaw := meta.hasFlag("raw")

	if hasRaw || !meta.hasFlag("onlyStrict") {
		//log.Printf("Running normal test: %s", name)
		t.Logf("Running normal test: %s", name)
		ctx.runTC39Test(name, src, meta, t)
	}

	if !hasRaw && !meta.hasFlag("noStrict") {
		//log.Printf("Running strict test: %s", name)
		t.Logf("Running strict test: %s", name)
		ctx.runTC39Test(name, "'use strict';\n"+src, meta, t)
	}

	if ctx.enableBench {
		ctx.benchLock.Lock()
		ctx.benchmark = append(ctx.benchmark, tc39BenchmarkItem{
			name:     name,
			duration: time.Since(startTime),
		})
		ctx.benchLock.Unlock()
	}

}

func (ctx *tc39TestCtx) init() {
	ctx.prgCache = make(map[string]*Program)
	ctx.sabStub = MustCompile("sabStub.js", `
		Object.defineProperty(this, "SharedArrayBuffer", {
			get: function() {
				throw IgnorableTestError;
			}
		});`,
		false)
}

func (ctx *tc39TestCtx) compile(base, name string) (*Program, error) {
	ctx.prgCacheLock.Lock()
	defer ctx.prgCacheLock.Unlock()

	prg := ctx.prgCache[name]
	if prg == nil {
		fname := path.Join(base, name)
		f, err := os.Open(fname)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		b, err := io.ReadAll(f)
		if err != nil {
			return nil, err
		}

		str := string(b)
		prg, err = Compile(name, str, false)
		if err != nil {
			return nil, err
		}
		ctx.prgCache[name] = prg
	}

	return prg, nil
}

func (ctx *tc39TestCtx) runFile(base, name string, vm *Runtime) error {
	prg, err := ctx.compile(base, name)
	if err != nil {
		return err
	}
	_, err = vm.RunProgram(prg)
	return err
}

func (ctx *tc39TestCtx) runTC39Script(name, src string, includes []string, vm *Runtime) (err error, early bool) {
	early = true
	err = ctx.runFile(ctx.base, path.Join("harness", "assert.js"), vm)
	if err != nil {
		return
	}

	err = ctx.runFile(ctx.base, path.Join("harness", "sta.js"), vm)
	if err != nil {
		return
	}

	for _, include := range includes {
		err = ctx.runFile(ctx.base, path.Join("harness", include), vm)
		if err != nil {
			return
		}
	}

	var p *Program
	p, err = Compile(name, src, false)

	if err != nil {
		return
	}

	early = false
	_, err = vm.RunProgram(p)

	return
}

func (ctx *tc39TestCtx) runTC39Tests(name string) {
	files, err := os.ReadDir(path.Join(ctx.base, name))
	if err != nil {
		ctx.t.Fatal(err)
	}

	for _, file := range files {
		if file.Name()[0] == '.' {
			continue
		}
		if file.IsDir() {
			ctx.runTC39Tests(path.Join(name, file.Name()))
		} else {
			fileName := file.Name()
			if strings.HasSuffix(fileName, ".js") && !strings.HasSuffix(fileName, "_FIXTURE.js") {
				name := path.Join(name, fileName)
				ctx.runTest(name, func(t *testing.T) {
					ctx.runTC39File(name, t)
				})
			}
		}
	}

}

func TestTC39(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	if _, err := os.Stat(tc39BASE); err != nil {
		t.Skipf("If you want to run tc39 tests, download them from https://github.com/tc39/test262 and put into %s. See .tc39_test262_checkout.sh for the latest working commit id. (%v)", tc39BASE, err)
	}

	ctx := &tc39TestCtx{
		base: tc39BASE,
	}
	ctx.init()
	//ctx.enableBench = true

	t.Run("tc39", func(t *testing.T) {
		ctx.t = t
		//ctx.runTC39File("test/language/types/number/8.5.1.js", t)
		ctx.runTC39Tests("test/language")
		ctx.runTC39Tests("test/built-ins")
		ctx.runTC39Tests("test/annexB/built-ins/String/prototype/substr")
		ctx.runTC39Tests("test/annexB/built-ins/String/prototype/trimLeft")
		ctx.runTC39Tests("test/annexB/built-ins/String/prototype/trimRight")
		ctx.runTC39Tests("test/annexB/built-ins/escape")
		ctx.runTC39Tests("test/annexB/built-ins/unescape")
		ctx.runTC39Tests("test/annexB/built-ins/RegExp")

		ctx.flush()
	})

	if ctx.enableBench {
		sort.Slice(ctx.benchmark, func(i, j int) bool {
			return ctx.benchmark[i].duration > ctx.benchmark[j].duration
		})
		bench := ctx.benchmark
		if len(bench) > 50 {
			bench = bench[:50]
		}
		for _, item := range bench {
			fmt.Printf("%s\t%d\n", item.name, item.duration/time.Millisecond)
		}
	}
}
