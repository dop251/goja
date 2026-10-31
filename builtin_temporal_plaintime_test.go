package goja

import "testing"

func TestTemporalPlainTimeBasics(t *testing.T) {
	const SCRIPT = `
	const t = new Temporal.PlainTime(15, 23, 30, 123, 456, 789);
	assert.sameValue(t.hour, 15);
	assert.sameValue(t.minute, 23);
	assert.sameValue(t.second, 30);
	assert.sameValue(t.millisecond, 123);
	assert.sameValue(t.microsecond, 456);
	assert.sameValue(t.nanosecond, 789);
	assert.sameValue(t.toString(), "15:23:30.123456789");
	assert.sameValue(Temporal.PlainTime.from("15:23:30.123456789").equals(t), true);
	assert.sameValue(Temporal.PlainTime.from("1976-11-18 15:23:30.123456789").equals(t), true);
	assert.sameValue(Temporal.PlainTime.from("15:23:60").second, 59);
	assert.sameValue(new Temporal.PlainTime(15, 23).toString(), "15:23:00");
	assert.throws(RangeError, () => new Temporal.PlainTime(24));
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}

func TestTemporalPlainTimeArithmetic(t *testing.T) {
	const SCRIPT = `
	assert.sameValue(new Temporal.PlainTime(23, 0).add({ hours: 2 }).toString(), "01:00:00");
	assert.sameValue(new Temporal.PlainTime(1, 0).subtract({ hours: 2 }).toString(), "23:00:00");
	assert.sameValue(new Temporal.PlainTime(0, 0).add({ days: 1 }).toString(), "00:00:00");
	const d = new Temporal.PlainTime(10, 0).until(new Temporal.PlainTime(12, 30));
	assert.sameValue(d.hours, 2);
	assert.sameValue(d.minutes, 30);
	assert.sameValue(new Temporal.PlainTime(12, 34, 56, 789, 987, 500).toString({ smallestUnit: "minute", roundingMode: "halfExpand" }), "12:35");
	assert.sameValue(new Temporal.PlainTime(12, 34, 56, 789, 987, 500).round({ smallestUnit: "second", roundingMode: "halfExpand" }).toString(), "12:34:57");
	`
	testScriptWithTestLib(SCRIPT, _undefined, t)
}
