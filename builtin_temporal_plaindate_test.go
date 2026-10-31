package goja

import "testing"

func TestTemporalPlainDateBasics(t *testing.T) {
	const SCRIPT = `
	const d = new Temporal.PlainDate(2020, 2, 29);
	assert.sameValue(d.year, 2020);
	assert.sameValue(d.month, 2);
	assert.sameValue(d.day, 29);
	assert.sameValue(d.calendarId, "iso8601");
	assert.sameValue(d.monthCode, "M02");
	assert.sameValue(d.dayOfWeek, 6);
	assert.sameValue(d.dayOfYear, 60);
	assert.sameValue(d.daysInMonth, 29);
	assert.sameValue(d.daysInYear, 366);
	assert.sameValue(d.inLeapYear, true);
	assert.sameValue(d.toString(), "2020-02-29");
	assert.sameValue(Temporal.PlainDate.from("2020-02-29").equals(d), true);
	assert.sameValue(Temporal.PlainDate.from("2020-02-29T12:34:56+01:00").equals(d), true);
	assert.throws(RangeError, () => new Temporal.PlainDate(2021, 2, 29));
	assert.throws(RangeError, () => new Temporal.PlainDate(2020, 13, 1));
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestTemporalPlainDateArithmetic(t *testing.T) {
	const SCRIPT2 = `
	assert.sameValue(Temporal.PlainDate.from("2019-01-31").add({ months: 1 }).toString(), "2019-02-28");
	assert.sameValue(Temporal.PlainDate.from("1976-11-18").add({ years: 43 }).toString(), "2019-11-18");
	assert.sameValue(Temporal.PlainDate.from("1976-11-18").add({ hours: 36 }).toString(), "1976-11-19");
	assert.sameValue(Temporal.PlainDate.from("1976-11-18").subtract({ days: 20 }).toString(), "1976-10-29");
	`
	testScriptWithTestLib(SCRIPT2, _undefined, t)

	const SCRIPT3 = `
	const a = Temporal.PlainDate.from("2021-01-31");
	const b = Temporal.PlainDate.from("2021-03-31");
	const d = a.until(b, { largestUnit: "months" });
	assert.sameValue(d.months, 2);
	assert.sameValue(d.days, 0);
	assert.sameValue(a.until(b).days, 59);
	assert.sameValue(b.since(a).days, 59);
	assert.sameValue(Temporal.PlainDate.compare(a, b), -1);
	`
	testScriptWithTestLib(SCRIPT3, _undefined, t)
}
