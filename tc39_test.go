package goja

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
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

		// Go 1.14 supports unicode 12
		"test/language/identifiers/start-unicode-13.0.0.js":         true,
		"test/language/identifiers/start-unicode-13.0.0-escaped.js": true,
		"test/language/identifiers/start-unicode-14.0.0.js":         true,
		"test/language/identifiers/start-unicode-14.0.0-escaped.js": true,
		"test/language/identifiers/part-unicode-13.0.0.js":          true,
		"test/language/identifiers/part-unicode-13.0.0-escaped.js":  true,
		"test/language/identifiers/part-unicode-14.0.0.js":          true,
		"test/language/identifiers/part-unicode-14.0.0-escaped.js":  true,

		// class
		"test/built-ins/Array/prototype/concat/Array.prototype.concat_non-array.js":                    true,
		"test/built-ins/ArrayBuffer/isView/arg-is-typedarray-subclass-instance.js":                     true,
		"test/built-ins/ArrayBuffer/isView/arg-is-dataview-subclass-instance.js":                       true,
		"test/language/expressions/object/method-definition/name-invoke-ctor.js":                       true,
		"test/language/expressions/object/method.js":                                                   true,
		"test/language/expressions/object/setter-super-prop.js":                                        true,
		"test/language/expressions/object/getter-super-prop.js":                                        true,
		"test/language/expressions/delete/super-property.js":                                           true,
		"test/language/statements/let/dstr/obj-ptrn-id-init-fn-name-class.js":                          true,
		"test/language/statements/let/dstr/ary-ptrn-elem-id-init-fn-name-class.js":                     true,
		"test/language/statements/for/dstr/var-obj-ptrn-id-init-fn-name-class.js":                      true,
		"test/language/statements/for/dstr/var-ary-ptrn-elem-id-init-fn-name-class.js":                 true,
		"test/language/statements/for/dstr/let-ary-ptrn-elem-id-init-fn-name-class.js":                 true,
		"test/language/statements/for/dstr/let-obj-ptrn-id-init-fn-name-class.js":                      true,
		"test/language/statements/const/dstr/ary-ptrn-elem-id-init-fn-name-class.js":                   true,
		"test/language/statements/for/dstr/const-obj-ptrn-id-init-fn-name-class.js":                    true,
		"test/language/statements/const/dstr/obj-ptrn-id-init-fn-name-class.js":                        true,
		"test/language/statements/for/dstr/const-ary-ptrn-elem-id-init-fn-name-class.js":               true,
		"test/language/statements/variable/dstr/obj-ptrn-id-init-fn-name-class.js":                     true,
		"test/language/statements/variable/dstr/ary-ptrn-elem-id-init-fn-name-class.js":                true,
		"test/language/expressions/object/method-definition/name-name-prop-symbol.js":                  true,
		"test/language/expressions/function/dstr/dflt-obj-ptrn-id-init-fn-name-class.js":               true,
		"test/language/expressions/function/dstr/dflt-ary-ptrn-elem-id-init-fn-name-class.js":          true,
		"test/language/expressions/function/dstr/ary-ptrn-elem-id-init-fn-name-class.js":               true,
		"test/language/expressions/function/dstr/obj-ptrn-id-init-fn-name-class.js":                    true,
		"test/language/statements/function/dstr/dflt-ary-ptrn-elem-id-init-fn-name-class.js":           true,
		"test/language/statements/function/dstr/obj-ptrn-id-init-fn-name-class.js":                     true,
		"test/language/statements/function/dstr/ary-ptrn-elem-id-init-fn-name-class.js":                true,
		"test/language/statements/function/dstr/dflt-obj-ptrn-id-init-fn-name-class.js":                true,
		"test/language/expressions/arrow-function/scope-paramsbody-var-open.js":                        true,
		"test/language/expressions/arrow-function/scope-paramsbody-var-close.js":                       true,
		"test/language/expressions/arrow-function/scope-body-lex-distinct.js":                          true,
		"test/language/statements/for-of/dstr/var-ary-ptrn-elem-id-init-fn-name-class.js":              true,
		"test/language/statements/for-of/dstr/var-obj-ptrn-id-init-fn-name-class.js":                   true,
		"test/language/statements/for-of/dstr/const-obj-ptrn-id-init-fn-name-class.js":                 true,
		"test/language/statements/for-of/dstr/let-obj-ptrn-id-init-fn-name-class.js":                   true,
		"test/language/statements/for-of/dstr/const-ary-ptrn-elem-id-init-fn-name-class.js":            true,
		"test/language/statements/for-of/dstr/let-ary-ptrn-elem-id-init-fn-name-class.js":              true,
		"test/language/statements/try/dstr/obj-ptrn-id-init-fn-name-class.js":                          true,
		"test/language/statements/try/dstr/ary-ptrn-elem-id-init-fn-name-class.js":                     true,
		"test/language/expressions/arrow-function/dstr/ary-ptrn-elem-id-init-fn-name-class.js":         true,
		"test/language/expressions/arrow-function/dstr/dflt-obj-ptrn-id-init-fn-name-class.js":         true,
		"test/language/expressions/arrow-function/dstr/obj-ptrn-id-init-fn-name-class.js":              true,
		"test/language/expressions/arrow-function/dstr/dflt-ary-ptrn-elem-id-init-fn-name-class.js":    true,
		"test/language/expressions/arrow-function/lexical-super-property-from-within-constructor.js":   true,
		"test/language/expressions/arrow-function/lexical-super-property.js":                           true,
		"test/language/expressions/arrow-function/lexical-supercall-from-immediately-invoked-arrow.js": true,
		"test/built-ins/Promise/prototype/finally/subclass-species-constructor-resolve-count.js":       true,
		"test/built-ins/Promise/prototype/finally/subclass-species-constructor-reject-count.js":        true,
		"test/built-ins/Promise/prototype/finally/subclass-resolve-count.js":                           true,
		"test/built-ins/Promise/prototype/finally/species-symbol.js":                                   true,
		"test/built-ins/Promise/prototype/finally/subclass-reject-count.js":                            true,
		"test/built-ins/Promise/prototype/finally/species-constructor.js":                              true,
		"test/language/statements/switch/scope-lex-class.js":                                           true,
		"test/language/expressions/arrow-function/lexical-super-call-from-within-constructor.js":       true,
		"test/language/expressions/object/dstr/meth-dflt-ary-ptrn-elem-id-init-fn-name-class.js":       true,
		"test/language/expressions/object/dstr/meth-ary-ptrn-elem-id-init-fn-name-class.js":            true,
		"test/language/expressions/object/dstr/meth-dflt-obj-ptrn-id-init-fn-name-class.js":            true,
		"test/language/expressions/object/dstr/meth-obj-ptrn-id-init-fn-name-class.js":                 true,
		"test/built-ins/Promise/prototype/finally/resolved-observable-then-calls-PromiseResolve.js":    true,
		"test/built-ins/Promise/prototype/finally/rejected-observable-then-calls-PromiseResolve.js":    true,
		"test/built-ins/Function/prototype/toString/class-expression-explicit-ctor.js":                 true,
		"test/built-ins/Function/prototype/toString/class-expression-implicit-ctor.js":                 true,
		"test/language/global-code/decl-lex.js":                                                        true,
		"test/language/global-code/decl-lex-deletion.js":                                               true,
		"test/language/global-code/script-decl-var-collision.js":                                       true,
		"test/language/global-code/script-decl-lex.js":                                                 true,
		"test/language/global-code/script-decl-lex-lex.js":                                             true,
		"test/language/global-code/script-decl-lex-deletion.js":                                        true,

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

		// x ** y
		"test/built-ins/Array/prototype/pop/clamps-to-integer-limit.js":                                        true,
		"test/built-ins/Array/prototype/pop/length-near-integer-limit.js":                                      true,
		"test/built-ins/Array/prototype/push/clamps-to-integer-limit.js":                                       true,
		"test/built-ins/Array/prototype/push/length-near-integer-limit.js":                                     true,
		"test/built-ins/Array/prototype/push/throws-if-integer-limit-exceeded.js":                              true,
		"test/built-ins/Array/prototype/reverse/length-exceeding-integer-limit-with-object.js":                 true,
		"test/built-ins/Array/prototype/reverse/length-exceeding-integer-limit-with-proxy.js":                  true,
		"test/built-ins/Array/prototype/slice/length-exceeding-integer-limit.js":                               true,
		"test/built-ins/Array/prototype/splice/clamps-length-to-integer-limit.js":                              true,
		"test/built-ins/Array/prototype/splice/length-and-deleteCount-exceeding-integer-limit.js":              true,
		"test/built-ins/Array/prototype/splice/length-exceeding-integer-limit-shrink-array.js":                 true,
		"test/built-ins/Array/prototype/splice/length-near-integer-limit-grow-array.js":                        true,
		"test/built-ins/Array/prototype/splice/throws-if-integer-limit-exceeded.js":                            true,
		"test/built-ins/Array/prototype/unshift/clamps-to-integer-limit.js":                                    true,
		"test/built-ins/Array/prototype/unshift/length-near-integer-limit.js":                                  true,
		"test/built-ins/Array/prototype/unshift/throws-if-integer-limit-exceeded.js":                           true,
		"test/built-ins/String/prototype/split/separator-undef-limit-custom.js":                                true,
		"test/built-ins/Array/prototype/splice/create-species-length-exceeding-integer-limit.js":               true,
		"test/built-ins/Array/prototype/slice/length-exceeding-integer-limit-proxied-array.js":                 true,
		"test/built-ins/String/prototype/split/separator-undef-limit-zero.js":                                  true,
		"test/language/expressions/object/cpn-obj-lit-computed-property-name-from-exponetiation-expression.js": true,
		"test/language/expressions/object/cpn-obj-lit-computed-property-name-from-math.js":                     true,
		"test/built-ins/RegExp/prototype/exec/failure-lastindex-set.js":                                        true,

		// generators
		"test/annexB/built-ins/RegExp/RegExp-control-escape-russian-letter.js":                                       true,
		"test/language/statements/switch/scope-lex-generator.js":                                                     true,
		"test/language/expressions/in/rhs-yield-present.js":                                                          true,
		"test/language/expressions/object/cpn-obj-lit-computed-property-name-from-yield-expression.js":               true,
		"test/language/expressions/object/cpn-obj-lit-computed-property-name-from-generator-function-declaration.js": true,
		"test/built-ins/TypedArrayConstructors/ctors/object-arg/as-generator-iterable-returns.js":                    true,
		"test/built-ins/Object/seal/seal-generatorfunction.js":                                                       true,

		// async
		"test/language/eval-code/direct/async-func-decl-a-preceding-parameter-is-named-arguments-declare-arguments-and-assign.js": true,
		"test/language/statements/switch/scope-lex-async-generator.js":                                                            true,
		"test/language/statements/switch/scope-lex-async-function.js":                                                             true,
		"test/language/statements/for-of/head-lhs-async-invalid.js":                                                               true,
		"test/language/expressions/object/cpn-obj-lit-computed-property-name-from-async-arrow-function-expression.js":             true,
		"test/language/expressions/object/cpn-obj-lit-computed-property-name-from-await-expression.js":                            true,
		"test/language/statements/async-function/evaluation-body.js":                                                              true,
		"test/language/expressions/object/method-definition/object-method-returns-promise.js":                                     true,
		"test/language/expressions/object/method-definition/async-super-call-param.js":                                            true,
		"test/language/expressions/object/method-definition/async-super-call-body.js":                                             true,
		"test/built-ins/Object/seal/seal-asyncgeneratorfunction.js":                                                               true,
		"test/built-ins/Object/seal/seal-asyncfunction.js":                                                                        true,
		"test/built-ins/Object/seal/seal-asyncarrowfunction.js":                                                                   true,
		"test/language/statements/for/head-init-async-of.js":                                                                      true,
		"test/language/reserved-words/await-module.js":                                                                            true,

		// legacy number literals
		"test/language/literals/numeric/non-octal-decimal-integer.js": true,

		// coalesce
		"test/language/expressions/object/cpn-obj-lit-computed-property-name-from-expression-coalesce.js": true,

		// integer separators
		"test/language/expressions/object/cpn-obj-lit-computed-property-name-from-integer-separators.js": true,

		// BigInt
		"test/built-ins/Object/seal/seal-biguint64array.js": true,
		"test/built-ins/Object/seal/seal-bigint64array.js":  true,

		// FIXME bugs

		// new.target availability
		"test/language/global-code/new.target-arrow.js":   true,
		"test/language/eval-code/direct/new.target-fn.js": true,

		// 'in' in a branch
		"test/language/expressions/conditional/in-branch-1.js": true,

		// Left-hand side as a CoverParenthesizedExpression
		"test/language/expressions/assignment/fn-name-lhs-cover.js": true,
	}

	featuresBlackList = []string{
		"async-iteration",
		"Symbol.asyncIterator",
		"async-functions",
		"BigInt",
		"class",
		"class-static-block",
		"class-fields-private",
		"class-fields-private-in",
		"super",
		"generators",
		"String.prototype.replaceAll",
		"String.prototype.at",
		"resizable-arraybuffer",
		"array-find-from-last",
		"Array.prototype.at",
		"TypedArray.prototype.at",
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
		"coalesce-expression",
		"import.meta",
		"optional-chaining",
		"Atomics",
		"Atomics.waitAsync",
		"FinalizationRegistry",
		"WeakRef",
		"numeric-separator-literal",
		"Object.fromEntries",
		"Object.hasOwn",
		"__getter__",
		"__setter__",
		"ShadowRealm",
		"SharedArrayBuffer",
		"error-cause",
	}
)

func init() {

	skip := func(prefixes ...string) {
		for _, prefix := range prefixes {
			skipPrefixes.Add(prefix)
		}
	}

	skip(
		// class
		"test/language/statements/class/",
		"test/language/expressions/class/",
		"test/language/expressions/super/",
		"test/language/expressions/assignment/target-super-",
		"test/language/arguments-object/cls-",
		"test/built-ins/Function/prototype/toString/class-",
		"test/built-ins/Function/prototype/toString/setter-class-",
		"test/built-ins/Function/prototype/toString/method-class-",
		"test/built-ins/Function/prototype/toString/getter-class-",

		// async
		"test/language/eval-code/direct/async-",
		"test/language/expressions/async-",
		"test/language/expressions/await/",
		"test/language/statements/async-function/",
		"test/built-ins/Async",

		// generators
		"test/language/eval-code/direct/gen-",
		"test/built-ins/GeneratorFunction/",
		"test/built-ins/Function/prototype/toString/generator-",

		// **
		"test/language/expressions/exponentiation",

		// BigInt
		"test/built-ins/TypedArrayConstructors/BigUint64Array/",
		"test/built-ins/TypedArrayConstructors/BigInt64Array/",
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
	testQueue    []tc39Test
	sabStub      *Program
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

	b, err := ioutil.ReadAll(f)
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
		if meta.Es6id == "" && meta.Esid == "" {
			t.Skip("No ids")
		}

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

		b, err := ioutil.ReadAll(f)
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
	files, err := ioutil.ReadDir(path.Join(ctx.base, name))
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
		t.Skipf("If you want to run tc39 tests, download them from https://github.com/tc39/test262 and put into %s. The current working commit is ddfe24afe3043388827aa220ef623b8540958bbd. (%v)", tc39BASE, err)
	}

	ctx := &tc39TestCtx{
		base: tc39BASE,
	}
	ctx.init()
	//ctx.enableBench = true

	t.Run("tc39", func(t *testing.T) {
		ctx.t = t
		//ctx.runTC39File("test/language/types/number/8.5.1.js", t)
		//ctx.runTC39Tests("test/language")
		ctx.runTC39Tests("test/language/expressions")
		ctx.runTC39Tests("test/language/arguments-object")
		ctx.runTC39Tests("test/language/asi")
		ctx.runTC39Tests("test/language/directive-prologue")
		ctx.runTC39Tests("test/language/function-code")
		ctx.runTC39Tests("test/language/eval-code")
		ctx.runTC39Tests("test/language/global-code")
		ctx.runTC39Tests("test/language/identifier-resolution")
		ctx.runTC39Tests("test/language/identifiers")
		//ctx.runTC39Tests("test/language/literals") // legacy octal escape in strings in strict mode and regexp
		ctx.runTC39Tests("test/language/literals/numeric")
		ctx.runTC39Tests("test/language/punctuators")
		ctx.runTC39Tests("test/language/reserved-words")
		ctx.runTC39Tests("test/language/source-text")
		ctx.runTC39Tests("test/language/statements")
		ctx.runTC39Tests("test/language/types")
		ctx.runTC39Tests("test/language/white-space")
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
