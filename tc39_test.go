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

	sabStub = MustCompile("sabStub.js", `
		Object.defineProperty(this, "SharedArrayBuffer", {
			get: function() {
				throw IgnorableTestError;
			}
		});`,
		false)
)

var (
	skipList = map[string]bool{
		"test/built-ins/Date/prototype/toISOString/15.9.5.43-0-8.js":  true, // timezone
		"test/built-ins/Date/prototype/toISOString/15.9.5.43-0-9.js":  true, // timezone
		"test/built-ins/Date/prototype/toISOString/15.9.5.43-0-10.js": true, // timezone

		// \u{xxxxx}
		"test/annexB/built-ins/escape/escape-above-astral.js": true,
		"test/built-ins/RegExp/prototype/source/value-u.js":   true,

		// SharedArrayBuffer
		"test/built-ins/ArrayBuffer/prototype/slice/this-is-sharedarraybuffer.js": true,

		// class
		"test/language/statements/class/subclass/builtin-objects/Symbol/symbol-valid-as-extends-value.js":            true,
		"test/language/statements/class/subclass/builtin-objects/Symbol/new-symbol-with-super-throws.js":             true,
		"test/language/statements/class/subclass/builtin-objects/WeakSet/super-must-be-called.js":                    true,
		"test/language/statements/class/subclass/builtin-objects/WeakSet/regular-subclassing.js":                     true,
		"test/language/statements/class/subclass/builtin-objects/WeakMap/super-must-be-called.js":                    true,
		"test/language/statements/class/subclass/builtin-objects/WeakMap/regular-subclassing.js":                     true,
		"test/language/statements/class/subclass/builtin-objects/Map/super-must-be-called.js":                        true,
		"test/language/statements/class/subclass/builtin-objects/Map/regular-subclassing.js":                         true,
		"test/language/statements/class/subclass/builtin-objects/Set/super-must-be-called.js":                        true,
		"test/language/statements/class/subclass/builtin-objects/Set/regular-subclassing.js":                         true,
		"test/language/statements/class/subclass/builtin-objects/Object/replacing-prototype.js":                      true,
		"test/language/statements/class/subclass/builtin-objects/Object/regular-subclassing.js":                      true,
		"test/built-ins/Array/prototype/concat/Array.prototype.concat_non-array.js":                                  true,
		"test/language/statements/class/subclass/builtin-objects/Array/length.js":                                    true,
		"test/language/statements/class/subclass/builtin-objects/TypedArray/super-must-be-called.js":                 true,
		"test/language/statements/class/subclass/builtin-objects/TypedArray/regular-subclassing.js":                  true,
		"test/language/statements/class/subclass/builtin-objects/DataView/super-must-be-called.js":                   true,
		"test/language/statements/class/subclass/builtin-objects/DataView/regular-subclassing.js":                    true,
		"test/language/statements/class/subclass/builtin-objects/String/super-must-be-called.js":                     true,
		"test/language/statements/class/subclass/builtin-objects/String/regular-subclassing.js":                      true,
		"test/language/statements/class/subclass/builtin-objects/String/length.js":                                   true,
		"test/language/statements/class/subclass/builtin-objects/Date/super-must-be-called.js":                       true,
		"test/language/statements/class/subclass/builtin-objects/Date/regular-subclassing.js":                        true,
		"test/language/statements/class/subclass/builtin-objects/Number/super-must-be-called.js":                     true,
		"test/language/statements/class/subclass/builtin-objects/Number/regular-subclassing.js":                      true,
		"test/language/statements/class/subclass/builtin-objects/Function/super-must-be-called.js":                   true,
		"test/language/statements/class/subclass/builtin-objects/Function/regular-subclassing.js":                    true,
		"test/language/statements/class/subclass/builtin-objects/Function/instance-name.js":                          true,
		"test/language/statements/class/subclass/builtin-objects/Function/instance-length.js":                        true,
		"test/language/statements/class/subclass/builtin-objects/Boolean/super-must-be-called.js":                    true,
		"test/language/statements/class/subclass/builtin-objects/Boolean/regular-subclassing.js":                     true,
		"test/language/statements/class/subclass/builtin-objects/NativeError/URIError-super.js":                      true,
		"test/language/statements/class/subclass/builtin-objects/NativeError/URIError-name.js":                       true,
		"test/language/statements/class/subclass/builtin-objects/NativeError/URIError-message.js":                    true,
		"test/language/statements/class/subclass/builtin-objects/NativeError/TypeError-super.js":                     true,
		"test/language/statements/class/subclass/builtin-objects/NativeError/TypeError-name.js":                      true,
		"test/language/statements/class/subclass/builtin-objects/NativeError/TypeError-message.js":                   true,
		"test/language/statements/class/subclass/builtin-objects/NativeError/SyntaxError-super.js":                   true,
		"test/language/statements/class/subclass/builtin-objects/NativeError/SyntaxError-name.js":                    true,
		"test/language/statements/class/subclass/builtin-objects/NativeError/SyntaxError-message.js":                 true,
		"test/language/statements/class/subclass/builtin-objects/NativeError/ReferenceError-super.js":                true,
		"test/language/statements/class/subclass/builtin-objects/NativeError/ReferenceError-name.js":                 true,
		"test/language/statements/class/subclass/builtin-objects/NativeError/ReferenceError-message.js":              true,
		"test/language/statements/class/subclass/builtin-objects/NativeError/RangeError-super.js":                    true,
		"test/language/statements/class/subclass/builtin-objects/NativeError/RangeError-name.js":                     true,
		"test/language/statements/class/subclass/builtin-objects/NativeError/RangeError-message.js":                  true,
		"test/language/statements/class/subclass/builtin-objects/NativeError/EvalError-super.js":                     true,
		"test/language/statements/class/subclass/builtin-objects/NativeError/EvalError-name.js":                      true,
		"test/language/statements/class/subclass/builtin-objects/NativeError/EvalError-message.js":                   true,
		"test/language/statements/class/subclass/builtin-objects/Error/super-must-be-called.js":                      true,
		"test/language/statements/class/subclass/builtin-objects/Error/regular-subclassing.js":                       true,
		"test/language/statements/class/subclass/builtin-objects/Error/message-property-assignment.js":               true,
		"test/language/statements/class/subclass/builtin-objects/Array/super-must-be-called.js":                      true,
		"test/language/statements/class/subclass/builtin-objects/Array/regular-subclassing.js":                       true,
		"test/language/statements/class/subclass/builtin-objects/Array/contructor-calls-super-single-argument.js":    true,
		"test/language/statements/class/subclass/builtin-objects/Array/contructor-calls-super-multiple-arguments.js": true,
		"test/language/statements/class/subclass/builtin-objects/ArrayBuffer/super-must-be-called.js":                true,
		"test/language/statements/class/subclass/builtin-objects/ArrayBuffer/regular-subclassing.js":                 true,
		"test/built-ins/ArrayBuffer/isView/arg-is-typedarray-subclass-instance.js":                                   true,
		"test/built-ins/ArrayBuffer/isView/arg-is-dataview-subclass-instance.js":                                     true,
		"test/language/statements/class/subclass/builtin-objects/RegExp/super-must-be-called.js":                     true,
		"test/language/statements/class/subclass/builtin-objects/RegExp/regular-subclassing.js":                      true,
		"test/language/statements/class/subclass/builtin-objects/RegExp/lastIndex.js":                                true,
		"TestTC39/tc39/test/language/statements/class/definition/fn-name-method.js":                                  true,
		"test/language/expressions/object/method-definition/name-invoke-ctor.js":                                     true,
		"test/language/expressions/object/method.js":                                                                 true,
		"test/language/expressions/object/setter-super-prop.js":                                                      true,
		"test/language/expressions/object/getter-super-prop.js":                                                      true,
		"test/language/expressions/delete/super-property.js":                                                         true,

		// object literals
		"test/built-ins/Array/from/source-object-iterator-1.js":                                                true,
		"test/built-ins/Array/from/source-object-iterator-2.js":                                                true,
		"test/built-ins/TypedArrays/of/argument-number-value-throws.js":                                        true,
		"test/built-ins/TypedArrays/from/set-value-abrupt-completion.js":                                       true,
		"test/built-ins/TypedArrays/from/property-abrupt-completion.js":                                        true,
		"test/built-ins/DataView/custom-proto-access-throws-sab.js":                                            true,
		"test/built-ins/Array/prototype/slice/length-exceeding-integer-limit-proxied-array.js":                 true,
		"test/built-ins/Array/prototype/splice/create-species-length-exceeding-integer-limit.js":               true,
		"test/built-ins/Array/prototype/splice/property-traps-order-with-species.js":                           true,
		"test/built-ins/String/prototype/indexOf/position-tointeger-errors.js":                                 true,
		"test/built-ins/String/prototype/indexOf/position-tointeger-toprimitive.js":                            true,
		"test/built-ins/String/prototype/indexOf/position-tointeger-wrapped-values.js":                         true,
		"test/built-ins/String/prototype/indexOf/searchstring-tostring-errors.js":                              true,
		"test/built-ins/String/prototype/indexOf/searchstring-tostring-toprimitive.js":                         true,
		"test/built-ins/String/prototype/indexOf/searchstring-tostring-wrapped-values.js":                      true,
		"test/built-ins/String/prototype/split/separator-undef-limit-zero.js":                                  true,
		"test/built-ins/String/prototype/trimEnd/this-value-object-cannot-convert-to-primitive-err.js":         true,
		"test/built-ins/String/prototype/trimEnd/this-value-object-toprimitive-call-err.js":                    true,
		"test/built-ins/String/prototype/trimEnd/this-value-object-toprimitive-meth-err.js":                    true,
		"test/built-ins/String/prototype/trimEnd/this-value-object-toprimitive-meth-priority.js":               true,
		"test/built-ins/String/prototype/trimEnd/this-value-object-toprimitive-returns-object-err.js":          true,
		"test/built-ins/String/prototype/trimEnd/this-value-object-tostring-call-err.js":                       true,
		"test/built-ins/String/prototype/trimEnd/this-value-object-tostring-meth-err.js":                       true,
		"test/built-ins/String/prototype/trimEnd/this-value-object-tostring-meth-priority.js":                  true,
		"test/built-ins/String/prototype/trimEnd/this-value-object-tostring-returns-object-err.js":             true,
		"test/built-ins/String/prototype/trimEnd/this-value-object-valueof-call-err.js":                        true,
		"test/built-ins/String/prototype/trimEnd/this-value-object-valueof-meth-err.js":                        true,
		"test/built-ins/String/prototype/trimEnd/this-value-object-valueof-meth-priority.js":                   true,
		"test/built-ins/String/prototype/trimEnd/this-value-object-valueof-returns-object-err.js":              true,
		"test/built-ins/String/prototype/trimStart/this-value-object-cannot-convert-to-primitive-err.js":       true,
		"test/built-ins/String/prototype/trimStart/this-value-object-toprimitive-call-err.js":                  true,
		"test/built-ins/String/prototype/trimStart/this-value-object-toprimitive-meth-err.js":                  true,
		"test/built-ins/String/prototype/trimStart/this-value-object-toprimitive-meth-priority.js":             true,
		"test/built-ins/String/prototype/trimStart/this-value-object-toprimitive-returns-object-err.js":        true,
		"test/built-ins/String/prototype/trimStart/this-value-object-tostring-call-err.js":                     true,
		"test/built-ins/String/prototype/trimStart/this-value-object-tostring-meth-err.js":                     true,
		"test/built-ins/String/prototype/trimStart/this-value-object-tostring-meth-priority.js":                true,
		"test/built-ins/String/prototype/trimStart/this-value-object-tostring-returns-object-err.js":           true,
		"test/built-ins/String/prototype/trimStart/this-value-object-valueof-call-err.js":                      true,
		"test/built-ins/String/prototype/trimStart/this-value-object-valueof-meth-err.js":                      true,
		"test/built-ins/String/prototype/trimStart/this-value-object-valueof-meth-priority.js":                 true,
		"test/built-ins/String/prototype/trimStart/this-value-object-valueof-returns-object-err.js":            true,
		"test/built-ins/TypedArray/prototype/sort/sort-tonumber.js":                                            true,
		"test/built-ins/Array/prototype/flatMap/array-like-objects.js":                                         true,
		"test/built-ins/Array/prototype/flatMap/array-like-objects-poisoned-length.js":                         true,
		"test/built-ins/Array/prototype/flatMap/this-value-ctor-object-species.js":                             true,
		"test/built-ins/Array/prototype/flatMap/this-value-ctor-object-species-custom-ctor.js":                 true,
		"test/built-ins/Array/prototype/flatMap/this-value-ctor-object-species-custom-ctor-poisoned-throws.js": true,
		"test/built-ins/Array/prototype/flatMap/this-value-ctor-object-species-bad-throws.js":                  true,
		"test/built-ins/Proxy/getPrototypeOf/instanceof-target-not-extensible-not-same-proto-throws.js":        true,
		"test/language/statements/class/definition/fn-name-method.js":                                          true,

		// arrow-function
		"test/built-ins/Object/prototype/toString/proxy-function.js":            true,
		"test/built-ins/Array/prototype/pop/throws-with-string-receiver.js":     true,
		"test/built-ins/Array/prototype/push/throws-with-string-receiver.js":    true,
		"test/built-ins/Array/prototype/shift/throws-with-string-receiver.js":   true,
		"test/built-ins/Array/prototype/unshift/throws-with-string-receiver.js": true,
		"test/built-ins/Date/prototype/toString/non-date-receiver.js":           true,
		"test/built-ins/Number/prototype/toExponential/range.js":                true,
		"test/built-ins/Number/prototype/toFixed/range.js":                      true,
		"test/built-ins/Number/prototype/toPrecision/range.js":                  true,
		"test/built-ins/TypedArray/prototype/sort/stability.js":                 true,
		"test/built-ins/RegExp/named-groups/functional-replace-global.js":       true,
		"test/built-ins/RegExp/named-groups/functional-replace-non-global.js":   true,
		"test/built-ins/Array/prototype/sort/stability-513-elements.js":         true,
		"test/built-ins/Array/prototype/sort/stability-5-elements.js":           true,
		"test/built-ins/Array/prototype/sort/stability-2048-elements.js":        true,
		"test/built-ins/Array/prototype/sort/stability-11-elements.js":          true,
		"test/language/statements/variable/fn-name-arrow.js":                    true,
		"test/language/statements/let/fn-name-arrow.js":                         true,
		"test/language/statements/const/fn-name-arrow.js":                       true,

		// template strings
		"test/built-ins/String/raw/zero-literal-segments.js":                                           true,
		"test/built-ins/String/raw/template-substitutions-are-appended-on-same-index.js":               true,
		"test/built-ins/String/raw/special-characters.js":                                              true,
		"test/built-ins/String/raw/return-the-string-value-from-template.js":                           true,
		"test/built-ins/TypedArray/prototype/fill/fill-values-conversion-operations-consistent-nan.js": true,

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

		// regexp named groups
		"test/built-ins/RegExp/prototype/Symbol.replace/named-groups-fn.js":               true,
		"test/built-ins/RegExp/prototype/Symbol.replace/result-coerce-groups-err.js":      true,
		"test/built-ins/RegExp/prototype/Symbol.replace/result-coerce-groups-prop-err.js": true,
		"test/built-ins/RegExp/prototype/Symbol.replace/result-coerce-groups-prop.js":     true,
		"test/built-ins/RegExp/prototype/Symbol.replace/result-coerce-groups.js":          true,
		"test/built-ins/RegExp/prototype/Symbol.replace/result-get-groups-err.js":         true,
		"test/built-ins/RegExp/prototype/Symbol.replace/result-get-groups-prop-err.js":    true,

		// Because goja parser works in UTF-8 it is not possible to pass strings containing invalid UTF-16 code points.
		// This is mitigated by escaping them as \uXXXX, however because of this the RegExp source becomes
		// `\uXXXX` instead of `<the actual UTF-16 code point of XXXX>`.
		// The resulting RegExp will work exactly the same, but it causes these two tests to fail.
		"test/annexB/built-ins/RegExp/RegExp-leading-escape-BMP.js":  true,
		"test/annexB/built-ins/RegExp/RegExp-trailing-escape-BMP.js": true,

		// Promise
		"test/built-ins/Symbol/species/builtin-getter-name.js": true,

		// x ** y
		"test/built-ins/Array/prototype/pop/clamps-to-integer-limit.js":                           true,
		"test/built-ins/Array/prototype/pop/length-near-integer-limit.js":                         true,
		"test/built-ins/Array/prototype/push/clamps-to-integer-limit.js":                          true,
		"test/built-ins/Array/prototype/push/length-near-integer-limit.js":                        true,
		"test/built-ins/Array/prototype/push/throws-if-integer-limit-exceeded.js":                 true,
		"test/built-ins/Array/prototype/reverse/length-exceeding-integer-limit-with-object.js":    true,
		"test/built-ins/Array/prototype/reverse/length-exceeding-integer-limit-with-proxy.js":     true,
		"test/built-ins/Array/prototype/slice/length-exceeding-integer-limit.js":                  true,
		"test/built-ins/Array/prototype/splice/clamps-length-to-integer-limit.js":                 true,
		"test/built-ins/Array/prototype/splice/length-and-deleteCount-exceeding-integer-limit.js": true,
		"test/built-ins/Array/prototype/splice/length-exceeding-integer-limit-shrink-array.js":    true,
		"test/built-ins/Array/prototype/splice/length-near-integer-limit-grow-array.js":           true,
		"test/built-ins/Array/prototype/splice/throws-if-integer-limit-exceeded.js":               true,
		"test/built-ins/Array/prototype/unshift/clamps-to-integer-limit.js":                       true,
		"test/built-ins/Array/prototype/unshift/length-near-integer-limit.js":                     true,
		"test/built-ins/Array/prototype/unshift/throws-if-integer-limit-exceeded.js":              true,
		"test/built-ins/String/prototype/split/separator-undef-limit-custom.js":                   true,

		// generators
		"test/annexB/built-ins/RegExp/RegExp-control-escape-russian-letter.js": true,

		// computed properties
		"test/language/expressions/object/__proto__-permitted-dup.js":                     true,
		"test/language/expressions/object/method-definition/name-name-prop-symbol.js":     true,
		"test/language/expressions/object/method-definition/name-prop-name-eval-error.js": true,
		"test/language/expressions/object/accessor-name-computed-yield-id.js":             true,
		"test/language/expressions/object/accessor-name-computed-in.js":                   true,

		// get [Symbol.*]
		"test/language/expressions/object/prop-def-id-eval-error.js": true,

		// destructing binding
		"test/language/statements/for-of/head-var-bound-names-dup.js": true,
		"test/language/statements/for-of/head-let-destructuring.js":   true,
		"test/language/statements/for-in/head-var-bound-names-dup.js": true,
		"test/language/statements/for/head-let-destructuring.js":      true,
		"test/language/statements/for-in/head-let-destructuring.js":   true,
	}

	featuresBlackList = []string{
		"arrow-function",
		"async-iteration",
		"BigInt",
		"class",
		"destructuring-binding",
		"generators",
		"String.prototype.replaceAll",
		"computed-property-names",
		"default-parameters",
		"super",
	}

	es6WhiteList = map[string]bool{}

	es6IdWhiteList = []string{
		"8.1.2.1",
		"9.5",
		"12.1",
		"12.2.1",
		"12.2.2",
		"12.2.5",
		"12.2.6.1",
		"12.2.6.8",
		"12.4",
		"12.5",
		"12.6",
		"12.7",
		"12.8",
		"12.9",
		"12.10",
		"13.1",
		"13.2",
		"13.3",
		"13.4",
		"13.5",
		"13.6",
		"13.7",
		"13.8",
		"13.9",
		"13.10",
		"13.11",
		"13.12",
		"13.13",
		"13.14",
		"13.15",
		"14.3.8",
		"18",
		"19",
		"20",
		"21",
		"22",
		"23",
		"24",
		"25.1",
		"26",
		"B.2.1",
		"B.2.2",
	}

	esIdPrefixWhiteList = []string{
		"sec-addition-*",
		"sec-array",
		"sec-%typedarray%",
		"sec-%typedarray%-of",
		"sec-@@iterator",
		"sec-@@tostringtag",
		"sec-string",
		"sec-date",
		"sec-json",
		"sec-number",
		"sec-math",
		"sec-arraybuffer-length",
		"sec-arraybuffer",
		"sec-regexp",
		"sec-string.prototype.trimLeft",
		"sec-string.prototype.trimRight",
		"sec-object.getownpropertydescriptor",
		"sec-object.getownpropertydescriptors",
		"sec-object.entries",
		"sec-object.values",
		"sec-object-initializer",
		"sec-proxy-*",
		"sec-for-statement-*",
		"sec-for-in-and-for-of-statements",
		"sec-do-while-statement",
		"sec-if-statement",
		"sec-while-statement",
		"sec-with-statement*",
		"sec-switch-*",
		"sec-try-*",
		"sec-strict-mode-of-ecmascript",
		"sec-let-and-const-declarations*",
		"sec-arguments-exotic-objects-defineownproperty-p-desc",
	}
)

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
	vm.Set("$262", _262)
	vm.Set("IgnorableTestError", ignorableTestError)
	vm.Set("print", t.Log)
	vm.RunProgram(sabStub)
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
}

func (ctx *tc39TestCtx) runTC39File(name string, t testing.TB) {
	if skipList[name] {
		t.Skip("Excluded")
	}
	p := path.Join(ctx.base, name)
	meta, src, err := parseTC39File(p)
	if err != nil {
		//t.Fatalf("Could not parse %s: %v", name, err)
		t.Errorf("Could not parse %s: %v", name, err)
		return
	}
	if meta.Es5id == "" {
		skip := true
		//t.Logf("%s: Not ES5, skipped", name)
		if es6WhiteList[name] {
			skip = false
		} else {
			if meta.Es6id != "" {
				for _, prefix := range es6IdWhiteList {
					if strings.HasPrefix(meta.Es6id, prefix) &&
						(len(meta.Es6id) == len(prefix) || meta.Es6id[len(prefix)] == '.') {

						skip = false
						break
					}
				}
			}
		}
		if skip {
			if meta.Esid != "" {
				for _, prefix := range esIdPrefixWhiteList {
					if strings.HasSuffix(prefix, "*") {
						if strings.HasPrefix(meta.Esid, prefix[:len(prefix)-1]) {
							skip = false
							break
						}
					} else {
						if strings.HasPrefix(meta.Esid, prefix) &&
							(len(meta.Esid) == len(prefix) || meta.Esid[len(prefix)] == '.') {
							skip = false
							break
						}
					}
				}
			}
		}
		if skip {
			t.Skip("Not ES5")
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
		//ctx.runTC39Tests("test/language/literals") // octal sequences in strict mode
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
