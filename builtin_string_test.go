package goja

import "testing"

func TestSubstr(t *testing.T) {
	const SCRIPT = `
assert.sameValue('abc'.substr(0, false), '', 'start: 0, length: false');
assert.sameValue('abc'.substr(1, false), '', 'start: 1, length: false');
assert.sameValue('abc'.substr(2, false), '', 'start: 2, length: false');
assert.sameValue('abc'.substr(3, false), '', 'start: 3, length: false');

assert.sameValue('abc'.substr(0, NaN), '', 'start: 0, length: NaN');
assert.sameValue('abc'.substr(1, NaN), '', 'start: 1, length: NaN');
assert.sameValue('abc'.substr(2, NaN), '', 'start: 2, length: NaN');
assert.sameValue('abc'.substr(3, NaN), '', 'start: 3, length: NaN');

assert.sameValue('abc'.substr(0, ''), '', 'start: 0, length: ""');
assert.sameValue('abc'.substr(1, ''), '', 'start: 1, length: ""');
assert.sameValue('abc'.substr(2, ''), '', 'start: 2, length: ""');
assert.sameValue('abc'.substr(3, ''), '', 'start: 3, length: ""');

assert.sameValue('abc'.substr(0, null), '', 'start: 0, length: null');
assert.sameValue('abc'.substr(1, null), '', 'start: 1, length: null');
assert.sameValue('abc'.substr(2, null), '', 'start: 2, length: null');
assert.sameValue('abc'.substr(3, null), '', 'start: 3, length: null');

assert.sameValue('abc'.substr(0, -1), '', '0, -1');
assert.sameValue('abc'.substr(0, -2), '', '0, -2');
assert.sameValue('abc'.substr(0, -3), '', '0, -3');
assert.sameValue('abc'.substr(0, -4), '', '0, -4');

assert.sameValue('abc'.substr(1, -1), '', '1, -1');
assert.sameValue('abc'.substr(1, -2), '', '1, -2');
assert.sameValue('abc'.substr(1, -3), '', '1, -3');
assert.sameValue('abc'.substr(1, -4), '', '1, -4');

assert.sameValue('abc'.substr(2, -1), '', '2, -1');
assert.sameValue('abc'.substr(2, -2), '', '2, -2');
assert.sameValue('abc'.substr(2, -3), '', '2, -3');
assert.sameValue('abc'.substr(2, -4), '', '2, -4');

assert.sameValue('abc'.substr(3, -1), '', '3, -1');
assert.sameValue('abc'.substr(3, -2), '', '3, -2');
assert.sameValue('abc'.substr(3, -3), '', '3, -3');
assert.sameValue('abc'.substr(3, -4), '', '3, -4');

assert.sameValue('abc'.substr(0, 1), 'a', '0, 1');
assert.sameValue('abc'.substr(0, 2), 'ab', '0, 1');
assert.sameValue('abc'.substr(0, 3), 'abc', '0, 1');
assert.sameValue('abc'.substr(0, 4), 'abc', '0, 1');

assert.sameValue('abc'.substr(1, 1), 'b', '1, 1');
assert.sameValue('abc'.substr(1, 2), 'bc', '1, 1');
assert.sameValue('abc'.substr(1, 3), 'bc', '1, 1');
assert.sameValue('abc'.substr(1, 4), 'bc', '1, 1');

assert.sameValue('abc'.substr(2, 1), 'c', '2, 1');
assert.sameValue('abc'.substr(2, 2), 'c', '2, 1');
assert.sameValue('abc'.substr(2, 3), 'c', '2, 1');
assert.sameValue('abc'.substr(2, 4), 'c', '2, 1');

assert.sameValue('abc'.substr(3, 1), '', '3, 1');
assert.sameValue('abc'.substr(3, 2), '', '3, 1');
assert.sameValue('abc'.substr(3, 3), '', '3, 1');
assert.sameValue('abc'.substr(3, 4), '', '3, 1');

assert.sameValue('abc'.substr(0), 'abc', 'start: 0, length: unspecified');
assert.sameValue('abc'.substr(1), 'bc', 'start: 1, length: unspecified');
assert.sameValue('abc'.substr(2), 'c', 'start: 2, length: unspecified');
assert.sameValue('abc'.substr(3), '', 'start: 3, length: unspecified');

assert.sameValue(
  'abc'.substr(0, undefined), 'abc', 'start: 0, length: undefined'
);
assert.sameValue(
  'abc'.substr(1, undefined), 'bc', 'start: 1, length: undefined'
);
assert.sameValue(
  'abc'.substr(2, undefined), 'c', 'start: 2, length: undefined'
);
assert.sameValue(
  'abc'.substr(3, undefined), '', 'start: 3, length: undefined'
);



	`

	testScript1(TESTLIB+SCRIPT, _undefined, t)
}
