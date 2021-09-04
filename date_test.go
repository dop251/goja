package goja

import (
	"testing"
	"time"
)

const TESTLIB = `
function $ERROR(message) {
	throw new Error(message);
}

function Test262Error() {
}

function assert(mustBeTrue, message) {
    if (mustBeTrue === true) {
        return;
    }

    if (message === undefined) {
        message = 'Expected true but got ' + String(mustBeTrue);
    }
    $ERROR(message);
}

assert._isSameValue = function (a, b) {
    if (a === b) {
        // Handle +/-0 vs. -/+0
        return a !== 0 || 1 / a === 1 / b;
    }

    // Handle NaN vs. NaN
    return a !== a && b !== b;
};

assert.sameValue = function (actual, expected, message) {
    if (assert._isSameValue(actual, expected)) {
        return;
    }

    if (message === undefined) {
        message = '';
    } else {
        message += ' ';
    }

    message += 'Expected SameValue(«' + String(actual) + '», «' + String(expected) + '») to be true';

    $ERROR(message);
};

assert.throws = function (expectedErrorConstructor, func, message) {
  if (typeof func !== "function") {
    $ERROR('assert.throws requires two arguments: the error constructor ' +
      'and a function to run');
    return;
  }
  if (message === undefined) {
    message = '';
  } else {
    message += ' ';
  }

  try {
    func();
  } catch (thrown) {
    if (typeof thrown !== 'object' || thrown === null) {
      message += 'Thrown value was not an object!';
      $ERROR(message);
    } else if (thrown.constructor !== expectedErrorConstructor) {
      message += 'Expected a ' + expectedErrorConstructor.name + ' but got a ' + thrown.constructor.name;
      $ERROR(message);
    }
    return;
  }

  message += 'Expected a ' + expectedErrorConstructor.name + ' to be thrown but no exception was thrown at all';
  $ERROR(message);
};

function compareArray(a, b) {
  if (b.length !== a.length) {
    return false;
  }

  for (var i = 0; i < a.length; i++) {
    if (b[i] !== a[i]) {
      return false;
    }
  }
  return true;
}
`

const TESTLIBX = TESTLIB +
	`function looksNative(fn) {
		return /native code/.test(Function.prototype.toString.call(fn));
	}

	function deepEqual(a, b) {
		if (typeof a === "object") {
			if (typeof b === "object") {
				if (a === b) {
					return true;
				}
				if (Reflect.getPrototypeOf(a) !== Reflect.getPrototypeOf(b)) {
					return false;
				}
				var keysA = Object.keys(a);
				var keysB = Object.keys(b);
				if (keysA.length !== keysB.length) {
					return false;
				}
				if (!compareArray(keysA.sort(), keysB.sort())) {
					return false;
				}
				for (var i = 0; i < keysA.length; i++) {
					var key = keysA[i];
					if (!deepEqual(a[key], b[key])) {
						return false;
					}
				}
				return true;
			} else {
				return false;
			}
		}
		return assert._isSameValue(a, b);
	}
`

func TestDateUTC(t *testing.T) {
	const SCRIPT = `
	assert.sameValue(Date.UTC(1970, 0), 0, '1970, 0');
	assert.sameValue(Date.UTC(2016, 0), 1451606400000, '2016, 0');
	assert.sameValue(Date.UTC(2016, 6), 1467331200000, '2016, 6');

	assert.sameValue(Date.UTC(2016, 6, 1), 1467331200000, '2016, 6, 1');
	assert.sameValue(Date.UTC(2016, 6, 5), 1467676800000, '2016, 6, 5');

	assert.sameValue(Date.UTC(2016, 6, 5, 0), 1467676800000, '2016, 6, 5, 0');
	assert.sameValue(Date.UTC(2016, 6, 5, 15), 1467730800000, '2016, 6, 5, 15');

	assert.sameValue(
  		Date.UTC(2016, 6, 5, 15, 0), 1467730800000, '2016, 6, 5, 15, 0'
	);
	assert.sameValue(
  		Date.UTC(2016, 6, 5, 15, 34), 1467732840000, '2016, 6, 5, 15, 34'
	);

	assert.sameValue(
  		Date.UTC(2016, 6, 5, 15, 34, 0), 1467732840000, '2016, 6, 5, 15, 34, 0'
	);
	assert.sameValue(
  		Date.UTC(2016, 6, 5, 15, 34, 45), 1467732885000, '2016, 6, 5, 15, 34, 45'
	);

	`

	testScript1(TESTLIB+SCRIPT, _undefined, t)
}

func TestNewDate(t *testing.T) {
	const SCRIPT = `
	var d1 = new Date("2016-09-01T12:34:56Z");
	d1.getUTCHours() === 12;

	`
	testScript1(SCRIPT, valueTrue, t)
}

func TestNewDate0(t *testing.T) {
	const SCRIPT = `
	(new Date(0)).toUTCString();

	`
	testScript1(SCRIPT, asciiString("Thu, 01 Jan 1970 00:00:00 GMT"), t)
}

func TestSetHour(t *testing.T) {
	l := time.Local
	defer func() {
		time.Local = l
	}()
	var err error
	time.Local, err = time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}

	const SCRIPT = `
	var d = new Date(2016, 8, 1, 12, 23, 45)
	assert.sameValue(d.getHours(), 12);
	assert.sameValue(d.getUTCHours(), 16);

	d.setHours(13);
	assert.sameValue(d.getHours(), 13);
	assert.sameValue(d.getMinutes(), 23);
	assert.sameValue(d.getSeconds(), 45);

	d.setUTCHours(13);
	assert.sameValue(d.getHours(), 9);
	assert.sameValue(d.getMinutes(), 23);
	assert.sameValue(d.getSeconds(), 45);

	`
	testScript1(TESTLIB+SCRIPT, _undefined, t)

}

func TestSetMinute(t *testing.T) {
	l := time.Local
	defer func() {
		time.Local = l
	}()
	time.Local = time.FixedZone("Asia/Delhi", 5*60*60+30*60)

	const SCRIPT = `
	var d = new Date(2016, 8, 1, 12, 23, 45)
	assert.sameValue(d.getHours(), 12);
	assert.sameValue(d.getUTCHours(), 6);
	assert.sameValue(d.getMinutes(), 23);
	assert.sameValue(d.getUTCMinutes(), 53);

	d.setMinutes(55);
	assert.sameValue(d.getMinutes(), 55);
	assert.sameValue(d.getSeconds(), 45);

	d.setUTCMinutes(52);
	assert.sameValue(d.getMinutes(), 22);
	assert.sameValue(d.getHours(), 13);

	`
	testScript1(TESTLIB+SCRIPT, _undefined, t)

}

func TestTimezoneOffset(t *testing.T) {
	const SCRIPT = `
	var d = new Date(0);
	d.getTimezoneOffset();
	`

	l := time.Local
	defer func() {
		time.Local = l
	}()
	var err error
	time.Local, err = time.LoadLocation("Europe/London")
	if err != nil {
		t.Fatal(err)
	}

	testScript1(SCRIPT, intToValue(-60), t)
}

func TestDateValueOf(t *testing.T) {
	const SCRIPT = `
	var d9 = new Date(1.23e15);
	d9.valueOf();
	`

	testScript1(SCRIPT, intToValue(1.23e15), t)
}

func TestDateSetters(t *testing.T) {
	const SCRIPT = `
	assert.sameValue((new Date(0)).setMilliseconds(2345), 2345, "setMilliseconds(2345)");
	assert.sameValue(new Date(1000).setMilliseconds(23450000000000), 23450000001000, "setMilliseconds(23450000000000)");
	assert.sameValue((new Date(0)).setUTCMilliseconds(2345), 2345, "setUTCMilliseconds()");
	assert.sameValue((new Date(0)).setSeconds(12), 12000, "setSeconds()");
	assert.sameValue((new Date(0)).setUTCSeconds(12), 12000, "setUTCSeconds()");
	assert.sameValue((new Date(0)).setMinutes(12), 12 * 60 * 1000, "setMinutes()");
	assert.sameValue((new Date(0)).setUTCMinutes(12), 12 * 60 * 1000, "setUTCMinutes()");
	assert.sameValue((new Date("2016-06-01")).setHours(1), 1464739200000, "setHours()");
	assert.sameValue((new Date("2016-06-01")).setUTCHours(1), 1464742800000, "setUTCHours()");
	assert.sameValue((new Date(0)).setDate(2), 86400000, "setDate()");
	assert.sameValue((new Date(0)).setUTCDate(2), 86400000, "setUTCDate()");
	assert.sameValue((new Date(0)).setMonth(2), 5097600000, "setMonth()");
	assert.sameValue((new Date(0)).setUTCMonth(2), 5097600000, "setUTCMonth()");
	assert.sameValue((new Date(0)).setFullYear(1971), 31536000000, "setFullYear()");
	assert.sameValue((new Date(0)).setFullYear(1971, 2, 3), 36806400000, "setFullYear(Y,M,D)");
	assert.sameValue((new Date(0)).setUTCFullYear(1971), 31536000000, "setUTCFullYear()");
	assert.sameValue((new Date(0)).setUTCFullYear(1971, 2, 3), 36806400000, "setUTCFullYear(Y,M,D)");

	var d = new Date();
	d.setTime(1151877845000);
	assert.sameValue(d.getHours(), 23, "d.getHours()");
	`

	l := time.Local
	defer func() {
		time.Local = l
	}()
	var err error
	time.Local, err = time.LoadLocation("Europe/London")
	if err != nil {
		t.Fatal(err)
	}

	testScript1(TESTLIB+SCRIPT, _undefined, t)
}

func TestDateParse(t *testing.T) {
	const SCRIPT = `
	var zero = new Date(0);

	assert.sameValue(zero.valueOf(), Date.parse(zero.toString()),
					 "Date.parse(zeroDate.toString())");
	assert.sameValue(zero.valueOf(), Date.parse(zero.toUTCString()),
					 "Date.parse(zeroDate.toUTCString())");
	assert.sameValue(zero.valueOf(), Date.parse(zero.toISOString()),
					 "Date.parse(zeroDate.toISOString())");

	function testParse(str, expected) {
		assert.sameValue(Date.parse(str), expected, str);
	}

	testParse("Mon, 02 Jan 2006 15:04:05 MST",							1136239445000);
	testParse("Tue, 22 Jun 2021 13:54:40 GMT",							1624370080000);
	testParse("Tuesday, 22 Jun 2021 13:54:40 GMT",						1624370080000);
	testParse("Mon, 02 Jan 2006 15:04:05 GMT-07:00 (MST)",				1136239445000);
	testParse("Mon, 02 Jan 2006 15:04:05 -07:00 (MST)",					1136239445000);
	testParse("Monday, 02 Jan 2006 15:04:05 -0700 (MST)",				1136239445000);
	testParse("Mon Jan 02 2006 15:04:05 GMT-0700 (GMT Standard Time)",	1136239445000);
	testParse("Mon Jan 2 15:04:05 MST 2006",							1136239445000);
	testParse("Mon Jan 02 15:04:05 MST 2006",							1136239445000);
	testParse("Mon Jan 02 15:04:05 -0700 2006",							1136239445000);

	testParse("December 04, 1986",	534038400000);
	testParse("Dec 04, 1986",		534038400000);
	testParse("Dec 4, 1986",		534038400000);

	testParse("2006-01-02T15:04:05.000Z",	1136214245000);
	testParse("2006-06-02T15:04:05.000",	1149275045000);
	testParse("2006-01-02T15:04:05",		1136232245000);
	testParse("2006-01-02",					1136160000000);
	testParse("2006T15:04-0700",			1136153040000);
	testParse("2006T15:04Z",				1136127840000);
	testParse("2019-01-01T12:00:00.52Z",	1546344000520);

	var d = new Date("Mon, 02 Jan 2006 15:04:05 MST");

	assert.sameValue(d.getUTCHours(), 22,
					"new Date(\"Mon, 02 Jan 2006 15:04:05 MST\").getUTCHours()");

	assert.sameValue(d.getHours(), 17,
					"new Date(\"Mon, 02 Jan 2006 15:04:05 MST\").getHours()");

	assert.sameValue(Date.parse("Mon, 02 Jan 2006 15:04:05 zzz"), NaN,
					 "Date.parse(\"Mon, 02 Jan 2006 15:04:05 zzz\")");

	assert.sameValue(Date.parse("Mon, 02 Jan 2006 15:04:05 ZZZ"), NaN,
					 "Date.parse(\"Mon, 02 Jan 2006 15:04:05 ZZZ\")");

	var minDateStr = "-271821-04-20T00:00:00.000Z";
	var minDate = new Date(-8640000000000000);

	assert.sameValue(minDate.toISOString(), minDateStr, "minDateStr");
	assert.sameValue(Date.parse(minDateStr), minDate.valueOf(), "parse minDateStr");

	var maxDateStr = "+275760-09-13T00:00:00.000Z";
	var maxDate = new Date(8640000000000000);

	assert.sameValue(maxDate.toISOString(), maxDateStr, "maxDateStr");
	assert.sameValue(Date.parse(maxDateStr), maxDate.valueOf(), "parse maxDateStr");

	var belowRange = "-271821-04-19T23:59:59.999Z";
	var aboveRange = "+275760-09-13T00:00:00.001Z";

	assert.sameValue(Date.parse(belowRange), NaN, "parse below minimum time value");
	assert.sameValue(Date.parse(aboveRange), NaN, "parse above maximum time value");
	`

	l := time.Local
	defer func() {
		time.Local = l
	}()
	var err error
	time.Local, err = time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}

	testScript1(TESTLIB+SCRIPT, _undefined, t)
}

func TestDateMaxValues(t *testing.T) {
	const SCRIPT = `
	assert.sameValue((new Date(0)).setUTCMilliseconds(8.64e15), 8.64e15);
	assert.sameValue((new Date(0)).setUTCSeconds(8640000000000), 8.64e15);
	assert.sameValue((new Date(0)).setUTCMilliseconds(-8.64e15), -8.64e15);
	assert.sameValue((new Date(0)).setUTCSeconds(-8640000000000), -8.64e15);
	`
	testScript1(TESTLIB+SCRIPT, _undefined, t)
}

func TestDateExport(t *testing.T) {
	vm := New()
	res, err := vm.RunString(`new Date(1000)`)
	if err != nil {
		t.Fatal(err)
	}
	exp := res.Export()
	if d, ok := exp.(time.Time); ok {
		if d.UnixNano()/1e6 != 1000 {
			t.Fatalf("Invalid exported date: %v", d)
		}
		if loc := d.Location(); loc != time.Local {
			t.Fatalf("Invalid timezone: %v", loc)
		}
	} else {
		t.Fatalf("Invalid export type: %T", exp)
	}
}

func TestDateToJSON(t *testing.T) {
	const SCRIPT = `
	Date.prototype.toJSON.call({ toISOString: function () { return 1; } })
	`
	testScript1(SCRIPT, intToValue(1), t)
}
