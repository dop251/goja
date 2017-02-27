package goja

import (
	"testing"
	"time"
)

const TESTLIB = `
function $ERROR(message) {
	throw new Error(message);
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
	testScript1(SCRIPT, asciiString("Thu Jan 01 1970 00:00:00 GMT+0000 (UTC)"), t)
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
	assert.sameValue((new Date(0)).setMilliseconds(2345), 2345, "setMilliseconds()");
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
