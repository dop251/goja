package goja

import "testing"

func TestTemporalPlainDateTimeBasics(t *testing.T) {
	const SCRIPT = `
	const dt = new Temporal.PlainDateTime(1976, 11, 18, 15, 23, 30, 123, 456, 789);
	assert.sameValue(dt.year, 1976);
	assert.sameValue(dt.month, 11);
	assert.sameValue(dt.monthCode, "M11");
	assert.sameValue(dt.day, 18);
	assert.sameValue(dt.hour, 15);
	assert.sameValue(dt.minute, 23);
	assert.sameValue(dt.second, 30);
	assert.sameValue(dt.millisecond, 123);
	assert.sameValue(dt.microsecond, 456);
	assert.sameValue(dt.nanosecond, 789);
	assert.sameValue(dt.dayOfWeek, 4);
	assert.sameValue(dt.toString(), "1976-11-18T15:23:30.123456789");
	assert.sameValue(dt.toPlainDate().toString(), "1976-11-18");
	assert.sameValue(dt.toPlainTime().toString(), "15:23:30.123456789");
	assert.sameValue(Temporal.PlainDateTime.from("1976-11-18T15:23:30.123456789").equals(dt), true);
	assert.sameValue(Temporal.PlainDateTime.from("19761118T15:23:30.123456789").equals(dt), true);
	assert.sameValue(Temporal.PlainDateTime.from(dt.toPlainDate()).equals(new Temporal.PlainDateTime(1976, 11, 18)), true);
	assert.throws(RangeError, () => new Temporal.PlainDateTime(-271821, 4, 19));
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestTemporalPlainDateTimeArithmetic(t *testing.T) {
	const SCRIPT = `
	const dt = new Temporal.PlainDateTime(2020, 1, 31, 15, 0);
	assert.sameValue(dt.add({ months: 1 }).toString(), "2020-02-29T15:00:00");
	assert.throws(RangeError, () => dt.add({ months: 1 }, { overflow: "reject" }));
	assert.sameValue(dt.subtract({ months: 1 }).toString(), "2019-12-31T15:00:00");
	assert.sameValue(new Temporal.PlainDateTime(2000, 5, 2).subtract(new Temporal.Duration(0, 0, 0, 0, 0, 0, 0, 0, 0, 2)).toString(), "2000-05-01T23:59:59.999999998");
	assert.sameValue(dt.add({ days: 1, hours: 9 }).toString(), "2020-02-02T00:00:00");
	assert.sameValue(dt.with({ month: 2, day: 15 }).toString(), "2020-02-15T15:00:00");
	assert.sameValue(dt.withPlainTime("12:34").toString(), "2020-01-31T12:34:00");
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestTemporalPlainDateTimeDifference(t *testing.T) {
	const SCRIPT = `
	const a = new Temporal.PlainDateTime(2019, 1, 8, 8, 22, 36, 123, 456, 789);
	const b = new Temporal.PlainDateTime(2021, 9, 7, 12, 39, 40, 987, 654, 289);
	let d = a.until(b, { smallestUnit: "years", roundingMode: "trunc" });
	assert.sameValue(d.years, 2);
	assert.sameValue(d.months, 0);
	d = a.until(b, { largestUnit: "days", smallestUnit: "hours", roundingMode: "trunc" });
	assert.sameValue(d.days, 973);
	assert.sameValue(d.hours, 4);
	assert.sameValue(d.minutes, 0);
	d = b.since(a, { largestUnit: "months" });
	assert.sameValue(d.months, 31);
	assert.sameValue(d.sign, 1);
	d = a.since(b, { largestUnit: "months" });
	assert.sameValue(d.months, -31);
	d = a.until(b, { largestUnit: "hours" });
	assert.sameValue(d.hours > 20000, true);
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestTemporalPlainDateTimeRound(t *testing.T) {
	const SCRIPT = `
	const dt = new Temporal.PlainDateTime(1976, 11, 18, 14, 23, 30, 123, 987, 500);
	assert.sameValue(dt.round({ smallestUnit: "day" }).toString(), "1976-11-19T00:00:00");
	assert.sameValue(dt.round({ smallestUnit: "hour" }).toString(), "1976-11-18T14:00:00");
	assert.sameValue(dt.round({ smallestUnit: "minute" }).toString(), "1976-11-18T14:24:00");
	assert.sameValue(dt.round({ smallestUnit: "second" }).toString(), "1976-11-18T14:23:30");
	assert.sameValue(dt.round({ smallestUnit: "millisecond" }).toString(), "1976-11-18T14:23:30.124");
	assert.sameValue(new Temporal.PlainDateTime(-99, 12, 15, 12, 0, 0, 500).round({ smallestUnit: "second", roundingMode: "trunc" }).toString(), "-000099-12-15T12:00:00");
	assert.sameValue(dt.toString({ smallestUnit: "minute" }), "1976-11-18T14:23");
	assert.sameValue(dt.toString({ smallestUnit: "microsecond" }), "1976-11-18T14:23:30.123987");
	assert.throws(RangeError, () => dt.round({ smallestUnit: "month" }));
	assert.throws(TypeError, () => dt.round());
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}
