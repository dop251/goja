package goja

import "testing"

func TestTemporalPlainYearMonth(t *testing.T) {
	const SCRIPT = `
	const ym = Temporal.PlainYearMonth.from("2019-10-31");
	assert.sameValue(ym.toString(), "2019-10");
	assert.sameValue(ym.toString({ calendarName: "always" }), "2019-10-01[u-ca=iso8601]");
	assert.sameValue(ym.year, 2019);
	assert.sameValue(ym.month, 10);
	assert.sameValue(ym.monthCode, "M10");
	assert.sameValue(ym.daysInMonth, 31);
	assert.sameValue(ym.add({ months: 3 }).toString(), "2020-01");
	assert.sameValue(ym.subtract({ years: 1 }).toString(), "2018-10");
	assert.sameValue(ym.toPlainDate({ day: 5 }).toString(), "2019-10-05");
	const d = ym.until(Temporal.PlainYearMonth.from("2021-01"), { largestUnit: "months" });
	assert.sameValue(d.months, 15);
	assert.sameValue(Temporal.PlainYearMonth.compare(ym, Temporal.PlainYearMonth.from("2020-01")), -1);
	assert.throws(RangeError, () => ym.add({ days: 1 }));
	assert.throws(RangeError, () => new Temporal.PlainYearMonth(2019, 1, "iso8601", 32));
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestTemporalPlainMonthDay(t *testing.T) {
	const SCRIPT = `
	const md = Temporal.PlainMonthDay.from("02-29");
	assert.sameValue(md.toString(), "02-29");
	assert.sameValue(md.toString({ calendarName: "always" }), "1972-02-29[u-ca=iso8601]");
	assert.sameValue(md.monthCode, "M02");
	assert.sameValue(md.day, 29);
	assert.sameValue(md.toPlainDate({ year: 2023 }).toString(), "2023-02-28");
	assert.sameValue(Temporal.PlainMonthDay.from("2019-06-15").toString(), "06-15");
	assert.sameValue(Temporal.PlainMonthDay.from({ month: 6, day: 15 }).equals(md), false);
	assert.sameValue(md.equals(Temporal.PlainMonthDay.from("02-29")), true);
	assert.throws(RangeError, () => Temporal.PlainMonthDay.from("13-01"));
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestTemporalNow(t *testing.T) {
	const SCRIPT = `
	assert(Temporal.Now.instant() instanceof Temporal.Instant);
	assert(Temporal.Now.zonedDateTimeISO("UTC") instanceof Temporal.ZonedDateTime);
	assert(Temporal.Now.plainDateTimeISO("UTC") instanceof Temporal.PlainDateTime);
	assert(Temporal.Now.plainDateISO("UTC") instanceof Temporal.PlainDate);
	assert(Temporal.Now.plainTimeISO("UTC") instanceof Temporal.PlainTime);
	assert.sameValue(Temporal.Now.timeZoneId("UTC"), "UTC");
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}
