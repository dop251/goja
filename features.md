# ECMAScript feature support

This is an inventory of language and built-in features in Goja, checked against
[6b6a90e5](https://github.com/dop251/goja/commit/6b6a90e519465e9b51198b64bff8aedc2cfd9b1e).
**Yes** means the feature is implemented, with the qualifications below; it is
not a claim that every associated conformance test passes. **No** means the
listed feature is not implemented. The [known incompatibilities](README.md#known-incompatibilities-and-caveats)
and [test262 exclusions](tc39_test.go) still apply.

## ECMAScript 2015 (ES6)

The rows group the language and API additions in
[ECMA-262, sixth edition](https://262.ecma-international.org/6.0/).
Compatibility changes to existing features are described in the specification's
[Annex D](https://262.ecma-international.org/6.0/#sec-corrections-and-clarifications-in-ecmascript-2015-with-possible-compatibility-impact)
and [Annex E](https://262.ecma-international.org/6.0/#sec-additions-and-changes-that-introduce-incompatibilities-with-prior-editions);
this inventory does not certify all those semantic edge cases.

### Syntax, functions and iteration

| Feature | Implemented? | Related issue |
| --- | --- | --- |
| `let`, `const`, block scope, temporal dead zone and per-iteration bindings | Yes | [#179](https://github.com/dop251/goja/issues/179) |
| Block-level function declarations (see the Annex B qualification below) | Yes | |
| Array/object destructuring in bindings and assignments, including nested patterns and defaults | Yes | [#280](https://github.com/dop251/goja/issues/280) |
| Default function parameters and rest parameters | Yes | [#302](https://github.com/dop251/goja/issues/302), [#301](https://github.com/dop251/goja/issues/301) |
| Iterable spread in arrays, function calls and constructor calls | Yes | [#301](https://github.com/dop251/goja/issues/301) |
| Arrow functions and lexical `this`/`arguments` | Yes | [#304](https://github.com/dop251/goja/issues/304) |
| Object-literal shorthand properties, computed names/accessors, concise methods, duplicate property names and `super` property access | Yes | |
| Template literals and tagged templates, including raw strings | Yes | [#260](https://github.com/dop251/goja/issues/260) |
| Class declarations/expressions, constructors, methods/accessors, static methods, `extends` and `super` | Yes | [#349](https://github.com/dop251/goja/issues/349) |
| `new.target`, subclassing built-in constructors and `Symbol.species` result constructors | Yes | |
| `for...of`, iterable/iterator protocols, built-in iterators and iterator closing on early completion | Yes | [#176](https://github.com/dop251/goja/issues/176) |
| Generator functions/methods, `yield`, `yield*`, and generator `next`/`return`/`throw` | Yes | [#436](https://github.com/dop251/goja/issues/436) |
| ES modules: static `import`/`export`, live bindings and module execution | No | [PR #430](https://github.com/dop251/goja/pull/430) |
| Proper tail calls | No | |
| Binary/octal literals (`0b`, `0o`) and binary/octal strings in Number conversion | Yes | |
| Unicode code-point escapes (`\u{...}`) and supplementary-plane identifiers | Yes | |

### Built-in objects

| Feature | Implemented? | Related issue |
| --- | --- | --- |
| Symbol primitives/property keys, `Symbol.for` and `Symbol.keyFor` | Yes | [#173](https://github.com/dop251/goja/issues/173) |
| Well-known symbols: `hasInstance`, `isConcatSpreadable`, `iterator`, `match`, `replace`, `search`, `species`, `split`, `toPrimitive`, `toStringTag`, `unscopables` | Yes | [#173](https://github.com/dop251/goja/issues/173) |
| `Object.assign`, `Object.is`, `Object.getOwnPropertySymbols`, `Object.setPrototypeOf` | Yes | |
| Primitive arguments to Object reflection/integrity methods; string/symbol own-key handling | Yes | |
| Function `name` inference, configurable `length`, and `Function.prototype[Symbol.hasInstance]` | Yes | |
| `Number.EPSILON`, `MAX_SAFE_INTEGER`, `MIN_SAFE_INTEGER`; `Number.isFinite`, `isInteger`, `isNaN`, `isSafeInteger`, `parseInt`, `parseFloat` | Yes | |
| `Math.acosh`, `asinh`, `atanh`, `cbrt`, `clz32`, `cosh`, `expm1`, `fround`, `hypot`, `imul`, `log1p`, `log2`, `log10`, `sign`, `sinh`, `tanh`, `trunc` | Yes | |
| `String.fromCodePoint`, `String.raw`; string `codePointAt`, `endsWith`, `includes`, `normalize`, `repeat`, `startsWith` and code-point iteration | Yes | |
| RegExp `u`/`y` flags, `flags`/`unicode`/`sticky` accessors, replacement constructor flags and symbol-based string dispatch (see limitations below) | Yes | |
| `Array.from`, `Array.of`; array `copyWithin`, `fill`, `find`, `findIndex`, `keys`, `values`, `entries`, iteration, species and unscopables | Yes | |
| Map and Set: iterable constructors, insertion-ordered iteration and methods | Yes | [#174](https://github.com/dop251/goja/issues/174) |
| WeakMap and WeakSet | Yes | [#174](https://github.com/dop251/goja/issues/174) |
| ArrayBuffer, `ArrayBuffer.isView`, `byteLength` and `slice` | Yes | [#177](https://github.com/dop251/goja/issues/177) |
| DataView and its integer/floating-point accessors with byte-order selection | Yes | [#177](https://github.com/dop251/goja/issues/177) |
| `Int8Array`, `Uint8Array`, `Uint8ClampedArray`, `Int16Array`, `Uint16Array`, `Int32Array`, `Uint32Array`, `Float32Array`, `Float64Array` | Yes | [#177](https://github.com/dop251/goja/issues/177) |
| Typed-array `from`/`of`, shared prototype methods, iterators and species | Yes | [#177](https://github.com/dop251/goja/issues/177) |
| Promise constructor, `resolve`, `reject`, `all`, `race`, `then`, `catch`, thenable resolution and reaction jobs | Yes | [#178](https://github.com/dop251/goja/issues/178) |
| Proxy traps/invariants, callable/constructable targets and `Proxy.revocable` | Yes | [#175](https://github.com/dop251/goja/issues/175) |
| Reflect `apply`, `construct`, `defineProperty`, `deleteProperty`, `get`, `getOwnPropertyDescriptor`, `getPrototypeOf`, `has`, `isExtensible`, `ownKeys`, `preventExtensions`, `set`, `setPrototypeOf` | Yes | [#181](https://github.com/dop251/goja/issues/181) |

Unicode RegExp matching is implemented, but some Unicode syntax restrictions and
full case-folding cases are excluded from the test suite. The `u` row does not
mean complete Unicode RegExp conformance. Unicode identifier properties also
depend on the Unicode tables in the Go version used to build Goja.

### Optional legacy behavior and historical APIs

Annex B is optional for non-browser hosts. Some Annex B functionality is missing
in Goja; support for the core features above does not imply complete Annex B
support. This is separate from the mandatory ES2015 features, such as modules
and proper tail calls.

| Feature | Implemented? | Related issue |
| --- | --- | --- |
| HTML-like comments in scripts | Yes | |
| Legacy leading-zero number literals and octal string escapes in non-strict code | Yes | |
| Object-literal `__proto__` special form | Yes | |
| `Object.prototype.__proto__` accessor | Yes | |
| Global `escape`/`unescape`; string `substr` and HTML wrapper methods | Yes | |
| Date `getYear`, `setYear`, `toGMTString`; `RegExp.prototype.compile` | Yes | |
| Redeclaring a simple catch parameter with `var` | Yes | |
| Complete Annex B compatibility, including legacy block/conditional-function and RegExp semantics | No | |
| Historical `Reflect.enumerate`, Proxy `enumerate` trap and module-namespace `Symbol.iterator` | No | |

The original 2015 edition also included `Reflect.enumerate`, the Proxy
`enumerate` trap and module-namespace `Symbol.iterator`. Goja does not implement
these historical APIs; they were subsequently removed from the language
([enumerate removal](https://github.com/tc39/ecma262/commit/d96e60a99a40fab2de0df329b3e5445ac27b8a8e),
[module-namespace iterator removal](https://github.com/tc39/ecma262/pull/747)).

## Beyond ES2015

This section lists implemented additions beyond the ES2015 baseline. It does
not assign an edition or proposal stage to each addition, or imply support for
all features of a later ECMAScript edition.

### Language

| Feature | Implemented? | Related issue |
| --- | --- | --- |
| Exponentiation (`**`, `**=`) | Yes | [#381](https://github.com/dop251/goja/issues/381) |
| Async functions/arrows/methods and `await` inside async functions | Yes | [#460](https://github.com/dop251/goja/issues/460) |
| Trailing commas in parameter and argument lists | Yes | |
| Object rest and spread | Yes | [#301](https://github.com/dop251/goja/issues/301) |
| Tagged templates with invalid cooked escapes (raw text remains available) | Yes | |
| `for await...of`, async iterables and async-from-sync iteration | Yes | |
| Optional catch binding (`catch {}`) | Yes | [#259](https://github.com/dop251/goja/issues/259) |
| U+2028/U+2029 in string literals (JSON-superset syntax) | Yes | |
| BigInt literals and operators | Yes | [#556](https://github.com/dop251/goja/issues/556) |
| Optional chaining (`?.`) | Yes | [#350](https://github.com/dop251/goja/issues/350) |
| Nullish coalescing (`??`) | Yes | [#382](https://github.com/dop251/goja/issues/382) |
| Logical/nullish assignment (`&&=`, `\|\|=`, `??=`) | Yes | [#647](https://github.com/dop251/goja/issues/647) |
| Numeric separators | Yes | |
| Public instance/static class fields; private fields, methods and accessors | Yes | |
| Private brand checks (`#name in value`) and class static initialization blocks | Yes | |
| Hashbang comments at the start of a script | Yes | |

Async iterable consumption does not imply async-generator support:
`async function*` is not implemented. Top-level `await`, dynamic `import()` and
`import.meta` are also unavailable in the absence of ES module support.

### Built-ins

| Feature | Implemented? | Related issue |
| --- | --- | --- |
| Array/TypedArray `includes` | Yes | |
| `Object.values`, `entries`, `getOwnPropertyDescriptors`, `fromEntries`, `hasOwn` | Yes | |
| Object prototype `__defineGetter__`, `__defineSetter__`, `__lookupGetter__`, `__lookupSetter__` | Yes | |
| String `padStart`, `padEnd`, `trimStart`, `trimEnd` (and `trimLeft`/`trimRight` aliases) | Yes | |
| `Symbol.prototype.description`, `Symbol.asyncIterator`, `Symbol.matchAll` | Yes | |
| AsyncFunction constructor/prototype, obtained through an async function | Yes | |
| `Function.prototype.toString` preserving source text | Yes | |
| Stable Array/TypedArray sorting | Yes | |
| Well-formed `JSON.stringify` output for lone surrogates | Yes | |
| BigInt constructor, `asIntN`, `asUintN` and prototype methods | Yes | [#556](https://github.com/dop251/goja/issues/556) |
| BigInt64Array/BigUint64Array and DataView `getBigInt64`, `getBigUint64`, `setBigInt64`, `setBigUint64` | Yes | |
| Array `flat` and `flatMap` | Yes | |
| String `matchAll`, RegExp `Symbol.matchAll` hook and String `replaceAll` | Yes | [PR #527](https://github.com/dop251/goja/pull/527) |
| RegExp dotAll (`s`), lookbehind and named capture groups | Yes | |
| `globalThis` | Yes | |
| `Promise.prototype.finally`; `Promise.allSettled`, `any`, `withResolvers`, `try`; AggregateError | Yes | |
| Array/String/TypedArray `at` | Yes | [PR #384](https://github.com/dop251/goja/pull/384) |
| Error `cause` option | Yes | |
| Array/TypedArray `findLast` and `findLastIndex` | Yes | |
| Array `toReversed`, `toSorted`, `toSpliced`, `with`; TypedArray `toReversed`, `toSorted`, `with` | Yes | |
| Non-registered symbols as WeakMap keys and WeakSet values | Yes | |
| `Object.groupBy`, `Map.groupBy` | Yes | |
| String `isWellFormed` and `toWellFormed` | Yes | |
| ArrayBuffer `detached`, `transfer` and `transferToFixedLength` | Yes | |
| Set `union`, `intersection`, `difference`, `symmetricDifference`, `isSubsetOf`, `isSupersetOf`, `isDisjointFrom` | Yes | |
| Iterator constructor, `Iterator.from`; iterator `map`, `filter`, `take`, `drop`, `flatMap`, `reduce`, `toArray`, `forEach`, `some`, `every`, `find` | Yes | |
| `Iterator.concat`, `Iterator.zip`, `Iterator.zipKeyed` | Yes | |
| `RegExp.escape` | Yes | |
| `Math.sumPrecise` | Yes | |
| `Uint8Array.fromHex`, `fromBase64`; Uint8Array prototype `toHex`, `toBase64`, `setFromHex`, `setFromBase64` | Yes | |
| Map/WeakMap `getOrInsert`, `getOrInsertComputed` | Yes | |
| `Error.isError` | Yes | |
| `JSON.rawJSON`, `JSON.isRawJSON` | Yes | |

The similarly named newer APIs are not interchangeable. ArrayBuffer transfers
do not imply resizable buffers, WeakMap/WeakSet do not imply WeakRef or
FinalizationRegistry, and synchronous Iterator helpers do not imply async
Iterator helpers. These latter features are not implemented.

RegExp Unicode property escapes have limited support (for example, `\p{L}`),
but not the complete ECMAScript feature. The `d` and `v` flags are not supported.
The JSON raw-text APIs do not include the `JSON.parse` reviver source-context
argument; the README's JSON parsing caveat still applies.

## Host environment

ECMA-402 internationalization and browser/Node.js APIs are separate from this
ECMA-262 inventory. For example, `setTimeout`, `fetch`, the DOM and CommonJS
`require` are not ES6 language features. See the [event-loop explanation](README.md#where-is-settimeoutsetinterval)
and [goja_nodejs](https://github.com/dop251/goja_nodejs) for host integrations.
